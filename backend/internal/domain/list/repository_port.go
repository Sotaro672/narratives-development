package list

import (
	"context"
	"errors"
	"time"

	common "narratives/internal/domain/common"
)

// Patch（部分更新）: nil のフィールドは更新しない
type ListPatch struct {
	Status     *ListStatus
	AssigneeID *string
	Title      *string
	ImageID    *string // ListImage.id を指す

	Description *string
	Prices      *map[string]ListPrice // key = inventoryId

	UpdatedAt *time.Time
	UpdatedBy *string
	DeletedAt *time.Time
	DeletedBy *string
}

// フィルタ/検索条件（実装側で適宜解釈）
type Filter struct {
	// フリーテキスト（id, title, description 等の部分一致などは実装側で解釈）
	SearchQuery string

	// 絞り込み
	IDs        []string
	AssigneeID *string
	Status     *ListStatus
	Statuses   []ListStatus

	// 価格条件（Prices[inventoryId].Price に対する閾値）
	// NOTE: 旧命名の互換のため ModelNumbers を残しているが、
	//       実際の意味は「Prices の key（= inventoryId）」として扱う。
	ModelNumbers []string
	MinPrice     *int
	MaxPrice     *int

	// 日付レンジ
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	UpdatedFrom *time.Time
	UpdatedTo   *time.Time
	DeletedFrom *time.Time
	DeletedTo   *time.Time

	// 論理削除の tri-state（nil: 全件 / true: 削除済のみ / false: 未削除のみ）
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

// 契約上の代表的エラー
var (
	ErrNotFound = errors.New("list: not found")
	ErrConflict = errors.New("list: conflict")
)

// Repository ポート（契約）
type Repository interface {
	// 一覧取得
	List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult[List], error)
	ListByCursor(ctx context.Context, filter Filter, sort Sort, cpage CursorPage) (CursorPageResult[List], error)
	Count(ctx context.Context, filter Filter) (int, error)

	// 取得
	GetByID(ctx context.Context, id string) (List, error)
	Exists(ctx context.Context, id string) (bool, error)

	// 変更
	Create(ctx context.Context, l List) (List, error)
	Update(ctx context.Context, id string, patch ListPatch) (List, error)
	Delete(ctx context.Context, id string) error

	// 任意: Upsert 等
	Save(ctx context.Context, l List, opts *SaveOptions) (List, error)
}
