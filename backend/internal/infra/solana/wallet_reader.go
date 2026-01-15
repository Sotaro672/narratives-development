package solana

import (
	"context"
	"fmt"
	"time"

	"strings"
)

// OnchainWalletReaderImpl implements usecase.OnchainWalletReader:
//
//	ListOwnedTokenMints(ctx, walletAddress) ([]string, error)
type OnchainWalletReaderImpl struct {
	Client RPCClient
}

// NewOnchainWalletReaderDevnet creates an onchain reader backed by Solana devnet (default).
// It honors SOLANA_RPC_ENDPOINT env for override.
func NewOnchainWalletReaderDevnet() *OnchainWalletReaderImpl {
	return &OnchainWalletReaderImpl{
		Client: NewJSONRPCClient(),
	}
}

// ListOwnedTokenMints fetches token accounts by owner from Solana RPC and returns
// a deduplicated list of mint addresses.
//
// Behavior notes:
//   - Uses getTokenAccountsByOwner with programId=Tokenkeg... and encoding=jsonParsed. :contentReference[oaicite:6]{index=6}
//   - Filters out zero-balance token accounts (Amount == "0") to match "walletが持つtoken" の直感に寄せる
//     (必要ならこの条件は外せます)
func (r *OnchainWalletReaderImpl) ListOwnedTokenMints(ctx context.Context, walletAddress string) ([]string, error) {
	if r == nil || r.Client == nil {
		return nil, fmt.Errorf("solana wallet reader: client not configured")
	}
	addr := strings.TrimSpace(walletAddress)
	if addr == "" {
		return nil, fmt.Errorf("solana wallet reader: walletAddress is empty")
	}

	// RPC call
	res, err := r.Client.GetTokenAccountsByOwner(ctx, addr, TokenProgramID)
	if err != nil {
		return nil, err
	}

	// Extract + dedup while keeping stable order
	seen := make(map[string]struct{}, len(res.Value))
	out := make([]string, 0, len(res.Value))

	for _, v := range res.Value {
		mint := strings.TrimSpace(v.Account.Data.Parsed.Info.Mint)
		amt := strings.TrimSpace(v.Account.Data.Parsed.Info.TokenAmount.Amount)

		if mint == "" {
			continue
		}
		// optional: skip zero-balance
		if amt == "0" {
			continue
		}

		if _, ok := seen[mint]; ok {
			continue
		}
		seen[mint] = struct{}{}
		out = append(out, mint)
	}

	_ = time.Now() // (keep import if you later add logging/metrics; otherwise remove)
	return out, nil
}
