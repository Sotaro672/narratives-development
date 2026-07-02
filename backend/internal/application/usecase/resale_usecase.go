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

	id := item.ID
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

	if id == "" {
		return resaledom.ErrInvalidID
	}

	return uc.resaleRepo.Delete(ctx, id)
}

func (uc *ResaleUsecase) CreateImage(
	ctx context.Context,
	img resaledom.ResaleConditionImage,
) (resaledom.ResaleConditionImage, error) {
	if uc == nil {
		return resaledom.ResaleConditionImage{}, ErrNotSupported("Resale.CreateImage")
	}

	if uc.imageRepo == nil {
		return resaledom.ResaleConditionImage{}, ErrNotSupported("Resale.CreateImage.ImageRepo")
	}

	if img.ResaleID == "" {
		return resaledom.ResaleConditionImage{}, resaledom.ErrInvalidConditionImageResaleID
	}

	if img.ID == "" {
		return resaledom.ResaleConditionImage{}, resaledom.ErrInvalidConditionImageID
	}

	if strings.Contains(img.ID, "/") || strings.Contains(img.ID, "://") {
		return resaledom.ResaleConditionImage{}, ErrInvalidArgument("invalid_image_id")
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
		return resaledom.ResaleConditionImage{}, err
	}

	created, err := uc.imageRepo.Create(ctx, img)
	if err != nil {
		return resaledom.ResaleConditionImage{}, err
	}

	return created, nil
}

func (uc *ResaleUsecase) UpdateImage(
	ctx context.Context,
	resaleID string,
	imageID string,
	patch resaledom.ResaleConditionImagePatch,
) (resaledom.ResaleConditionImage, error) {
	if uc == nil {
		return resaledom.ResaleConditionImage{}, ErrNotSupported("Resale.UpdateImage")
	}

	if uc.imageRepo == nil {
		return resaledom.ResaleConditionImage{}, ErrNotSupported("Resale.UpdateImage.ImageRepo")
	}

	if resaleID == "" {
		return resaledom.ResaleConditionImage{}, resaledom.ErrInvalidConditionImageResaleID
	}

	if imageID == "" {
		return resaledom.ResaleConditionImage{}, resaledom.ErrInvalidConditionImageID
	}

	if strings.Contains(imageID, "/") || strings.Contains(imageID, "://") {
		return resaledom.ResaleConditionImage{}, ErrInvalidArgument("invalid_image_id")
	}

	if patch.DisplayOrder != nil && *patch.DisplayOrder < 0 {
		return resaledom.ResaleConditionImage{}, resaledom.ErrInvalidConditionImageDisplayOrder
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
		return resaledom.ResaleConditionImage{}, err
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
		if !errors.Is(err, resaledom.ErrNotFound) {
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

	rid := resaleID
	iid := imageID

	if rid == "" {
		return resaledom.Resale{}, resaledom.ErrInvalidID
	}

	if iid == "" {
		return resaledom.Resale{}, resaledom.ErrEmptyImageID
	}

	if strings.Contains(iid, "/") || strings.Contains(iid, "://") {
		return resaledom.Resale{}, resaledom.ErrInvalidImageID
	}

	images, err := uc.imageRepo.ListByResaleID(ctx, rid)
	if err != nil {
		return resaledom.Resale{}, err
	}

	var selected resaledom.ResaleConditionImage
	found := false

	for _, img := range images {
		if img.ID != iid {
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

	if selected.ResaleID != "" && selected.ResaleID != rid {
		return resaledom.Resale{}, errors.New("resale: image belongs to other resale")
	}

	if selected.URL == "" {
		return resaledom.Resale{}, resaledom.ErrInvalidConditionImageURL
	}

	if now.IsZero() {
		now = time.Now().UTC()
	}

	r, err := uc.resaleRepo.GetByID(ctx, rid)
	if err != nil {
		return resaledom.Resale{}, err
	}

	updatedAt := now.UTC()

	r.ImageID = iid
	r.UpdatedAt = &updatedAt

	if updatedBy != nil {
		r.UpdatedBy = updatedBy
	}

	updated, err := uc.resaleRepo.Update(ctx, rid, r)
	if err != nil {
		return resaledom.Resale{}, err
	}

	return updated, nil
}
