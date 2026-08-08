// backend/internal/application/query/mall/market_query.go
package mall

import (
	"context"
	"errors"

	mallshared "narratives/internal/application/query/mall/shared"
	avatardom "narratives/internal/domain/avatar"
	resaledom "narratives/internal/domain/resale"
)

// MarketResaleRepository is the repository dependency used by MarketQuery.
type MarketResaleRepository interface {
	List(ctx context.Context, filter resaledom.Filter, sort resaledom.Sort, page resaledom.Page) (resaledom.PageResult[resaledom.Resale], error)
	ListByCursor(ctx context.Context, filter resaledom.Filter, sort resaledom.Sort, cpage resaledom.CursorPage) (resaledom.CursorPageResult[resaledom.Resale], error)
	GetByID(ctx context.Context, id string) (resaledom.Resale, error)
}

// MarketQuery is the buyer-facing market read model.
//
// Market policy:
// - Only listing resales are visible.
// - Suspended resales are never returned from List / ListByCursor.
// - Own resales are excluded from List / ListByCursor by ExcludeAvatarIDs.
// - Detail visibility is guarded by status and viewer avatarId.
// - Display fields are enriched here.
// - Images are visible only when the parent resale is listing and not owned by the viewer.
type MarketQuery struct {
	resaleRepo      MarketResaleRepository
	imageRepo       resaledom.ImageRepository
	displayResolver mallshared.MallDisplayResolver
	avatarRepo      avatardom.Repository
}

func NewMarketQuery(
	resaleRepo MarketResaleRepository,
	imageRepo resaledom.ImageRepository,
	displayResolver mallshared.MallDisplayResolver,
	avatarRepo ...avatardom.Repository,
) *MarketQuery {
	q := &MarketQuery{
		resaleRepo:      resaleRepo,
		imageRepo:       imageRepo,
		displayResolver: displayResolver,
	}

	if len(avatarRepo) > 0 {
		q.avatarRepo = avatarRepo[0]
	}

	return q
}

func (q *MarketQuery) List(
	ctx context.Context,
	filter resaledom.Filter,
	sort resaledom.Sort,
	page resaledom.Page,
) (resaledom.PageResult[resaledom.Resale], error) {
	if q == nil || q.resaleRepo == nil {
		return resaledom.PageResult[resaledom.Resale]{}, errors.New("not supported: MarketQuery.List")
	}

	filter = forcePublicMarketFilter(filter)
	filter.ExcludeAvatarIDs = normalizeMarketAvatarIDs(filter.ExcludeAvatarIDs)

	sort = normalizePublicMarketSort(sort)
	page = normalizePublicMarketPage(page)

	result, err := q.resaleRepo.List(ctx, filter, sort, page)
	if err != nil {
		return resaledom.PageResult[resaledom.Resale]{}, err
	}

	result.Items = q.enrichResalesForDisplay(ctx, result.Items)

	return result, nil
}

func (q *MarketQuery) ListByCursor(
	ctx context.Context,
	filter resaledom.Filter,
	sort resaledom.Sort,
	cpage resaledom.CursorPage,
) (resaledom.CursorPageResult[resaledom.Resale], error) {
	if q == nil || q.resaleRepo == nil {
		return resaledom.CursorPageResult[resaledom.Resale]{}, errors.New("not supported: MarketQuery.ListByCursor")
	}

	filter = forcePublicMarketFilter(filter)
	filter.ExcludeAvatarIDs = normalizeMarketAvatarIDs(filter.ExcludeAvatarIDs)

	sort = normalizePublicMarketSort(sort)
	cpage = normalizePublicMarketCursorPage(cpage)

	result, err := q.resaleRepo.ListByCursor(ctx, filter, sort, cpage)
	if err != nil {
		return resaledom.CursorPageResult[resaledom.Resale]{}, err
	}

	result.Items = q.enrichResalesForDisplay(ctx, result.Items)

	return result, nil
}

func (q *MarketQuery) GetByID(
	ctx context.Context,
	id string,
	viewerAvatarID string,
) (resaledom.Resale, error) {
	if q == nil || q.resaleRepo == nil {
		return resaledom.Resale{}, errors.New("not supported: MarketQuery.GetByID")
	}

	if id == "" {
		return resaledom.Resale{}, resaledom.ErrInvalidID
	}

	if viewerAvatarID == "" {
		return resaledom.Resale{}, resaledom.ErrNotFound
	}

	item, err := q.resaleRepo.GetByID(ctx, id)
	if err != nil {
		return resaledom.Resale{}, err
	}

	if item.Status != resaledom.StatusListing {
		return resaledom.Resale{}, resaledom.ErrNotFound
	}

	if item.AvatarID == viewerAvatarID {
		return resaledom.Resale{}, resaledom.ErrNotFound
	}

	item = q.enrichResaleForDisplay(ctx, item)

	return item, nil
}

func (q *MarketQuery) ListImagesByResaleID(
	ctx context.Context,
	resaleID string,
	viewerAvatarID string,
) ([]resaledom.ResaleImage, error) {
	if q == nil || q.resaleRepo == nil || q.imageRepo == nil {
		return nil, errors.New("not supported: MarketQuery.ListImagesByResaleID")
	}

	if resaleID == "" {
		return nil, resaledom.ErrInvalidID
	}

	if viewerAvatarID == "" {
		return nil, resaledom.ErrNotFound
	}

	item, err := q.resaleRepo.GetByID(ctx, resaleID)
	if err != nil {
		return nil, err
	}

	if item.Status != resaledom.StatusListing {
		return nil, resaledom.ErrNotFound
	}

	if item.AvatarID == viewerAvatarID {
		return nil, resaledom.ErrNotFound
	}

	images, err := q.imageRepo.ListByResaleID(ctx, resaleID)
	if err != nil {
		return nil, err
	}

	return images, nil
}

func (q *MarketQuery) enrichResalesForDisplay(
	ctx context.Context,
	items []resaledom.Resale,
) []resaledom.Resale {
	return q.newDisplayEnricher().enrichResalesForDisplay(ctx, items)
}

func (q *MarketQuery) enrichResaleForDisplay(
	ctx context.Context,
	item resaledom.Resale,
) resaledom.Resale {
	return q.newDisplayEnricher().enrichResaleForDisplay(ctx, item)
}

func (q *MarketQuery) newDisplayEnricher() *resaleDisplayEnricher {
	if q == nil {
		return newResaleDisplayEnricher(resaleDisplayEnricherConfig{})
	}

	return newResaleDisplayEnricher(resaleDisplayEnricherConfig{
		displayResolver: q.displayResolver,
		imageRepo:       q.imageRepo,
		avatarRepo:      q.avatarRepo,

		// MarketQuery の既存挙動:
		// - avatarName/avatarIcon を補完する
		// - primary resale image URL を補完する
		// - tokenBlueprint.IconURL は TokenIcon にのみ入れる
		includeAvatar:               true,
		includeImage:                true,
		useTokenIconAsImageFallback: false,
	})
}

func forcePublicMarketFilter(filter resaledom.Filter) resaledom.Filter {
	status := resaledom.StatusListing

	filter.Status = &status
	filter.Statuses = nil

	return filter
}

func normalizeMarketAvatarIDs(ids []string) []string {
	if len(ids) == 0 {
		return []string{}
	}

	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))

	for _, id := range ids {
		if id == "" {
			continue
		}

		if _, ok := seen[id]; ok {
			continue
		}

		seen[id] = struct{}{}
		out = append(out, id)
	}

	return out
}

func normalizePublicMarketSort(sort resaledom.Sort) resaledom.Sort {
	allowedColumns := map[string]string{
		"createdAt":   "createdAt",
		"price":       "price",
		"productName": "productName",
		"brandName":   "brandName",
		"tokenName":   "tokenName",

		// aliases
		"updatedAt":  "createdAt",
		"updated_at": "createdAt",
	}

	column, order := mallshared.NormalizeSortParts(
		sort.Column,
		string(sort.Order),
		allowedColumns,
		"createdAt",
		string(resaledom.SortDesc),
	)

	return resaledom.Sort{
		Column: column,
		Order:  resaledom.SortOrder(order),
	}
}

func normalizePublicMarketPage(page resaledom.Page) resaledom.Page {
	number, perPage := mallshared.NormalizeIntPage(
		page.Number,
		page.PerPage,
		1,
		20,
		100,
	)

	page.Number = number
	page.PerPage = perPage

	return page
}

func normalizePublicMarketCursorPage(page resaledom.CursorPage) resaledom.CursorPage {
	page.Limit = mallshared.NormalizeLimit(page.Limit, 20, 100)

	return page
}
