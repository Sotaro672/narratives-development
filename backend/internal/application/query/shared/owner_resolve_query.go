// backend/internal/application/query/shared/owner_resolve_query.go
package shared

import (
	"context"
	"errors"
	"strings"
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

// AvatarNameReader resolves avatarName by avatarId.
// ✅ brand.Service.GetNameByID と同じ思想で導入する port。
type AvatarNameReader interface {
	GetNameByID(ctx context.Context, avatarID string) (string, error)
}

// BrandNameReader resolves brandName by brandId.
// ✅ brand.Service.GetNameByID と同じ思想で導入する port。
type BrandNameReader interface {
	GetNameByID(ctx context.Context, brandID string) (string, error)
}

// ------------------------------------------------------------
// DTO
// ------------------------------------------------------------

// OwnerType describes which entity matched the walletAddress.
type OwnerType string

const (
	OwnerTypeUnknown OwnerType = "unknown"
	OwnerTypeAvatar  OwnerType = "avatar"
	OwnerTypeBrand   OwnerType = "brand"
)

// OwnerResolveResult is the unified response for "who owns this wallet address?".
type OwnerResolveResult struct {
	WalletAddress string    `json:"walletAddress"`
	OwnerType     OwnerType `json:"ownerType"`

	// Only one of them is expected to be set.
	BrandID  string `json:"brandId,omitempty"`
	AvatarID string `json:"avatarId,omitempty"`

	// ✅ NEW: resolved display names (non-fatal if empty)
	BrandName  string `json:"brandName,omitempty"`
	AvatarName string `json:"avatarName,omitempty"`
}

// ------------------------------------------------------------
// Query
// ------------------------------------------------------------

// OwnerResolveQuery resolves (brandId or avatarId) from a wallet address.
// ✅ 方針:
// - 既に購入済み（tokens/{productId}.toAddress が buyer avatar wallet に更新済み）なら avatarId がヒット
// - まだ誰にも購入されていない在庫（toAddress が brand wallet のまま）なら brandId がヒット
//
// NOTE:
// - 競合した場合の優先順位は avatar を優先（購入済みの解決を優先）。
type OwnerResolveQuery struct {
	AvatarRepo AvatarWalletAddressReader
	BrandRepo  BrandWalletAddressReader

	// ✅ NEW: ID -> Name（nil 許容 / Resolve は継続）
	AvatarName AvatarNameReader
	BrandName  BrandNameReader
}

// NewOwnerResolveQuery constructs OwnerResolveQuery.
// AvatarRepo / BrandRepo はどちらも nil 許容だが、Resolve には最低1つ必要。
// AvatarName / BrandName は nil でも Resolve は動作する（名前は埋めない）。
func NewOwnerResolveQuery(
	avatarRepo AvatarWalletAddressReader,
	brandRepo BrandWalletAddressReader,
	avatarName AvatarNameReader,
	brandName BrandNameReader,
) *OwnerResolveQuery {
	return &OwnerResolveQuery{
		AvatarRepo: avatarRepo,
		BrandRepo:  brandRepo,
		AvatarName: avatarName,
		BrandName:  brandName,
	}
}

// Resolve resolves owner by wallet address.
// - avatar が見つかれば avatar を返す（+ 可能なら avatarName も埋める）
// - 見つからなければ brand を返す（+ 可能なら brandName も埋める）
// - どちらも見つからなければ ErrOwnerNotFound
func (q *OwnerResolveQuery) Resolve(
	ctx context.Context,
	walletAddress string,
) (*OwnerResolveResult, error) {
	if q == nil || (q.AvatarRepo == nil && q.BrandRepo == nil) {
		return nil, ErrOwnerResolveNotConfigured
	}

	addr := strings.TrimSpace(walletAddress)
	if !looksLikeSolanaAddress(addr) {
		return nil, ErrInvalidWalletAddress
	}

	// 1) avatar 優先（購入済みのケース）
	if q.AvatarRepo != nil {
		avatarID, err := q.AvatarRepo.FindAvatarIDByWalletAddress(ctx, addr)
		if err != nil {
			return nil, err
		}
		avatarID = strings.TrimSpace(avatarID)
		if avatarID != "" {
			res := &OwnerResolveResult{
				WalletAddress: addr,
				OwnerType:     OwnerTypeAvatar,
				AvatarID:      avatarID,
			}

			// ✅ optional: avatarId -> avatarName
			if q.AvatarName != nil {
				if name, err := q.AvatarName.GetNameByID(ctx, avatarID); err == nil {
					res.AvatarName = strings.TrimSpace(name)
				}
			}

			return res, nil
		}
	}

	// 2) brand（未購入在庫のケース）
	if q.BrandRepo != nil {
		brandID, err := q.BrandRepo.FindBrandIDByWalletAddress(ctx, addr)
		if err != nil {
			return nil, err
		}
		brandID = strings.TrimSpace(brandID)
		if brandID != "" {
			res := &OwnerResolveResult{
				WalletAddress: addr,
				OwnerType:     OwnerTypeBrand,
				BrandID:       brandID,
			}

			// ✅ optional: brandId -> brandName
			if q.BrandName != nil {
				if name, err := q.BrandName.GetNameByID(ctx, brandID); err == nil {
					res.BrandName = strings.TrimSpace(name)
				}
			}

			return res, nil
		}
	}

	return nil, ErrOwnerNotFound
}

// ResolveIDs is a compatibility helper if you only need IDs.
// Returns (brandId, avatarId, ownerType, error).
func (q *OwnerResolveQuery) ResolveIDs(
	ctx context.Context,
	walletAddress string,
) (brandID string, avatarID string, ownerType OwnerType, err error) {
	r, err := q.Resolve(ctx, walletAddress)
	if err != nil {
		return "", "", OwnerTypeUnknown, err
	}
	return r.BrandID, r.AvatarID, r.OwnerType, nil
}

// ------------------------------------------------------------
// Helpers
// ------------------------------------------------------------

// looksLikeSolanaAddress performs a light validation for Solana base58 public key.
// - 空/空白は NG
// - 長さはざっくり 32〜64
// - base58 文字だけ許容（0,O,I,l を除外）
func looksLikeSolanaAddress(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	// Solana pubkey は通常 32 bytes -> base58 で 32〜44 文字程度。
	// 将来の拡張を踏まえゆるめに。
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
	// Bitcoin base58 alphabet
	// 123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz
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
