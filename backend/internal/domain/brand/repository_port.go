package brand

import (
	"context"
	"errors"
	"time"

	common "narratives/internal/domain/common"
)

// RepositoryPort defines the data access interface for brand domain (query-friendly).
type RepositoryPort interface {
	List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult[Brand], error)
	ListByCursor(ctx context.Context, filter Filter, sort Sort, cpage CursorPage) (CursorPageResult[Brand], error)
	GetByID(ctx context.Context, id string) (Brand, error)
	Exists(ctx context.Context, id string) (bool, error)
	Count(ctx context.Context, filter Filter) (int, error)
	Create(ctx context.Context, b Brand) (Brand, error)
	Update(ctx context.Context, id string, patch BrandPatch) (Brand, error)
	Delete(ctx context.Context, id string) error
	Save(ctx context.Context, b Brand, opts *SaveOptions) (Brand, error)
}

// Filter / Sort / Page 構造体（一覧取得用）
type Filter struct {
	// フリーテキスト検索（name, description, websiteUrl など実装側で解釈）
	SearchQuery string

	// 絞り込み
	CompanyID     *string
	CompanyIDs    []string
	ManagerID     *string
	ManagerIDs    []string
	IsActive      *bool
	WalletAddress *string

	// 期間
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	UpdatedFrom *time.Time
	UpdatedTo   *time.Time
	DeletedFrom *time.Time
	DeletedTo   *time.Time

	// 論理削除の tri-state
	// nil: 全件 / true: 削除済のみ / false: 未削除のみ
	Deleted *bool
}

// 共通型エイリアス（インフラ非依存）
type Sort = common.Sort
type SortOrder = common.SortOrder
type Page = common.Page
type PageResult[T any] = common.PageResult[T]
type CursorPage = common.CursorPage
type CursorPageResult[T any] = common.CursorPageResult[T]
type SaveOptions = common.SaveOptions

const (
	SortAsc  = common.SortAsc
	SortDesc = common.SortDesc
)

// 代表エラー（契約）
var (
	ErrNotFound = errors.New("brand: not found")
	ErrConflict = errors.New("brand: conflict")
)

// ========================================
// Port (Repository)
// ========================================

type Repository interface {
	// 一覧取得
	List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult[Brand], error)
	ListByCursor(ctx context.Context, filter Filter, sort Sort, cpage CursorPage) (CursorPageResult[Brand], error)

	// 取得
	GetByID(ctx context.Context, id string) (Brand, error)
	Exists(ctx context.Context, id string) (bool, error)
	Count(ctx context.Context, filter Filter) (int, error)

	// 変更
	Create(ctx context.Context, b Brand) (Brand, error)
	Update(ctx context.Context, id string, patch BrandPatch) (Brand, error)
	Delete(ctx context.Context, id string) error

	// 任意: Upsert 等
	Save(ctx context.Context, b Brand, opts *SaveOptions) (Brand, error)
}
