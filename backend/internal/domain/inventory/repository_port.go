package inventory

import (
	"context"
	"errors"
	"time"

	common "narratives/internal/domain/common"
)

// Patch（部分更新）: nil のフィールドは更新しない
// ConnectedToken は nil=変更なし、空文字ポインタ("")=クリア、非空ポインタ=設定 の運用を想定
type InventoryPatch struct {
	Models         *[]InventoryModel
	Location       *string
	Status         *InventoryStatus
	ConnectedToken *string

	UpdatedAt *time.Time
	UpdatedBy *string
}

// フィルタ/検索条件（実装側で適宜解釈）
type Filter struct {
	// フリーテキスト（id, location などへの部分一致等は実装側に委譲）
	SearchQuery string

	// 絞り込み
	IDs            []string
	ConnectedToken *string
	Location       *string
	Status         *InventoryStatus
	Statuses       []InventoryStatus
	CreatedBy      *string
	UpdatedBy      *string

	// 日付レンジ
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	UpdatedFrom *time.Time
	UpdatedTo   *time.Time
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

// 契約上の代表的エラー
var (
	ErrNotFound = errors.New("inventory: not found")
	ErrConflict = errors.New("inventory: conflict")
)

// Repository ポート（契約）
type Repository interface {
	// 一覧取得
	List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult[Inventory], error)
	ListByCursor(ctx context.Context, filter Filter, sort Sort, cpage CursorPage) (CursorPageResult[Inventory], error)

	// 取得
	GetByID(ctx context.Context, id string) (Inventory, error)
	Exists(ctx context.Context, id string) (bool, error)
	Count(ctx context.Context, filter Filter) (int, error)

	// 変更
	Create(ctx context.Context, inv Inventory) (Inventory, error)
	Update(ctx context.Context, id string, patch InventoryPatch) (Inventory, error)
	Delete(ctx context.Context, id string) error

	// 任意: Upsert 等
	Save(ctx context.Context, inv Inventory, opts *SaveOptions) (Inventory, error)
}
