package listImage

import (
	"context"
	"errors"
	"time"

	common "narratives/internal/domain/common"
)

// アップロード入力（インフラ非依存の契約）
type UploadImageInput struct {
	ImageData string `json:"imageData"`        // Base64エンコード済み画像
	FileName  string `json:"fileName"`         // ファイル名
	ListID    string `json:"listId,omitempty"` // 紐づくリストID（任意）
}

// 部分更新: nil のフィールドは更新しない
type ListImagePatch struct {
	URL          *string
	FileName     *string
	Size         *int64
	DisplayOrder *int

	UpdatedAt *time.Time
	UpdatedBy *string
	DeletedAt *time.Time
	DeletedBy *string
}

// フィルタ/検索条件（実装側で解釈）
type Filter struct {
	SearchQuery string // fileName/url などの部分一致は実装側で解釈

	IDs     []string
	ListID  *string
	ListIDs []string

	FileNameLike   *string
	MinSize        *int64
	MaxSize        *int64
	MinDisplayOrd  *int
	MaxDisplayOrd  *int
	CreatedBy      *string
	UpdatedBy      *string
	DeletedBy      *string
	CreatedFrom    *time.Time
	CreatedTo      *time.Time
	UpdatedFrom    *time.Time
	UpdatedTo      *time.Time
	DeletedFrom    *time.Time
	DeletedTo      *time.Time
	Deleted        *bool // nil: 全件 / true: 削除済のみ / false: 未削除のみ
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

// 代表的なエラー（契約）
var (
	ErrNotFound = errors.New("listImage: not found")
	ErrConflict = errors.New("listImage: conflict")
)

// RepositoryPort - 画像管理のデータアクセスを抽象化（契約）
type RepositoryPort interface {
	// 一覧・検索
	List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult[ListImage], error)
	ListByCursor(ctx context.Context, filter Filter, sort Sort, cpage CursorPage) (CursorPageResult[ListImage], error)
	Count(ctx context.Context, filter Filter) (int, error)

	// 取得
	FindByID(ctx context.Context, imageID string) (*ListImage, error)
	FindByListID(ctx context.Context, listID string) ([]ListImage, error)
	Exists(ctx context.Context, imageID string) (bool, error)

	// 作成/変更
	Create(ctx context.Context, img ListImage) (ListImage, error)
	Update(ctx context.Context, imageID string, patch ListImagePatch) (ListImage, error)
	Save(ctx context.Context, img ListImage, opts *SaveOptions) (ListImage, error)

	// アップロード（データURL等を受け取り、保存＋レコード作成まで行うユースケース向け）
	Upload(ctx context.Context, in UploadImageInput) (*ListImage, error)

	// 削除
	Delete(ctx context.Context, imageID string) error
}
