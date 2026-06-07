// backend/internal/domain/wallet/repository_port.go
package wallet

import "context"

// Repository は Wallet 集約の永続化ポートです。
//
// Collection design:
// - collection: wallets
// - docId: avatarId
// - fields: walletAddress, tokens, lastUpdatedAt, status
// - avatarId field is NOT stored. docId is the source of truth.
type Repository interface {
	// GetByAvatarID は wallets/{avatarId} から Wallet を取得します。
	GetByAvatarID(ctx context.Context, avatarID string) (Wallet, error)

	// Save は wallets/{avatarId} に Wallet を保存します。
	Save(ctx context.Context, avatarID string, w Wallet) error
}

// OnchainReader は Solana 上のウォレットが保持するトークンの mint 一覧を取得するポートです。
type OnchainReader interface {
	ListOwnedTokenMints(ctx context.Context, walletAddress string) ([]string, error)
}
