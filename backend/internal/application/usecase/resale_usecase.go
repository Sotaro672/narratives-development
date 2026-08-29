// backend/internal/application/usecase/resale_usecase.go
package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	applicationport "narratives/internal/application/port"
	resaledom "narratives/internal/domain/resale"
	resalereview "narratives/internal/domain/resale_review"
)

type ResaleUsecase struct {
	resaleRepo    resaledom.Repository
	imageRepo     resaledom.ImageRepository
	imageStorage  applicationport.ResaleImageStorage
	reviewCleanup resalereview.CleanupRepository
}

func NewResaleUsecase(
	resaleRepo resaledom.Repository,
	imageRepo resaledom.ImageRepository,
	imageStorage applicationport.ResaleImageStorage,
) *ResaleUsecase {
	return &ResaleUsecase{
		resaleRepo:   resaleRepo,
		imageRepo:    imageRepo,
		imageStorage: imageStorage,
	}
}

func (uc *ResaleUsecase) WithReviewCleanup(
	reviewCleanup resalereview.CleanupRepository,
) *ResaleUsecase {
	if uc == nil {
		return nil
	}

	uc.reviewCleanup = reviewCleanup
	return uc
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

	id := item.ID
	if id == "" {
		return resaledom.Resale{}, resaledom.ErrInvalidID
	}

	item.ID = id

	return uc.resaleRepo.Update(ctx, id, item)
}

func (uc *ResaleUsecase) Delete(
	ctx context.Context,
	id string,
) error {
	if uc == nil || uc.resaleRepo == nil {
		return ErrNotSupported("Resale.Delete")
	}

	if id == "" {
		return resaledom.ErrInvalidID
	}

	item, err := uc.resaleRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := item.ValidateDelete(); err != nil {
		return err
	}

	if uc.imageStorage == nil {
		return ErrNotSupported("Resale.Delete.ImageStorage")
	}

	if err := uc.imageStorage.DeleteAll(ctx, id); err != nil {
		return err
	}

	if uc.reviewCleanup != nil {
		if err := uc.reviewCleanup.DeleteByResaleID(ctx, id); err != nil {
			return err
		}
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

func (uc *ResaleUsecase) DeleteImage(
	ctx context.Context,
	resaleID string,
	imageID string,
) error {
	if uc == nil {
		return ErrNotSupported("Resale.DeleteImage")
	}

	if uc.imageRepo == nil {
		return ErrNotSupported("Resale.DeleteImage.ImageRepo")
	}

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
		return resaledom.Resale{}, errors.New(
			"resale: image belongs to other resale",
		)
	}

	if selected.URL == "" {
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
		v := *updatedBy
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
