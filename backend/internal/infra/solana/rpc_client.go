// backend\internal\infra\solana\rpc_client.go
package solana

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// Solana Devnet RPC endpoint (default)
const DevnetEndpoint = "https://api.devnet.solana.com" // :contentReference[oaicite:1]{index=1}

// SPL Token Program ID (Tokenkeg...)
const TokenProgramID = "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA" // :contentReference[oaicite:2]{index=2}

// RPCClient defines the minimal Solana RPC methods we need.
// (You can extend this interface later as needed.)
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

// NewJSONRPCClient creates a Solana JSON-RPC client.
// Endpoint resolution order:
// 1) SOLANA_RPC_ENDPOINT env (if set)
// 2) DevnetEndpoint (default)
func NewJSONRPCClient() *JSONRPCClient {
	ep := os.Getenv("SOLANA_RPC_ENDPOINT")
	if ep == "" {
		ep = DevnetEndpoint
	}
	return &JSONRPCClient{
		Endpoint: ep,
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
		return fmt.Errorf("solana rpc: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("solana rpc: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("solana rpc: http do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("solana rpc: http status=%d", resp.StatusCode)
	}

	var rr rpcResponse
	if err := json.NewDecoder(resp.Body).Decode(&rr); err != nil {
		return fmt.Errorf("solana rpc: decode response: %w", err)
	}
	if rr.Error != nil {
		return fmt.Errorf("solana rpc: error code=%d message=%s", rr.Error.Code, rr.Error.Message)
	}

	if out != nil {
		if err := json.Unmarshal(rr.Result, out); err != nil {
			return fmt.Errorf("solana rpc: unmarshal result: %w", err)
		}
	}
	return nil
}

// GetTokenAccountsByOwnerResult is the decoded `result` object for getTokenAccountsByOwner (jsonParsed).
// Based on Solana RPC docs response shape. :contentReference[oaicite:3]{index=3}
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
							Amount   string `json:"amount"`   // string integer
							Decimals int    `json:"decimals"` // for UI conversion
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

	owner = stringsTrim(owner)
	if owner == "" {
		return out, fmt.Errorf("solana rpc: owner is empty")
	}
	if programID == "" {
		programID = TokenProgramID
	}

	params := []any{
		owner,
		map[string]any{
			"programId": programID, // Tokenkeg... :contentReference[oaicite:4]{index=4}
		},
		map[string]any{
			"commitment": "finalized",
			"encoding":   "jsonParsed", // :contentReference[oaicite:5]{index=5}
		},
	}

	if err := c.call(ctx, "getTokenAccountsByOwner", params, &out); err != nil {
		return GetTokenAccountsByOwnerResult{}, err
	}
	return out, nil
}

func stringsTrim(s string) string {
	// tiny helper to avoid importing strings everywhere in this file
	// (feel free to replace with strings.TrimSpace)
	n := len(s)
	i := 0
	for i < n && (s[i] == ' ' || s[i] == '\n' || s[i] == '\t' || s[i] == '\r') {
		i++
	}
	j := n - 1
	for j >= i && (s[j] == ' ' || s[j] == '\n' || s[j] == '\t' || s[j] == '\r') {
		j--
	}
	if i > j {
		return ""
	}
	return s[i : j+1]
}
