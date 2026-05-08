// backend/internal/application/usecase/list/feature_images.go
//
// Responsibility:
// - ListImage の保存を提供する。
// - 保存後に Firestore の /lists/{listId}/images サブコレクションへ永続化する（複数画像対応）。
// - 画像削除は Firestore record の削除を usecase 経由で提供する。
// - Firebase Storage の実体削除は frontend 側 deleteObject、または将来 Firebase Admin SDK 専用 endpoint に寄せる。
//
// Firebase Storage migration policy:
// - backend は GCS signed URL を発行しない
// - backend は GCS object / bucket / metadata を扱わない
// - frontend が Firebase Storage へ直接 upload する
// - frontend が取得した downloadURL / objectPath / fileName / contentType / size を backend に送る
// - backend は domain/list.ListImage を Firestore record として保存・取得・削除する
//
// Features:
// - SaveImage
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
)

// --- optional capability interfaces (type-assert) ---

// ListImageRecordDeleter deletes Firestore record (/lists/{listId}/images/{imageId}).
type ListImageRecordDeleter interface {
	Delete(ctx context.Context, listID string, imageID string) error
}

// ListReader is already your port in this package; we only need GetByID here.
type listReaderForPrimary interface {
	GetByID(ctx context.Context, id string) (listdom.List, error)
}

// SaveImage persists an already-uploaded Firebase Storage image record.
//
// Expected flow:
// 1) frontend uploads file to Firebase Storage.
// 2) frontend calls getDownloadURL().
// 3) frontend sends image metadata to backend.
// 4) backend validates and persists /lists/{listId}/images/{imageId} record.
//
// Required image fields:
// - ID: Firestore docID for image record
// - ListID: list document id
// - URL: Firebase Storage downloadURL
// - ObjectPath: Firebase Storage object path
// - FileName: original/safe file name
// - ContentType: image/*
// - CreatedBy: actor/member id
//
// Canonical Firebase Storage objectPath:
//
//	lists/{listId}/images/{imageId}/{fileName}
func (uc *ListUsecase) SaveImage(
	ctx context.Context,
	img listdom.ListImage,
) (listdom.ListImage, error) {
	if uc == nil {
		return listdom.ListImage{}, usecase.ErrNotSupported("List.SaveImage")
	}

	if uc.listImageRecordRepo == nil {
		return listdom.ListImage{}, usecase.ErrNotSupported("List.SaveImage.RecordRepo")
	}

	img.ID = strings.TrimSpace(img.ID)
	img.ListID = strings.TrimSpace(img.ListID)
	img.URL = strings.TrimSpace(img.URL)
	img.ObjectPath = strings.TrimLeft(strings.TrimSpace(img.ObjectPath), "/")
	img.FileName = strings.TrimSpace(img.FileName)
	img.ContentType = strings.ToLower(strings.TrimSpace(img.ContentType))
	img.CreatedBy = strings.TrimSpace(img.CreatedBy)

	if img.ListID == "" {
		return listdom.ListImage{}, listdom.ErrInvalidListImageListID
	}

	if img.ID == "" {
		return listdom.ListImage{}, listdom.ErrInvalidListImageID
	}

	if strings.Contains(img.ID, "/") {
		return listdom.ListImage{}, usecase.ErrInvalidArgument("invalid_image_id")
	}

	if img.DisplayOrder < 0 {
		img.DisplayOrder = 0
	}

	if img.CreatedAt.IsZero() {
		img.CreatedAt = time.Now().UTC()
	} else {
		img.CreatedAt = img.CreatedAt.UTC()
	}

	expectedPrefix := "lists/" + img.ListID + "/images/" + img.ID + "/"
	if !strings.HasPrefix(img.ObjectPath, expectedPrefix) {
		return listdom.ListImage{}, usecase.ErrInvalidArgument("objectPath_not_canonical")
	}

	if err := img.Validate(); err != nil {
		return listdom.ListImage{}, err
	}

	saved, err := uc.listImageRecordRepo.Upsert(ctx, img)
	if err != nil {
		return listdom.ListImage{}, err
	}

	log.Printf(
		"[list_usecase] firebase listImage saved listID=%s imageID=%s url=%q objectPath=%s size=%d displayOrder=%d",
		saved.ListID,
		saved.ID,
		saved.URL,
		saved.ObjectPath,
		saved.Size,
		saved.DisplayOrder,
	)

	return saved, nil
}

// DeleteImage deletes a list image by IDs.
//
// Firebase Storage migration behavior:
// 1) Delete Firestore record (/lists/{listId}/images/{imageId}).
// 2) Do NOT delete Firebase Storage object here.
//   - frontend may call deleteObject(ref(storage, objectPath))
//   - or a future backend endpoint can use Firebase Admin SDK
//
// 3) If list.imageId(primaryImageID) == imageId, clear primary.
//
// imageID is Firestore docID only. URL/objectPath is not accepted.
func (uc *ListUsecase) DeleteImage(ctx context.Context, listID string, imageID string) error {
	listID = strings.TrimSpace(listID)
	imageID = strings.TrimSpace(imageID)

	if listID == "" {
		return listdom.ErrInvalidListImageListID
	}

	if imageID == "" {
		return listdom.ErrInvalidListImageID
	}

	if strings.Contains(imageID, "/") {
		return usecase.ErrInvalidArgument("invalid_image_id")
	}

	if uc == nil {
		return usecase.ErrNotSupported("List.DeleteImage")
	}

	if uc.listImageRecordRepo == nil {
		return usecase.ErrNotSupported("List.DeleteImage.RecordRepo")
	}

	// 1) Firestore record delete
	if deleter, ok := any(uc.listImageRecordRepo).(ListImageRecordDeleter); ok {
		if err := deleter.Delete(ctx, listID, imageID); err != nil {
			if !errors.Is(err, listdom.ErrListImageNotFound) {
				log.Printf("[list_usecase] delete image record failed listID=%s imageID=%s err=%v", listID, imageID, err)
				return err
			}
		}
	} else {
		return usecase.ErrNotSupported("List.DeleteImage.RecordRepo.Delete")
	}

	// 2) Firebase Storage object delete is intentionally not handled here.
	// backend no longer has a GCS object deleter in this flow.

	// 3) primary fix
	// if list.imageId == deleted imageID, unset it.
	if uc.listPrimaryImageSetter != nil && uc.listReader != nil {
		if r, ok := any(uc.listReader).(listReaderForPrimary); ok && r != nil {
			l, err := r.GetByID(ctx, listID)
			if err == nil && l.ImageID == imageID {
				now := time.Now().UTC()
				_ = uc.listPrimaryImageSetter.SetPrimaryImageID(ctx, listID, "", now)
			}
		}
	}

	log.Printf("[list_usecase] delete image record done listID=%s imageID=%s", listID, imageID)
	return nil
}
