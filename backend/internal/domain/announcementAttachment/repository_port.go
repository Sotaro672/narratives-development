package announcementAttachment

import (
	"context"
	"errors"

	common "narratives/internal/domain/common"
)

// AttachmentFile は entity.go を参照（GCS: DefaultBucket, BuildObjectPath, Bucket/ObjectPath を保持）

// 部分更新用パッチ（nil は更新しない）
type AttachmentFilePatch struct {
	FileURL  *string
	FileSize *int64
	MimeType *string
	// Bucket/ObjectPath は通常不変（オブジェクト移動は新規作成+旧削除で対処）
}

// 一覧取得用フィルタ（entity に合わせて GCS 参照もフィルタ可能に）
type Filter struct {
	SearchQuery    string   // fileName/fileUrl 等の部分一致（実装側で解釈）
	AnnouncementID *string  // 特定のお知らせに限定
	FileName       *string  // 完全一致（必要に応じて実装側で LIKE）
	MimeTypes      []string // IN
	SizeMin        *int64
	SizeMax        *int64

	// GCS 参照
	Bucket         *string // 例: narratives_development_announcement_attachment
	ObjectPathLike string  // 例: "announcements/{id}/%" のような前方一致
}

// 共通型（インフラ非依存）
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

// 代表的なエラー（リポジトリ層）
var (
	ErrNotFound = errors.New("announcementAttachment: not found")
	ErrConflict = errors.New("announcementAttachment: conflict")
)

// Repository ポート（契約）
type Repository interface {
	// 一覧
	List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult[AttachmentFile], error)
	ListByCursor(ctx context.Context, filter Filter, sort Sort, cpage CursorPage) (CursorPageResult[AttachmentFile], error)

	// 取得系
	Get(ctx context.Context, announcementID, fileName string) (AttachmentFile, error)
	GetByAnnouncementID(ctx context.Context, announcementID string) ([]AttachmentFile, error)
	Exists(ctx context.Context, announcementID, fileName string) (bool, error)
	Count(ctx context.Context, filter Filter) (int, error)

	// 変更系
	Create(ctx context.Context, f AttachmentFile) (AttachmentFile, error)
	Update(ctx context.Context, announcementID, fileName string, patch AttachmentFilePatch) (AttachmentFile, error)
	Delete(ctx context.Context, announcementID, fileName string) error
	DeleteAllByAnnouncementID(ctx context.Context, announcementID string) error

	// 任意: Upsert 等
	Save(ctx context.Context, f AttachmentFile, opts *SaveOptions) (AttachmentFile, error)
}

// オブジェクトストレージ操作ポート（GCS 等）
// 実体の削除/移動はアダプタ層で実装します。
// entity.go の GCSDeleteOp/BuildGCSDeleteOps を併用してください。
type ObjectStoragePort interface {
	// 単一削除
	DeleteObject(ctx context.Context, bucket, objectPath string) error
	// 複数削除（バッチ）
	DeleteObjects(ctx context.Context, ops []GCSDeleteOp) error
}
