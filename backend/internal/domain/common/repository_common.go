package common

import (
	"context"
	"time"
)

// Timestamps は作成・更新時刻を共通で保持するための埋め込み用構造体
type Timestamps struct {
	CreatedAt time.Time  // 作成日時
	UpdatedAt *time.Time // 更新日時（未更新なら nil）
}

// TimeRange は期間フィルタのための共通構造体
type TimeRange struct {
	From *time.Time
	To   *time.Time
}

// FilterCommon は多くの一覧取得で使える共通フィルタ
// 各ドメイン固有の条件は別途拡張用の Filter に追加してください。
type FilterCommon struct {
	SearchQuery string    // 部分一致検索など
	Created     TimeRange // 作成日時の範囲
	Updated     TimeRange // 更新日時の範囲
}

// Sort はソート指定の共通表現
type Sort struct {
	Column string    // カラム名（各ドメイン側で許可カラムをバリデート）
	Order  SortOrder // 昇順/降順
}

// SortOrder はソート順
type SortOrder string

const (
	SortAsc  SortOrder = "asc"
	SortDesc SortOrder = "desc"
)

// Page はオフセットページング指定
type Page struct {
	Number  int // 1-based
	PerPage int // 0 以下は実装側デフォルト
}

// PageResult はページング結果（ジェネリクスでアイテム型を受け取る）
type PageResult[T any] struct {
	Items      []T
	TotalCount int
	TotalPages int
	Page       int
	PerPage    int
}

// CursorPage はカーソルページング指定
type CursorPage struct {
	After string // 直前ページの最後のカーソル（空なら先頭）
	Limit int
}

// CursorPageResult はカーソルページング結果（ジェネリクス対応）
type CursorPageResult[T any] struct {
	Items      []T
	NextCursor *string // 次ページがなければ nil
	Limit      int
}

// SaveOptions は保存時の前提条件（楽観ロック等）を受け取るためのオプション
type SaveOptions struct {
	// IfMatchVersion が指定されていれば、現在の Version と一致した場合のみ更新
	// （一致しなければ ErrPreconditionFailed）
	IfMatchVersion *int64
}

// RepositoryCRUD は基本的なCRUD操作の共通インターフェース
// P は部分更新のための Patch 型（ドメインごとに定義）
type RepositoryCRUD[T any, P any] interface {
	GetByID(ctx context.Context, id string) (T, error)
	Create(ctx context.Context, entity T) (T, error)
	Update(ctx context.Context, id string, patch P) (T, error)
	Delete(ctx context.Context, id string) error
}

// RepositoryList は Filter + Sort + Page を伴う一覧取得の共通インターフェース
// F はフィルタ型（各ドメインで FilterCommon を内包/埋め込みして拡張することを推奨）
type RepositoryList[T any, F any] interface {
	List(ctx context.Context, filter F, sort Sort, page Page) (PageResult[T], error)
}

// Repository は CRUD と List を合成した共通の包括インターフェース
type Repository[T any, F any, P any] interface {
	RepositoryCRUD[T, P]
	RepositoryList[T, F]
}
