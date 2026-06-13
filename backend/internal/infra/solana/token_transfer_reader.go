// backend\internal\infra\solana\token_transfer_reader.go
package solana

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"time"
)

var (
	ErrTokenTransferReaderNotConfigured = errors.New("token_transfer_reader: not configured")
	ErrTokenTransferReaderMintEmpty     = errors.New("token_transfer_reader: mintAddress is empty")
)

const (
	defaultTokenTransferReaderLimitPerAccount = 50
	defaultTokenTransferReaderHTTPTimeout     = 20 * time.Second
)

type TokenTransferReaderSolana struct {
	RPCURL     string
	HTTPClient *http.Client
	Commitment string
	Timeout    time.Duration
}

func NewTokenTransferReaderSolana(rpcURL string) *TokenTransferReaderSolana {
	u := rpcURL
	if u == "" {
		u = os.Getenv("SOLANA_RPC_URL")
	}
	if u == "" {
		u = os.Getenv("SOLANA_RPC_ENDPOINT")
	}
	if u == "" {
		u = DevnetEndpoint
	}

	timeout := defaultTokenTransferReaderHTTPTimeout

	return &TokenTransferReaderSolana{
		RPCURL:     u,
		HTTPClient: &http.Client{Timeout: timeout},
		Commitment: "finalized",
		Timeout:    timeout,
	}
}

type ListMintTransfersInput struct {
	MintAddress     string
	LimitPerAccount int
	BeforeSignature string
	UntilSignature  string
}

type ListMintTransfersResult struct {
	MintAddress string               `json:"mintAddress"`
	Transfers   []MintTransferRecord `json:"transfers"`
}

type MintTransferRecord struct {
	FromWalletAddress string     `json:"fromWalletAddress"`
	ToWalletAddress   string     `json:"toWalletAddress"`
	TransferredAt     *time.Time `json:"transferredAt,omitempty"`
}

func (e *TokenTransferReaderSolana) ListMintTransfers(
	ctx context.Context,
	in ListMintTransfersInput,
) (ListMintTransfersResult, error) {
	if e == nil || e.RPCURL == "" {
		return ListMintTransfersResult{}, ErrTokenTransferReaderNotConfigured
	}

	mintAddress := in.MintAddress
	if mintAddress == "" {
		return ListMintTransfersResult{}, ErrTokenTransferReaderMintEmpty
	}

	limitPerAccount := in.LimitPerAccount
	if limitPerAccount <= 0 {
		limitPerAccount = defaultTokenTransferReaderLimitPerAccount
	}

	rpcCtx := ctx
	if e.Timeout > 0 {
		var cancel context.CancelFunc
		rpcCtx, cancel = context.WithTimeout(ctx, e.Timeout)
		defer cancel()
	}

	tokenAccounts, err := e.getTokenAccountsByMint(rpcCtx, mintAddress)
	if err != nil {
		return ListMintTransfersResult{}, fmt.Errorf("token_transfer_reader: getTokenAccountsByMint: %w", err)
	}

	signatureSet := map[string]struct{}{}
	signatures := make([]string, 0, len(tokenAccounts)*4)

	for _, tokenAccount := range tokenAccounts {
		sigs, err := e.getSignaturesForAddress(
			rpcCtx,
			tokenAccount,
			limitPerAccount,
			in.BeforeSignature,
			in.UntilSignature,
		)
		if err != nil {
			return ListMintTransfersResult{}, fmt.Errorf(
				"token_transfer_reader: getSignaturesForAddress(%s): %w",
				tokenAccount,
				err,
			)
		}

		for _, sig := range sigs {
			if sig == "" {
				continue
			}
			if _, ok := signatureSet[sig]; ok {
				continue
			}
			signatureSet[sig] = struct{}{}
			signatures = append(signatures, sig)
		}
	}

	type sortableTransfer struct {
		Record MintTransferRecord
		Unix   int64
	}

	records := make([]sortableTransfer, 0, len(signatures))

	for _, sig := range signatures {
		tx, err := e.getTransaction(rpcCtx, sig)
		if err != nil {
			return ListMintTransfersResult{}, fmt.Errorf("token_transfer_reader: getTransaction(%s): %w", sig, err)
		}
		if tx == nil {
			continue
		}

		items := extractMintTransferRecordsFromTransaction(tx, mintAddress)
		for _, it := range items {
			unix := int64(0)
			if it.TransferredAt != nil {
				unix = it.TransferredAt.Unix()
			}
			records = append(records, sortableTransfer{
				Record: it,
				Unix:   unix,
			})
		}
	}

	sort.SliceStable(records, func(i, j int) bool {
		if records[i].Unix != records[j].Unix {
			return records[i].Unix > records[j].Unix
		}
		if records[i].Record.FromWalletAddress != records[j].Record.FromWalletAddress {
			return records[i].Record.FromWalletAddress > records[j].Record.FromWalletAddress
		}
		return records[i].Record.ToWalletAddress > records[j].Record.ToWalletAddress
	})

	out := make([]MintTransferRecord, 0, len(records))
	for _, r := range records {
		out = append(out, r.Record)
	}

	return ListMintTransfersResult{
		MintAddress: mintAddress,
		Transfers:   out,
	}, nil
}

func (e *TokenTransferReaderSolana) getTokenAccountsByMint(ctx context.Context, mintAddress string) ([]string, error) {
	var out rpcGetProgramAccountsResponse

	if err := e.rpcCall(ctx, "getProgramAccounts", []any{
		TokenProgramID,
		map[string]any{
			"encoding":   "base64",
			"commitment": e.commitment(),
			"filters": []any{
				map[string]any{
					"dataSize": 165,
				},
				map[string]any{
					"memcmp": map[string]any{
						"offset": 0,
						"bytes":  mintAddress,
					},
				},
			},
		},
	}, &out); err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(out))
	for _, row := range out {
		if row.Pubkey == "" {
			continue
		}
		keys = append(keys, row.Pubkey)
	}

	return keys, nil
}

func (e *TokenTransferReaderSolana) getSignaturesForAddress(
	ctx context.Context,
	address string,
	limit int,
	before string,
	until string,
) ([]string, error) {
	cfg := map[string]any{
		"commitment": e.commitment(),
		"limit":      limit,
	}

	if before != "" {
		cfg["before"] = before
	}
	if until != "" {
		cfg["until"] = until
	}

	var out []rpcSignatureInfo
	if err := e.rpcCall(ctx, "getSignaturesForAddress", []any{
		address,
		cfg,
	}, &out); err != nil {
		return nil, err
	}

	res := make([]string, 0, len(out))
	for _, s := range out {
		if s.Signature == "" {
			continue
		}
		res = append(res, s.Signature)
	}

	return res, nil
}

func (e *TokenTransferReaderSolana) getTransaction(
	ctx context.Context,
	signature string,
) (*rpcTransactionResponse, error) {
	var out *rpcTransactionResponse
	if err := e.rpcCall(ctx, "getTransaction", []any{
		signature,
		map[string]any{
			"encoding":                       "jsonParsed",
			"commitment":                     e.commitment(),
			"maxSupportedTransactionVersion": 0,
		},
	}, &out); err != nil {
		return nil, err
	}

	return out, nil
}

func (e *TokenTransferReaderSolana) rpcCall(
	ctx context.Context,
	method string,
	params []any,
	out any,
) error {
	if e == nil || e.RPCURL == "" {
		return ErrTokenTransferReaderNotConfigured
	}

	reqBody := rpcRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal rpc request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.RPCURL, bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("new http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := e.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: defaultTokenTransferReaderHTTPTimeout}
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("rpc http call: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read rpc response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("rpc http status=%d body=%s", resp.StatusCode, string(respBody))
	}

	var rpcResp rpcResponse
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return fmt.Errorf("unmarshal rpc envelope: %w body=%s", err, string(respBody))
	}
	if rpcResp.Error != nil {
		return fmt.Errorf("rpc error code=%d message=%s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	if out == nil {
		return nil
	}
	if len(rpcResp.Result) == 0 || string(rpcResp.Result) == "null" {
		return nil
	}
	if err := json.Unmarshal(rpcResp.Result, out); err != nil {
		return fmt.Errorf("unmarshal rpc result: %w body=%s", err, string(respBody))
	}

	return nil
}

func (e *TokenTransferReaderSolana) commitment() string {
	c := e.Commitment
	if c == "" {
		return "finalized"
	}

	return c
}

func extractMintTransferRecordsFromTransaction(
	tx *rpcTransactionResponse,
	mintAddress string,
) []MintTransferRecord {
	if tx == nil {
		return nil
	}

	if tx.Meta != nil && tx.Meta.Err != nil {
		return nil
	}

	var transferredAt *time.Time
	if tx.BlockTime != nil {
		t := time.Unix(*tx.BlockTime, 0).UTC()
		transferredAt = &t
	}

	accountIndexToOwner := make(map[int]string)
	if tx.Meta != nil {
		for _, tb := range tx.Meta.PostTokenBalances {
			if tb.Owner != "" {
				accountIndexToOwner[tb.AccountIndex] = tb.Owner
			}
		}
		for _, tb := range tx.Meta.PreTokenBalances {
			if tb.Owner != "" {
				if _, ok := accountIndexToOwner[tb.AccountIndex]; !ok {
					accountIndexToOwner[tb.AccountIndex] = tb.Owner
				}
			}
		}
	}

	out := make([]MintTransferRecord, 0, 4)

	appendFromInstructions := func(ixs []rpcParsedInstruction) {
		for _, ix := range ixs {
			if !isSPLTokenTransferInstruction(ix) {
				continue
			}

			info, ok := ix.Parsed.Info.(map[string]any)
			if !ok {
				continue
			}

			mint := stringValue(info["mint"])
			if mint == "" {
				mint = mintAddress
			}
			if mint != mintAddress {
				continue
			}

			sourceATA := stringValue(info["source"])
			destinationATA := stringValue(info["destination"])
			if sourceATA == "" || destinationATA == "" {
				continue
			}

			fromWallet := resolveOwnerByTokenAccount(tx, sourceATA, accountIndexToOwner)
			toWallet := resolveOwnerByTokenAccount(tx, destinationATA, accountIndexToOwner)
			if fromWallet == "" || toWallet == "" {
				continue
			}

			out = append(out, MintTransferRecord{
				FromWalletAddress: fromWallet,
				ToWalletAddress:   toWallet,
				TransferredAt:     transferredAt,
			})
		}
	}

	if tx.Transaction.Message.Instructions != nil {
		appendFromInstructions(tx.Transaction.Message.Instructions)
	}
	if tx.Meta != nil {
		for _, inner := range tx.Meta.InnerInstructions {
			appendFromInstructions(inner.Instructions)
		}
	}

	return dedupeTransferRecords(out)
}

func dedupeTransferRecords(in []MintTransferRecord) []MintTransferRecord {
	if len(in) == 0 {
		return in
	}

	seen := make(map[string]struct{}, len(in))
	out := make([]MintTransferRecord, 0, len(in))

	for _, r := range in {
		ts := ""
		if r.TransferredAt != nil {
			ts = r.TransferredAt.UTC().Format(time.RFC3339)
		}
		key := r.FromWalletAddress + "|" + r.ToWalletAddress + "|" + ts
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, r)
	}

	return out
}

func isSPLTokenTransferInstruction(ix rpcParsedInstruction) bool {
	program := ix.Program
	if program != "spl-token" && ix.ProgramID != TokenProgramID {
		return false
	}

	t := ix.Parsed.Type
	return t == "transfer" || t == "transferChecked"
}

func resolveOwnerByTokenAccount(
	tx *rpcTransactionResponse,
	tokenAccount string,
	byIndex map[int]string,
) string {
	if tx == nil || tokenAccount == "" {
		return ""
	}

	keys := tx.Transaction.Message.AccountKeys
	for i, k := range keys {
		pubkey := k.Pubkey
		if pubkey == "" {
			pubkey = k.String
		}
		if pubkey != tokenAccount {
			continue
		}
		if owner, ok := byIndex[i]; ok {
			return owner
		}
		break
	}

	return ""
}

func stringValue(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case json.Number:
		return t.String()
	case float64:
		return fmt.Sprintf("%.0f", t)
	case float32:
		return fmt.Sprintf("%.0f", t)
	case int:
		return fmt.Sprintf("%d", t)
	case int64:
		return fmt.Sprintf("%d", t)
	case uint64:
		return fmt.Sprintf("%d", t)
	default:
		return ""
	}
}

type rpcGetProgramAccountsResponse []struct {
	Pubkey  string `json:"pubkey"`
	Account struct {
		Lamports   uint64 `json:"lamports"`
		Owner      string `json:"owner"`
		Executable bool   `json:"executable"`
		RentEpoch  uint64 `json:"rentEpoch"`
		Data       []any  `json:"data"`
	} `json:"account"`
}

type rpcSignatureInfo struct {
	Signature string `json:"signature"`
	Slot      uint64 `json:"slot"`
	Err       any    `json:"err"`
}

type rpcTransactionResponse struct {
	Slot      uint64 `json:"slot"`
	BlockTime *int64 `json:"blockTime"`
	Meta      *struct {
		Err               any                    `json:"err"`
		PreTokenBalances  []rpcTokenBalance      `json:"preTokenBalances"`
		PostTokenBalances []rpcTokenBalance      `json:"postTokenBalances"`
		InnerInstructions []rpcInnerInstructions `json:"innerInstructions"`
	} `json:"meta"`
	Transaction struct {
		Signatures []string `json:"signatures"`
		Message    struct {
			AccountKeys  []rpcAccountKey        `json:"accountKeys"`
			Instructions []rpcParsedInstruction `json:"instructions"`
		} `json:"message"`
	} `json:"transaction"`
}

type rpcTokenBalance struct {
	AccountIndex int    `json:"accountIndex"`
	Mint         string `json:"mint"`
	Owner        string `json:"owner"`
}

type rpcInnerInstructions struct {
	Index        int                    `json:"index"`
	Instructions []rpcParsedInstruction `json:"instructions"`
}

type rpcAccountKey struct {
	Pubkey string `json:"pubkey"`
	String string `json:"-"`
}

func (k *rpcAccountKey) UnmarshalJSON(b []byte) error {
	if len(b) == 0 {
		return nil
	}

	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		k.String = s
		k.Pubkey = s
		return nil
	}

	var v struct {
		Pubkey string `json:"pubkey"`
	}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}

	k.Pubkey = v.Pubkey
	return nil
}

type rpcParsedInstruction struct {
	Program   string `json:"program"`
	ProgramID string `json:"programId"`
	Parsed    struct {
		Type string `json:"type"`
		Info any    `json:"info"`
	} `json:"parsed"`
}
