// backend/internal/infra/solana/rpc_client.go
package solana

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// SPL Token Program ID (Tokenkeg...)
const TokenProgramID = "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA"

// RPCClient defines the minimal Solana RPC methods we need.
// You can extend this interface later as needed.
type RPCClient interface {
	// GetTokenAccountsByOwner calls `getTokenAccountsByOwner` with:
	// params: [owner, {"programId": programID}, {"encoding":"jsonParsed","commitment":"finalized"}]
	GetTokenAccountsByOwner(ctx context.Context, owner string, programID string) (GetTokenAccountsByOwnerResult, error)
}

// JSONRPCClient is a simple HTTP JSON-RPC client for Solana.
type JSONRPCClient struct {
	Endpoint string
	HTTP     *http.Client
}

// solanaRPCURLFromEnv returns the Solana RPC URL used by all Solana clients.
//
// 本番事故防止のため、public devnet endpoint への暗黙fallbackは行いません。
// ローカル開発でも SOLANA_RPC_URL を明示してください。
func solanaRPCURLFromEnv() (string, error) {
	ep := strings.TrimSpace(os.Getenv("SOLANA_RPC_URL"))
	if ep == "" {
		return "", fmt.Errorf("SOLANA_RPC_URL is not set")
	}

	return ep, nil
}

// NewJSONRPCClient creates a Solana JSON-RPC client.
// Endpoint resolution order:
// 1) SOLANA_RPC_URL env
// 2) error
func NewJSONRPCClient() *JSONRPCClient {
	ep, _ := solanaRPCURLFromEnv()
	return NewJSONRPCClientWithEndpoint(ep)
}

// NewJSONRPCClientWithEndpoint creates a Solana JSON-RPC client with a specific endpoint.
func NewJSONRPCClientWithEndpoint(endpoint string) *JSONRPCClient {
	return &JSONRPCClient{
		Endpoint: strings.TrimSpace(endpoint),
		HTTP: &http.Client{
			Timeout: 12 * time.Second,
		},
	}
}

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type solanaRPCError struct {
	Method  string
	Code    int
	Message string
}

func (e *solanaRPCError) Error() string {
	if e == nil {
		return ""
	}

	return fmt.Sprintf(
		"solana rpc: method=%s error code=%d message=%s",
		e.Method,
		e.Code,
		e.Message,
	)
}

func (e *solanaRPCError) retryable() bool {
	if e == nil {
		return false
	}

	switch e.Code {
	case -32603:
		// JSON-RPC Internal error.
		// devnet RPC / faucet の一時障害として bounded retry する。
		return true
	default:
		return false
	}
}

type solanaHTTPStatusError struct {
	StatusCode int
	Body       string
}

func (e *solanaHTTPStatusError) Error() string {
	if e == nil {
		return ""
	}
	if e.Body == "" {
		return fmt.Sprintf("solana rpc: http status=%d", e.StatusCode)
	}
	return fmt.Sprintf("solana rpc: http status=%d body=%s", e.StatusCode, e.Body)
}

func (e *solanaHTTPStatusError) retryable() bool {
	if e == nil {
		return false
	}

	return e.StatusCode == http.StatusTooManyRequests ||
		e.StatusCode == http.StatusInternalServerError ||
		e.StatusCode == http.StatusBadGateway ||
		e.StatusCode == http.StatusServiceUnavailable ||
		e.StatusCode == http.StatusGatewayTimeout
}

func isRetryableSolanaError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var rpcErr *solanaRPCError
	if errors.As(err, &rpcErr) && rpcErr.retryable() {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	var statusErr *solanaHTTPStatusError
	if errors.As(err, &statusErr) && statusErr.retryable() {
		return true
	}

	msg := strings.ToLower(err.Error())

	retryablePatterns := []string{
		"429",
		"too many requests",
		"rate limit",
		"rate limits",
		"connection rate limits exceeded",
		"connection reset",
		"connection refused",
		"connection timed out",
		"i/o timeout",
		"temporary failure",
		"timeout",
		"eof",
		"status=500",
		"status=502",
		"status=503",
		"status=504",
	}

	for _, p := range retryablePatterns {
		if strings.Contains(msg, p) {
			return true
		}
	}

	return false
}

func withSolanaRPCRetry(ctx context.Context, operation string, fn func() error) error {
	if fn == nil {
		return fmt.Errorf("solana rpc: retry operation %s has nil function", operation)
	}

	delays := []time.Duration{
		500 * time.Millisecond,
		1 * time.Second,
		2 * time.Second,
		4 * time.Second,
		8 * time.Second,
	}

	var lastErr error

	for attempt := 0; attempt <= len(delays); attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		if !isRetryableSolanaError(err) {
			return fmt.Errorf("solana rpc %s failed: %w", operation, err)
		}

		if attempt == len(delays) {
			break
		}

		delay := delays[attempt]

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}

	return fmt.Errorf("solana rpc %s failed after retries: %w", operation, lastErr)
}

func (c *JSONRPCClient) call(ctx context.Context, method string, params any, out any) error {
	if c == nil || c.Endpoint == "" || c.HTTP == nil {
		return fmt.Errorf("solana rpc: client not configured")
	}

	reqBody, err := json.Marshal(rpcRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	})
	if err != nil {
		return fmt.Errorf("solana rpc: marshal request method=%s: %w", method, err)
	}

	return c.callRawOnce(ctx, method, reqBody, out)
}

func (c *JSONRPCClient) callRawOnce(ctx context.Context, method string, reqBody []byte, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("solana rpc: new request method=%s: %w", method, err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("solana rpc: http do method=%s: %w", method, err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &solanaHTTPStatusError{
			StatusCode: resp.StatusCode,
			Body:       string(bodyBytes),
		}
	}

	var rr rpcResponse
	if err := json.Unmarshal(bodyBytes, &rr); err != nil {
		return fmt.Errorf("solana rpc: decode response method=%s: %w", method, err)
	}

	if rr.Error != nil {
		return &solanaRPCError{
			Method:  method,
			Code:    rr.Error.Code,
			Message: rr.Error.Message,
		}
	}

	if out != nil {
		if len(rr.Result) == 0 {
			return fmt.Errorf("solana rpc: method=%s returned empty result", method)
		}
		if err := json.Unmarshal(rr.Result, out); err != nil {
			return fmt.Errorf("solana rpc: unmarshal result method=%s: %w", method, err)
		}
	}

	return nil
}

// GetTokenAccountsByOwnerResult is the decoded `result` object for getTokenAccountsByOwner (jsonParsed).
type GetTokenAccountsByOwnerResult struct {
	Context struct {
		Slot uint64 `json:"slot"`
	} `json:"context"`
	Value []struct {
		Pubkey  string `json:"pubkey"`
		Account struct {
			Data struct {
				Program string `json:"program"`
				Parsed  struct {
					Info struct {
						Mint        string `json:"mint"`
						Owner       string `json:"owner"`
						TokenAmount struct {
							Amount   string `json:"amount"`
							Decimals int    `json:"decimals"`
						} `json:"tokenAmount"`
					} `json:"info"`
					Type string `json:"type"`
				} `json:"parsed"`
				Space uint64 `json:"space"`
			} `json:"data"`
			Owner string `json:"owner"`
		} `json:"account"`
	} `json:"value"`
}

func (c *JSONRPCClient) GetTokenAccountsByOwner(ctx context.Context, owner string, programID string) (GetTokenAccountsByOwnerResult, error) {
	var out GetTokenAccountsByOwnerResult

	if owner == "" {
		return out, fmt.Errorf("solana rpc: owner is empty")
	}

	if programID == "" {
		programID = TokenProgramID
	}

	params := []any{
		owner,
		map[string]any{
			"programId": programID,
		},
		map[string]any{
			"commitment": "finalized",
			"encoding":   "jsonParsed",
		},
	}

	if err := withSolanaRPCRetry(ctx, "getTokenAccountsByOwner", func() error {
		return c.call(ctx, "getTokenAccountsByOwner", params, &out)
	}); err != nil {
		return GetTokenAccountsByOwnerResult{}, err
	}

	return out, nil
}

type getSignatureStatusesResult struct {
	Context struct {
		Slot uint64 `json:"slot"`
	} `json:"context"`
	Value []struct {
		Slot               uint64 `json:"slot"`
		Confirmations      *int   `json:"confirmations"`
		Err                any    `json:"err"`
		ConfirmationStatus string `json:"confirmationStatus"`
	} `json:"value"`
}

func waitForSignatureConfirmed(ctx context.Context, endpoint string, signature string) error {
	if signature == "" {
		return fmt.Errorf("signature is empty")
	}

	rpcClient := NewJSONRPCClientWithEndpoint(endpoint)
	if rpcClient == nil || rpcClient.Endpoint == "" {
		return fmt.Errorf("solana rpc client is not configured")
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		var out getSignatureStatusesResult
		params := []any{
			[]string{signature},
			map[string]any{
				"searchTransactionHistory": true,
			},
		}

		err := rpcClient.call(ctx, "getSignatureStatuses", params, &out)
		if err == nil {
			if len(out.Value) > 0 {
				status := out.Value[0]

				if status.Err != nil {
					errBytes, _ := json.Marshal(status.Err)
					return fmt.Errorf("transaction failed on chain: %s", string(errBytes))
				}

				switch status.ConfirmationStatus {
				case "confirmed", "finalized":
					return nil
				}
			}
		} else if !isRetryableSolanaError(err) {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
