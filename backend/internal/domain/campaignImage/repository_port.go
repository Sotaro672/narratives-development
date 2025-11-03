package campaignImage

import (
	"context"
	"errors"

	common "narratives/internal/domain/common"
)

// ========================================
// Patch（部分更新）: nil のフィールドは更新しない
// ========================================
type CampaignImagePatch struct {
	// 再割当て（任意）
	CampaignID *string

	// 画像情報
	ImageURL *string
	Width    *int
	Height   *int
	FileSize *int64
	MimeType *string
}

// ========================================
// フィルタ/検索条件（実装側で適宜解釈）
// ========================================
type Filter struct {
	// フリーテキスト（imageUrl, mimeType 等、実装側で解釈）
	SearchQuery string

	// 絞り込み
	CampaignID  *string
	CampaignIDs []string
	MimeTypes   []string

	// サイズ/寸法レンジ
	WidthMin    *int
	WidthMax    *int
	HeightMin   *int
	HeightMax   *int
	FileSizeMin *int64
	FileSizeMax *int64
}

// ========================================
// 共通型エイリアス（インフラ非依存）
// ========================================
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
	ErrNotFound = errors.New("campaignImage: not found")
	ErrConflict = errors.New("campaignImage: conflict")
)

// ========================================
// Repository ポート（契約）
// ========================================
type Repository interface {
	// 一覧取得
	List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult[CampaignImage], error)
	ListByCursor(ctx context.Context, filter Filter, sort Sort, cpage CursorPage) (CursorPageResult[CampaignImage], error)

	// 取得
	GetByID(ctx context.Context, id string) (CampaignImage, error)
	Exists(ctx context.Context, id string) (bool, error)
	Count(ctx context.Context, filter Filter) (int, error)

	// 変更
	Create(ctx context.Context, img CampaignImage) (CampaignImage, error)
	Update(ctx context.Context, id string, patch CampaignImagePatch) (CampaignImage, error)
	Delete(ctx context.Context, id string) error

	// 任意: Upsert 等
	Save(ctx context.Context, img CampaignImage, opts *SaveOptions) (CampaignImage, error)
}

// ========================================
// 任意の拡張ポート（GCS 連携向け）
// 実装が提供していれば、型アサーションで利用可能。
// ========================================

// BulkDeleter はキャンペーン単位や複数IDの一括削除を提供します。
// Repository が未対応の場合はこのインターフェイスを実装しないでください。
type BulkDeleter interface {
	// DeleteByCampaignID は指定キャンペーンの画像を一括削除します（メタデータ）。
	// 戻り値は削除件数。
	DeleteByCampaignID(ctx context.Context, campaignID string) (int, error)

	// DeleteByIDs は指定ID群の画像を一括削除します（メタデータ）。
	// 戻り値は削除件数。
	DeleteByIDs(ctx context.Context, ids []string) (int, error)
}

// GCSDeleteOpsProvider はストレージオブジェクト削除用の情報を一括で提供します。
// entity.CampaignImage にも ToGCSDeleteOp があるため、一覧取得後に呼び出し側で
// 生成しても構いません。大量件数で効率化したい場合のみ実装してください。
type GCSDeleteOpsProvider interface {
	// BuildDeleteOps は指定ID群に対応する GCS の削除ターゲットを返します。
	// バケットやオブジェクトパスは entity 側の URL 情報から解決されます。
	BuildDeleteOps(ctx context.Context, ids []string) ([]GCSDeleteOp, error)

	// BuildDeleteOpsByCampaignID はキャンペーンID配下の GCS 削除ターゲットを返します。
	BuildDeleteOpsByCampaignID(ctx context.Context, campaignID string) ([]GCSDeleteOp, error)
}
