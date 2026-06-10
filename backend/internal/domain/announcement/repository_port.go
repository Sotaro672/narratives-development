// backend/internal/domain/announcement/repository_port.go
package announcement

import (
	"context"
	"errors"
	"time"

	common "narratives/internal/domain/common"
)

// ========================================
// Patch（部分更新）: nil のフィールドは更新しない
// ========================================
type AnnouncementPatch struct {
	Title       *string
	Content     *string
	TargetToken *string
	Published   *bool
	PublishedAt *time.Time
	Attachments *[]string
	UpdatedBy   *string
}

// avatars サブコレクション用
type AnnouncementAvatarPatch struct {
	IsRead *bool
}

type AnnouncementAvatarFilter struct {
	AvatarIDs []string
	IsRead    *bool
}

// attachment 部分更新用パッチ（nil は更新しない）
type AttachmentFilePatch struct {
	FileURL    *string
	FileSize   *int64
	MimeType   *string
	ObjectPath *string
}

// attachment 一覧取得用フィルタ
type AttachmentFilter struct {
	SearchQuery    string   // fileName/fileUrl/objectPath 等の部分一致（実装側で解釈）
	AnnouncementID *string  // 特定のお知らせに限定
	FileName       *string  // 完全一致（必要に応じて実装側で LIKE）
	MimeTypes      []string // IN
	SizeMin        *int64
	SizeMax        *int64
	ObjectPathLike string // 例: "announcements/{id}/attachments/" のような前方一致
}

// ========================================
// フィルタ/ソート/ページング（契約）
// ========================================
type Filter struct {
	TargetToken *string

	// 公開状態
	Published *bool

	// 日付範囲
	CreatedFrom   *time.Time
	CreatedTo     *time.Time
	UpdatedFrom   *time.Time
	UpdatedTo     *time.Time
	PublishedFrom *time.Time
	PublishedTo   *time.Time
}

// 共通型エイリアス（インフラに依存しない）
type Sort = common.Sort
type SortOrder = common.SortOrder
type Page = common.Page
type PageResult[T any] = common.PageResult[T]
type CursorPage = common.CursorPage
type CursorPageResult[T any] = common.CursorPageResult[T]

const (
	SortAsc  = common.SortAsc
	SortDesc = common.SortDesc
)

// 代表的なリポジトリエラー
var (
	ErrNotFound = errors.New("announcement: not found")
	ErrConflict = errors.New("announcement: conflict")
)

// ========================================
// Repository ポート（契約）
// ========================================
type Repository interface {
	// 一覧取得
	List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult[Announcement], error)
	ListByCursor(ctx context.Context, filter Filter, sort Sort, cpage CursorPage) (CursorPageResult[Announcement], error)

	// 取得
	GetByID(ctx context.Context, id string) (Announcement, error)

	// 変更
	Create(ctx context.Context, a Announcement) (Announcement, error)
	Update(ctx context.Context, id string, patch AnnouncementPatch) (Announcement, error)
	Delete(ctx context.Context, id string) error

	// avatars サブコレクション
	ListAvatars(ctx context.Context, announcementID string, filter AnnouncementAvatarFilter) ([]AnnouncementAvatar, error)
	GetAvatar(ctx context.Context, announcementID, avatarID string) (AnnouncementAvatar, error)
	UpsertAvatar(ctx context.Context, announcementID string, avatar AnnouncementAvatar) (AnnouncementAvatar, error)
	UpdateAvatar(ctx context.Context, announcementID, avatarID string, patch AnnouncementAvatarPatch) (AnnouncementAvatar, error)
	DeleteAvatar(ctx context.Context, announcementID, avatarID string) error
}

// AttachmentRepository は Announcement 添付ファイル metadata の repository 契約。
// Firebase Storage の実体操作は frontend 側で行い、backend は metadata のみ扱う。
type AttachmentRepository interface {
	// 一覧
	ListAttachments(ctx context.Context, filter AttachmentFilter, sort Sort, page Page) (PageResult[AttachmentFile], error)
	ListAttachmentsByCursor(ctx context.Context, filter AttachmentFilter, sort Sort, cpage CursorPage) (CursorPageResult[AttachmentFile], error)

	// 取得系
	GetAttachment(ctx context.Context, announcementID, fileName string) (AttachmentFile, error)
	GetAttachmentsByAnnouncementID(ctx context.Context, announcementID string) ([]AttachmentFile, error)

	// 変更系
	CreateAttachment(ctx context.Context, f AttachmentFile) (AttachmentFile, error)
	UpdateAttachment(ctx context.Context, announcementID, fileName string, patch AttachmentFilePatch) (AttachmentFile, error)
	DeleteAttachment(ctx context.Context, announcementID, fileName string) error
	DeleteAllAttachmentsByAnnouncementID(ctx context.Context, announcementID string) error
}
