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
	FirebaseUID   *string    `json:"firebaseUid,omitempty"`
	AvatarName    *string    `json:"avatarName,omitempty"`
	AvatarIcon    *string    `json:"avatarIcon,omitempty"`
	WalletAddress *string    `json:"walletAddress,omitempty"`
	Profile       *string    `json:"profile,omitempty"`
	ExternalLink  *string    `json:"externalLink,omitempty"`
	DeletedAt     *time.Time `json:"deletedAt,omitempty"` // soft delete/restore 用（必要な場合のみ使用）
}

// ========================================
// フィルタ/ソート/ページング
// ========================================
//
// ✅ entity.go を正として検索対象も更新（bio/website → profile/externalLink）
// ✅ avatarIconUrl/avatarIconPath → avatarIcon
// ✅ firebaseUid を検索・絞り込み対象に追加（必要に応じて実装側で利用）
type Filter struct {
	// 部分一致検索対象: id, firebaseUid, avatarName, profile, externalLink, walletAddress, avatarIcon
	SearchQuery string

	// 絞り込み
	UserID        *string
	FirebaseUID   *string
	WalletAddress *string

	// 日付範囲（created_at ベース）
	// 既存実装互換のため JoinedFrom/JoinedTo を使用（PG実装が参照）
	JoinedFrom *time.Time
	JoinedTo   *time.Time

	// 追加で使いたい場合の汎用的な範囲（必要ならPG側で対応）
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	UpdatedFrom *time.Time
	UpdatedTo   *time.Time

	// 論理削除フィルタ
	// nil: すべて / true: DeletedAt IS NOT NULL / false: DeletedAt IS NULL
	Deleted *bool
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
type RepositoryList = common.RepositoryList[Avatar, Filter]

// カーソルページング（PG実装が使用）
type CursorPage = common.CursorPage
type CursorPageResult = common.CursorPageResult[Avatar]

// ========================================
// Repository ポート（契約）
// ========================================

type Repository interface {
	// 共通CRUD/一覧
	RepositoryCRUD
	RepositoryList

	// 追加要件（必要に応じて実装側で活用）
	GetByWalletAddress(ctx context.Context, wallet string) (Avatar, error)
	Search(ctx context.Context, query string) ([]Avatar, error)
	Exists(ctx context.Context, id string) (bool, error)
	Count(ctx context.Context, filter Filter) (int, error)
	Save(ctx context.Context, a Avatar, opts *SaveOptions) (Avatar, error)
	Reset(ctx context.Context) error
}
