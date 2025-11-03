package avatarIcon

import (
	"context"
	"errors"

	common "narratives/internal/domain/common"
)

// Patch（部分更新）: nil のフィールドは更新しない
// entity.go 準拠（監査系は排除）
type AvatarIconPatch struct {
	AvatarID *string
	URL      *string
	FileName *string
	Size     *int64
}

// フィルタ（契約）
// entity.go に合わせて監査系や削除フラグ等は持たない
type Filter struct {
	// 部分一致: id, url, fileName（実装側で解釈）
	SearchQuery string

	// 絞り込み
	AvatarID    *string
	HasAvatarID *bool // true: NOT NULL, false: IS NULL

	// サイズ範囲
	SizeMin *int64
	SizeMax *int64
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
	ErrNotFound = errors.New("avatarIcon: not found")
	ErrConflict = errors.New("avatarIcon: conflict")
)

// Repository ポート（契約）
type Repository interface {
	// 一覧取得
	List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult[AvatarIcon], error)
	ListByCursor(ctx context.Context, filter Filter, sort Sort, cpage CursorPage) (CursorPageResult[AvatarIcon], error)

	// 取得
	GetByID(ctx context.Context, id string) (AvatarIcon, error)
	GetByAvatarID(ctx context.Context, avatarID string) ([]AvatarIcon, error)

	// 変更
	Create(ctx context.Context, a AvatarIcon) (AvatarIcon, error)
	Update(ctx context.Context, id string, patch AvatarIconPatch) (AvatarIcon, error)
	Delete(ctx context.Context, id string) error

	// 補助
	Count(ctx context.Context, filter Filter) (int, error)

	// 任意: Upsert 等（実装側で opts を無視してもよい）
	Save(ctx context.Context, a AvatarIcon, opts *SaveOptions) (AvatarIcon, error)
}
