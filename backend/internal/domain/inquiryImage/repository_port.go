package inquiryimage

import (
	"context"
	"errors"
	"time"

	common "narratives/internal/domain/common"
)

// リクエストDTO: 画像追加（必須項目のみ）
type AddImageRequest struct {
	FileName string `json:"fileName"`
	FileURL  string `json:"fileUrl"`
	FileSize int64  `json:"fileSize"`
	MimeType string `json:"mimeType"`
	Width    *int   `json:"width,omitempty"`
	Height   *int   `json:"height,omitempty"`
}

// リクエストDTO: 画像一括置換（Inquiry単位）
type UpdateImagesRequest struct {
	Images []ImageFile `json:"images"`
}

// Patch（部分更新）: nil のフィールドは更新しない（画像メタ更新などに使用）
type ImagePatch struct {
	FileName *string
	FileURL  *string
	FileSize *int64
	MimeType *string
	Width    *int
	Height   *int

	UpdatedAt *time.Time
	UpdatedBy *string
	DeletedAt *time.Time
	DeletedBy *string
}

// フィルタ/検索条件（実装側で解釈）
type Filter struct {
	SearchQuery string // fileName, fileUrl などに対する部分一致等は実装側で解釈

	// 絞り込み
	InquiryID   *string
	InquiryIDs  []string
	FileName    *string
	MimeType    *string
	CreatedBy   *string
	UpdatedBy   *string
	DeletedBy   *string

	// 日付レンジ
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	UpdatedFrom *time.Time
	UpdatedTo   *time.Time
	DeletedFrom *time.Time
	DeletedTo   *time.Time

	// 論理削除の tri-state（nil: 全件 / true: 削除済のみ / false: 未削除のみ）
	Deleted *bool
}

// 共通型エイリアス（インフラ非依存）
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

// 代表的なエラー（契約上の表現）
var (
	ErrNotFound = errors.New("inquiryImage: not found")
	ErrConflict = errors.New("inquiryImage: conflict")
)

// Repository ポート（契約）
type Repository interface {
	// 集約取得
	GetImagesByInquiryID(ctx context.Context, inquiryID string) (*InquiryImage, error)

	// 画像一覧（横断的検索）
	ListImages(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult[ImageFile], error)
	ListImagesByCursor(ctx context.Context, filter Filter, sort Sort, cpage CursorPage) (CursorPageResult[ImageFile], error)
	Count(ctx context.Context, filter Filter) (int, error)
	Exists(ctx context.Context, inquiryID, fileName string) (bool, error)

	// 作成・更新（集約単位）
	AddImage(ctx context.Context, inquiryID string, req AddImageRequest) (*InquiryImage, error)
	UpdateImages(ctx context.Context, inquiryID string, req UpdateImagesRequest) (*InquiryImage, error)

	// 画像メタの部分更新（任意実装）
	PatchImage(ctx context.Context, inquiryID, fileName string, patch ImagePatch) (*ImageFile, error)

	// 削除
	DeleteImage(ctx context.Context, inquiryID, fileName string) (*InquiryImage, error)
	DeleteAllImages(ctx context.Context, inquiryID string) error

	// 任意: Upsert 等（集約保存）
	Save(ctx context.Context, agg InquiryImage, opts *SaveOptions) (*InquiryImage, error)
}

// ImageKey は 1 件の Inquiry 画像を一意に識別する複合キーです。
type ImageKey struct {
	InquiryID string `json:"inquiryId"`
	FileName  string `json:"fileName"`
}

// ================================
// 任意の拡張ポート（GCS 連携向け）
// 既存の Repository 実装は変更不要です。
// 実装が提供される場合のみ、型アサーションで利用してください。
// ================================

// GCSObjectSaver は、GCS に格納済みオブジェクトの bucket/objectPath から
// 公開URLを組み立てて画像メタを保存するための拡張契約です。
// bucket が空文字の場合は entity.go の DefaultBucket (narratives_development_inquiry_image) を用いる実装を推奨します。
type GCSObjectSaver interface {
	SaveImageFromBucketObject(
		ctx context.Context,
		inquiryID string,
		fileName string,
		bucket string,
		objectPath string,
		fileSize int64,
		mimeType string,
		width, height *int,
		createdAt time.Time,
		createdBy string,
	) (*ImageFile, error)
}

// GCSDeleteOpsProvider は削除対象オブジェクトの解決を一括で提供する拡張契約です。
// entity.ImageFile の ToGCSDeleteOp で呼び出し側が組み立てることも可能ですが、
// 大量件数で効率化したい場合に実装してください。
type GCSDeleteOpsProvider interface {
	// 指定キー群に対応する GCS の削除ターゲットを返します。
	BuildDeleteOps(ctx context.Context, keys []ImageKey) ([]GCSDeleteOp, error)

	// 問い合わせID配下の全画像に対する GCS 削除ターゲットを返します。
	BuildDeleteOpsByInquiryID(ctx context.Context, inquiryID string) ([]GCSDeleteOp, error)
}
