// backend/internal/application/usecase/list/feature_images.go
//
// Responsibility:
// - ListImage の保存（GCS object -> domain ListImage）を提供する。
// - 保存後に Firestore の /lists/{listId}/images サブコレクションへ永続化する（複数画像対応）。
// - 画像削除（Firestore record + GCS object）を usecase 経由で提供する（handler から deleter を撤去する方針）。
// - URL 解決結果をログへ出す（運用上の検証用）。
//
// Features:
// - SaveImageFromGCS
// - DeleteImage
package list

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	usecase "narratives/internal/application/usecase"
	listdom "narratives/internal/domain/list"
	listimgdom "narratives/internal/domain/listImage"
)

// --- optional capability interfaces (type-assert) ---

// ListImageRecordDeleter deletes Firestore record (/lists/{listId}/images/{imageId}).
type ListImageRecordDeleter interface {
	Delete(ctx context.Context, listID string, imageID string) error
}

// ListImageObjectDeleter deletes underlying object (GCS object) by IDs.
// Bucket resolution must be done inside the adapter (env/DI).
type ListImageObjectDeleter interface {
	Delete(ctx context.Context, listID string, imageID string) error
}

// ListReader is already your port in this package; we only need GetByID here.
type listReaderForPrimary interface {
	GetByID(ctx context.Context, id string) (listdom.List, error)
}

// SaveImageFromGCS finalizes an already-uploaded GCS object:
//
// 1) Validate inputs + canonical objectPath (Policy A).
// 2) Normalize/validate metadata on GCS and build domain ListImage via imageObjectSaver (GCS adapter).
// 3) Persist the ListImage record into Firestore subcollection via listImageRecordRepo.
//
// IMPORTANT:
// - This method assumes the actual bytes are already uploaded to GCS (signed URL PUT done).
// - Policy A canonical objectPath MUST be: "lists/{listId}/images/{imageId}"
func (uc *ListUsecase) SaveImageFromGCS(
	ctx context.Context,
	imageID string,
	listID string,
	bucket string,
	objectPath string,
	size int64,
	displayOrder int,
) (listimgdom.ListImage, error) {
	// required deps
	if uc.imageObjectSaver == nil {
		return listimgdom.ListImage{}, usecase.ErrNotSupported("List.SaveImageFromGCS")
	}
	if uc.listImageRecordRepo == nil {
		return listimgdom.ListImage{}, usecase.ErrNotSupported("List.SaveImageFromGCS.RecordRepo")
	}

	// normalize inputs
	listID = strings.TrimSpace(listID)
	imageID = strings.TrimSpace(imageID)
	bucket = strings.TrimSpace(bucket)
	objectPath = strings.TrimLeft(strings.TrimSpace(objectPath), "/")

	// required fields
	if listID == "" {
		return listimgdom.ListImage{}, listimgdom.ErrInvalidListID
	}
	if imageID == "" {
		return listimgdom.ListImage{}, listimgdom.ErrInvalidID
	}
	// imageId は docID 前提（URL/objectPath を入れない）
	if strings.Contains(imageID, "/") {
		return listimgdom.ListImage{}, usecase.ErrInvalidArgument("invalid_image_id")
	}
	if bucket == "" {
		// handler は 400 にしている想定だが、usecase でも守る
		return listimgdom.ListImage{}, usecase.ErrInvalidArgument("bucket is required")
	}
	if objectPath == "" {
		return listimgdom.ListImage{}, usecase.ErrInvalidArgument("objectPath is required")
	}

	// Guard: displayOrder should not be negative
	if displayOrder < 0 {
		displayOrder = 0
	}

	// Policy A canonical objectPath validation:
	// MUST be "lists/{listId}/images/{imageId}"
	expectedPrefix := "lists/" + listID + "/images/"
	if !strings.HasPrefix(objectPath, expectedPrefix) {
		return listimgdom.ListImage{}, usecase.ErrInvalidArgument("objectPath_not_canonical")
	}
	if objectPath != expectedPrefix+imageID {
		return listimgdom.ListImage{}, usecase.ErrInvalidArgument("objectPath_id_mismatch")
	}

	// 1) GCS object -> domain ListImage (also updates GCS metadata best-effort)
	img, err := uc.imageObjectSaver.SaveFromBucketObject(
		ctx,
		imageID,    // ✅ imageId (docID)
		listID,     // ✅ listId
		bucket,     // ✅ bucket (required)
		objectPath, // ✅ canonical objectPath
		size,
		displayOrder,
	)
	if err != nil {
		return listimgdom.ListImage{}, err
	}

	// Derive imageID from objectPath for logging/sanity (Policy A only)
	derivedImageID := extractImageIDFromCanonicalObjectPath(objectPath, listID)

	// 2) Persist to Firestore subcollection (multi-image feature)
	saved, err := uc.listImageRecordRepo.Upsert(ctx, img)
	if err != nil {
		return listimgdom.ListImage{}, err
	}
	img = saved

	// NOTE:
	// Policy A 方針では、primary の更新は専用 endpoint（SetPrimaryImage）でのみ行う。
	// ここでは副作用を避けるため primary を触らない。

	log.Printf(
		"[list_usecase] listImage finalized saved=%t url=%q listID=%s imageID(in)=%s imageID(derived)=%s img.ID=%s img.ObjectPath=%s bucket=%s size=%d displayOrder=%d",
		strings.TrimSpace(img.URL) != "",
		strings.TrimSpace(img.URL),
		listID,
		imageID,
		derivedImageID,
		strings.TrimSpace(img.ID),
		strings.TrimSpace(img.ObjectPath),
		bucket,
		img.Size,
		img.DisplayOrder,
	)

	return img, nil
}

// DeleteImage deletes a list image by IDs (Policy A).
//
// ✅ Important behavior:
// 1) Delete Firestore record (/lists/{listId}/images/{imageId}) FIRST (must).
// 2) Delete GCS object best-effort (ErrObjectNotExist is treated as success).
// 3) If list.image_id(primaryImageID) == imageId, clear primary (or you can implement "pick next").
//
// - imageID は Firestore docID（"63b5..."）のみを受け付け、URL/objectPath は受け付けない。
func (uc *ListUsecase) DeleteImage(ctx context.Context, listID string, imageID string) error {
	listID = strings.TrimSpace(listID)
	imageID = strings.TrimSpace(imageID)

	if listID == "" {
		return listimgdom.ErrInvalidListID
	}
	if imageID == "" {
		return listimgdom.ErrInvalidID
	}
	if strings.Contains(imageID, "/") {
		return usecase.ErrInvalidArgument("invalid_image_id")
	}

	// required deps
	if uc.listImageRecordRepo == nil {
		return usecase.ErrNotSupported("List.DeleteImage.RecordRepo")
	}

	// 1) Firestore record delete (MUST)
	if deleter, ok := any(uc.listImageRecordRepo).(ListImageRecordDeleter); ok {
		if err := deleter.Delete(ctx, listID, imageID); err != nil {
			// if already not found, treat as success (idempotent)
			if !errors.Is(err, listimgdom.ErrNotFound) {
				log.Printf("[list_usecase] delete image record failed listID=%s imageID=%s err=%v", listID, imageID, err)
				return err
			}
		}
	} else {
		return usecase.ErrNotSupported("List.DeleteImage.RecordRepo.Delete")
	}

	// 2) GCS delete (best-effort)
	// bucket resolution must be inside adapter (env/DI)
	if objDel, ok := any(uc.imageObjectSaver).(ListImageObjectDeleter); ok && objDel != nil {
		if err := objDel.Delete(ctx, listID, imageID); err != nil {
			// best-effort: DO NOT fail user-facing deletion because Firestore is already correct
			log.Printf("[list_usecase] delete gcs object failed listID=%s imageID=%s err=%v", listID, imageID, err)
		}
	}

	// 3) primary fix (if wired)
	// if list.image_id == deleted imageID, unset it.
	if uc.listPrimaryImageSetter != nil && uc.listReader != nil {
		if r, ok := any(uc.listReader).(listReaderForPrimary); ok && r != nil {
			l, err := r.GetByID(ctx, listID)
			if err == nil {
				if strings.TrimSpace(l.ImageID) == imageID {
					now := time.Now().UTC()
					_ = uc.listPrimaryImageSetter.SetPrimaryImageID(ctx, listID, "", now)
				}
			}
		}
	}

	log.Printf("[list_usecase] delete image done listID=%s imageID=%s", listID, imageID)
	return nil
}

// extractImageIDFromCanonicalObjectPath returns imageId from Policy A canonical objectPath.
//
// Canonical:
//
//	lists/{listId}/images/{imageId}
func extractImageIDFromCanonicalObjectPath(objectPath string, listID string) string {
	p := strings.TrimLeft(strings.TrimSpace(objectPath), "/")
	if p == "" {
		return ""
	}
	parts := strings.Split(p, "/")

	// must be: ["lists", "{listId}", "images", "{imageId}"]
	if len(parts) != 4 {
		return ""
	}
	if strings.TrimSpace(parts[0]) != "lists" {
		return ""
	}
	if strings.TrimSpace(parts[1]) != strings.TrimSpace(listID) {
		return ""
	}
	if strings.TrimSpace(parts[2]) != "images" {
		return ""
	}
	return strings.TrimSpace(parts[3])
}
