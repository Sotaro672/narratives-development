// backend/internal/adapters/out/firestore/image_document_helper_fs.go
package firestore

import (
	"context"
	"errors"
	"strings"
	"time"

	gfs "cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// imageDocument は、List画像とResale画像で共通する
// Firestore上の画像レコードを表します。
//
// OwnerIDには、呼び出し元に応じて次の値が入ります。
//   - List画像: ListID
//   - Resale画像: ResaleID
type imageDocument struct {
	ID           string
	OwnerID      string
	URL          string
	DisplayOrder int
	CreatedAt    time.Time
	CreatedBy    string
	UpdatedAt    *time.Time
	UpdatedBy    *string
}

// normalizeImageDocumentID は、画像ドキュメントIDを正規化して検証します。
//
// FirestoreのドキュメントIDとして不正になり得る次の値は許可しません。
//   - 空文字
//   - "/" を含む値
//   - URL形式の値
func normalizeImageDocumentID(
	imageID string,
) (string, bool) {
	id := strings.TrimSpace(imageID)
	if id == "" {
		return "", false
	}

	if strings.Contains(id, "/") ||
		strings.Contains(id, "://") {
		return "", false
	}

	return id, true
}

// normalizeImageCreatedAt は、作成日時をUTCへ正規化します。
// Zero値の場合は現在時刻を使用します。
func normalizeImageCreatedAt(
	value time.Time,
) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}

	return value.UTC()
}

// normalizeImageUpdatedAt は、更新日時をUTCへ正規化します。
// nilまたはZero値の場合はnilを返します。
func normalizeImageUpdatedAt(
	value *time.Time,
) *time.Time {
	if value == nil || value.IsZero() {
		return nil
	}

	utc := value.UTC()

	return &utc
}

// normalizeImageUpdatedBy は、更新者IDを正規化します。
// nilまたは空文字の場合はnilを返します。
func normalizeImageUpdatedBy(
	value *string,
) *string {
	if value == nil {
		return nil
	}

	updatedBy := strings.TrimSpace(*value)
	if updatedBy == "" {
		return nil
	}

	return &updatedBy
}

// encodeImageDocument は、画像レコードをFirestore保存用データへ変換します。
//
// ownerFieldには次のいずれかを指定します。
//   - List画像: "list_id"
//   - Resale画像: "resale_id"
func encodeImageDocument(
	ownerField string,
	image imageDocument,
) map[string]any {
	data := map[string]any{
		"id":            image.ID,
		ownerField:      image.OwnerID,
		"url":           image.URL,
		"display_order": image.DisplayOrder,
		"created_at":    image.CreatedAt.UTC(),
		"created_by":    image.CreatedBy,
	}

	if image.UpdatedAt != nil &&
		!image.UpdatedAt.IsZero() {
		data["updated_at"] =
			image.UpdatedAt.UTC()
	}

	if image.UpdatedBy != nil {
		updatedBy := strings.TrimSpace(
			*image.UpdatedBy,
		)
		if updatedBy != "" {
			data["updated_by"] = updatedBy
		}
	}

	return data
}

// decodeImageDocument は、Firestoreドキュメントを
// List画像・Resale画像共通のimageDocumentへ変換します。
//
// ownerFieldには次のいずれかを指定します。
//   - List画像: "list_id"
//   - Resale画像: "resale_id"
func decodeImageDocument(
	doc *gfs.DocumentSnapshot,
	fallbackOwnerID string,
	ownerField string,
) (imageDocument, bool) {
	if doc == nil || doc.Ref == nil {
		return imageDocument{}, false
	}

	data := doc.Data()
	if data == nil {
		return imageDocument{}, false
	}

	imageID, ok :=
		normalizeImageDocumentID(doc.Ref.ID)
	if !ok {
		imageID, ok =
			normalizeImageDocumentID(
				asString(data["id"]),
			)
		if !ok {
			return imageDocument{}, false
		}
	}

	ownerID := strings.TrimSpace(
		asString(data[ownerField]),
	)
	if ownerID == "" {
		ownerID = strings.TrimSpace(
			fallbackOwnerID,
		)
	}
	if ownerID == "" {
		return imageDocument{}, false
	}

	createdAt := time.Time{}
	if value, ok := asTime(data["created_at"]); ok {
		createdAt = value
	}
	createdAt =
		normalizeImageCreatedAt(createdAt)

	var updatedAt *time.Time

	if value, ok := asTime(data["updated_at"]); ok {
		updatedAt =
			normalizeImageUpdatedAt(&value)
	}

	var updatedBy *string

	if value := strings.TrimSpace(
		asString(data["updated_by"]),
	); value != "" {
		updatedBy = &value
	}

	return imageDocument{
		ID: imageID,

		OwnerID: ownerID,

		URL: strings.TrimSpace(
			asString(data["url"]),
		),

		DisplayOrder: asInt(
			data["display_order"],
		),

		CreatedAt: createdAt,

		CreatedBy: strings.TrimSpace(
			asString(data["created_by"]),
		),

		UpdatedAt: updatedAt,
		UpdatedBy: updatedBy,
	}, true
}

// listOrderedImageDocuments は、画像サブコレクションを
// display_order、document IDの昇順で取得します。
//
// ドメインエンティティへの変換はdecodeに委譲します。
func listOrderedImageDocuments[T any](
	ctx context.Context,
	collection *gfs.CollectionRef,
	decode func(
		*gfs.DocumentSnapshot,
	) (T, bool),
) ([]T, error) {
	if collection == nil {
		return nil,
			errors.New(
				"firestore image collection is nil",
			)
	}

	if decode == nil {
		return nil,
			errors.New(
				"firestore image decoder is nil",
			)
	}

	it := collection.
		OrderBy(
			"display_order",
			gfs.Asc,
		).
		OrderBy(
			gfs.DocumentID,
			gfs.Asc,
		).
		Documents(ctx)
	defer it.Stop()

	out := make([]T, 0, 8)

	for {
		doc, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}

		if err != nil {
			return nil, err
		}

		item, ok := decode(doc)
		if !ok {
			continue
		}

		out = append(out, item)
	}

	return out, nil
}

// deleteImageDocument は、画像ドキュメントを
// Firestoreトランザクション内で削除します。
//
// notFoundErrがnilの場合は、対象が存在しなくても成功扱いにします。
// notFoundErrが指定されている場合は、そのエラーを返します。
func deleteImageDocument(
	ctx context.Context,
	client *gfs.Client,
	ref *gfs.DocumentRef,
	notFoundErr error,
) error {
	if client == nil {
		return errors.New(
			"firestore client is nil",
		)
	}

	if ref == nil {
		return errors.New(
			"firestore image document reference is nil",
		)
	}

	return client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			tx *gfs.Transaction,
		) error {
			_, err := tx.Get(ref)
			if err != nil {
				if status.Code(err) ==
					codes.NotFound {
					if notFoundErr == nil {
						return nil
					}

					return notFoundErr
				}

				return err
			}

			return tx.Delete(ref)
		},
	)
}
