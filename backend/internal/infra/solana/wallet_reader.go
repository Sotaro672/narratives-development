// internal/platform/solana/wallet_reader.go
package solana

import (
	"context"
	// Solana RPC クライアントのライブラリをここで import
)

type RPCClient interface {
	// getTokenAccountsByOwner などを叩けるインターフェース
}

type OnchainWalletReaderImpl struct {
	Client RPCClient
}

func (r *OnchainWalletReaderImpl) ListOwnedTokenMints(ctx context.Context, walletAddress string) ([]string, error) {
	// 1. Solana RPC の getTokenAccountsByOwner を呼ぶ
	// 2. 各 token account から mint address を抜き出す
	// 3. []string として返す
	return nil, nil
}
