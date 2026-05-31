// backend/internal/application/query/console/list_management_query.go
package query

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	resolver "narratives/internal/application/resolver"
	listdom "narratives/internal/domain/list"
)

// ============================================================
// Ports (read-only) - list management
// ============================================================

type ListManagementLister interface {
	GetByID(ctx context.Context, id string) (listdom.List, error)
	ListIDsByInventoryID(ctx context.Context, inventoryID string) ([]string, error)
}

// ============================================================
// DTO (query -> handler)
// ============================================================

type ListRowDTO struct {
	ID          string `json:"id"`
	InventoryID string `json:"inventoryId"`

	Title string `json:"title"`

	ProductBlueprintID string `json:"productBlueprintId"`
	TokenBlueprintID   string `json:"tokenBlueprintId"`

	ProductName      string `json:"productName"`
	ProductBrandID   string `json:"productBrandId"`
	ProductBrandName string `json:"productBrandName"`

	TokenName      string `json:"tokenName"`
	TokenBrandID   string `json:"tokenBrandId"`
	TokenBrandName string `json:"tokenBrandName"`

	AssigneeID   string `json:"assigneeId"`
	AssigneeName string `json:"assigneeName"`

	Status string `json:"status"`

	// listManagement に表示する作成日（RFC3339）
	CreatedAt string `json:"createdAt,omitempty"`
}

// ============================================================
// ListManagementQuery (listManagement.tsx)
// ============================================================

type ListManagementQuery struct {
	lister       ListManagementLister
	nameResolver *resolver.NameResolver

	pbGetter ProductBlueprintGetter
	tbGetter TokenBlueprintGetter

	// company boundary source
	invRows InventoryRowsLister
}

// ============================================================
// SINGLE ENTRYPOINT
//
// 要件:
// - companyId を使わない「単純な list」は禁止（＝invRows が必須）
// - List 全体 scan は禁止
// - current company 境界の inventoryID を列挙し、
//   ListIDsByInventoryID -> GetByID の順で該当 List のみ取得する
// - この ctor のみを公開し、配線を集中させる
// ============================================================

type NewListManagementQueryParams struct {
	Lister       ListManagementLister
	NameResolver *resolver.NameResolver

	PBGetter ProductBlueprintGetter
	TBGetter TokenBlueprintGetter

	InvRows InventoryRowsLister // REQUIRED
}

func NewListManagementQuery(p NewListManagementQueryParams) *ListManagementQuery {
	return &ListManagementQuery{
		lister:       p.Lister,
		nameResolver: p.NameResolver,
		pbGetter:     p.PBGetter,
		tbGetter:     p.TBGetter,
		invRows:      p.InvRows,
	}
}

// ============================================================
// Query
// ============================================================

func (q *ListManagementQuery) ListRows(
	ctx context.Context,
	filter listdom.Filter,
	sort listdom.Sort,
	page listdom.Page,
) (listdom.PageResult[ListRowDTO], error) {
	page = NormalizePage(page)
	_ = sort // 現状は company boundary first の取得順を維持する

	// company boundary を使わない単純 list は禁止
	if q == nil || q.lister == nil || q.invRows == nil {
		return listdom.PageResult[ListRowDTO]{}, errors.New("ListManagementQuery.ListRows: wiring is incomplete (lister/invRows required)")
	}

	allowedInventoryIDs, allowedSet, err := allowedInventoryIDsFromContext(ctx, q.invRows)
	if err != nil {
		log.Printf("[ListManagementQuery] ERROR company boundary (inventory_query) failed: %v", err)
		return listdom.PageResult[ListRowDTO]{}, err
	}

	if len(allowedInventoryIDs) == 0 {
		return listdom.PageResult[ListRowDTO]{
			Items:      []ListRowDTO{},
			Page:       page.Number,
			PerPage:    page.PerPage,
			TotalCount: 0,
			TotalPages: 0,
		}, nil
	}

	tokenNameCache := map[string]string{}
	memberNameCache := map[string]string{}
	productNameCache := map[string]string{}
	brandIDCachePB := map[string]string{}
	brandIDCacheTB := map[string]string{}
	brandNameByIDCache := map[string]string{}

	allowedAll := make([]ListRowDTO, 0, page.PerPage)
	seenListID := map[string]struct{}{}

	for _, inventoryID := range allowedInventoryIDs {
		if inventoryID == "" {
			continue
		}

		listIDs, e := q.lister.ListIDsByInventoryID(ctx, inventoryID)
		if e != nil {
			log.Printf("[ListManagementQuery] ERROR ListIDsByInventoryID failed inventoryID=%q: %v", inventoryID, e)
			return listdom.PageResult[ListRowDTO]{}, e
		}

		for _, listID := range listIDs {
			listID = strings.TrimSpace(listID)
			if listID == "" {
				continue
			}

			if _, ok := seenListID[listID]; ok {
				continue
			}
			seenListID[listID] = struct{}{}

			it, e := q.lister.GetByID(ctx, listID)
			if e != nil {
				if errors.Is(e, listdom.ErrNotFound) {
					continue
				}

				log.Printf("[ListManagementQuery] ERROR GetByID failed listID=%q inventoryID=%q: %v", listID, inventoryID, e)
				return listdom.PageResult[ListRowDTO]{}, e
			}

			id := it.ID
			invID := it.InventoryID

			// Safety:
			// ListIDsByInventoryID で取った後も、List 実体の InventoryID が
			// current company 境界内か必ず再確認する。
			if !InventoryAllowed(allowedSet, invID) {
				continue
			}

			if !listMatchesFilter(it, filter) {
				continue
			}

			assigneeID := it.AssigneeID

			pbID, tbID, ok := ParseInventoryIDStrict(invID)
			if !ok {
				continue
			}

			title := it.Title

			productName := title
			if pbID != "" && q.nameResolver != nil {
				if cached, ok := productNameCache[pbID]; ok {
					if cached != "" {
						productName = cached
					}
				} else {
					resolved := q.nameResolver.ResolveProductName(ctx, pbID)
					productNameCache[pbID] = resolved
					if resolved != "" {
						productName = resolved
					}
				}
			}

			tokenName := ""
			if tbID != "" && q.nameResolver != nil {
				if cached, ok := tokenNameCache[tbID]; ok {
					tokenName = cached
				} else {
					resolved := q.nameResolver.ResolveTokenName(ctx, tbID)
					tokenNameCache[tbID] = resolved
					tokenName = resolved
				}
			}

			assigneeName := ""
			if assigneeID != "" && q.nameResolver != nil {
				if cached, ok := memberNameCache[assigneeID]; ok {
					assigneeName = cached
				} else {
					resolved := q.nameResolver.ResolveAssigneeName(ctx, assigneeID)
					memberNameCache[assigneeID] = resolved
					assigneeName = resolved
				}
			}
			if assigneeName == "" {
				assigneeName = "未設定"
			}

			productBrandID := ""
			tokenBrandID := ""

			if pbID != "" && q.pbGetter != nil {
				if cached, ok := brandIDCachePB[pbID]; ok {
					productBrandID = cached
				} else {
					pb, ee := q.pbGetter.GetByID(ctx, pbID)
					if ee == nil {
						productBrandID = pb.BrandID
					}
					brandIDCachePB[pbID] = productBrandID
				}
			}

			if tbID != "" && q.tbGetter != nil {
				if cached, ok := brandIDCacheTB[tbID]; ok {
					tokenBrandID = cached
				} else {
					tb, ee := q.tbGetter.GetByID(ctx, tbID)
					if ee == nil && tb != nil {
						tokenBrandID = tb.BrandID
					}
					brandIDCacheTB[tbID] = tokenBrandID
				}
			}

			productBrandName := ""
			tokenBrandName := ""

			if q.nameResolver != nil {
				if productBrandID != "" {
					if cached, ok := brandNameByIDCache[productBrandID]; ok {
						productBrandName = cached
					} else {
						resolved := q.nameResolver.ResolveBrandName(ctx, productBrandID)
						brandNameByIDCache[productBrandID] = resolved
						productBrandName = resolved
					}
				}

				if tokenBrandID != "" {
					if cached, ok := brandNameByIDCache[tokenBrandID]; ok {
						tokenBrandName = cached
					} else {
						resolved := q.nameResolver.ResolveBrandName(ctx, tokenBrandID)
						brandNameByIDCache[tokenBrandID] = resolved
						tokenBrandName = resolved
					}
				}
			}

			createdAt := ""
			if !it.CreatedAt.IsZero() {
				createdAt = it.CreatedAt.UTC().Format(time.RFC3339)
			}

			allowedAll = append(allowedAll, ListRowDTO{
				ID:          NonEmpty(id, "(missing id)"),
				InventoryID: invID,
				Title:       title,

				ProductBlueprintID: pbID,
				TokenBlueprintID:   tbID,

				ProductName:      productName,
				ProductBrandID:   productBrandID,
				ProductBrandName: productBrandName,

				TokenName:      tokenName,
				TokenBrandID:   tokenBrandID,
				TokenBrandName: tokenBrandName,

				AssigneeID:   assigneeID,
				AssigneeName: assigneeName,

				Status: string(it.Status),

				CreatedAt: createdAt,
			})
		}
	}

	totalCount := len(allowedAll)
	tp := TotalPages(totalCount, page.PerPage)

	start := (page.Number - 1) * page.PerPage
	if start < 0 {
		start = 0
	}

	if start >= totalCount {
		return listdom.PageResult[ListRowDTO]{
			Items:      []ListRowDTO{},
			Page:       page.Number,
			PerPage:    page.PerPage,
			TotalCount: totalCount,
			TotalPages: tp,
		}, nil
	}

	end := MinInt(start+page.PerPage, totalCount)

	return listdom.PageResult[ListRowDTO]{
		Items:      allowedAll[start:end],
		Page:       page.Number,
		PerPage:    page.PerPage,
		TotalCount: totalCount,
		TotalPages: tp,
	}, nil
}

// ============================================================
// local helpers
// ============================================================

func allowedInventoryIDsFromContext(
	ctx context.Context,
	invRows InventoryRowsLister,
) ([]string, map[string]struct{}, error) {
	if invRows == nil {
		return nil, nil, errors.New("inventory rows lister is nil (company boundary via inventory_query is not configured)")
	}

	rows, err := invRows.ListByCurrentCompany(ctx)
	if err != nil {
		return nil, nil, err
	}

	ids := make([]string, 0, len(rows))
	set := map[string]struct{}{}

	for _, r := range rows {
		pbID := r.ProductBlueprintID
		tbID := r.TokenBlueprintID
		if pbID == "" || tbID == "" {
			continue
		}

		invID := pbID + "__" + tbID
		if _, ok := set[invID]; ok {
			continue
		}

		set[invID] = struct{}{}
		ids = append(ids, invID)
	}

	return ids, set, nil
}

func listMatchesFilter(it listdom.List, filter listdom.Filter) bool {
	if len(filter.IDs) > 0 && !stringIn(filter.IDs, it.ID) {
		return false
	}

	if filter.AssigneeID != nil && *filter.AssigneeID != "" {
		if it.AssigneeID != *filter.AssigneeID {
			return false
		}
	}

	if filter.Status != nil && *filter.Status != "" {
		if it.Status != *filter.Status {
			return false
		}
	}

	if len(filter.Statuses) > 0 && !statusIn(filter.Statuses, it.Status) {
		return false
	}

	if len(filter.InventoryIDs) > 0 && !stringIn(filter.InventoryIDs, it.InventoryID) {
		return false
	}

	if filter.SearchQuery != "" {
		q := strings.ToLower(strings.TrimSpace(filter.SearchQuery))
		if q != "" {
			haystacks := []string{
				it.ID,
				it.InventoryID,
				it.Title,
				it.Description,
				it.AssigneeID,
				string(it.Status),
			}

			found := false
			for _, h := range haystacks {
				if strings.Contains(strings.ToLower(h), q) {
					found = true
					break
				}
			}

			if !found {
				return false
			}
		}
	}

	if hasPriceFilter(filter) && !listMatchesPriceFilter(it, filter) {
		return false
	}

	return true
}

func hasPriceFilter(filter listdom.Filter) bool {
	return len(filter.ModelNumbers) > 0 ||
		filter.MinPrice != nil ||
		filter.MaxPrice != nil
}

func listMatchesPriceFilter(it listdom.List, filter listdom.Filter) bool {
	if len(it.Prices) == 0 {
		return false
	}

	for _, row := range it.Prices {
		if row.ModelID == "" {
			continue
		}

		if len(filter.ModelNumbers) > 0 && !stringIn(filter.ModelNumbers, row.ModelID) {
			continue
		}

		if filter.MinPrice != nil && row.Price < *filter.MinPrice {
			continue
		}

		if filter.MaxPrice != nil && row.Price > *filter.MaxPrice {
			continue
		}

		return true
	}

	return false
}

func stringIn(values []string, target string) bool {
	if len(values) == 0 {
		return false
	}
	if target == "" {
		return false
	}

	for _, v := range values {
		if v == target {
			return true
		}
	}

	return false
}

func statusIn(values []listdom.ListStatus, target listdom.ListStatus) bool {
	if len(values) == 0 {
		return false
	}
	if target == "" {
		return false
	}

	for _, v := range values {
		if v == target {
			return true
		}
	}

	return false
}
