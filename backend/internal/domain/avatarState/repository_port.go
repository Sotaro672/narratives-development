// backend\internal\domain\avatarState\repository_port.go
package avatarState

import (
	"context"
	"errors"
	"time"

	common "narratives/internal/domain/common"
)

// Patch（部分更新）: nil のフィールドは更新しない
type AvatarStatePatch struct {
	FollowerCount  *int64
	FollowingCount *int64
	PostCount      *int64
	LastActiveAt   *time.Time
	UpdatedAt      *time.Time
}

// フィルタ/検索条件（実装側で適宜解釈）
type Filter struct {
	// 部分一致（id 等）。実装側で対象カラムを決定
	SearchQuery string

	// 絞り込み
	AvatarID  *string
	AvatarIDs []string

	// カウント範囲
	FollowerMin  *int64
	FollowerMax  *int64
	FollowingMin *int64
	FollowingMax *int64
	PostMin      *int64
	PostMax      *int64

	// 日付範囲
	LastActiveFrom *time.Time
	LastActiveTo   *time.Time
	UpdatedFrom    *time.Time
	UpdatedTo      *time.Time
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
	ErrNotFound = errors.New("avatarState: not found")
	ErrConflict = errors.New("avatarState: conflict")
)

// Repository ポート（契約）
type Repository interface {
	// 一覧取得
	List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult[AvatarState], error)
	ListByCursor(ctx context.Context, filter Filter, sort Sort, cpage CursorPage) (CursorPageResult[AvatarState], error)

	// 取得
	GetByID(ctx context.Context, id string) (AvatarState, error)
	GetByAvatarID(ctx context.Context, avatarID string) (AvatarState, error)
	Exists(ctx context.Context, id string) (bool, error)
	Count(ctx context.Context, filter Filter) (int, error)

	// 変更
	Create(ctx context.Context, s AvatarState) (AvatarState, error)
	Update(ctx context.Context, id string, patch AvatarStatePatch) (AvatarState, error)
	UpdateByAvatarID(ctx context.Context, avatarID string, patch AvatarStatePatch) (AvatarState, error)
	Delete(ctx context.Context, id string) error
	DeleteByAvatarID(ctx context.Context, avatarID string) error

	// 任意: Upsert 等
	Save(ctx context.Context, s AvatarState, opts *SaveOptions) (AvatarState, error)
}
