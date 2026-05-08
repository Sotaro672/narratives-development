// backend/internal/application/usecase/list/feature_primary_image.go

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

// ListImageRecordByIDReader is an optional extended contract for Firestore subcollection.
//
// Expected store:
//
//	/lists/{listId}/images/{imageId}
//
// Firebase Storage migration policy:
// - backend does not resolve GCS objectPath.
// - backend does not fabricate public URL.
// - imageId must be Firestore docID.
// - List.ImageID stores imageId only.
type ListImageRecordByIDReader interface {
	GetByListIDAndID(ctx context.Context, listID string, imageID string) (listdom.ListImage, error)
}

func (uc *ListUsecase) SetPrimaryImage(
	ctx context.Context,
	listID string,
	imageID string,
	now time.Time,
	updatedBy *string,
) (listdom.List, error) {
	if uc == nil {
		return listdom.List{}, usecase.ErrNotSupported("List.SetPrimaryImage")
	}

	if uc.listPatcher == nil {
		return listdom.List{}, usecase.ErrNotSupported("List.SetPrimaryImage")
	}

	lid := strings.TrimSpace(listID)
	iid := strings.TrimSpace(imageID)

	if lid == "" {
		return listdom.List{}, listdom.ErrInvalidID
	}

	if iid == "" {
		return listdom.List{}, listdom.ErrEmptyImageID
	}

	// Firebase Storage 移行後:
	// - URL は受け付けない
	// - objectPath も受け付けない
	// - imageId(docID) のみ受け付ける
	if strings.Contains(iid, "/") || strings.Contains(iid, "://") {
		return listdom.List{}, listdom.ErrInvalidImageID
	}

	if uc.listImageRecordRepo == nil {
		return listdom.List{}, usecase.ErrNotSupported("List.SetPrimaryImage.RecordRepo")
	}

	reader, ok := uc.listImageRecordRepo.(ListImageRecordByIDReader)
	if !ok || reader == nil {
		return listdom.List{}, usecase.ErrNotSupported("List.SetPrimaryImage.RecordRepo.GetByListIDAndID")
	}

	img, err := reader.GetByListIDAndID(ctx, lid, iid)
	if err != nil {
		return listdom.List{}, err
	}

	if img.ID == "" {
		return listdom.List{}, listdom.ErrInvalidImageID
	}

	if img.ListID != "" && img.ListID != lid {
		return listdom.List{}, errors.New("list: image belongs to other list")
	}

	if strings.TrimSpace(img.URL) == "" {
		return listdom.List{}, listdom.ErrInvalidListImageURL
	}

	if strings.TrimSpace(img.ObjectPath) == "" {
		return listdom.List{}, listdom.ErrInvalidListImageObjectPath
	}

	if now.IsZero() {
		now = time.Now().UTC()
	}

	patch := listdom.ListPatch{
		ImageID:   &iid,
		UpdatedAt: ptrTime(now.UTC()),
		UpdatedBy: updatedBy,
	}

	updated, err := uc.listPatcher.Update(ctx, lid, patch)
	if err != nil {
		return listdom.List{}, err
	}

	log.Printf(
		"[list_usecase] primaryImage set listID=%s imageID=%s imageURL=%q objectPath=%q",
		lid,
		iid,
		img.URL,
		img.ObjectPath,
	)

	return updated, nil
}

func ptrTime(t time.Time) *time.Time {
	tt := t
	return &tt
}
