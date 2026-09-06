// backend/internal/application/usecase/resale_usecase.go
package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	applicationport "narratives/internal/application/port"
	productdom "narratives/internal/domain/product"
	productblueprintdom "narratives/internal/domain/productBlueprint"
	resaledom "narratives/internal/domain/resale"
	resalereview "narratives/internal/domain/resale_review"
)

type ResaleUsecase struct {
	resaleRepo                resaledom.Repository
	imageRepo                 resaledom.ImageRepository
	imageStorage              applicationport.ResaleImageStorage
	reviewCleanup             resalereview.CleanupRepository
	cartItemCleanup           CartItemCleanup
	productRepo               productdom.Repository
	productBlueprintRepo      productblueprintdom.Repository
	avatarResaleAccessChecker AvatarResaleAccessChecker
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

func (uc *ResaleUsecase) WithReviewCleanup(reviewCleanup resalereview.CleanupRepository) *ResaleUsecase {
	if uc == nil {
		return nil
	}

	uc.reviewCleanup = reviewCleanup
	return uc
}

func (uc *ResaleUsecase) WithCartItemCleanup(cartItemCleanup CartItemCleanup) *ResaleUsecase {
	if uc == nil {
		return nil
	}

	uc.cartItemCleanup = cartItemCleanup
	return uc
}

func (uc *ResaleUsecase) WithProductIdentityRepositories(
	productRepo productdom.Repository,
	productBlueprintRepo productblueprintdom.Repository,
) *ResaleUsecase {
	if uc == nil {
		return nil
	}

	uc.productRepo = productRepo
	uc.productBlueprintRepo = productBlueprintRepo
	return uc
}

func (uc *ResaleUsecase) WithAvatarResaleAccessChecker(
	checker AvatarResaleAccessChecker,
) *ResaleUsecase {
	if uc == nil {
		return nil
	}

	uc.avatarResaleAccessChecker = checker
	return uc
}

func (uc *ResaleUsecase) Create(
	ctx context.Context,
	item resaledom.Resale,
) (resaledom.Resale, error) {
	if uc == nil || uc.resaleRepo == nil {
		return resaledom.Resale{}, ErrNotSupported("Resale.Create")
	}
	if uc.productRepo == nil || uc.productBlueprintRepo == nil {
		return resaledom.Resale{}, ErrNotSupported("Resale.Create.ProductIdentity")
	}
	if item.AvatarID == "" {
		return resaledom.Resale{}, resaledom.ErrInvalidAvatarID
	}
	if item.ProductID == "" {
		return resaledom.Resale{}, resaledom.ErrInvalidProductID
	}

	if err := checkAvatarResaleAccess(
		ctx,
		uc.avatarResaleAccessChecker,
		item.AvatarID,
	); err != nil {
		return resaledom.Resale{}, err
	}

	product, err := uc.productRepo.GetByID(ctx, item.ProductID)
	if err != nil {
		return resaledom.Resale{}, err
	}
	if product.ID != item.ProductID || product.ModelID == "" {
		return resaledom.Resale{}, resaledom.ErrInvalidProductID
	}

	productBlueprintID, _, err := uc.productBlueprintRepo.GetIDByModelID(
		ctx,
		product.ModelID,
	)
	if err != nil {
		return resaledom.Resale{}, err
	}
	if productBlueprintID == "" {
		return resaledom.Resale{}, resaledom.ErrInvalidProductBlueprintID
	}

	productBlueprint, err := uc.productBlueprintRepo.GetByID(
		ctx,
		productBlueprintID,
	)
	if err != nil {
		return resaledom.Resale{}, err
	}
	if productBlueprint.ID != productBlueprintID {
		return resaledom.Resale{}, resaledom.ErrInvalidProductBlueprintID
	}
	if productBlueprint.BrandID == "" {
		return resaledom.Resale{}, resaledom.ErrInvalidBrandID
	}

	// BrandID / ProductBlueprintID are canonical server-side values.
	// Client-provided values, if any, are intentionally ignored.
	item.ProductBlueprintID = productBlueprint.ID
	item.BrandID = productBlueprint.BrandID

	if err := item.ValidateForCreate(); err != nil {
		return resaledom.Resale{}, err
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

	if item.Status == resaledom.StatusListing {
		// 再公開時の所有Avatarは呼び出し側から渡された値を信用せず、
		// 永続化済みResaleから取得したcanonical値を使用する。
		existing, err := uc.resaleRepo.GetByID(ctx, id)
		if err != nil {
			return resaledom.Resale{}, err
		}
		if existing.ID != id {
			return resaledom.Resale{}, resaledom.ErrInvalidID
		}
		if existing.AvatarID == "" {
			return resaledom.Resale{}, resaledom.ErrInvalidAvatarID
		}

		if err := checkAvatarResaleAccess(
			ctx,
			uc.avatarResaleAccessChecker,
			existing.AvatarID,
		); err != nil {
			return resaledom.Resale{}, err
		}

		item.AvatarID = existing.AvatarID
	}

	if item.Status == resaledom.StatusSuspended && uc.cartItemCleanup == nil {
		return resaledom.Resale{}, ErrNotSupported("Resale.Update.CartItemCleanup")
	}

	item.ID = id

	updated, err := uc.resaleRepo.Update(ctx, id, item)
	if err != nil {
		return resaledom.Resale{}, err
	}

	// RemoveItemsByResaleID is idempotent.
	// Always execute it while the resulting resale is suspended so a retry
	// can complete cleanup even when the previous update already persisted
	// status=suspended.
	if updated.Status == resaledom.StatusSuspended {
		if err := uc.cartItemCleanup.RemoveItemsByResaleID(ctx, id); err != nil {
			return resaledom.Resale{}, err
		}
	}

	return updated, nil
}

// SuspendAvatarResaleByAdmin suspends only the resale service for an avatar.
//
// Policy:
// - Avatar itself is not deleted or disabled.
// - Only currently listing resales are changed to suspended.
// - Already suspended resales remain unchanged.
// - Sold resales and completed/past trades remain unchanged.
// - Suspended resale items are removed from carts through Update.
// - The operation is idempotent; retrying completes any remaining listings.
func (uc *ResaleUsecase) SuspendAvatarResaleByAdmin(
	ctx context.Context,
	input SuspendAvatarResaleByAdminInput,
) error {
	if uc == nil || uc.resaleRepo == nil {
		return ErrNotSupported("Resale.SuspendAvatarResaleByAdmin")
	}
	if input.AvatarID == "" {
		return resaledom.ErrInvalidAvatarID
	}

	items, err := uc.resaleRepo.ListByAvatarID(ctx, input.AvatarID)
	if err != nil {
		return err
	}

	now := time.Now().UTC()

	for _, item := range items {
		if item.AvatarID != input.AvatarID {
			return resaledom.ErrInvalidAvatarID
		}
		if item.Status != resaledom.StatusListing {
			continue
		}

		if err := item.Suspend(now); err != nil {
			return err
		}

		if input.AdminID != "" {
			updatedBy := input.AdminID
			item.UpdatedBy = &updatedBy
		}

		if _, err := uc.Update(ctx, item); err != nil {
			return err
		}
	}

	return nil
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
	if uc.cartItemCleanup == nil {
		return ErrNotSupported("Resale.Delete.CartItemCleanup")
	}

	if err := uc.imageStorage.DeleteAll(ctx, id); err != nil {
		return err
	}

	if uc.reviewCleanup != nil {
		if err := uc.reviewCleanup.DeleteByResaleID(ctx, id); err != nil {
			return err
		}
	}

	if err := uc.resaleRepo.Delete(ctx, id); err != nil {
		return err
	}

	return uc.cartItemCleanup.RemoveItemsByResaleID(ctx, id)
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
		return resaledom.Resale{}, errors.New("resale: image belongs to other resale")
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
