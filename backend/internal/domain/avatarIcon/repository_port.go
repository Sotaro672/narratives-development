// backend/internal/domain/avatarIcon/port.go
package avatarIcon

import (
	"context"
	"errors"

	common "narratives/internal/domain/common"
)

// ------------------------------
// Signed URL DTOs
// ------------------------------

// IssueSignedURLInput is a domain-level DTO for issuing signed PUT URL.
type IssueSignedURLInput struct {
	AvatarID         string
	FileName         string
	ContentType      string
	Size             int64
	ExpiresInSeconds int
}

// IssueSignedURLOutput is a domain-level DTO for signed PUT URL response.
type IssueSignedURLOutput struct {
	ID          string
	Bucket      string
	ObjectPath  string
	UploadURL   string
	PublicURL   string
	FileName    string
	ContentType string
	Size        int64
	ExpiresAt   string
}

// ------------------------------
// Patch
// ------------------------------

// Patch（部分更新）: nil のフィールドは更新しない
type AvatarIconPatch struct {
	AvatarID *string
	URL      *string
	FileName *string
	Size     *int64
}

type SaveOptions = common.SaveOptions

// 代表的なエラー（契約上の表現）
var (
	ErrNotFound = errors.New("avatarIcon: not found")
	ErrConflict = errors.New("avatarIcon: conflict")
)

// Repository ポート（契約）
type Repository interface {
	// 取得
	GetByID(ctx context.Context, id string) (AvatarIcon, error)
	GetByAvatarID(ctx context.Context, avatarID string) ([]AvatarIcon, error)

	// 変更
	Create(ctx context.Context, a AvatarIcon) (AvatarIcon, error)
	Update(ctx context.Context, id string, patch AvatarIconPatch) (AvatarIcon, error)
	Delete(ctx context.Context, id string) error

	// 任意: Upsert 等（実装側で opts を無視してもよい）
	Save(ctx context.Context, a AvatarIcon, opts *SaveOptions) (AvatarIcon, error)
}
