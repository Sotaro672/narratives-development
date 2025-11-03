package discount

import (
	"context"
	"errors"
	"time"

	common "narratives/internal/domain/common"
)

// ==============================
// Patch（部分更新）: nil のフィールドは更新しない
// ==============================
type DiscountPatch struct {
	ListID       *string
	Discounts    *[]DiscountItem
	Description  *string
	DiscountedBy *string
	DiscountedAt *time.Time
	UpdatedAt    *time.Time
	UpdatedBy    *string
}

// ==============================
// フィルタ/検索条件（実装側で適宜解釈）
// ==============================
type Filter struct {
	// フリーテキスト（id, listId, description, discountedBy, updatedBy など実装側で解釈）
	SearchQuery string

	// 絞り込み
	IDs        []string
	ListID     *string
	ListIDs    []string
	DiscountedBy *string
	UpdatedBy    *string

	// Item 条件（配列 discounts 内を実装側で解釈）
	ModelNumbers []string // 例: discounts[].modelNumber に含まれるもの
	PercentMin   *int     // 例: discounts[].discount の最小
	PercentMax   *int     // 例: discounts[].discount の最大

	// 日付レンジ
	DiscountedFrom *time.Time
	DiscountedTo   *time.Time
	UpdatedFrom    *time.Time
	UpdatedTo      *time.Time
}

// ==============================
// 共通型エイリアス（インフラ非依存）
// ==============================
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

// 代表的なエラー（契約上の表現）
var (
	ErrNotFound = errors.New("discount: not found")
	ErrConflict = errors.New("discount: conflict")
)

// ==============================
// Repository ポート（契約）
// ==============================
type Repository interface {
	// 一覧取得
	List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult[Discount], error)
	ListByCursor(ctx context.Context, filter Filter, sort Sort, cpage CursorPage) (CursorPageResult[Discount], error)

	// 取得
	GetByID(ctx context.Context, id string) (Discount, error)
	GetByListID(ctx context.Context, listID string, sort Sort, page Page) (PageResult[Discount], error)
	Exists(ctx context.Context, id string) (bool, error)
	Count(ctx context.Context, filter Filter) (int, error)

	// 変更
	Create(ctx context.Context, d Discount) (Discount, error)
	Update(ctx context.Context, id string, patch DiscountPatch) (Discount, error)
	Delete(ctx context.Context, id string) error

	// 任意: Upsert 等
	Save(ctx context.Context, d Discount, opts *SaveOptions) (Discount, error)
}
