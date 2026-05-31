// backend/internal/application/usecase/list_usecase.go
package usecase

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	listdom "narratives/internal/domain/list"
)

type ListUsecase struct {
	listRepo  listdom.Repository
	imageRepo listdom.ImageRepository
}

func NewListUsecase(
	listRepo listdom.Repository,
	imageRepo listdom.ImageRepository,
) *ListUsecase {
	return &ListUsecase{
		listRepo:  listRepo,
		imageRepo: imageRepo,
	}
}

func generateReadableID(listID string, createdAt time.Time) string {
	t := createdAt
	if t.IsZero() {
		t = time.Now().UTC()
	}

	date := t.UTC().Format("20060102")

	base := listID
	if base == "" {
		base = fmt.Sprintf("noid-%d", time.Now().UTC().UnixNano())
	}

	sum := sha1.Sum([]byte(base))
	hex6 := hex.EncodeToString(sum[:])
	if len(hex6) > 6 {
		hex6 = hex6[:6]
	}

	return fmt.Sprintf("L-%s-%s", date, hex6)
}

func (uc *ListUsecase) Create(
	ctx context.Context,
	item listdom.List,
) (listdom.List, error) {
	if uc == nil || uc.listRepo == nil {
		return listdom.List{}, ErrNotSupported("List.Create")
	}

	created, err := uc.listRepo.Create(ctx, item)
	if err != nil {
		return listdom.List{}, err
	}

	if created.ReadableID == "" {
		rid := generateReadableID(created.ID, created.CreatedAt)
		now := time.Now().UTC()

		created.ReadableID = rid
		created.UpdatedAt = &now

		updated, err := uc.listRepo.Update(ctx, created.ID, created)
		if err == nil {
			created = updated
		}
	}

	return created, nil
}

func (uc *ListUsecase) Update(
	ctx context.Context,
	item listdom.List,
) (listdom.List, error) {
	if uc == nil || uc.listRepo == nil {
		return listdom.List{}, ErrNotSupported("List.Update")
	}

	id := item.ID
	if id == "" {
		return listdom.List{}, listdom.ErrInvalidID
	}

	item.ID = id

	return uc.listRepo.Update(ctx, id, item)
}

func (uc *ListUsecase) Delete(ctx context.Context, id string) error {
	if uc == nil || uc.listRepo == nil {
		return ErrNotSupported("List.Delete")
	}

	if id == "" {
		return listdom.ErrInvalidID
	}

	return uc.listRepo.Delete(ctx, id)
}

func (uc *ListUsecase) CreateImage(
	ctx context.Context,
	img listdom.ListImage,
) (listdom.ListImage, error) {
	if uc == nil {
		return listdom.ListImage{}, ErrNotSupported("List.CreateImage")
	}

	if uc.imageRepo == nil {
		return listdom.ListImage{}, ErrNotSupported("List.CreateImage.ImageRepo")
	}

	if img.ListID == "" {
		return listdom.ListImage{}, listdom.ErrInvalidListImageListID
	}

	if img.ID == "" {
		return listdom.ListImage{}, listdom.ErrInvalidListImageID
	}

	if strings.Contains(img.ID, "/") || strings.Contains(img.ID, "://") {
		return listdom.ListImage{}, ErrInvalidArgument("invalid_image_id")
	}

	if img.DisplayOrder < 0 {
		img.DisplayOrder = 0
	}

	if img.CreatedAt.IsZero() {
		img.CreatedAt = time.Now().UTC()
	} else {
		img.CreatedAt = img.CreatedAt.UTC()
	}

	if err := img.Validate(); err != nil {
		return listdom.ListImage{}, err
	}

	created, err := uc.imageRepo.Create(ctx, img)
	if err != nil {
		return listdom.ListImage{}, err
	}

	return created, nil
}

func (uc *ListUsecase) UpdateImage(
	ctx context.Context,
	listID string,
	imageID string,
	patch listdom.ListImagePatch,
) (listdom.ListImage, error) {
	if uc == nil {
		return listdom.ListImage{}, ErrNotSupported("List.UpdateImage")
	}

	if uc.imageRepo == nil {
		return listdom.ListImage{}, ErrNotSupported("List.UpdateImage.ImageRepo")
	}

	if listID == "" {
		return listdom.ListImage{}, listdom.ErrInvalidListImageListID
	}

	if imageID == "" {
		return listdom.ListImage{}, listdom.ErrInvalidListImageID
	}

	if strings.Contains(imageID, "/") || strings.Contains(imageID, "://") {
		return listdom.ListImage{}, ErrInvalidArgument("invalid_image_id")
	}

	if patch.DisplayOrder != nil && *patch.DisplayOrder < 0 {
		return listdom.ListImage{}, listdom.ErrInvalidListImageDisplayOrder
	}

	if patch.UpdatedAt == nil {
		now := time.Now().UTC()
		patch.UpdatedAt = &now
	} else if !patch.UpdatedAt.IsZero() {
		t := patch.UpdatedAt.UTC()
		patch.UpdatedAt = &t
	}

	updated, err := uc.imageRepo.Update(ctx, listID, imageID, patch)
	if err != nil {
		return listdom.ListImage{}, err
	}

	return updated, nil
}

func (uc *ListUsecase) DeleteImage(ctx context.Context, listID string, imageID string) error {
	if uc == nil {
		return ErrNotSupported("List.DeleteImage")
	}

	if uc.imageRepo == nil {
		return ErrNotSupported("List.DeleteImage.ImageRepo")
	}

	if listID == "" {
		return listdom.ErrInvalidListImageListID
	}

	if imageID == "" {
		return listdom.ErrInvalidListImageID
	}

	if strings.Contains(imageID, "/") || strings.Contains(imageID, "://") {
		return ErrInvalidArgument("invalid_image_id")
	}

	if err := uc.imageRepo.Delete(ctx, listID, imageID); err != nil {
		if !errors.Is(err, listdom.ErrListImageNotFound) {
			return err
		}
	}

	if uc.listRepo != nil {
		l, err := uc.listRepo.GetByID(ctx, listID)
		if err == nil && l.ImageID == imageID {
			now := time.Now().UTC()

			l.ImageID = ""
			l.UpdatedAt = &now

			_, _ = uc.listRepo.Update(ctx, listID, l)
		}
	}

	return nil
}

func (uc *ListUsecase) SetPrimaryImage(
	ctx context.Context,
	listID string,
	imageID string,
	now time.Time,
	updatedBy *string,
) (listdom.List, error) {
	if uc == nil {
		return listdom.List{}, ErrNotSupported("List.SetPrimaryImage")
	}

	if uc.listRepo == nil {
		return listdom.List{}, ErrNotSupported("List.SetPrimaryImage.ListRepo")
	}

	if uc.imageRepo == nil {
		return listdom.List{}, ErrNotSupported("List.SetPrimaryImage.ImageRepo")
	}

	lid := listID
	iid := imageID

	if lid == "" {
		return listdom.List{}, listdom.ErrInvalidID
	}

	if iid == "" {
		return listdom.List{}, listdom.ErrEmptyImageID
	}

	if strings.Contains(iid, "/") || strings.Contains(iid, "://") {
		return listdom.List{}, listdom.ErrInvalidImageID
	}

	img, err := uc.imageRepo.GetByListIDAndID(ctx, lid, iid)
	if err != nil {
		return listdom.List{}, err
	}

	if img.ID == "" {
		return listdom.List{}, listdom.ErrInvalidImageID
	}

	if img.ListID != "" && img.ListID != lid {
		return listdom.List{}, errors.New("list: image belongs to other list")
	}

	if img.URL == "" {
		return listdom.List{}, listdom.ErrInvalidListImageURL
	}

	if now.IsZero() {
		now = time.Now().UTC()
	}

	l, err := uc.listRepo.GetByID(ctx, lid)
	if err != nil {
		return listdom.List{}, err
	}

	updatedAt := now.UTC()

	l.ImageID = iid
	l.UpdatedAt = &updatedAt

	if updatedBy != nil {
		l.UpdatedBy = updatedBy
	}

	updated, err := uc.listRepo.Update(ctx, lid, l)
	if err != nil {
		return listdom.List{}, err
	}

	return updated, nil
}
