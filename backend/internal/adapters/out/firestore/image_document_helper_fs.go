// backend/internal/adapters/out/firestore/image_document_helper_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
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

// imageDocumentRecord はFirestore上の共通フィールドを
// 型変換せずに受け取るためのrecordです。
// owner fieldはlist_id / resale_idで動的に変わるため、
// decodeImageDocument内で個別に検証します。
type imageDocumentRecord struct {
	ID           string     `firestore:"id"`
	URL          string     `firestore:"url"`
	DisplayOrder int        `firestore:"display_order"`
	CreatedAt    time.Time  `firestore:"created_at"`
	CreatedBy    string     `firestore:"created_by"`
	UpdatedAt    *time.Time `firestore:"updated_at"`
	UpdatedBy    *string    `firestore:"updated_by"`
}

// normalizeImageDocumentID は、書き込み・入力値として渡された
// 画像ドキュメントIDを正規化して検証します。
//
// FirestoreのドキュメントIDとして不正になり得る次の値は許可しません。
//   - 空文字
//   - "/" を含む値
//   - URL形式の値
//
// NOTE:
// Firestore read時にはこの関数による補正値を採用しません。
// doc.Ref.IDが正規化によって変化する場合は破損データとして扱います。
func normalizeImageDocumentID(imageID string) (string, bool) {
	id := strings.TrimSpace(imageID)
	if id == "" {
		return "", false
	}
	if strings.Contains(id, "/") || strings.Contains(id, "://") {
		return "", false
	}
	return id, true
}

// normalizeImageCreatedAt はwrite側で使用する日時正規化です。
// Zero値の場合は現在時刻を使用します。
// Firestore read時には使用しません。
func normalizeImageCreatedAt(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}
	return value.UTC()
}

// normalizeImageUpdatedAt はwrite側で使用する日時正規化です。
// nilまたはZero値の場合はnilを返します。
// Firestore read時には使用しません。
func normalizeImageUpdatedAt(value *time.Time) *time.Time {
	if value == nil || value.IsZero() {
		return nil
	}
	utc := value.UTC()
	return &utc
}

// normalizeImageUpdatedBy はwrite側で使用する更新者ID正規化です。
// nilまたは空文字の場合はnilを返します。
// Firestore read時には使用しません。
func normalizeImageUpdatedBy(value *string) *string {
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
func encodeImageDocument(ownerField string, image imageDocument) map[string]any {
	data := map[string]any{
		"id":            image.ID,
		ownerField:      image.OwnerID,
		"url":           image.URL,
		"display_order": image.DisplayOrder,
		"created_at":    image.CreatedAt.UTC(),
		"created_by":    image.CreatedBy,
	}

	if image.UpdatedAt != nil && !image.UpdatedAt.IsZero() {
		data["updated_at"] = image.UpdatedAt.UTC()
	}

	if image.UpdatedBy != nil {
		updatedBy := strings.TrimSpace(*image.UpdatedBy)
		if updatedBy != "" {
			data["updated_by"] = updatedBy
		}
	}

	return data
}

// decodeImageDocument はFirestoreドキュメントを
// List画像・Resale画像共通のimageDocumentへ厳密に変換します。
//
// read時には以下を行いません。
//   - docIdからidへの補正
//   - id fieldからdocIdへのfallback
//   - ownerIdのfallback
//   - string trimによる補正
//   - 型coercion
//   - createdAtの現在時刻補完
//   - updatedAt / updatedByの正規化
//
// expectedOwnerIDは親ドキュメントから確定しているListID / ResaleIDです。
// Firestoreに保存されたowner fieldと一致しない場合は不正データとして扱います。
func decodeImageDocument(doc *gfs.DocumentSnapshot, expectedOwnerID string, ownerField string) (imageDocument, bool) {
	if doc == nil || doc.Ref == nil {
		return imageDocument{}, false
	}
	if ownerField == "" || expectedOwnerID == "" {
		return imageDocument{}, false
	}

	imageID := doc.Ref.ID
	normalizedImageID, ok := normalizeImageDocumentID(imageID)
	if !ok || normalizedImageID != imageID {
		return imageDocument{}, false
	}

	var rec imageDocumentRecord
	if err := doc.DataTo(&rec); err != nil {
		return imageDocument{}, false
	}

	// id fieldとdocument IDは同一でなければならない。
	// id fieldが欠落していてもdoc.Ref.IDへfallbackしない。
	if rec.ID == "" || rec.ID != imageID {
		return imageDocument{}, false
	}

	data := doc.Data()
	if data == nil {
		return imageDocument{}, false
	}

	rawOwnerID, exists := data[ownerField]
	if !exists || rawOwnerID == nil {
		return imageDocument{}, false
	}

	ownerID, ok := rawOwnerID.(string)
	if !ok || ownerID == "" {
		return imageDocument{}, false
	}

	// 親pathから確定しているownerIdとDB内ownerIdは一致必須。
	if ownerID != expectedOwnerID {
		return imageDocument{}, false
	}

	// createdAtは必須。
	// Zero値を現在時刻へ補完しない。
	if rec.CreatedAt.IsZero() {
		return imageDocument{}, false
	}

	// updatedAtはoptionalだが、存在するならZero値を許可しない。
	if rec.UpdatedAt != nil && rec.UpdatedAt.IsZero() {
		return imageDocument{}, false
	}

	// updatedByはoptionalだが、存在する空文字をnilへ補正しない。
	if rec.UpdatedBy != nil && *rec.UpdatedBy == "" {
		return imageDocument{}, false
	}

	return imageDocument{
		ID:           imageID,
		OwnerID:      ownerID,
		URL:          rec.URL,
		DisplayOrder: rec.DisplayOrder,
		CreatedAt:    rec.CreatedAt,
		CreatedBy:    rec.CreatedBy,
		UpdatedAt:    rec.UpdatedAt,
		UpdatedBy:    rec.UpdatedBy,
	}, true
}

// listOrderedImageDocuments は画像サブコレクションを
// display_order、document IDの昇順で取得します。
//
// decodeに失敗したドキュメントは黙って除外せず、
// Firestore上の不正データとしてエラーを返します。
func listOrderedImageDocuments[T any](
	ctx context.Context,
	collection *gfs.CollectionRef,
	decode func(*gfs.DocumentSnapshot) (T, bool),
) ([]T, error) {
	if collection == nil {
		return nil, errors.New("firestore image collection is nil")
	}
	if decode == nil {
		return nil, errors.New("firestore image decoder is nil")
	}

	it := collection.
		OrderBy("display_order", gfs.Asc).
		OrderBy(gfs.DocumentID, gfs.Asc).
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
			if doc == nil || doc.Ref == nil {
				return nil, errors.New("failed to decode firestore image document")
			}
			return nil, fmt.Errorf("failed to decode firestore image document: %s", doc.Ref.ID)
		}

		out = append(out, item)
	}

	return out, nil
}

// deleteImageDocument は画像ドキュメントを
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
		return errors.New("firestore client is nil")
	}
	if ref == nil {
		return errors.New("firestore image document reference is nil")
	}

	return client.RunTransaction(ctx, func(ctx context.Context, tx *gfs.Transaction) error {
		_, err := tx.Get(ref)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				if notFoundErr == nil {
					return nil
				}
				return notFoundErr
			}
			return err
		}

		return tx.Delete(ref)
	})
}
