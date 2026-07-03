// backend/internal/application/usecase/resale_usecase.go
package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	resaledom "narratives/internal/domain/resale"
)

type ResaleUsecase struct {
	resaleRepo resaledom.Repository
	imageRepo  resaledom.ImageRepository
}

func NewResaleUsecase(
	resaleRepo resaledom.Repository,
	imageRepo resaledom.ImageRepository,
) *ResaleUsecase {
	return &ResaleUsecase{
		resaleRepo: resaleRepo,
		imageRepo:  imageRepo,
	}
}

func (uc *ResaleUsecase) Create(
	ctx context.Context,
	item resaledom.Resale,
) (resaledom.Resale, error) {
	if uc == nil || uc.resaleRepo == nil {
		return resaledom.Resale{}, ErrNotSupported("Resale.Create")
	}

	return uc.resaleRepo.Create(ctx, item)
}

func (uc *ResaleUsecase) Update(
	ctx context.Context,
	item resaledom.Resale,
) (resaledom.Resale, error) {
	if uc == nil || uc.resaleRepo == nil {
		return resaledom.Resale{}, ErrNotSupported("Resale.Update")
	}

	id := strings.TrimSpace(item.ID)
	if id == "" {
		return resaledom.Resale{}, resaledom.ErrInvalidID
	}

	item.ID = id

	return uc.resaleRepo.Update(ctx, id, item)
}

func (uc *ResaleUsecase) Delete(ctx context.Context, id string) error {
	if uc == nil || uc.resaleRepo == nil {
		return ErrNotSupported("Resale.Delete")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return resaledom.ErrInvalidID
	}

	return uc.resaleRepo.Delete(ctx, id)
}

func (uc *ResaleUsecase) CreateImage(
	ctx context.Context,
	img resaledom.ResaleImage,
) (resaledom.ResaleImage, error) {
	if uc == nil {
		return resaledom.ResaleImage{}, ErrNotSupported("Resale.CreateImage")
	}

	if uc.imageRepo == nil {
		return resaledom.ResaleImage{}, ErrNotSupported("Resale.CreateImage.ImageRepo")
	}

	img.ResaleID = strings.TrimSpace(img.ResaleID)
	img.ID = strings.TrimSpace(img.ID)
	img.URL = strings.TrimSpace(img.URL)
	img.CreatedBy = strings.TrimSpace(img.CreatedBy)

	if img.ResaleID == "" {
		return resaledom.ResaleImage{}, resaledom.ErrInvalidConditionImageResaleID
	}

	if img.ID == "" {
		return resaledom.ResaleImage{}, resaledom.ErrInvalidConditionImageID
	}

	if strings.Contains(img.ID, "/") || strings.Contains(img.ID, "://") {
		return resaledom.ResaleImage{}, ErrInvalidArgument("invalid_image_id")
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
		return resaledom.ResaleImage{}, err
	}

	created, err := uc.imageRepo.Create(ctx, img)
	if err != nil {
		return resaledom.ResaleImage{}, err
	}

	return created, nil
}

func (uc *ResaleUsecase) UpdateImage(
	ctx context.Context,
	resaleID string,
	imageID string,
	patch resaledom.ResaleImagePatch,
) (resaledom.ResaleImage, error) {
	if uc == nil {
		return resaledom.ResaleImage{}, ErrNotSupported("Resale.UpdateImage")
	}

	if uc.imageRepo == nil {
		return resaledom.ResaleImage{}, ErrNotSupported("Resale.UpdateImage.ImageRepo")
	}

	resaleID = strings.TrimSpace(resaleID)
	imageID = strings.TrimSpace(imageID)

	if resaleID == "" {
		return resaledom.ResaleImage{}, resaledom.ErrInvalidConditionImageResaleID
	}

	if imageID == "" {
		return resaledom.ResaleImage{}, resaledom.ErrInvalidConditionImageID
	}

	if strings.Contains(imageID, "/") || strings.Contains(imageID, "://") {
		return resaledom.ResaleImage{}, ErrInvalidArgument("invalid_image_id")
	}

	if patch.URL != nil {
		v := strings.TrimSpace(*patch.URL)
		patch.URL = &v
	}

	if patch.DisplayOrder != nil && *patch.DisplayOrder < 0 {
		return resaledom.ResaleImage{}, resaledom.ErrInvalidConditionImageDisplayOrder
	}

	if patch.UpdatedBy != nil {
		v := strings.TrimSpace(*patch.UpdatedBy)
		patch.UpdatedBy = &v
	}

	if patch.UpdatedAt == nil {
		now := time.Now().UTC()
		patch.UpdatedAt = &now
	} else if !patch.UpdatedAt.IsZero() {
		t := patch.UpdatedAt.UTC()
		patch.UpdatedAt = &t
	}

	updated, err := uc.imageRepo.Update(ctx, resaleID, imageID, patch)
	if err != nil {
		return resaledom.ResaleImage{}, err
	}

	return updated, nil
}

func (uc *ResaleUsecase) DeleteImage(ctx context.Context, resaleID string, imageID string) error {
	if uc == nil {
		return ErrNotSupported("Resale.DeleteImage")
	}

	if uc.imageRepo == nil {
		return ErrNotSupported("Resale.DeleteImage.ImageRepo")
	}

	resaleID = strings.TrimSpace(resaleID)
	imageID = strings.TrimSpace(imageID)

	if resaleID == "" {
		return resaledom.ErrInvalidConditionImageResaleID
	}

	if imageID == "" {
		return resaledom.ErrInvalidConditionImageID
	}

	if strings.Contains(imageID, "/") || strings.Contains(imageID, "://") {
		return ErrInvalidArgument("invalid_image_id")
	}

	if err := uc.imageRepo.Delete(ctx, resaleID, imageID); err != nil {
		if !errors.Is(err, resaledom.ErrNotFound) &&
			!errors.Is(err, resaledom.ErrConditionImageNotFound) {
			return err
		}
	}

	if uc.resaleRepo != nil {
		r, err := uc.resaleRepo.GetByID(ctx, resaleID)
		if err == nil && r.ImageID == imageID {
			now := time.Now().UTC()

			r.ImageID = ""
			r.UpdatedAt = &now

			_, _ = uc.resaleRepo.Update(ctx, resaleID, r)
		}
	}

	return nil
}

func (uc *ResaleUsecase) SetPrimaryImage(
	ctx context.Context,
	resaleID string,
	imageID string,
	now time.Time,
	updatedBy *string,
) (resaledom.Resale, error) {
	if uc == nil {
		return resaledom.Resale{}, ErrNotSupported("Resale.SetPrimaryImage")
	}

	if uc.resaleRepo == nil {
		return resaledom.Resale{}, ErrNotSupported("Resale.SetPrimaryImage.ResaleRepo")
	}

	if uc.imageRepo == nil {
		return resaledom.Resale{}, ErrNotSupported("Resale.SetPrimaryImage.ImageRepo")
	}

	resaleID = strings.TrimSpace(resaleID)
	imageID = strings.TrimSpace(imageID)

	if resaleID == "" {
		return resaledom.Resale{}, resaledom.ErrInvalidID
	}

	if imageID == "" {
		return resaledom.Resale{}, resaledom.ErrEmptyImageID
	}

	if strings.Contains(imageID, "/") || strings.Contains(imageID, "://") {
		return resaledom.Resale{}, resaledom.ErrInvalidImageID
	}

	images, err := uc.imageRepo.ListByResaleID(ctx, resaleID)
	if err != nil {
		return resaledom.Resale{}, err
	}

	var selected resaledom.ResaleImage
	found := false

	for _, img := range images {
		if img.ID != imageID {
			continue
		}

		selected = img
		found = true
		break
	}

	if !found {
		return resaledom.Resale{}, resaledom.ErrNotFound
	}

	if selected.ID == "" {
		return resaledom.Resale{}, resaledom.ErrInvalidImageID
	}

	if selected.ResaleID != "" && selected.ResaleID != resaleID {
		return resaledom.Resale{}, errors.New("resale: image belongs to other resale")
	}

	if strings.TrimSpace(selected.URL) == "" {
		return resaledom.Resale{}, resaledom.ErrInvalidConditionImageURL
	}

	if now.IsZero() {
		now = time.Now().UTC()
	}

	r, err := uc.resaleRepo.GetByID(ctx, resaleID)
	if err != nil {
		return resaledom.Resale{}, err
	}

	updatedAt := now.UTC()

	r.ImageID = imageID
	r.UpdatedAt = &updatedAt

	if updatedBy != nil {
		v := strings.TrimSpace(*updatedBy)
		if v == "" {
			r.UpdatedBy = nil
		} else {
			r.UpdatedBy = &v
		}
	}

	updated, err := uc.resaleRepo.Update(ctx, resaleID, r)
	if err != nil {
		return resaledom.Resale{}, err
	}

	return updated, nil
}
