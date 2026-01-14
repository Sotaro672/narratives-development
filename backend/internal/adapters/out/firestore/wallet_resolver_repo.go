// backend/internal/adapters/out/firestore/wallet_resolver_repo.go
package firestore

import (
	"context"
	"errors"
	"strings"

	usecase "narratives/internal/application/usecase"
	branddom "narratives/internal/domain/brand"
	walletdom "narratives/internal/domain/wallet"
)

var (
	ErrWalletResolverNotConfigured = errors.New("wallet_resolver_repo: not configured")
)

// WalletResolverRepoFS provides BOTH:
// - usecase.BrandWalletResolver
// - usecase.AvatarWalletResolver
//
// ✅ Brand: brands/{brandId}.walletAddress (via BrandRepositoryFS)
// ✅ Avatar: wallets/{avatarId}.walletAddress (via WalletRepositoryFS)
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

// Compile-time interface checks
var _ usecase.BrandWalletResolver = (*WalletResolverRepoFS)(nil)
var _ usecase.AvatarWalletResolver = (*WalletResolverRepoFS)(nil)

// ResolveBrandWalletAddress implements usecase.BrandWalletResolver.
func (r *WalletResolverRepoFS) ResolveBrandWalletAddress(ctx context.Context, brandID string) (string, error) {
	if r == nil || r.BrandRepo == nil {
		return "", ErrWalletResolverNotConfigured
	}

	id := strings.TrimSpace(brandID)
	if id == "" {
		return "", branddom.ErrNotFound
	}

	b, err := r.BrandRepo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(b.WalletAddress), nil
}

// ResolveAvatarWalletAddress implements usecase.AvatarWalletResolver.
func (r *WalletResolverRepoFS) ResolveAvatarWalletAddress(ctx context.Context, avatarID string) (string, error) {
	if r == nil || r.WalletRepo == nil {
		return "", ErrWalletResolverNotConfigured
	}

	id := strings.TrimSpace(avatarID)
	if id == "" {
		return "", ErrInvalidAvatarID // from wallet_repository_fs.go
	}

	w, err := r.WalletRepo.GetByAvatarID(ctx, id)
	if err != nil {
		return "", err
	}

	addr := strings.TrimSpace(w.WalletAddress)
	if addr == "" {
		return "", walletdom.ErrInvalidWalletAddress
	}

	return addr, nil
}
