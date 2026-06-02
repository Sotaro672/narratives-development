// backend/internal/domain/avatarIcon/port.go
package avatarIcon

import (
	"context"
	"errors"
)

// ------------------------------
// Patch
// ------------------------------

// Patch（部分更新）: nil のフィールドは更新しない
type AvatarIconPatch struct {
	AvatarID *string
	URL      *string
}

// 代表的なエラー（契約上の表現）
var (
	ErrNotFound = errors.New("avatarIcon: not found")
	ErrConflict = errors.New("avatarIcon: conflict")
)

// Repository ポート（契約）
type Repository interface {
	// 取得
	GetByAvatarID(ctx context.Context, avatarID string) ([]AvatarIcon, error)

	// 変更
	Create(ctx context.Context, a AvatarIcon) (AvatarIcon, error)
	Update(ctx context.Context, id string, patch AvatarIconPatch) (AvatarIcon, error)
	Delete(ctx context.Context, id string) error
}
