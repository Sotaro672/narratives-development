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
	return q.newDisplayEnricher().enrichResalesForDisplay(ctx, items)
}

func (q *ResaleQuery) enrichResaleForDisplay(
	ctx context.Context,
	item resaledom.Resale,
) resaledom.Resale {
	return q.newDisplayEnricher().enrichResaleForDisplay(ctx, item)
}

func (q *ResaleQuery) newDisplayEnricher() *resaleDisplayEnricher {
	if q == nil {
		return newResaleDisplayEnricher(resaleDisplayEnricherConfig{})
	}

	return newResaleDisplayEnricher(resaleDisplayEnricherConfig{
		productRepo:          q.productRepo,
		modelRepo:            q.modelRepo,
		productBlueprintRepo: q.productBlueprintRepo,
		tokenBlueprintRepo:   q.tokenBlueprintRepo,
		brandRepo:            q.brandRepo,

		// ResaleQuery の既存挙動:
		// - avatarName/avatarIcon は補完しない
		// - resale image は ListImages API 側で取得する
		// - tokenBlueprint.IconURL を ImageURL fallback として使う
		includeAvatar:               false,
		includeImage:                false,
		useTokenIconAsImageFallback: true,
	})
}
