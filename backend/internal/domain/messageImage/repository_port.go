package messageImage

import (
	"context"
	"errors"
	"time"
)

// RepositoryPort - MessageImage のメタデータ永続化ポート（ドメイン層）
// データストア技術に依存しない抽象インターフェースです。
type RepositoryPort interface {
	// 取得系
	ListByMessageID(ctx context.Context, messageID string) ([]ImageFile, error)
	Get(ctx context.Context, messageID, fileName string) (*ImageFile, error)
	List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult, error)
	Count(ctx context.Context, filter Filter) (int, error)

	// 変更系（メタデータ）
	Add(ctx context.Context, img ImageFile) (ImageFile, error)
	ReplaceAll(ctx context.Context, messageID string, images []ImageFile) ([]ImageFile, error)
	Update(ctx context.Context, messageID, fileName string, patch ImageFilePatch) (ImageFile, error)
	Delete(ctx context.Context, messageID, fileName string) error
	DeleteAll(ctx context.Context, messageID string) error
}

// ObjectStoragePort - オブジェクトストレージ操作のポート（GCS 等）
// 実体の削除はアダプタ層で実装します。ドメインでは削除対象の組み立てのみ行います。
// entity.go の GCSDeleteOp/BuildGCSDeleteOpsFromMessage を併用してください。
type ObjectStoragePort interface {
	// 単一削除
	DeleteObject(ctx context.Context, bucket, objectPath string) error
	// 複数削除（バッチ）
	DeleteObjects(ctx context.Context, ops []GCSDeleteOp) error
}

// 部分更新用パッチ（nil は更新しない）
type ImageFilePatch struct {
	FileName  *string
	FileURL   *string
	FileSize  *int64
	MimeType  *string
	Width     *int
	Height    *int
	UpdatedAt *time.Time
	DeletedAt *time.Time
}

// 一覧用フィルタ/ソート/ページング
type Filter struct {
	MessageID    string
	FileNameLike string
	MimeType     *string
	MinSize      *int64
	MaxSize      *int64

	CreatedFrom *time.Time
	CreatedTo   *time.Time
	UpdatedFrom *time.Time
	UpdatedTo   *time.Time
	Deleted     *bool // nil: 全件 / true: 削除済のみ / false: 未削除のみ
}

type Sort struct {
	Column SortColumn
	Order  SortOrder
}

type SortColumn string

const (
	SortByCreatedAt SortColumn = "createdAt"
	SortByFileName  SortColumn = "fileName"
	SortByFileSize  SortColumn = "fileSize"
	SortByUpdatedAt SortColumn = "updatedAt"
)

type SortOrder string

const (
	SortAsc  SortOrder = "asc"
	SortDesc SortOrder = "desc"
)

type Page struct {
	Number  int
	PerPage int
}

type PageResult struct {
	Items      []ImageFile
	TotalCount int
	TotalPages int
	Page       int
	PerPage    int
}

// 代表的なエラー
var (
	ErrNotFound = errors.New("messageImage: not found")
	ErrConflict = errors.New("messageImage: conflict")
)
