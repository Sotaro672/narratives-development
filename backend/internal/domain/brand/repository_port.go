// backend/internal/domain/brand/repository_port.go
package brand

import (
	"context"
	"errors"
	"time"

	common "narratives/internal/domain/common"
)

// ========================================
// Ports (RepositoryPort: query-friendly repository interface)
// ========================================

// RepositoryPort defines the data access interface for brand domain.
type RepositoryPort interface {
	ListByCompanyID(ctx context.Context, companyID string, page Page) (PageResult[Brand], error)
	GetByID(ctx context.Context, id string) (Brand, error)
	Create(ctx context.Context, b Brand) (Brand, error)
	Update(ctx context.Context, id string, patch BrandPatch) (Brand, error)
	Delete(ctx context.Context, id string) error
}

// ========================================
// Filter (align to common.FilterCommon / common.TimeRange)
// ========================================

// Filter は一覧取得条件です。
// common.FilterCommon を内包して SearchQuery/Created/Updated の表現を統一します。
// Brand 固有条件はこの struct に追加します。
type Filter struct {
	common.FilterCommon

	CompanyID     *string
	CompanyIDs    []string
	ManagerID     *string
	ManagerIDs    []string
	IsActive      *bool
	WalletAddress *string

	// Brand 固有: deleted の期間やフラグは必要なら残す
	DeletedFrom *time.Time
	DeletedTo   *time.Time
	Deleted     *bool
}

// 共通型エイリアス（インフラ非依存）
type Page = common.Page
type PageResult[T any] = common.PageResult[T]

// 代表エラー（契約）
var (
	ErrNotFound = errors.New("brand: not found")
	ErrConflict = errors.New("brand: conflict")

	// ErrInvalidID は entity.go 側で定義済みのものを利用する
)

// ========================================
// Brand summary value objects
// ========================================

type NameIcon struct {
	Name      string `json:"name"`
	BrandIcon string `json:"brandIcon"`
}

type NameIconBackground struct {
	Name                 string `json:"name"`
	BrandIcon            string `json:"brandIcon"`
	BrandBackgroundImage string `json:"brandBackgroundImage"`
}

type BrandProfile struct {
	Name                 string `json:"name"`
	Description          string `json:"description"`
	URL                  string `json:"websiteUrl"`
	BrandIcon            string `json:"brandIcon"`
	BrandBackgroundImage string `json:"brandBackgroundImage"`
	CompanyID            string `json:"companyId"`
}

// ========================================
// Port (Repository)
// ========================================

type Repository interface {
	ListByCompanyID(ctx context.Context, companyID string, page Page) (PageResult[Brand], error)
	GetByID(ctx context.Context, id string) (Brand, error)

	Create(ctx context.Context, b Brand) (Brand, error)
	Update(ctx context.Context, id string, patch BrandPatch) (Brand, error)
	Delete(ctx context.Context, id string) error
}

// ========================================
// Solana Brand Wallet 関連 (Domain メタ情報)
// ========================================

type SolanaBrandWallet struct {
	BrandID    string
	Address    string
	SecretName string
}

type MintAuthorityKey struct {
	Address    string
	SecretName string
}

type SolanaBrandWalletService interface {
	OpenBrandWallet(ctx context.Context, b Brand) (SolanaBrandWallet, error)
	FreezeBrandWallet(ctx context.Context, wallet SolanaBrandWallet) error
	DelegateTokenOperation(
		ctx context.Context,
		brandWallet SolanaBrandWallet,
		master MintAuthorityKey,
	) error
}
