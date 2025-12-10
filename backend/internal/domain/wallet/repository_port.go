// backend/internal/domain/wallet/repository_port.go
package wallet

import "context"

// Repository は Wallet 集約の永続化ポートです。
type Repository interface {
	GetByAddress(ctx context.Context, addr string) (Wallet, error)
	Save(ctx context.Context, w Wallet) error
}

// OnchainReader は Solana 上のウォレットが保持するトークンの mint 一覧を取得するポートです。
type OnchainReader interface {
	ListOwnedTokenMints(ctx context.Context, walletAddress string) ([]string, error)
}
