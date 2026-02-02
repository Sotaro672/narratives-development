// backend/internal/application/usecase/list/feature_images.go
//
// Responsibility:
// - ListImage の保存（GCS object -> domain ListImage）を提供する。
// - 保存後に Firestore の /lists/{listId}/images サブコレクションへ永続化する（複数画像対応）。
// - 必要に応じて list 本体の primary(imageId=URL cache) を更新する。
// - URL 解決結果をログへ出す（運用上の検証用）。
//
// Features:
// - SaveImageFromGCS
package list

import (
	"context"
	"log"
	"strings"
	"time"

	usecase "narratives/internal/application/usecase"
	listimgdom "narratives/internal/domain/listImage"
)

// ListImageRecordRepository is a persistence port for list images (Firestore subcollection).
// Expected target:
// - /lists/{listId}/images/{imageId}
//
// NOTE:
// - This is intentionally small; implement it in adapters/out/firestore.
// - imageId should typically be derived from objectPath "{listId}/{imageId}/{fileName}".
type ListImageRecordRepository interface {
	// Upsert stores the ListImage record (idempotent).
	// Recommended behavior:
	// - docID = imageId (or img.ID if you choose to store objectPath as ID)
	Upsert(ctx context.Context, img listimgdom.ListImage) (listimgdom.ListImage, error)
}

// ListPrimaryImageSetter updates list's primary image URL cache (List.ImageID).
// Recommended behavior:
// - If list.imageId is empty, set it to imageURL
// - Or if displayOrder==0, set it to imageURL
type ListPrimaryImageSetter interface {
	// SetPrimaryImageIfEmpty sets primary image only when current primary is empty.
	SetPrimaryImageIfEmpty(ctx context.Context, listID string, imageURL string, now time.Time) error
}

// SaveImageFromGCS finalizes an already-uploaded GCS object:
// 1) Normalize/validate metadata on GCS and build domain ListImage via imageObjectSaver (GCS adapter).
// 2) Persist the ListImage record into Firestore subcollection via listImageRecordRepo (new).
// 3) Optionally set List.ImageID (primary URL cache) when empty.
//
// IMPORTANT:
// - This method assumes the actual bytes are already uploaded to GCS (signed URL PUT done).
func (uc *ListUsecase) SaveImageFromGCS(
	ctx context.Context,
	id string,
	listID string,
	bucket string,
	objectPath string,
	size int64,
	displayOrder int,
) (listimgdom.ListImage, error) {
	if uc.imageObjectSaver == nil {
		return listimgdom.ListImage{}, usecase.ErrNotSupported("List.SaveImageFromGCS")
	}

	listID = strings.TrimSpace(listID)
	if listID == "" {
		return listimgdom.ListImage{}, listimgdom.ErrInvalidListID
	}

	// Guard: displayOrder should not be negative
	if displayOrder < 0 {
		displayOrder = 0
	}

	// 1) GCS object -> domain ListImage (also updates GCS metadata best-effort)
	img, err := uc.imageObjectSaver.SaveFromBucketObject(
		ctx,
		strings.TrimSpace(id),
		listID,
		strings.TrimSpace(bucket),
		strings.TrimSpace(objectPath),
		size,
		displayOrder,
	)
	if err != nil {
		return listimgdom.ListImage{}, err
	}

	// Resolve imageId (for logging / Firestore doc id decision)
	// Prefer objectPath if provided, otherwise derive from img.ID when it's an objectPath.
	derivedImageID := ""
	if op := strings.TrimSpace(objectPath); op != "" {
		derivedImageID = extractImageIDFromObjectPath(op)
	}
	if derivedImageID == "" {
		derivedImageID = extractImageIDFromObjectPath(strings.TrimSpace(img.ID))
	}

	// 2) Persist to Firestore subcollection (recommended for multi-image feature)
	if uc.listImageRecordRepo == nil {
		// If you want to allow running without Firestore persistence, change this to just a log.
		// For multi-image feature, it's better to fail fast.
		return listimgdom.ListImage{}, usecase.ErrNotSupported("List.SaveImageFromGCS: listImageRecordRepo is nil")
	}

	saved, err := uc.listImageRecordRepo.Upsert(ctx, img)
	if err != nil {
		return listimgdom.ListImage{}, err
	}
	img = saved

	// 3) Optionally set list primary image cache (only if empty)
	now := time.Now().UTC()
	if uc.listPrimaryImageSetter != nil {
		// Only set when empty (safe, avoids overwriting primary unintentionally)
		_ = uc.listPrimaryImageSetter.SetPrimaryImageIfEmpty(ctx, listID, strings.TrimSpace(img.URL), now)
	}

	log.Printf(
		"[list_usecase] listImage finalized saved=%t url=%q listID=%s imageID=%s objectPath=%s bucket=%s size=%d displayOrder=%d",
		strings.TrimSpace(img.URL) != "",
		strings.TrimSpace(img.URL),
		listID,
		derivedImageID,
		strings.TrimSpace(objectPath),
		strings.TrimSpace(bucket),
		img.Size,
		img.DisplayOrder,
	)

	return img, nil
}

// extractImageIDFromObjectPath expects "{listId}/{imageId}/{fileName}" and returns imageId.
func extractImageIDFromObjectPath(objectPath string) string {
	p := strings.TrimLeft(strings.TrimSpace(objectPath), "/")
	if p == "" {
		return ""
	}
	parts := strings.Split(p, "/")
	if len(parts) < 2 {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
