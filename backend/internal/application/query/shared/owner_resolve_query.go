// backend/internal/application/query/shared/owner_resolve_query.go
package shared

import (
	"context"
	"errors"

	avatardom "narratives/internal/domain/avatar"
	branddom "narratives/internal/domain/brand"
)

// ------------------------------------------------------------
// Errors
// ------------------------------------------------------------

var (
	ErrOwnerResolveNotConfigured = errors.New("owner_resolve_query: not configured")
	ErrInvalidWalletAddress      = errors.New("owner_resolve_query: invalid walletAddress")
	ErrOwnerNotFound             = errors.New("owner_resolve_query: owner not found")
)

// ------------------------------------------------------------
// Ports (dependency interfaces)
// ------------------------------------------------------------

// AvatarWalletAddressReader resolves avatarId by walletAddress.
// 想定: avatars コレクションを walletAddress で検索して avatarId を返す。
// 見つからない場合は ("", nil) を返してOK。
type AvatarWalletAddressReader interface {
	FindAvatarIDByWalletAddress(ctx context.Context, walletAddress string) (string, error)
}

// BrandWalletAddressReader resolves brandId by walletAddress.
// 想定: brands コレクションを walletAddress で検索して brandId を返す。
// 見つからない場合は ("", nil) を返してOK。
type BrandWalletAddressReader interface {
	FindBrandIDByWalletAddress(ctx context.Context, walletAddress string) (string, error)
}

// AvatarReader resolves avatar by avatarId.
// avatar 側は GetByID port に統一する。
type AvatarReader interface {
	GetByID(ctx context.Context, avatarID string) (avatardom.Avatar, error)
}

// BrandReader resolves brand by brandId.
// brand.Service をそのまま注入できるように GetByID を使う。
type BrandReader interface {
	GetByID(ctx context.Context, brandID string) (branddom.Brand, error)
}

// ------------------------------------------------------------
// DTO
// ------------------------------------------------------------

// OwnerType describes which entity matched the walletAddress.
type OwnerType string

const (
	OwnerTypeAvatar OwnerType = "avatar"
	OwnerTypeBrand  OwnerType = "brand"
)

// OwnerResolveResult is the unified response for "who owns this wallet address?".
type OwnerResolveResult struct {
	WalletAddress string    `json:"walletAddress"`
	OwnerType     OwnerType `json:"ownerType"`

	// Only one of them is expected to be set.
	BrandID  string `json:"brandId,omitempty"`
	AvatarID string `json:"avatarId,omitempty"`

	// resolved display names (non-fatal if empty)
	BrandName  string `json:"brandName,omitempty"`
	AvatarName string `json:"avatarName,omitempty"`
}

// ------------------------------------------------------------
// Query
// ------------------------------------------------------------

// OwnerResolveQuery resolves brandId or avatarId from a wallet address.
//
// 方針:
// - 既に購入済み（tokens/{productId}.toAddress が buyer avatar wallet に更新済み）なら avatarId がヒット
// - まだ誰にも購入されていない在庫（toAddress が brand wallet のまま）なら brandId がヒット
// - 競合した場合は avatar を優先する
type OwnerResolveQuery struct {
	AvatarRepo AvatarWalletAddressReader
	BrandRepo  BrandWalletAddressReader

	Avatar AvatarReader
	Brand  BrandReader
}

// NewOwnerResolveQuery constructs OwnerResolveQuery.
//
// AvatarRepo / BrandRepo はどちらも nil 許容だが、Resolve には最低1つ必要。
// Avatar / Brand は nil でも Resolve は動作する（名前は埋めない）。
func NewOwnerResolveQuery(
	avatarRepo AvatarWalletAddressReader,
	brandRepo BrandWalletAddressReader,
	avatarReader AvatarReader,
	brandReader BrandReader,
) *OwnerResolveQuery {
	return &OwnerResolveQuery{
		AvatarRepo: avatarRepo,
		BrandRepo:  brandRepo,
		Avatar:     avatarReader,
		Brand:      brandReader,
	}
}

// Resolve resolves owner by wallet address.
//
// - avatar が見つかれば avatar を返す
// - 見つからなければ brand を返す
// - どちらも見つからなければ ErrOwnerNotFound
func (q *OwnerResolveQuery) Resolve(
	ctx context.Context,
	walletAddress string,
) (*OwnerResolveResult, error) {
	if q == nil || (q.AvatarRepo == nil && q.BrandRepo == nil) {
		return nil, ErrOwnerResolveNotConfigured
	}

	addr := walletAddress
	if !looksLikeSolanaAddress(addr) {
		return nil, ErrInvalidWalletAddress
	}

	if q.AvatarRepo != nil {
		avatarID, err := q.AvatarRepo.FindAvatarIDByWalletAddress(ctx, addr)
		if err != nil {
			return nil, err
		}

		if avatarID != "" {
			result := &OwnerResolveResult{
				WalletAddress: addr,
				OwnerType:     OwnerTypeAvatar,
				AvatarID:      avatarID,
			}

			if q.Avatar != nil {
				if avatar, err := q.Avatar.GetByID(ctx, avatarID); err == nil {
					result.AvatarName = avatar.AvatarName
				}
			}

			return result, nil
		}
	}

	if q.BrandRepo != nil {
		brandID, err := q.BrandRepo.FindBrandIDByWalletAddress(ctx, addr)
		if err != nil {
			return nil, err
		}

		if brandID != "" {
			result := &OwnerResolveResult{
				WalletAddress: addr,
				OwnerType:     OwnerTypeBrand,
				BrandID:       brandID,
			}

			if q.Brand != nil {
				if brand, err := q.Brand.GetByID(ctx, brandID); err == nil {
					result.BrandName = brand.Name
				}
			}

			return result, nil
		}
	}

	return nil, ErrOwnerNotFound
}

// ------------------------------------------------------------
// Helpers
// ------------------------------------------------------------

// looksLikeSolanaAddress performs a light validation for Solana base58 public key.
//
// - 空/空白は NG
// - 長さは 32〜64
// - base58 文字だけ許容（0,O,I,l を除外）
func looksLikeSolanaAddress(s string) bool {
	if s == "" {
		return false
	}

	if len(s) < 32 || len(s) > 64 {
		return false
	}

	for _, r := range s {
		if !isBase58Rune(r) {
			return false
		}
	}

	return true
}

func isBase58Rune(r rune) bool {
	switch {
	case r >= '1' && r <= '9':
		return true
	case r >= 'A' && r <= 'H':
		return true
	case r >= 'J' && r <= 'N':
		return true
	case r >= 'P' && r <= 'Z':
		return true
	case r >= 'a' && r <= 'k':
		return true
	case r >= 'm' && r <= 'z':
		return true
	default:
		return false
	}
}
