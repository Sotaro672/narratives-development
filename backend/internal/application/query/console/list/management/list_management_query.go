// backend/internal/application/query/console/list/management/list_management_query.go
package listManagement

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	querydto "narratives/internal/application/query/console/dto"
	resolver "narratives/internal/application/resolver"
	listdom "narratives/internal/domain/list"
	pbpdom "narratives/internal/domain/productBlueprint"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

// ============================================================
// Ports (read-only) - shared
// ============================================================

type ListLister interface {
	List(ctx context.Context, filter listdom.Filter, sort listdom.Sort, page listdom.Page) (listdom.PageResult[listdom.List], error)
}

type ProductBlueprintGetter interface {
	GetByID(ctx context.Context, id string) (pbpdom.ProductBlueprint, error)
}

type TokenBlueprintGetter interface {
	GetByID(ctx context.Context, id string) (tbdom.TokenBlueprint, error)
}

// company boundary
// - currentCompany の inventory rows を列挙できることが必須
type InventoryRowsLister interface {
	ListByCurrentCompany(ctx context.Context) ([]querydto.InventoryManagementRowDTO, error)
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

type ListCreateSeedDTO struct {
	InventoryID        string           `json:"inventoryId"`
	ProductBlueprintID string           `json:"productBlueprintId"`
	TokenBlueprintID   string           `json:"tokenBlueprintId"`
	ProductName        string           `json:"productName"`
	TokenName          string           `json:"tokenName"`
	Prices             map[string]int64 `json:"prices"` // modelId -> price value
}

// ============================================================
// ListManagementQuery (listManagement.tsx / listCreate.tsx)
// ============================================================

type ListManagementQuery struct {
	lister       ListLister
	nameResolver *resolver.NameResolver

	pbGetter ProductBlueprintGetter
	tbGetter TokenBlueprintGetter

	// company boundary source
	invRows InventoryRowsLister
}

// ============================================================
// ✅ SINGLE ENTRYPOINT (唯一の出入り口)
//
// 要件:
// - companyId を使わない「単純な list」は禁止（＝invRows が必須）
// - この ctor のみを公開し、配線を集中させる
// ============================================================

type NewListManagementQueryParams struct {
	Lister       ListLister
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

func (q *ListManagementQuery) ListRows(ctx context.Context, filter listdom.Filter, sort listdom.Sort, page listdom.Page) (listdom.PageResult[ListRowDTO], error) {
	page = normalizePage(page)

	// company boundary を使わない単純 list は禁止
	if q == nil || q.lister == nil || q.invRows == nil {
		return listdom.PageResult[ListRowDTO]{}, errors.New("ListManagementQuery.ListRows: wiring is incomplete (lister/invRows required)")
	}

	allowedSet, err := allowedInventoryIDSetFromContext(ctx, q.invRows)
	if err != nil {
		log.Printf("[ListManagementQuery] ERROR company boundary (inventory_query) failed: %v", err)
		return listdom.PageResult[ListRowDTO]{}, err
	}
	if len(allowedSet) == 0 {
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

	const maxScanPages = 500
	srcPage := 1

	for {
		if srcPage > maxScanPages {
			log.Printf("[ListManagementQuery] WARN scan page limit reached (max=%d). results may be truncated.", maxScanPages)
			break
		}

		pr, e := q.lister.List(ctx, filter, sort, listdom.Page{Number: srcPage, PerPage: page.PerPage})
		if e != nil {
			log.Printf("[ListManagementQuery] ERROR lister.List failed (scan page=%d): %v", srcPage, e)
			return listdom.PageResult[ListRowDTO]{}, e
		}
		if pr.Items == nil {
			pr.Items = []listdom.List{}
		}

		for _, it := range pr.Items {
			id := strings.TrimSpace(it.ID)
			invID := strings.TrimSpace(it.InventoryID)

			if !inventoryAllowed(allowedSet, invID) {
				continue
			}

			assigneeID := strings.TrimSpace(it.AssigneeID)

			pbID, tbID, ok := parseInventoryIDStrict(invID)
			if !ok {
				continue
			}

			title := strings.TrimSpace(it.Title)

			productName := title
			if pbID != "" && q.nameResolver != nil {
				if cached, ok := productNameCache[pbID]; ok {
					if cached != "" {
						productName = cached
					}
				} else {
					resolved := strings.TrimSpace(q.nameResolver.ResolveProductName(ctx, pbID))
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
					resolved := strings.TrimSpace(q.nameResolver.ResolveTokenName(ctx, tbID))
					tokenNameCache[tbID] = resolved
					tokenName = resolved
				}
			}

			assigneeName := ""
			if assigneeID != "" && q.nameResolver != nil {
				if cached, ok := memberNameCache[assigneeID]; ok {
					assigneeName = cached
				} else {
					resolved := strings.TrimSpace(q.nameResolver.ResolveAssigneeName(ctx, assigneeID))
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
						productBrandID = strings.TrimSpace(pb.BrandID)
					}
					brandIDCachePB[pbID] = productBrandID
				}
			}
			if tbID != "" && q.tbGetter != nil {
				if cached, ok := brandIDCacheTB[tbID]; ok {
					tokenBrandID = cached
				} else {
					tb, ee := q.tbGetter.GetByID(ctx, tbID)
					if ee == nil {
						tokenBrandID = strings.TrimSpace(tb.BrandID)
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
						resolved := strings.TrimSpace(q.nameResolver.ResolveBrandName(ctx, productBrandID))
						brandNameByIDCache[productBrandID] = resolved
						productBrandName = resolved
					}
				}
				if tokenBrandID != "" {
					if cached, ok := brandNameByIDCache[tokenBrandID]; ok {
						tokenBrandName = cached
					} else {
						resolved := strings.TrimSpace(q.nameResolver.ResolveBrandName(ctx, tokenBrandID))
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
				ID:          nonEmpty(id, "(missing id)"),
				InventoryID: invID,
				Title:       title,

				ProductBlueprintID: pbID,
				TokenBlueprintID:   tbID,

				ProductName:      strings.TrimSpace(productName),
				ProductBrandID:   productBrandID,
				ProductBrandName: strings.TrimSpace(productBrandName),

				TokenName:      strings.TrimSpace(tokenName),
				TokenBrandID:   tokenBrandID,
				TokenBrandName: strings.TrimSpace(tokenBrandName),

				AssigneeID:   assigneeID,
				AssigneeName: assigneeName,

				Status: strings.TrimSpace(string(it.Status)),

				CreatedAt: createdAt,
			})
		}

		if len(pr.Items) == 0 {
			break
		}
		if pr.TotalPages > 0 {
			if srcPage >= pr.TotalPages {
				break
			}
		} else {
			if len(pr.Items) < page.PerPage {
				break
			}
		}

		srcPage++
	}

	totalCount := len(allowedAll)
	tp := totalPages(totalCount, page.PerPage)

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
	end := minInt(start+page.PerPage, totalCount)

	return listdom.PageResult[ListRowDTO]{
		Items:      allowedAll[start:end],
		Page:       page.Number,
		PerPage:    page.PerPage,
		TotalCount: totalCount,
		TotalPages: tp,
	}, nil
}

func (q *ListManagementQuery) BuildCreateSeed(ctx context.Context, inventoryID string, modelIDs []string) (ListCreateSeedDTO, error) {
	// company boundary を使わない単純 list は禁止
	if q == nil || q.invRows == nil {
		return ListCreateSeedDTO{}, errors.New("ListManagementQuery.BuildCreateSeed: wiring is incomplete (invRows required)")
	}

	allowedSet, err := allowedInventoryIDSetFromContext(ctx, q.invRows)
	if err != nil {
		log.Printf("[ListManagementQuery] ERROR company boundary (inventory_query) failed (seed): %v", err)
		return ListCreateSeedDTO{}, err
	}

	inventoryID = strings.TrimSpace(inventoryID)
	if !inventoryAllowed(allowedSet, inventoryID) {
		return ListCreateSeedDTO{}, listdom.ErrNotFound
	}

	pbID, tbID, ok := parseInventoryIDStrict(inventoryID)
	if !ok {
		log.Printf("[ListManagementQuery] BuildCreateSeed invalid inventoryID (expected {pbId}__{tbId}) inventoryID=%q", inventoryID)
		return ListCreateSeedDTO{}, listdom.ErrInvalidInventoryID
	}

	productName := ""
	tokenName := ""
	// ✅ FIX: ここでは q は必ず non-nil（上で guard 済み）なので q != nil チェックを外す
	if q.nameResolver != nil {
		productName = strings.TrimSpace(q.nameResolver.ResolveProductName(ctx, pbID))
		tokenName = strings.TrimSpace(q.nameResolver.ResolveTokenName(ctx, tbID))
	}

	prices := map[string]int64{}
	for _, mid := range modelIDs {
		mid = strings.TrimSpace(mid)
		if mid == "" {
			continue
		}
		prices[mid] = 0
	}

	return ListCreateSeedDTO{
		InventoryID:        inventoryID,
		ProductBlueprintID: pbID,
		TokenBlueprintID:   tbID,
		ProductName:        productName,
		TokenName:          tokenName,
		Prices:             prices,
	}, nil
}

// ============================================================
// local helpers (all lowercase)
// ============================================================

func allowedInventoryIDSetFromContext(ctx context.Context, invRows InventoryRowsLister) (map[string]struct{}, error) {
	if invRows == nil {
		return nil, errors.New("inventory rows lister is nil (company boundary via inventory_query is not configured)")
	}

	rows, err := invRows.ListByCurrentCompany(ctx)
	if err != nil {
		return nil, err
	}

	set := map[string]struct{}{}
	for _, r := range rows {
		pbID := strings.TrimSpace(r.ProductBlueprintID)
		tbID := strings.TrimSpace(r.TokenBlueprintID)
		if pbID == "" || tbID == "" {
			continue
		}
		invID := pbID + "__" + tbID
		set[invID] = struct{}{}
	}
	return set, nil
}

func inventoryAllowed(set map[string]struct{}, inventoryID string) bool {
	if len(set) == 0 {
		return false
	}
	id := strings.TrimSpace(inventoryID)
	if id == "" {
		return false
	}
	_, ok := set[id]
	return ok
}

func normalizePage(p listdom.Page) listdom.Page {
	if p.Number <= 0 {
		p.Number = 1
	}
	if p.PerPage <= 0 {
		p.PerPage = 20
	}
	return p
}

func totalPages(totalCount int, perPage int) int {
	if perPage <= 0 {
		return 0
	}
	if totalCount <= 0 {
		return 0
	}
	return (totalCount + perPage - 1) / perPage
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func nonEmpty(v string, fallback string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return fallback
	}
	return v
}

func parseInventoryIDStrict(invID string) (pbID string, tbID string, ok bool) {
	invID = strings.TrimSpace(invID)
	if invID == "" {
		return "", "", false
	}
	if !strings.Contains(invID, "__") {
		return "", "", false
	}
	parts := strings.Split(invID, "__")
	if len(parts) < 2 {
		return "", "", false
	}
	pb := strings.TrimSpace(parts[0])
	tb := strings.TrimSpace(parts[1])
	if pb == "" || tb == "" {
		return "", "", false
	}
	return pb, tb, true
}
