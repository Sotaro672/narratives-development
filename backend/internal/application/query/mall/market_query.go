// backend/internal/application/query/mall/market_query.go
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

// MarketResaleRepository is the repository dependency used by MarketQuery.
type MarketResaleRepository interface {
	List(ctx context.Context, filter resaledom.Filter, sort resaledom.Sort, page resaledom.Page) (resaledom.PageResult[resaledom.Resale], error)
	ListByCursor(ctx context.Context, filter resaledom.Filter, sort resaledom.Sort, cpage resaledom.CursorPage) (resaledom.CursorPageResult[resaledom.Resale], error)
	GetByID(ctx context.Context, id string) (resaledom.Resale, error)
	ListByAvatarID(ctx context.Context, avatarID string) ([]resaledom.Resale, error)
}

// MarketQuery is the buyer-facing public market read model.
//
// Public market policy:
// - Only listing resales are visible.
// - Suspended resales are never returned from List / ListByCursor.
// - Own resales are excluded from List / ListByCursor when viewer avatarId is provided.
// - Detail visibility is guarded by status.
// - Display fields are enriched here.
//
// NOTE:
// Current implementation treats filter.AvatarIDs as viewer avatarIds for exclusion.
// Public market listing itself does not support seller avatar filtering because returning
// the viewer's own listings in buyer-facing market is contradictory.
type MarketQuery struct {
	resaleRepo           MarketResaleRepository
	imageRepo            resaledom.ImageRepository
	productBlueprintRepo productblueprintdom.Repository
	tokenBlueprintRepo   tokenblueprintdom.RepositoryPort
	brandRepo            branddom.Repository
}

func NewMarketQuery(
	resaleRepo MarketResaleRepository,
	imageRepo resaledom.ImageRepository,
	productBlueprintRepo productblueprintdom.Repository,
	tokenBlueprintRepo tokenblueprintdom.RepositoryPort,
	brandRepo branddom.Repository,
) *MarketQuery {
	return &MarketQuery{
		resaleRepo:           resaleRepo,
		imageRepo:            imageRepo,
		productBlueprintRepo: productBlueprintRepo,
		tokenBlueprintRepo:   tokenBlueprintRepo,
		brandRepo:            brandRepo,
	}
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

	viewerAvatarIDs := normalizeViewerAvatarIDs(filter.AvatarIDs)

	filter = forcePublicMarketFilter(filter)
	filter = removeViewerAvatarIDsFromPublicMarketFilter(filter)

	sort = normalizePublicMarketSort(sort)
	page = normalizePublicMarketPage(page)

	result, err := q.resaleRepo.List(ctx, filter, sort, page)
	if err != nil {
		return resaledom.PageResult[resaledom.Resale]{}, err
	}

	result.Items, err = q.excludeOwnResales(ctx, result.Items, viewerAvatarIDs)
	if err != nil {
		return resaledom.PageResult[resaledom.Resale]{}, err
	}

	result.Items = q.enrichResalesForDisplay(ctx, result.Items)
	result = normalizePageResultCount(result, page)

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

	viewerAvatarIDs := normalizeViewerAvatarIDs(filter.AvatarIDs)

	filter = forcePublicMarketFilter(filter)
	filter = removeViewerAvatarIDsFromPublicMarketFilter(filter)

	sort = normalizePublicMarketSort(sort)
	cpage = normalizePublicMarketCursorPage(cpage)

	result, err := q.resaleRepo.ListByCursor(ctx, filter, sort, cpage)
	if err != nil {
		return resaledom.CursorPageResult[resaledom.Resale]{}, err
	}

	result.Items, err = q.excludeOwnResales(ctx, result.Items, viewerAvatarIDs)
	if err != nil {
		return resaledom.CursorPageResult[resaledom.Resale]{}, err
	}

	result.Items = q.enrichResalesForDisplay(ctx, result.Items)

	return result, nil
}

func (q *MarketQuery) GetByID(ctx context.Context, id string) (resaledom.Resale, error) {
	if q == nil || q.resaleRepo == nil {
		return resaledom.Resale{}, errors.New("not supported: MarketQuery.GetByID")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return resaledom.Resale{}, resaledom.ErrInvalidID
	}

	item, err := q.resaleRepo.GetByID(ctx, id)
	if err != nil {
		return resaledom.Resale{}, err
	}

	if item.Status != resaledom.StatusListing {
		return resaledom.Resale{}, resaledom.ErrNotFound
	}

	item = q.enrichResaleForDisplay(ctx, item)

	return item, nil
}

func (q *MarketQuery) excludeOwnResales(
	ctx context.Context,
	items []resaledom.Resale,
	viewerAvatarIDs []string,
) ([]resaledom.Resale, error) {
	if len(items) == 0 || len(viewerAvatarIDs) == 0 {
		return items, nil
	}

	if q == nil || q.resaleRepo == nil {
		return items, nil
	}

	ownIDs := make(map[string]struct{})

	for _, avatarID := range viewerAvatarIDs {
		ownItems, err := q.resaleRepo.ListByAvatarID(ctx, avatarID)
		if err != nil {
			return nil, err
		}

		for _, own := range ownItems {
			id := strings.TrimSpace(own.ID)
			if id == "" {
				continue
			}

			ownIDs[id] = struct{}{}
		}
	}

	if len(ownIDs) == 0 {
		return items, nil
	}

	out := make([]resaledom.Resale, 0, len(items))
	for _, item := range items {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}

		if _, ok := ownIDs[id]; ok {
			continue
		}

		out = append(out, item)
	}

	return out, nil
}

func (q *MarketQuery) enrichResalesForDisplay(
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

func (q *MarketQuery) enrichResaleForDisplay(
	ctx context.Context,
	item resaledom.Resale,
) resaledom.Resale {
	item = q.enrichResaleWithProductName(ctx, item)
	item = q.enrichResaleWithTokenName(ctx, item)
	item = q.enrichResaleWithBrandName(ctx, item)
	item = q.enrichResaleWithImageURL(ctx, item)

	return item
}

func (q *MarketQuery) enrichResaleWithProductName(
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

func (q *MarketQuery) enrichResaleWithTokenName(
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

func (q *MarketQuery) enrichResaleWithBrandName(
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

func (q *MarketQuery) enrichResaleWithImageURL(
	ctx context.Context,
	item resaledom.Resale,
) resaledom.Resale {
	if q == nil || q.imageRepo == nil {
		return item
	}

	resaleID := strings.TrimSpace(item.ID)
	if resaleID == "" {
		return item
	}

	images, err := q.imageRepo.ListByResaleID(ctx, resaleID)
	if err != nil || len(images) == 0 {
		return item
	}

	primaryImageID := strings.TrimSpace(item.ImageID)

	if primaryImageID != "" {
		for _, image := range images {
			if strings.TrimSpace(image.ID) == primaryImageID {
				item.ImageURL = strings.TrimSpace(image.URL)
				return item
			}
		}
	}

	for _, image := range images {
		if strings.TrimSpace(image.URL) != "" {
			item.ImageURL = strings.TrimSpace(image.URL)
			return item
		}
	}

	return item
}

func forcePublicMarketFilter(filter resaledom.Filter) resaledom.Filter {
	status := resaledom.StatusListing

	filter.Status = &status
	filter.Statuses = nil

	return filter
}

func removeViewerAvatarIDsFromPublicMarketFilter(filter resaledom.Filter) resaledom.Filter {
	filter.AvatarIDs = nil

	return filter
}

func normalizeViewerAvatarIDs(ids []string) []string {
	if len(ids) == 0 {
		return []string{}
	}

	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))

	for _, id := range ids {
		normalized := strings.TrimSpace(id)
		if normalized == "" {
			continue
		}

		if _, ok := seen[normalized]; ok {
			continue
		}

		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}

	return out
}

func normalizePublicMarketSort(sort resaledom.Sort) resaledom.Sort {
	column := strings.TrimSpace(sort.Column)
	order := sort.Order

	if order != resaledom.SortAsc {
		order = resaledom.SortDesc
	}

	switch column {
	case "createdAt", "price", "productName", "brandName", "tokenName":
		return resaledom.Sort{
			Column: column,
			Order:  order,
		}

	case "updatedAt", "updated_at", "":
		return resaledom.Sort{
			Column: "createdAt",
			Order:  order,
		}

	default:
		return resaledom.Sort{
			Column: "createdAt",
			Order:  order,
		}
	}
}

func normalizePublicMarketPage(page resaledom.Page) resaledom.Page {
	if page.Number <= 0 {
		page.Number = 1
	}

	if page.PerPage <= 0 {
		page.PerPage = 20
	}

	if page.PerPage > 100 {
		page.PerPage = 100
	}

	return page
}

func normalizePageResultCount(
	result resaledom.PageResult[resaledom.Resale],
	page resaledom.Page,
) resaledom.PageResult[resaledom.Resale] {
	result.TotalCount = len(result.Items)

	if page.PerPage <= 0 {
		result.TotalPages = 1
		return result
	}

	if result.TotalCount == 0 {
		result.TotalPages = 0
		return result
	}

	totalPages := result.TotalCount / page.PerPage
	if result.TotalCount%page.PerPage != 0 {
		totalPages++
	}

	result.TotalPages = totalPages

	return result
}

func normalizePublicMarketCursorPage(page resaledom.CursorPage) resaledom.CursorPage {
	page.After = strings.TrimSpace(page.After)

	if page.Limit <= 0 {
		page.Limit = 20
	}

	if page.Limit > 100 {
		page.Limit = 100
	}

	return page
}
