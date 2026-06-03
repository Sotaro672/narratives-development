// backend/internal/domain/avatarState/repository_port.go
package avatarState

import (
	"context"
	"errors"
)

// 代表的なエラー（契約上の表現）
var (
	ErrNotFound = errors.New("avatarState: not found")
	ErrConflict = errors.New("avatarState: conflict")
)

// Repository ポート（契約）
type Repository interface {
	// 取得
	GetByAvatarID(ctx context.Context, avatarID string) (AvatarState, error)

	// 作成または更新
	Upsert(ctx context.Context, s AvatarState) (AvatarState, error)

	// 削除
	Delete(ctx context.Context, avatarID string) error
}
