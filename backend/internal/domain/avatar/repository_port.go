// backend/internal/domain/avatar/repository_port.go
package avatar

import "context"

// AvatarPatch はAvatarの部分更新入力です。
// nilのフィールドは更新しません。
type AvatarPatch struct {
	UserID        string  `json:"userId"`
	AvatarName    *string `json:"avatarName,omitempty"`
	AvatarIcon    *string `json:"avatarIcon,omitempty"`
	WalletAddress *string `json:"walletAddress,omitempty"`
	Profile       *string `json:"profile,omitempty"`
	ExternalLink  *string `json:"externalLink,omitempty"`
}

// Repository はAvatar集約の永続化契約です。
type Repository interface {
	// GetByID はavatarIdでAvatarを取得します。
	GetByID(ctx context.Context, id string) (Avatar, error)

	// GetByUserID はuserIdでAvatarを取得します。
	// Avatar document IDはavatarIdであり、userIdではありません。
	GetByUserID(ctx context.Context, userID string) (Avatar, error)

	// Create はAvatarを作成します。
	Create(ctx context.Context, a Avatar) (Avatar, error)

	// Update はAvatarを部分更新します。
	Update(ctx context.Context, id string, patch AvatarPatch) (Avatar, error)

	// Delete はAvatarを削除します。
	Delete(ctx context.Context, id string) error

	// ExistsByUserID はuserIdに対応するAvatarの存在を確認します。
	ExistsByUserID(ctx context.Context, userID string) (bool, error)
}
