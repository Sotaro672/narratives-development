// backend/internal/application/query/mall/resale_query.go
package mall

import (
	"context"
	"errors"

	applicationport "narratives/internal/application/port"
	mallshared "narratives/internal/application/query/mall/shared"
	avatardom "narratives/internal/domain/avatar"
	resaledom "narratives/internal/domain/resale"
)

type ResaleQuery struct {
	resaleRepo      resaledom.Repository
	imageRepo       applicationport.ResaleImageLister
	displayResolver mallshared.MallDisplayResolver
	avatarRepo      avatardom.Repository
}

func NewResaleQuery(
	resaleRepo resaledom.Repository,
	imageRepo applicationport.ResaleImageLister,
	displayResolver mallshared.MallDisplayResolver,
	avatarRepo avatardom.Repository,
) *ResaleQuery {
	return &ResaleQuery{
		resaleRepo:      resaleRepo,
		imageRepo:       imageRepo,
		displayResolver: displayResolver,
		avatarRepo:      avatarRepo,
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
		displayResolver: q.displayResolver,
		avatarRepo:      q.avatarRepo,

		// ResaleQuery の表示補完:
		// - avatarName/avatarIcon を補完する
		// - resale image は ListImages API 側で取得する
		// - tokenBlueprint.IconURL を ImageURL fallback として使う
		includeAvatar:               true,
		includeImage:                false,
		useTokenIconAsImageFallback: true,
	})
}
