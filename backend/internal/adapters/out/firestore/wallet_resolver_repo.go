// backend/internal/adapters/out/firestore/wallet_resolver_repo.go
package firestore

import (
	"context"
	"errors"

	applicationport "narratives/internal/application/port"
	branddom "narratives/internal/domain/brand"
	walletdom "narratives/internal/domain/wallet"
)

var (
	ErrWalletResolverNotConfigured = errors.New("wallet_resolver_repo: not configured")
)

// WalletResolverRepoFS provides BOTH:
// - applicationport.BrandWalletResolver
// - applicationport.AvatarWalletResolver
//
// Brand: brands/{brandId}.walletAddress (via BrandRepositoryFS)
// Avatar: wallets/{avatarId}.walletAddress (via WalletRepositoryFS)
type WalletResolverRepoFS struct {
	BrandRepo  *BrandRepositoryFS
	WalletRepo *WalletRepositoryFS
}

func NewWalletResolverRepoFS(
	brandRepo *BrandRepositoryFS,
	walletRepo *WalletRepositoryFS,
) *WalletResolverRepoFS {
	return &WalletResolverRepoFS{
		BrandRepo:  brandRepo,
		WalletRepo: walletRepo,
	}
}

// Compile-time interface checks.
var _ applicationport.BrandWalletResolver = (*WalletResolverRepoFS)(nil)
var _ applicationport.AvatarWalletResolver = (*WalletResolverRepoFS)(nil)

// ResolveBrandWalletAddress implements applicationport.BrandWalletResolver.
func (r *WalletResolverRepoFS) ResolveBrandWalletAddress(ctx context.Context, brandID string) (string, error) {
	if r == nil || r.BrandRepo == nil {
		return "", ErrWalletResolverNotConfigured
	}

	id := brandID
	if id == "" {
		return "", branddom.ErrNotFound
	}

	b, err := r.BrandRepo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}

	return b.WalletAddress, nil
}

// ResolveAvatarWalletAddress implements applicationport.AvatarWalletResolver.
func (r *WalletResolverRepoFS) ResolveAvatarWalletAddress(ctx context.Context, avatarID string) (string, error) {
	if r == nil || r.WalletRepo == nil {
		return "", ErrWalletResolverNotConfigured
	}

	id := avatarID
	if id == "" {
		return "", ErrInvalidAvatarID // from wallet_repository_fs.go
	}

	w, err := r.WalletRepo.GetByAvatarID(ctx, id)
	if err != nil {
		return "", err
	}

	addr := w.WalletAddress
	if addr == "" {
		return "", walletdom.ErrInvalidWalletAddress
	}

	return addr, nil
}
