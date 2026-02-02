// backend/internal/application/usecase/list/types.go
//
// Responsibility:
// - List Usecase の「型定義」を集約する（DTO / Port / Aggregate / Usecase struct）。
// - ビジネス処理（メソッド実装）は置かず、他ファイルから参照される共通定義のみを保持する。
//
// Features:
// - Signed URL 発行用 DTO
// - List / ListImage の各 Port（Reader/Lister/Creator/...）
// - ListUsecase / ListAggregate の定義
package list

import (
	"context"
	"time"

	listdom "narratives/internal/domain/list"
	listimgdom "narratives/internal/domain/listImage"
)

// ==============================
// Signed URL (A) DTOs
// ==============================

type ListImageIssueSignedURLInput struct {
	ListID           string `json:"listId"`
	FileName         string `json:"fileName"`
	ContentType      string `json:"contentType"`
	Size             int64  `json:"size"`
	DisplayOrder     int    `json:"displayOrder"`
	ExpiresInSeconds int    `json:"expiresInSeconds"` // optional
}

type ListImageIssueSignedURLOutput struct {
	// id は objectPath を採用（SaveFromBucketObject / GetByID で一意に引ける）
	ID         string `json:"id"`
	Bucket     string `json:"bucket"`
	ObjectPath string `json:"objectPath"`

	// signed url
	UploadURL string `json:"uploadUrl"`

	// public url
	PublicURL string `json:"publicUrl"`

	FileName     string `json:"fileName"`
	ContentType  string `json:"contentType"`
	Size         int64  `json:"size"`
	DisplayOrder int    `json:"displayOrder"`
	ExpiresAt    string `json:"expiresAt"` // RFC3339
}

// ==============================
// Ports
// ==============================

// GCS adapter 側が実装する（例：IssueSignedURL）
type ListImageSignedURLIssuer interface {
	IssueSignedURL(ctx context.Context, in ListImageIssueSignedURLInput) (ListImageIssueSignedURLOutput, error)
}

// ListReader は List 単体取得の契約です。
type ListReader interface {
	GetByID(ctx context.Context, id string) (listdom.List, error)
}

// ListLister は List 一覧取得の契約です（GET /lists 用）。
type ListLister interface {
	List(ctx context.Context, filter listdom.Filter, sort listdom.Sort, page listdom.Page) (listdom.PageResult[listdom.List], error)
	Count(ctx context.Context, filter listdom.Filter) (int, error)
}

// ListCreator は List 作成の契約です。
type ListCreator interface {
	Create(ctx context.Context, item listdom.List) (listdom.List, error)
}

// ListUpdater は List 本体更新の契約です（PUT/PATCH /lists/{id} 用）。
type ListUpdater interface {
	Update(ctx context.Context, item listdom.List) (listdom.List, error)
}

// ★ domain.Repository 互換の「patch Update」(Update(ctx, id, patch)) を直接叩ける場合に使う。
type ListPatchUpdater interface {
	Update(ctx context.Context, id string, patch listdom.ListPatch) (listdom.List, error)
}

// ListPatcher は List.ImageID を更新できる契約です。
type ListPatcher interface {
	// NOTE: 現方針では imageID は「画像URL（bucket上のURL）」を格納する。
	UpdateImageID(ctx context.Context, listID string, imageID string, now time.Time, updatedBy *string) (listdom.List, error)
}

// ListImageReader は ListID に紐づく ListImage 一覧の取得契約です。
// NOTE:
// - 今後の推奨: Firestore の /lists/{listId}/images サブコレクションを source of truth とする。
type ListImageReader interface {
	ListByListID(ctx context.Context, listID string) ([]listimgdom.ListImage, error)
}

// ListImageByIDReader は ListImage を主キーで取得する契約です。
type ListImageByIDReader interface {
	GetByID(ctx context.Context, id string) (listimgdom.ListImage, error)
}

// ListImageObjectSaver は GCS に存在するオブジェクトから ListImage を保存する契約です。
type ListImageObjectSaver interface {
	SaveFromBucketObject(
		ctx context.Context,
		id string,
		listID string,
		bucket string,
		objectPath string,
		size int64,
		displayOrder int,
	) (listimgdom.ListImage, error)
}

// ✅ Create 時に「listId の名前のバケット」を初期化したい場合のオプショナル契約。
type ListImageBucketInitializer interface {
	EnsureListBucket(ctx context.Context, listID string) error
}

// ==============================
// Aggregate / Usecase struct
// ==============================

type ListAggregate struct {
	List   listdom.List           `json:"list"`
	Images []listimgdom.ListImage `json:"images"`
}

// ListUsecase は List と ListImage をまとめて扱います。
type ListUsecase struct {
	listReader  ListReader
	listLister  ListLister  // GET /lists
	listCreator ListCreator // POST /lists (optional)
	listUpdater ListUpdater // PUT/PATCH /lists/{id} (optional)
	listPatcher ListPatcher

	// images
	// - imageReader / imageByIDReader は「画像一覧/単体取得」の source-of-truth 側（推奨: Firestore subcollection）
	imageReader     ListImageReader
	imageByIDReader ListImageByIDReader

	// - imageObjectSaver / imageSignedURLIssuer は「GCS 実体ファイル」の操作
	imageObjectSaver     ListImageObjectSaver
	imageSignedURLIssuer ListImageSignedURLIssuer // signed-url issuer

	// ✅ NEW (for multi-image persistence):
	// Firestore subcollection (/lists/{listId}/images) へ画像レコードを永続化するための port。
	// ※ interface 定義は feature_images.go 側で宣言済み（同一packageなので参照可能）
	listImageRecordRepo ListImageRecordRepository

	// ✅ NEW (for primary image cache update):
	// list 本体の primary image(URL cache) を更新するための port。
	// ※ interface 定義は feature_images.go 側で宣言済み（同一packageなので参照可能）
	listPrimaryImageSetter ListPrimaryImageSetter
}
