// backend/internal/application/query/mall/resale_display_enricher.go
package mall

import (
	"context"

	mallshared "narratives/internal/application/query/mall/shared"
	avatardom "narratives/internal/domain/avatar"
	branddom "narratives/internal/domain/brand"
	modeldom "narratives/internal/domain/model"
	productdom "narratives/internal/domain/product"
	productblueprintdom "narratives/internal/domain/productBlueprint"
	resaledom "narratives/internal/domain/resale"
	tokenblueprintdom "narratives/internal/domain/tokenBlueprint"
)

// resaleDisplayEnricher centralizes resale display enrichment shared by
// ResaleQuery and MarketQuery.
//
// Responsibilities:
// - productId -> modelId
// - modelId -> variation display fields
// - productBlueprintId -> productName
// - tokenBlueprintId -> tokenName / tokenIcon / imageUrl fallback
// - brandId -> brandName
// - optional avatar display enrichment
// - optional primary resale image URL enrichment
//
// Policy such as public listing visibility, own listing exclusion, and
// permission checks must stay in the caller query service.
type resaleDisplayEnricher struct {
	displayResolver mallshared.MallDisplayResolver

	imageRepo  resaledom.ImageRepository
	avatarRepo avatardom.Repository

	includeAvatar bool
	includeImage  bool

	// When true, tokenBlueprint.IconURL is copied to Resale.ImageURL.
	// This preserves the old ResaleQuery behavior.
	useTokenIconAsImageFallback bool
}

type resaleDisplayEnricherConfig struct {
	displayResolver mallshared.MallDisplayResolver

	productRepo          productdom.Repository
	modelRepo            modeldom.RepositoryPort
	productBlueprintRepo productblueprintdom.Repository
	tokenBlueprintRepo   tokenblueprintdom.RepositoryPort
	brandRepo            branddom.Repository

	imageRepo  resaledom.ImageRepository
	avatarRepo avatardom.Repository

	includeAvatar bool
	includeImage  bool

	useTokenIconAsImageFallback bool
}

func newResaleDisplayEnricher(
	cfg resaleDisplayEnricherConfig,
) *resaleDisplayEnricher {
	displayResolver := cfg.displayResolver
	if displayResolver == nil {
		displayResolver = mallshared.NewDisplayResolver(
			cfg.productRepo,
			cfg.modelRepo,
			cfg.productBlueprintRepo,
			cfg.tokenBlueprintRepo,
			cfg.brandRepo,
		)
	}

	return &resaleDisplayEnricher{
		displayResolver: displayResolver,

		imageRepo:  cfg.imageRepo,
		avatarRepo: cfg.avatarRepo,

		includeAvatar: cfg.includeAvatar,
		includeImage:  cfg.includeImage,

		useTokenIconAsImageFallback: cfg.useTokenIconAsImageFallback,
	}
}

func (e *resaleDisplayEnricher) enrichResalesForDisplay(
	ctx context.Context,
	items []resaledom.Resale,
) []resaledom.Resale {
	if len(items) == 0 {
		return items
	}

	for i := range items {
		items[i] = e.enrichResaleForDisplay(ctx, items[i])
	}

	return items
}

func (e *resaleDisplayEnricher) enrichResaleForDisplay(
	ctx context.Context,
	item resaledom.Resale,
) resaledom.Resale {
	item = e.enrichResaleWithProductAndModel(ctx, item)
	item = e.enrichResaleWithProductName(ctx, item)
	item = e.enrichResaleWithTokenBlueprint(ctx, item)
	item = e.enrichResaleWithBrandName(ctx, item)

	if e != nil && e.includeAvatar {
		item = e.enrichResaleWithAvatar(ctx, item)
	}

	if e != nil && e.includeImage {
		item = e.enrichResaleWithImageURL(ctx, item)
	}

	return item
}

func (e *resaleDisplayEnricher) enrichResaleWithProductAndModel(
	ctx context.Context,
	item resaledom.Resale,
) resaledom.Resale {
	if e == nil || e.displayResolver == nil {
		return item
	}

	productID := item.ProductID
	if productID == "" {
		return item
	}

	model, err := e.displayResolver.ResolveModelByProductID(ctx, productID)
	if err != nil {
		return item
	}

	return applyModelDisplayToResale(item, model)
}

func applyModelDisplayToResale(
	item resaledom.Resale,
	model mallshared.ModelDisplay,
) resaledom.Resale {
	if model.ModelID != "" {
		item.ModelID = model.ModelID
	}

	if model.ProductBlueprintID != "" {
		item.ProductBlueprintID = firstNonEmpty(
			item.ProductBlueprintID,
			model.ProductBlueprintID,
		)
	}

	if model.Kind != "" {
		item.Kind = model.Kind
	}

	if model.ModelNumber != "" {
		item.ModelNumber = model.ModelNumber
	}

	if model.Size != "" {
		item.Size = model.Size
	}

	if model.ColorName != "" {
		item.Color = &resaledom.ResaleColor{
			Name: model.ColorName,
			RGB:  model.ColorRGB,
		}
	}

	if len(model.Measurements) > 0 {
		item.Measurements = model.Measurements
	}

	if model.VolumeValue != nil || model.VolumeUnit != "" {
		amount := 0
		if model.VolumeValue != nil {
			amount = *model.VolumeValue
		}

		item.Volume = &resaledom.ResaleVolume{
			Amount: amount,
			Unit:   model.VolumeUnit,
		}
	}

	return item
}

func (e *resaleDisplayEnricher) enrichResaleWithProductName(
	ctx context.Context,
	item resaledom.Resale,
) resaledom.Resale {
	if e == nil || e.displayResolver == nil {
		return item
	}

	productBlueprintID := item.ProductBlueprintID
	if productBlueprintID == "" {
		return item
	}

	pb, err := e.displayResolver.ResolveProductBlueprintInfo(ctx, productBlueprintID)
	if err != nil {
		return item
	}

	if pb.ProductName != "" {
		item.ProductName = pb.ProductName
	}

	if item.BrandID == "" && pb.BrandID != "" {
		item.BrandID = pb.BrandID
	}

	return item
}

func (e *resaleDisplayEnricher) enrichResaleWithTokenBlueprint(
	ctx context.Context,
	item resaledom.Resale,
) resaledom.Resale {
	if e == nil || e.displayResolver == nil {
		return item
	}

	tokenBlueprintID := item.TokenBlueprintID
	if tokenBlueprintID == "" {
		return item
	}

	tb, err := e.displayResolver.ResolveTokenBlueprintInfo(ctx, tokenBlueprintID)
	if err != nil {
		return item
	}

	if tb.TokenName != "" {
		item.TokenName = tb.TokenName
	}

	if tb.TokenIcon != "" {
		item.TokenIcon = tb.TokenIcon

		if e.useTokenIconAsImageFallback && item.ImageURL == "" {
			item.ImageURL = tb.TokenIcon
		}
	}

	if item.BrandID == "" && tb.BrandID != "" {
		item.BrandID = tb.BrandID
	}

	return item
}

func (e *resaleDisplayEnricher) enrichResaleWithBrandName(
	ctx context.Context,
	item resaledom.Resale,
) resaledom.Resale {
	if e == nil || e.displayResolver == nil {
		return item
	}

	brandID := item.BrandID
	if brandID == "" {
		return item
	}

	brand, err := e.displayResolver.ResolveBrandInfo(ctx, brandID)
	if err != nil {
		return item
	}

	if brand.BrandName != "" {
		item.BrandName = brand.BrandName
	}

	return item
}

func (e *resaleDisplayEnricher) enrichResaleWithAvatar(
	ctx context.Context,
	item resaledom.Resale,
) resaledom.Resale {
	if e == nil || e.avatarRepo == nil {
		return item
	}

	avatarID := item.AvatarID
	if avatarID == "" {
		return item
	}

	avatar, err := e.avatarRepo.GetByID(ctx, avatarID)
	if err != nil {
		return item
	}

	item.AvatarName = avatar.AvatarName

	if avatar.AvatarIcon != nil {
		item.AvatarIcon = *avatar.AvatarIcon
	}

	return item
}

func (e *resaleDisplayEnricher) enrichResaleWithImageURL(
	ctx context.Context,
	item resaledom.Resale,
) resaledom.Resale {
	if e == nil || e.imageRepo == nil {
		return item
	}

	resaleID := item.ID
	if resaleID == "" {
		return item
	}

	images, err := e.imageRepo.ListByResaleID(ctx, resaleID)
	if err != nil || len(images) == 0 {
		return item
	}

	imageURL := mallshared.SelectPrimaryResaleImageURL(item, images)
	if imageURL != "" {
		item.ImageURL = imageURL
	}

	return item
}

func firstNonEmpty(primary string, fallback string) string {
	if primary != "" {
		return primary
	}

	return fallback
}
