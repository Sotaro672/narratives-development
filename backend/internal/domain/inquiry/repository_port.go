package inquiry

import (
	"context"
	"errors"
	"time"

	common "narratives/internal/domain/common"
)

// Patch（部分更新）: nil のフィールドは更新しない
type InquiryPatch struct {
	Subject     *string
	Content     *string
	Status      *InquiryStatus
	InquiryType *InquiryType

	ProductBlueprintID *string
	TokenBlueprintID   *string
	AssigneeID         *string
	Image              *string

	UpdatedAt *time.Time
	UpdatedBy *string
	DeletedAt *time.Time
	DeletedBy *string
}

// フィルタ/検索条件（実装側で適宜解釈）
type Filter struct {
	// フリーテキスト（subject, content などに対して部分一致など、実装側で解釈）
	SearchQuery string

	// 絞り込み
	IDs                []string
	AvatarID           *string
	AssigneeID         *string
	Status             *InquiryStatus
	Statuses           []InquiryStatus
	InquiryType        *InquiryType
	InquiryTypes       []InquiryType
	ProductBlueprintID *string
	TokenBlueprintID   *string
	HasImage           *bool
	UpdatedBy          *string
	DeletedBy          *string

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

// 代表的なエラー（契約上の表現）
var (
	ErrNotFound = errors.New("inquiry: not found")
	ErrConflict = errors.New("inquiry: conflict")
)

// Repository ポート（契約）
type Repository interface {
	// 一覧取得
	List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult[Inquiry], error)
	ListByCursor(ctx context.Context, filter Filter, sort Sort, cpage CursorPage) (CursorPageResult[Inquiry], error)

	// 取得
	GetByID(ctx context.Context, id string) (Inquiry, error)
	Exists(ctx context.Context, id string) (bool, error)
	Count(ctx context.Context, filter Filter) (int, error)

	// 変更
	Create(ctx context.Context, inq Inquiry) (Inquiry, error)
	Update(ctx context.Context, id string, patch InquiryPatch) (Inquiry, error)
	Delete(ctx context.Context, id string) error

	// 任意: Upsert 等
	Save(ctx context.Context, inq Inquiry, opts *SaveOptions) (Inquiry, error)
}
