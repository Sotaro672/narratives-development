// backend/internal/application/query/console/list_management_query.go
package query

import (
	"context"
	"errors"
	"time"

	applicationport "narratives/internal/application/port"
	resolver "narratives/internal/application/resolver"
	listdom "narratives/internal/domain/list"
	memberdom "narratives/internal/domain/member"
)

// ============================================================
// Ports (read-only) - list management
// ============================================================

type ListManagementLister interface {
	ListByInventoryID(ctx context.Context, inventoryID string) ([]listdom.List, error)
}

// ============================================================
// DTO (query -> handler)
// ============================================================

type ListRowDTO struct {
	ID          string `json:"id"`
	ReadableID  string `json:"readableId"`
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
	memberRepo   memberdom.Repository

	pbGetter applicationport.ProductBlueprintGetter
	tbGetter applicationport.TokenBlueprintGetter

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
//   ListByInventoryID の順で該当 List のみ取得する
// - filter.InventoryIDs が指定された場合も current company 境界内だけに限定する
// - assigneeId は Firestore members の document ID を正とする
// - assigneeName は member.Repository.GetByID で解決する
// - この ctor のみを公開し、配線を集中させる
// ============================================================

type NewListManagementQueryParams struct {
	Lister       ListManagementLister
	NameResolver *resolver.NameResolver
	MemberRepo   memberdom.Repository

	PBGetter applicationport.ProductBlueprintGetter
	TBGetter applicationport.TokenBlueprintGetter

	InvRows InventoryRowsLister // REQUIRED
}

func NewListManagementQuery(p NewListManagementQueryParams) *ListManagementQuery {
	return &ListManagementQuery{
		lister:       p.Lister,
		nameResolver: p.NameResolver,
		memberRepo:   p.MemberRepo,
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

	if q == nil || q.lister == nil || q.invRows == nil {
		return listdom.PageResult[ListRowDTO]{}, errors.New("ListManagementQuery.ListRows: wiring is incomplete (lister/invRows required)")
	}

	allowedInventoryIDs, allowedSet, err := AllowedInventoryIDsFromContext(ctx, q.invRows)
	if err != nil {
		return listdom.PageResult[ListRowDTO]{}, err
	}

	if len(allowedInventoryIDs) == 0 {
		return emptyListManagementPage(page), nil
	}

	targetInventoryIDs := allowedInventoryIDs

	// inventoryId が指定された場合は、current company が保持する inventoryId の中から
	// 指定されたものだけを取得対象にする。
	if len(filter.InventoryIDs) > 0 {
		requestedInventoryIDs := make(map[string]struct{}, len(filter.InventoryIDs))
		for _, inventoryID := range filter.InventoryIDs {
			if inventoryID == "" {
				continue
			}

			requestedInventoryIDs[inventoryID] = struct{}{}
		}

		targetInventoryIDs = make([]string, 0, len(requestedInventoryIDs))
		for _, inventoryID := range allowedInventoryIDs {
			if _, ok := requestedInventoryIDs[inventoryID]; !ok {
				continue
			}

			targetInventoryIDs = append(targetInventoryIDs, inventoryID)
		}
	}

	if len(targetInventoryIDs) == 0 {
		return emptyListManagementPage(page), nil
	}

	tokenNameCache := map[string]string{}
	memberNameCache := map[string]string{}
	productNameCache := map[string]string{}
	brandIDCachePB := map[string]string{}
	brandIDCacheTB := map[string]string{}
	brandNameByIDCache := map[string]string{}

	allowedAll := make([]ListRowDTO, 0, page.PerPage)
	seenListID := map[string]struct{}{}

	for _, inventoryID := range targetInventoryIDs {
		if inventoryID == "" {
			continue
		}

		items, err := q.lister.ListByInventoryID(ctx, inventoryID)
		if err != nil {
			return listdom.PageResult[ListRowDTO]{}, err
		}

		for _, it := range items {
			if it.ID == "" {
				continue
			}

			if _, ok := seenListID[it.ID]; ok {
				continue
			}

			seenListID[it.ID] = struct{}{}

			id := it.ID
			invID := it.InventoryID

			// ListByInventoryID 後も List 実体の InventoryID が
			// current company 境界内か必ず再確認する。
			if !InventoryAllowed(allowedSet, invID) {
				continue
			}

			// filter.InventoryIDs 指定時は List 実体側も一致を再確認する。
			if len(filter.InventoryIDs) > 0 && invID != inventoryID {
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
			if assigneeID != "" && q.memberRepo != nil {
				if cached, ok := memberNameCache[assigneeID]; ok {
					assigneeName = cached
				} else {
					rec, err := q.memberRepo.GetByID(ctx, assigneeID)
					if err == nil {
						assigneeName = memberdom.FormatLastFirst(
							rec.Member.LastName,
							rec.Member.FirstName,
						)
					}

					memberNameCache[assigneeID] = assigneeName
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
					pb, err := q.pbGetter.GetByID(ctx, pbID)
					if err == nil {
						productBrandID = pb.BrandID
					}

					brandIDCachePB[pbID] = productBrandID
				}
			}

			if tbID != "" && q.tbGetter != nil {
				if cached, ok := brandIDCacheTB[tbID]; ok {
					tokenBrandID = cached
				} else {
					tb, err := q.tbGetter.GetByID(ctx, tbID)
					if err == nil && tb != nil {
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
				ReadableID:  it.ReadableID,
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

				Status:    string(it.Status),
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

func emptyListManagementPage(page listdom.Page) listdom.PageResult[ListRowDTO] {
	return listdom.PageResult[ListRowDTO]{
		Items:      []ListRowDTO{},
		Page:       page.Number,
		PerPage:    page.PerPage,
		TotalCount: 0,
		TotalPages: 0,
	}
}
