// backend/internal/domain/avatarState/repository_port.go
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
	Followers      *[]AvatarFollowRef
	Following      *[]AvatarFollowRef
	LastActiveAt   *time.Time
	UpdatedAt      *time.Time
}

// 共通型エイリアス（インフラ非依存）
type SaveOptions = common.SaveOptions

// 代表的なエラー（契約上の表現）
var (
	ErrNotFound = errors.New("avatarState: not found")
	ErrConflict = errors.New("avatarState: conflict")
)

// Repository ポート（契約）
type Repository interface {
	// 取得
	GetByID(ctx context.Context, id string) (AvatarState, error)
	GetByAvatarID(ctx context.Context, avatarID string) (AvatarState, error)
	Exists(ctx context.Context, id string) (bool, error)

	// 変更
	Create(ctx context.Context, s AvatarState) (AvatarState, error)
	Update(ctx context.Context, id string, patch AvatarStatePatch) (AvatarState, error)
	UpdateByAvatarID(ctx context.Context, avatarID string, patch AvatarStatePatch) (AvatarState, error)
	Delete(ctx context.Context, id string) error
	DeleteByAvatarID(ctx context.Context, avatarID string) error

	// 任意: Upsert 等
	Save(ctx context.Context, s AvatarState, opts *SaveOptions) (AvatarState, error)
}
