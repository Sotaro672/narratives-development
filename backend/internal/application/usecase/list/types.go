// backend/internal/application/usecase/list/types.go
//
// Responsibility:
// - List Usecase の「型定義」を集約する（DTO / Port / Aggregate / Usecase struct）。
// - ビジネス処理（メソッド実装）は置かず、他ファイルから参照される共通定義のみを保持する。
//
// Firebase Storage migration policy:
// - backend は GCS signed URL を発行しない
// - backend は GCS bucket / GCS object / GCS metadata を扱わない
// - frontend が Firebase Storage へ直接 upload する
// - frontend が取得した downloadURL / objectPath / fileName / contentType / size を backend に送る
// - backend は domain/list.ListImage を Firestore record として保存・取得・削除する
//
// Features:
// - List / ListImage の各 Port（Reader/Lister/Creator/...）
// - ListUsecase / ListAggregate の定義
package list

import (
	"context"
	"time"

	listdom "narratives/internal/domain/list"
)

// ==============================
// Ports: List
// ==============================

// ListReader は List 単体取得の契約です。
type ListReader interface {
	GetByID(ctx context.Context, id string) (listdom.List, error)
}

// ListLister は List 一覧取得の契約です（GET /lists 用）。
type ListLister interface {
	List(
		ctx context.Context,
		filter listdom.Filter,
		sort listdom.Sort,
		page listdom.Page,
	) (listdom.PageResult[listdom.List], error)

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

// ListPatcher は List を patch で部分更新する契約です。
type ListPatcher interface {
	Update(ctx context.Context, id string, patch listdom.ListPatch) (listdom.List, error)
}

// ListPrimaryImageSetter updates list's primary image docID.
//
// Firebase Storage 移行後:
// - List.ImageID stores primary imageId, not URL.
// - imageId == "" means unset.
type ListPrimaryImageSetter interface {
	// SetPrimaryImageID sets list.imageId to imageID.
	// imageID is /lists/{listId}/images/{imageId} docID.
	SetPrimaryImageID(ctx context.Context, listID string, imageID string, now time.Time) error

	// SetPrimaryImageIfEmpty sets primary image only when current primary is empty.
	SetPrimaryImageIfEmpty(ctx context.Context, listID string, imageID string, now time.Time) error
}

// ==============================
// Ports: ListImage
// ==============================

// ListImageReader は ListID に紐づく ListImage 一覧の取得契約です。
//
// Expected source of truth:
// - Firestore subcollection: /lists/{listId}/images/{imageId}
//
// Firebase Storage 移行後:
// - ListImage.URL is Firebase Storage downloadURL.
// - ListImage.ObjectPath is Firebase Storage objectPath.
// - backend does not generate URLs from bucket/objectPath.
type ListImageReader interface {
	ListByListID(ctx context.Context, listID string) ([]listdom.ListImage, error)
}

// ListImageByIDReader は ListImage を imageId で取得する契約です。
type ListImageByIDReader interface {
	GetByID(ctx context.Context, imageID string) (listdom.ListImage, error)
}

// ListImageRecordRepository is a persistence port for list images.
//
// Expected target:
// - /lists/{listId}/images/{imageId}
// - docID = imageId
//
// Firebase Storage migration:
// - Upsert stores metadata sent from frontend after Firebase Storage upload.
// - Delete deletes Firestore record only.
// - Storage object deletion is handled by frontend deleteObject(), or by a future Firebase Admin endpoint.
type ListImageRecordRepository interface {
	Upsert(ctx context.Context, img listdom.ListImage) (listdom.ListImage, error)
	ListByListID(ctx context.Context, listID string) ([]listdom.ListImage, error)
	GetByID(ctx context.Context, imageID string) (listdom.ListImage, error)
	Delete(ctx context.Context, listID string, imageID string) error
}

// ==============================
// Aggregate / Usecase struct
// ==============================

type ListAggregate struct {
	List   listdom.List        `json:"list"`
	Images []listdom.ListImage `json:"images"`
}

// ListUsecase は List と ListImage をまとめて扱います。
type ListUsecase struct {
	listReader  ListReader
	listLister  ListLister
	listCreator ListCreator
	listUpdater ListUpdater
	listPatcher ListPatcher

	imageReader     ListImageReader
	imageByIDReader ListImageByIDReader

	// Firestore subcollection repository for list image records.
	listImageRecordRepo ListImageRecordRepository

	// list 本体の primary imageId を更新するための port。
	listPrimaryImageSetter ListPrimaryImageSetter
}
