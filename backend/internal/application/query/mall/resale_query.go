// backend/internal/application/query/mall/resale_query.go
package mall

import (
	"context"
	"errors"
	"strings"

	branddom "narratives/internal/domain/brand"
	productblueprintdom "narratives/internal/domain/productBlueprint"
	resaledom "narratives/internal/domain/resale"
	tokenblueprintdom "narratives/internal/domain/tokenBlueprint"
)

type ResaleQuery struct {
	resaleRepo           resaledom.Repository
	imageRepo            resaledom.ImageRepository
	productBlueprintRepo productblueprintdom.Repository
	tokenBlueprintRepo   tokenblueprintdom.RepositoryPort
	brandRepo            branddom.Repository
}

func NewResaleQuery(
	resaleRepo resaledom.Repository,
	imageRepo resaledom.ImageRepository,
	productBlueprintRepo productblueprintdom.Repository,
	tokenBlueprintRepo tokenblueprintdom.RepositoryPort,
	brandRepo branddom.Repository,
) *ResaleQuery {
	return &ResaleQuery{
		resaleRepo:           resaleRepo,
		imageRepo:            imageRepo,
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

	id = strings.TrimSpace(id)
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

	avatarID = strings.TrimSpace(avatarID)
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

	resaleID = strings.TrimSpace(resaleID)
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
	item = q.enrichResaleWithProductName(ctx, item)
	item = q.enrichResaleWithTokenName(ctx, item)
	item = q.enrichResaleWithBrandName(ctx, item)

	return item
}

func (q *ResaleQuery) enrichResaleWithProductName(
	ctx context.Context,
	item resaledom.Resale,
) resaledom.Resale {
	if q == nil || q.productBlueprintRepo == nil {
		return item
	}

	productBlueprintID := strings.TrimSpace(item.ProductBlueprintID)
	if productBlueprintID == "" {
		return item
	}

	pb, err := q.productBlueprintRepo.GetByID(ctx, productBlueprintID)
	if err != nil {
		return item
	}

	item.ProductName = strings.TrimSpace(pb.ProductName)

	return item
}

func (q *ResaleQuery) enrichResaleWithTokenName(
	ctx context.Context,
	item resaledom.Resale,
) resaledom.Resale {
	if q == nil || q.tokenBlueprintRepo == nil {
		return item
	}

	tokenBlueprintID := strings.TrimSpace(item.TokenBlueprintID)
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

	item.TokenName = strings.TrimSpace(tb.Name)

	return item
}

func (q *ResaleQuery) enrichResaleWithBrandName(
	ctx context.Context,
	item resaledom.Resale,
) resaledom.Resale {
	if q == nil || q.brandRepo == nil {
		return item
	}

	brandID := strings.TrimSpace(item.BrandID)
	if brandID == "" {
		return item
	}

	brand, err := q.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return item
	}

	item.BrandName = strings.TrimSpace(brand.Name)

	return item
}
