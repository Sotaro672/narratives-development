// backend/internal/domain/avatar/repository_port.go
package avatar

import (
	"context"
	"time"

	common "narratives/internal/domain/common"
)

// ========================================
// Patch（部分更新）
// ========================================
// nil のフィールドは更新しない契約
//
// ✅ entity.go を正として:
// - AvatarIconURL / AvatarIconPath → AvatarIcon
// - Bio          → Profile
// - Website      → ExternalLink
// - FirebaseUID  を追加
type AvatarPatch struct {
	UserID        string     `json:"userId"`
	AvatarName    *string    `json:"avatarName,omitempty"`
	AvatarIcon    *string    `json:"avatarIcon,omitempty"`
	WalletAddress *string    `json:"walletAddress,omitempty"`
	Profile       *string    `json:"profile,omitempty"`
	ExternalLink  *string    `json:"externalLink,omitempty"`
	DeletedAt     *time.Time `json:"deletedAt,omitempty"` // soft delete/restore 用（必要な場合のみ使用）
}

// Sanitize keeps patch fields as-is.
func (p *AvatarPatch) Sanitize() {
	if p == nil {
		return
	}
	// DeletedAt: keep as-is (nil means "no change")
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
	// 共通CRUD
	RepositoryCRUD

	// ✅ NEW: avatarId -> avatarName (best-effort lightweight getter)
	GetNameByID(ctx context.Context, id string) (string, error)

	// ✅ NEW: avatarId -> (avatarName, avatarIcon) (best-effort lightweight getter)
	// - 一覧表示やコメント表示などで N+1 を軽量化する用途を想定
	// - 見つからない場合は error を返す（実装側で NotFound を返却）
	GetNameAndIconByID(ctx context.Context, id string) (name string, icon string, err error)

	// 追加要件（必要に応じて実装側で活用）
	GetByWalletAddress(ctx context.Context, wallet string) (Avatar, error)
	Exists(ctx context.Context, id string) (bool, error)
	Save(ctx context.Context, a Avatar, opts *SaveOptions) (Avatar, error)
}
