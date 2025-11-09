package announcement

import (
	"context"
	"errors"
	"time"

	common "narratives/internal/domain/common"
)

// ========================================
// Patch（部分更新）: nil のフィールドは更新しない
// ========================================
type AnnouncementPatch struct {
	Title          *string
	Content        *string
	Category       *AnnouncementCategory
	TargetAudience *TargetAudience
	TargetToken    *string
	TargetProducts *[]string
	TargetAvatars  *[]string
	IsPublished    *bool
	PublishedAt    *time.Time
	Attachments    *[]string
	Status         *AnnouncementStatus
	UpdatedBy      *string
	DeletedAt      *time.Time
	DeletedBy      *string
}

// ========================================
// フィルタ/ソート/ページング（契約）
// ========================================
type Filter struct {
	// キーワード検索（title, content, attachments など実装側で解釈）
	SearchQuery string

	// 絞り込み
	Categories []AnnouncementCategory
	Audiences  []TargetAudience
	Statuses   []AnnouncementStatus

	TargetToken   *string
	TargetProducts []string
	TargetAvatars  []string

	// 公開状態
	IsPublished *bool

	// 日付範囲
	CreatedFrom  *time.Time
	CreatedTo    *time.Time
	UpdatedFrom  *time.Time
	UpdatedTo    *time.Time
	PublishedFrom *time.Time
	PublishedTo   *time.Time

	// 論理削除フィルタ
	// nil: 全件 / true: 削除済のみ / false: 未削除のみ
	Deleted *bool
}

// 共通型エイリアス（インフラに依存しない）
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

// 代表的なリポジトリエラー
var (
	ErrNotFound = errors.New("announcement: not found")
	ErrConflict = errors.New("announcement: conflict")
)

// ========================================
// Repository ポート（契約）
// ========================================
type Repository interface {
	// 一覧取得
	List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult[Announcement], error)
	ListByCursor(ctx context.Context, filter Filter, sort Sort, cpage CursorPage) (CursorPageResult[Announcement], error)

	// 取得
	GetByID(ctx context.Context, id string) (Announcement, error)
	Exists(ctx context.Context, id string) (bool, error)
	Count(ctx context.Context, filter Filter) (int, error)
	Search(ctx context.Context, query string) ([]Announcement, error)

	// 変更
	Create(ctx context.Context, a Announcement) (Announcement, error)
	Update(ctx context.Context, id string, patch AnnouncementPatch) (Announcement, error)
	Delete(ctx context.Context, id string) error

	// 任意: Upsert 等
	Save(ctx context.Context, a Announcement, opts *SaveOptions) (Announcement, error)
}
