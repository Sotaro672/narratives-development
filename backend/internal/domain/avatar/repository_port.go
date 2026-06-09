// backend/internal/domain/avatar/repository_port.go
package avatar

import (
	"context"

	common "narratives/internal/domain/common"
)

// ========================================
// Patch（部分更新）
// ========================================
// nil のフィールドは更新しない契約
//
// entity.go を正として:
// - AvatarIconURL / AvatarIconPath → AvatarIcon
// - Bio          → Profile
// - Website      → ExternalLink
type AvatarPatch struct {
	UserID        string  `json:"userId"`
	AvatarName    *string `json:"avatarName,omitempty"`
	AvatarIcon    *string `json:"avatarIcon,omitempty"`
	WalletAddress *string `json:"walletAddress,omitempty"`
	Profile       *string `json:"profile,omitempty"`
	ExternalLink  *string `json:"externalLink,omitempty"`
}

// Sanitize keeps patch fields as-is.
func (p *AvatarPatch) Sanitize() {
	if p == nil {
		return
	}
}

type Sort struct {
	Column SortColumn
	Order  SortOrder
}

type SortColumn string
type SortOrder string

const (
	SortAsc  SortOrder = "asc"
	SortDesc SortOrder = "desc"
)

// 共通定義のエイリアス（ドメイン層はインフラ未依存）
type Page = common.Page
type PageResult = common.PageResult[Avatar]
type SaveOptions = common.SaveOptions
type RepositoryCRUD = common.RepositoryCRUD[Avatar, AvatarPatch]

// カーソルページング（PG実装が使用）
type CursorPage = common.CursorPage
type CursorPageResult = common.CursorPageResult[Avatar]

// ========================================
// Repository ポート（契約）
// ========================================

type Repository interface {
	// avatarId による取得
	GetByID(ctx context.Context, id string) (Avatar, error)

	// userId による取得
	// avatar document id は avatarId であり userId ではない。
	// uid -> avatarId 解決や mall/me/avatar 判定で使用する。
	GetByUserID(ctx context.Context, userID string) (Avatar, error)

	// 作成
	Create(ctx context.Context, a Avatar) (Avatar, error)

	// 更新
	Update(ctx context.Context, id string, patch AvatarPatch) (Avatar, error)

	// 削除
	Delete(ctx context.Context, id string) error

	// userId による存在確認
	// avatar document id は avatarId であり userId ではない。
	// setup status や mall/me/avatar 判定では userId で存在確認する。
	ExistsByUserID(ctx context.Context, userID string) (bool, error)
}
