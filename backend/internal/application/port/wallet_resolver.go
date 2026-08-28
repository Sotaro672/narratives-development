// backend/internal/application/port/wallet_resolver.go
package port

import "context"

// BrandWalletResolver resolves the wallet address owned by a brand.
type BrandWalletResolver interface {
	ResolveBrandWalletAddress(
		ctx context.Context,
		brandID string,
	) (string, error)
}

// AvatarWalletResolver resolves the wallet address owned by an avatar.
type AvatarWalletResolver interface {
	ResolveAvatarWalletAddress(
		ctx context.Context,
		avatarID string,
	) (string, error)
}
