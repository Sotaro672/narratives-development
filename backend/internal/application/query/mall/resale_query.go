// backend/internal/application/query/mall/resale_query.go
package mall

import (
	"context"
	"errors"

	branddom "narratives/internal/domain/brand"
	modeldom "narratives/internal/domain/model"
	productdom "narratives/internal/domain/product"
	productblueprintdom "narratives/internal/domain/productBlueprint"
	resaledom "narratives/internal/domain/resale"
	tokenblueprintdom "narratives/internal/domain/tokenBlueprint"
)

type ResaleQuery struct {
	resaleRepo           resaledom.Repository
	imageRepo            resaledom.ImageRepository
	productRepo          productdom.Repository
	modelRepo            modeldom.RepositoryPort
	productBlueprintRepo productblueprintdom.Repository
	tokenBlueprintRepo   tokenblueprintdom.RepositoryPort
	brandRepo            branddom.Repository
}

func NewResaleQuery(
	resaleRepo resaledom.Repository,
	imageRepo resaledom.ImageRepository,
	productRepo productdom.Repository,
	modelRepo modeldom.RepositoryPort,
	productBlueprintRepo productblueprintdom.Repository,
	tokenBlueprintRepo tokenblueprintdom.RepositoryPort,
	brandRepo branddom.Repository,
) *ResaleQuery {
	return &ResaleQuery{
		resaleRepo:           resaleRepo,
		imageRepo:            imageRepo,
		productRepo:          productRepo,
		modelRepo:            modelRepo,
		productBlueprintRepo: productBlueprintRepo,
		tokenBlueprintRepo:   tokenBlueprintRepo,
		brandRepo:            brandRepo,
	}
}

func (q *ResaleQuery) List(
	ctx context.Context,
	filter resaledom.Filter,
	sort resaledom.Sort,
	page resaledom.Page,
) (resaledom.PageResult[resaledom.Resale], error) {
	if q == nil || q.resaleRepo == nil {
		return resaledom.PageResult[resaledom.Resale]{}, errors.New("not supported: ResaleQuery.List")
	}

	result, err := q.resaleRepo.List(ctx, filter, sort, page)
	if err != nil {
		return resaledom.PageResult[resaledom.Resale]{}, err
	}

	result.Items = q.enrichResalesForDisplay(ctx, result.Items)

	return result, nil
}

func (q *ResaleQuery) ListByCursor(
	ctx context.Context,
	filter resaledom.Filter,
	sort resaledom.Sort,
	cpage resaledom.CursorPage,
) (resaledom.CursorPageResult[resaledom.Resale], error) {
	if q == nil || q.resaleRepo == nil {
		return resaledom.CursorPageResult[resaledom.Resale]{}, errors.New("not supported: ResaleQuery.ListByCursor")
	}

	result, err := q.resaleRepo.ListByCursor(ctx, filter, sort, cpage)
	if err != nil {
		return resaledom.CursorPageResult[resaledom.Resale]{}, err
	}

	result.Items = q.enrichResalesForDisplay(ctx, result.Items)

	return result, nil
}

func (q *ResaleQuery) GetByID(
	ctx context.Context,
	id string,
) (resaledom.Resale, error) {
	if q == nil || q.resaleRepo == nil {
		return resaledom.Resale{}, errors.New("not supported: ResaleQuery.GetByID")
	}

	if id == "" {
		return resaledom.Resale{}, resaledom.ErrInvalidID
	}

	item, err := q.resaleRepo.GetByID(ctx, id)
	if err != nil {
		return resaledom.Resale{}, err
	}

	item = q.enrichResaleForDisplay(ctx, item)

	return item, nil
}

func (q *ResaleQuery) ListByAvatarID(
	ctx context.Context,
	avatarID string,
) ([]resaledom.Resale, error) {
	if q == nil || q.resaleRepo == nil {
		return nil, errors.New("not supported: ResaleQuery.ListByAvatarID")
	}

	if avatarID == "" {
		return []resaledom.Resale{}, nil
	}

	items, err := q.resaleRepo.ListByAvatarID(ctx, avatarID)
	if err != nil {
		return nil, err
	}

	items = q.enrichResalesForDisplay(ctx, items)

	return items, nil
}

func (q *ResaleQuery) ListImages(
	ctx context.Context,
	resaleID string,
) ([]resaledom.ResaleImage, error) {
	if q == nil || q.imageRepo == nil {
		return nil, errors.New("not supported: ResaleQuery.ListImages")
	}

	if resaleID == "" {
		return nil, resaledom.ErrInvalidConditionImageResaleID
	}

	return q.imageRepo.ListByResaleID(ctx, resaleID)
}

func (q *ResaleQuery) enrichResalesForDisplay(
	ctx context.Context,
	items []resaledom.Resale,
) []resaledom.Resale {
	if len(items) == 0 {
		return items
	}

	for i := range items {
		items[i] = q.enrichResaleForDisplay(ctx, items[i])
	}

	return items
}

func (q *ResaleQuery) enrichResaleForDisplay(
	ctx context.Context,
	item resaledom.Resale,
) resaledom.Resale {
	item = q.enrichResaleWithProductAndModel(ctx, item)
	item = q.enrichResaleWithProductName(ctx, item)
	item = q.enrichResaleWithTokenBlueprint(ctx, item)
	item = q.enrichResaleWithBrandName(ctx, item)

	return item
}

func (q *ResaleQuery) enrichResaleWithProductAndModel(
	ctx context.Context,
	item resaledom.Resale,
) resaledom.Resale {
	if q == nil || q.productRepo == nil {
		return item
	}

	productID := item.ProductID
	if productID == "" {
		return item
	}

	product, err := q.productRepo.GetByID(ctx, productID)
	if err != nil {
		return item
	}

	item.ModelID = product.ModelID

	if q.modelRepo == nil || product.ModelID == "" {
		return item
	}

	modelVariation, err := q.modelRepo.GetByID(ctx, product.ModelID)
	if err != nil {
		return item
	}

	item = applyModelVariationToResale(item, modelVariation)

	return item
}

func applyModelVariationToResale(
	item resaledom.Resale,
	modelVariation modeldom.ModelVariation,
) resaledom.Resale {
	if modelVariation == nil {
		return item
	}

	item.ModelID = modelVariation.GetID()
	item.ProductBlueprintID = firstNonEmpty(
		item.ProductBlueprintID,
		modelVariation.GetProductBlueprintID(),
	)
	item.ModelNumber = modelVariation.GetModelNumber()

	switch mv := modelVariation.(type) {
	case modeldom.ApparelModelVariation:
		item = applyApparelModelVariationToResale(item, mv)

	case *modeldom.ApparelModelVariation:
		if mv != nil {
			item = applyApparelModelVariationToResale(item, *mv)
		}

	case modeldom.AlcoholModelVariation:
		item = applyAlcoholModelVariationToResale(item, mv)

	case *modeldom.AlcoholModelVariation:
		if mv != nil {
			item = applyAlcoholModelVariationToResale(item, *mv)
		}
	}

	return item
}

func applyApparelModelVariationToResale(
	item resaledom.Resale,
	modelVariation modeldom.ApparelModelVariation,
) resaledom.Resale {
	item.Kind = string(modeldom.ModelVariationKindApparel)
	item.ModelID = firstNonEmpty(item.ModelID, modelVariation.ID)
	item.ProductBlueprintID = firstNonEmpty(item.ProductBlueprintID, modelVariation.ProductBlueprintID)
	item.ModelNumber = firstNonEmpty(item.ModelNumber, modelVariation.ModelNumber)
	item.Size = modelVariation.Size
	item.Color = &resaledom.ResaleColor{
		Name: modelVariation.Color.Name,
		RGB:  modelVariation.Color.RGB,
	}
	item.Measurements = modelVariation.Measurements

	return item
}

func applyAlcoholModelVariationToResale(
	item resaledom.Resale,
	modelVariation modeldom.AlcoholModelVariation,
) resaledom.Resale {
	item.Kind = string(modeldom.ModelVariationKindAlcohol)
	item.ModelID = firstNonEmpty(item.ModelID, modelVariation.ID)
	item.ProductBlueprintID = firstNonEmpty(item.ProductBlueprintID, modelVariation.ProductBlueprintID)
	item.ModelNumber = firstNonEmpty(item.ModelNumber, modelVariation.ModelNumber)
	item.Volume = &resaledom.ResaleVolume{
		Amount: modelVariation.Volume.Value,
		Unit:   modelVariation.Volume.Unit,
	}

	return item
}

func (q *ResaleQuery) enrichResaleWithProductName(
	ctx context.Context,
	item resaledom.Resale,
) resaledom.Resale {
	if q == nil || q.productBlueprintRepo == nil {
		return item
	}

	productBlueprintID := item.ProductBlueprintID
	if productBlueprintID == "" {
		return item
	}

	pb, err := q.productBlueprintRepo.GetByID(ctx, productBlueprintID)
	if err != nil {
		return item
	}

	item.ProductName = pb.ProductName

	return item
}

func (q *ResaleQuery) enrichResaleWithTokenBlueprint(
	ctx context.Context,
	item resaledom.Resale,
) resaledom.Resale {
	if q == nil || q.tokenBlueprintRepo == nil {
		return item
	}

	tokenBlueprintID := item.TokenBlueprintID
	if tokenBlueprintID == "" {
		return item
	}

	tb, err := q.tokenBlueprintRepo.GetByID(ctx, tokenBlueprintID)
	if err != nil {
		return item
	}

	if tb == nil {
		return item
	}

	item.TokenName = tb.Name

	if tb.IconURL != "" {
		item.ImageURL = tb.IconURL
	}

	return item
}

func (q *ResaleQuery) enrichResaleWithBrandName(
	ctx context.Context,
	item resaledom.Resale,
) resaledom.Resale {
	if q == nil || q.brandRepo == nil {
		return item
	}

	brandID := item.BrandID
	if brandID == "" {
		return item
	}

	brand, err := q.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return item
	}

	item.BrandName = brand.Name

	return item
}

func firstNonEmpty(primary string, fallback string) string {
	if primary != "" {
		return primary
	}

	return fallback
}
