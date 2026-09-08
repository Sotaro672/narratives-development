// backend/internal/application/usecase/resale_crud_usecase.go
package usecase

import (
	"context"

	resaledom "narratives/internal/domain/resale"
)

// CreateResaleInput is the application input for creating a resale listing.
//
// Server-owned values such as status, audit timestamps, BrandID and
// ProductBlueprintID are intentionally not accepted from the caller.
type CreateResaleInput struct {
	AvatarID         string
	AssetID          string
	TokenBlueprintID string
	ProductID        string
	Price            int
	Condition        resaledom.ResaleCondition
	Description      string
}

// CreateResale creates a new resale listing.
//
// Policy:
// - Status is always listing on creation.
// - Condition defaults to ConditionLikeNew when omitted.
// - AvatarID is the authenticated seller avatar.
// - CreatedBy / CreatedAt / UpdatedAt are application-owned.
// - BrandID / ProductBlueprintID are resolved canonically by Create.
// - Client-provided server-owned fields are never accepted.
func (uc *ResaleUsecase) CreateResale(
	ctx context.Context,
	input CreateResaleInput,
) (resaledom.Resale, error) {
	if uc == nil || uc.resaleRepo == nil {
		return resaledom.Resale{}, ErrNotSupported("Resale.CreateResale")
	}
	if input.AvatarID == "" {
		return resaledom.Resale{}, resaledom.ErrInvalidAvatarID
	}
	if input.AssetID == "" {
		return resaledom.Resale{}, resaledom.ErrInvalidAssetID
	}
	if input.TokenBlueprintID == "" {
		return resaledom.Resale{}, resaledom.ErrInvalidTokenBlueprintID
	}
	if input.ProductID == "" {
		return resaledom.Resale{}, resaledom.ErrInvalidProductID
	}
	if input.Price <= 0 {
		return resaledom.Resale{}, resaledom.ErrInvalidPrice
	}

	condition := input.Condition
	if condition == "" {
		condition = resaledom.ConditionLikeNew
	}

	now := uc.nowUTC()
	item := resaledom.Resale{
		Status:           resaledom.StatusListing,
		AssetID:          input.AssetID,
		TokenBlueprintID: input.TokenBlueprintID,
		ProductID:        input.ProductID,
		AvatarID:         input.AvatarID,
		Price:            input.Price,
		Condition:        condition,
		Description:      input.Description,
		CreatedBy:        input.AvatarID,
		CreatedAt:        now,
		UpdatedAt:        &now,
	}

	return uc.Create(ctx, item)
}

// UpdateOwnedResaleInput is the application input for updating an owned resale.
//
// Only mutable fields are accepted. Identity, ownership, canonical product
// linkage, primary image and audit fields are always preserved or generated
// server-side.
type UpdateOwnedResaleInput struct {
	ResaleID    string
	AvatarID    string
	Price       int
	Status      resaledom.ResaleStatus
	Condition   resaledom.ResaleCondition
	Description string
}

// UpdateOwnedResale updates a resale after verifying ownership.
//
// Existing behavior is preserved:
// - Price <= 0 keeps the persisted price.
// - Empty Status keeps the persisted status.
// - Empty Condition keeps the persisted condition.
// - Description is replaced with the supplied value, including empty string.
// - Immutable/server-owned fields are restored from the persisted resale.
// - UpdatedAt / UpdatedBy are application-owned.
func (uc *ResaleUsecase) UpdateOwnedResale(
	ctx context.Context,
	input UpdateOwnedResaleInput,
) (resaledom.Resale, error) {
	if uc == nil || uc.resaleRepo == nil {
		return resaledom.Resale{}, ErrNotSupported("Resale.UpdateOwnedResale")
	}
	if input.ResaleID == "" {
		return resaledom.Resale{}, resaledom.ErrInvalidID
	}
	if input.AvatarID == "" {
		return resaledom.Resale{}, resaledom.ErrInvalidAvatarID
	}

	existing, err := uc.GetOwned(
		ctx,
		input.ResaleID,
		input.AvatarID,
	)
	if err != nil {
		return resaledom.Resale{}, err
	}

	price := input.Price
	if price <= 0 {
		price = existing.Price
	}

	status := input.Status
	if status == "" {
		status = existing.Status
	}

	condition := input.Condition
	if condition == "" {
		condition = existing.Condition
	}

	now := uc.nowUTC()
	updatedBy := input.AvatarID

	item := resaledom.Resale{
		ID:                 existing.ID,
		Status:             status,
		AssetID:            existing.AssetID,
		TokenBlueprintID:   existing.TokenBlueprintID,
		ProductID:          existing.ProductID,
		BrandID:            existing.BrandID,
		ProductBlueprintID: existing.ProductBlueprintID,
		AvatarID:           existing.AvatarID,
		Price:              price,
		Condition:          condition,
		Description:        input.Description,
		ImageID:            existing.ImageID,
		CreatedBy:          existing.CreatedBy,
		CreatedAt:          existing.CreatedAt,
		UpdatedBy:          &updatedBy,
		UpdatedAt:          &now,
	}

	return uc.Update(ctx, item)
}

// DeleteOwned physically deletes a resale after verifying that the
// authenticated avatar owns it.
//
// The underlying Delete method retains deletion policy, image cleanup,
// review cleanup and cart cleanup responsibilities.
func (uc *ResaleUsecase) DeleteOwned(
	ctx context.Context,
	resaleID string,
	avatarID string,
) error {
	if uc == nil || uc.resaleRepo == nil {
		return ErrNotSupported("Resale.DeleteOwned")
	}
	if resaleID == "" {
		return resaledom.ErrInvalidID
	}
	if avatarID == "" {
		return resaledom.ErrInvalidAvatarID
	}

	if _, err := uc.GetOwned(
		ctx,
		resaleID,
		avatarID,
	); err != nil {
		return err
	}

	return uc.Delete(ctx, resaleID)
}
