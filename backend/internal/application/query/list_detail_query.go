// backend/internal/application/query/list_detail_query.go
package query

import (
	"context"
	"errors"
	"log"
	"strings"

	querydto "narratives/internal/application/query/dto"
	resolver "narratives/internal/application/resolver"
	listdom "narratives/internal/domain/list"
)

// ============================================================
// Ports (read-only) - detail
// ============================================================

type ListGetter interface {
	GetByID(ctx context.Context, id string) (listdom.List, error)
}

type InventoryDetailGetter interface {
	GetDetailByID(ctx context.Context, inventoryID string) (*querydto.InventoryDetailDTO, error)
}

// ============================================================
// ListDetailQuery (listDetail.tsx)
// ============================================================

type ListDetailQuery struct {
	getter       ListGetter
	nameResolver *resolver.NameResolver

	pbGetter ProductBlueprintGetter
	tbGetter TokenBlueprintGetter

	invGetter InventoryDetailGetter
	invRows   InventoryRowsLister
}

func NewListDetailQuery(getter ListGetter, nameResolver *resolver.NameResolver) *ListDetailQuery {
	return NewListDetailQueryWithBrandAndInventoryGetters(getter, nameResolver, nil, nil, nil)
}

func NewListDetailQueryWithBrandGetters(
	getter ListGetter,
	nameResolver *resolver.NameResolver,
	pbGetter ProductBlueprintGetter,
	tbGetter TokenBlueprintGetter,
) *ListDetailQuery {
	return NewListDetailQueryWithBrandAndInventoryGetters(getter, nameResolver, pbGetter, tbGetter, nil)
}

func NewListDetailQueryWithBrandAndInventoryGetters(
	getter ListGetter,
	nameResolver *resolver.NameResolver,
	pbGetter ProductBlueprintGetter,
	tbGetter TokenBlueprintGetter,
	invGetter InventoryDetailGetter,
) *ListDetailQuery {
	return &ListDetailQuery{
		getter:       getter,
		nameResolver: nameResolver,
		pbGetter:     pbGetter,
		tbGetter:     tbGetter,
		invGetter:    invGetter,
		invRows:      nil,
	}
}

func NewListDetailQueryWithBrandInventoryAndInventoryRows(
	getter ListGetter,
	nameResolver *resolver.NameResolver,
	pbGetter ProductBlueprintGetter,
	tbGetter TokenBlueprintGetter,
	invGetter InventoryDetailGetter,
	invRows InventoryRowsLister,
) *ListDetailQuery {
	q := NewListDetailQueryWithBrandAndInventoryGetters(getter, nameResolver, pbGetter, tbGetter, invGetter)
	q.invRows = invRows
	return q
}

func (q *ListDetailQuery) BuildListDetailDTO(ctx context.Context, listID string) (querydto.ListDetailDTO, error) {
	if q == nil || q.getter == nil {
		return querydto.ListDetailDTO{}, errors.New("ListDetailQuery.BuildListDetailDTO: getter is nil (wire list repo to ListDetailQuery)")
	}

	allowedSet, err := allowedInventoryIDSetFromContext(ctx, q.invRows)
	if err != nil {
		log.Printf("[ListDetailQuery] ERROR company boundary (inventory_query) failed (detail): %v", err)
		return querydto.ListDetailDTO{}, err
	}

	listID = strings.TrimSpace(listID)
	if listID == "" {
		return querydto.ListDetailDTO{}, errors.New("ListDetailQuery.BuildListDetailDTO: listID is empty")
	}

	it, err := q.getter.GetByID(ctx, listID)
	if err != nil {
		return querydto.ListDetailDTO{}, err
	}

	invID := strings.TrimSpace(it.InventoryID)
	if !inventoryAllowed(allowedSet, invID) {
		return querydto.ListDetailDTO{}, listdom.ErrNotFound
	}

	pbID, tbID, ok := parseInventoryIDStrict(invID)
	if !ok {
		return querydto.ListDetailDTO{}, listdom.ErrNotFound
	}

	// ---- names ----
	productName := ""
	tokenName := ""
	assigneeName := ""

	if q.nameResolver != nil {
		if pbID != "" {
			productName = strings.TrimSpace(q.nameResolver.ResolveProductName(ctx, pbID))
		}
		if tbID != "" {
			tokenName = strings.TrimSpace(q.nameResolver.ResolveTokenName(ctx, tbID))
		}
		if strings.TrimSpace(it.AssigneeID) != "" {
			assigneeName = strings.TrimSpace(q.nameResolver.ResolveAssigneeName(ctx, it.AssigneeID))
		}
	}
	if assigneeName == "" && strings.TrimSpace(it.AssigneeID) != "" {
		assigneeName = "未設定"
	}

	// ---- brand ----
	productBrandID := ""
	tokenBrandID := ""
	if pbID != "" && q.pbGetter != nil {
		pb, e := q.pbGetter.GetByID(ctx, pbID)
		if e == nil {
			productBrandID = strings.TrimSpace(pb.BrandID)
		}
	}
	if tbID != "" && q.tbGetter != nil {
		tb, e := q.tbGetter.GetByID(ctx, tbID)
		if e == nil {
			tokenBrandID = strings.TrimSpace(tb.BrandID)
		}
	}

	productBrandName := ""
	tokenBrandName := ""
	if q.nameResolver != nil {
		if productBrandID != "" {
			productBrandName = strings.TrimSpace(q.nameResolver.ResolveBrandName(ctx, productBrandID))
		}
		if tokenBrandID != "" {
			tokenBrandName = strings.TrimSpace(q.nameResolver.ResolveBrandName(ctx, tokenBrandID))
		}
	}

	// ---- priceRows + stock(size/color/rgb) ----
	priceRows, totalStock, metaLog := q.buildDetailPriceRows(ctx, it, invID)
	if metaLog != "" {
		log.Printf("[ListDetailQuery] [modelMetadata] listID=%q %s", strings.TrimSpace(it.ID), metaLog)
	}

	dto := querydto.ListDetailDTO{
		ID:          strings.TrimSpace(it.ID),
		InventoryID: invID,

		Status:   strings.TrimSpace(string(it.Status)),
		Decision: strings.TrimSpace(string(it.Status)),

		Title:       strings.TrimSpace(it.Title),
		Description: strings.TrimSpace(it.Description),

		AssigneeID:   strings.TrimSpace(it.AssigneeID),
		AssigneeName: strings.TrimSpace(assigneeName),

		CreatedBy: strings.TrimSpace(it.CreatedBy),
		CreatedAt: it.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: it.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),

		ImageID: strings.TrimSpace(it.ImageID),

		ProductBlueprintID: pbID,
		TokenBlueprintID:   tbID,

		ProductBrandID:   productBrandID,
		ProductBrandName: productBrandName,
		ProductName:      productName,

		TokenBrandID:   tokenBrandID,
		TokenBrandName: tokenBrandName,
		TokenName:      tokenName,

		ImageURLs: []string{},
		PriceRows: priceRows,

		TotalStock:  totalStock,
		CurrencyJPY: true,
	}

	return dto, nil
}

func (q *ListDetailQuery) buildDetailPriceRows(ctx context.Context, it listdom.List, inventoryID string) ([]querydto.ListDetailPriceRowDTO, int, string) {
	rowsAny := extractPriceRowsFromList(it)
	if len(rowsAny) == 0 {
		return []querydto.ListDetailPriceRowDTO{}, 0, "rows=0"
	}

	stockByModel := map[string]int{}
	attrByModel := map[string]resolver.ModelResolved{}
	invUsed := false
	invErrMsg := ""

	invID := strings.TrimSpace(inventoryID)
	if invID != "" && q != nil && q.invGetter != nil {
		if invDTO, err := q.invGetter.GetDetailByID(ctx, invID); err != nil {
			invErrMsg = "invErr=" + strings.TrimSpace(err.Error())
		} else if invDTO != nil {
			invUsed = true
			for _, r := range invDTO.Rows {
				mid := strings.TrimSpace(r.ModelID)
				if mid == "" {
					continue
				}
				stockByModel[mid] = r.Stock
				attrByModel[mid] = resolver.ModelResolved{
					Size:  strings.TrimSpace(r.Size),
					Color: strings.TrimSpace(r.Color),
					RGB:   r.RGB,
				}
			}
		}
	}

	modelResolvedCache := map[string]resolver.ModelResolved{}

	out := make([]querydto.ListDetailPriceRowDTO, 0, len(rowsAny))
	total := 0

	resolvedNonEmpty := 0
	resolvedEmpty := 0
	stockFromInv := 0
	stockFromList := 0

	for _, r := range rowsAny {
		modelID := strings.TrimSpace(readStringField(r, "ModelID", "ModelId", "ID", "Id"))
		if modelID == "" {
			continue
		}

		pricePtr := readIntPtrField(r, "Price", "price")

		stock := 0
		if invUsed {
			if v, ok := stockByModel[modelID]; ok {
				stock = v
				stockFromInv++
			} else {
				stock = 0
				stockFromInv++
			}
		} else {
			stock = readIntField(r, "Stock", "stock")
			stockFromList++
		}

		dtoRow := querydto.ListDetailPriceRowDTO{
			ModelID: modelID,
			Stock:   stock,
			Price:   pricePtr,
		}

		if mr, ok := attrByModel[modelID]; ok {
			dtoRow.Size = strings.TrimSpace(mr.Size)
			dtoRow.Color = strings.TrimSpace(mr.Color)
			dtoRow.RGB = mr.RGB
		} else {
			mr := q.resolveModelResolvedCached(ctx, modelID, modelResolvedCache)
			dtoRow.Size = strings.TrimSpace(mr.Size)
			dtoRow.Color = strings.TrimSpace(mr.Color)
			dtoRow.RGB = mr.RGB
		}

		if dtoRow.Size != "" || dtoRow.Color != "" || dtoRow.RGB != nil {
			resolvedNonEmpty++
		} else {
			resolvedEmpty++
		}

		out = append(out, dtoRow)
		total += stock
	}

	meta := "rows=" + itoa(len(out)) +
		" resolvedNonEmpty=" + itoa(resolvedNonEmpty) +
		" resolvedEmpty=" + itoa(resolvedEmpty) +
		" invUsed=" + bool01(invUsed) +
		" stockFromInv=" + itoa(stockFromInv) +
		" stockFromList=" + itoa(stockFromList)
	if invErrMsg != "" {
		meta += " " + invErrMsg
	}

	return out, total, meta
}

func (q *ListDetailQuery) resolveModelResolvedCached(
	ctx context.Context,
	variationID string,
	cache map[string]resolver.ModelResolved,
) resolver.ModelResolved {
	id := strings.TrimSpace(variationID)
	if id == "" {
		return resolver.ModelResolved{}
	}
	if cache != nil {
		if v, ok := cache[id]; ok {
			return v
		}
	}

	var v resolver.ModelResolved
	if q != nil && q.nameResolver != nil {
		v = q.nameResolver.ResolveModelResolved(ctx, id)
	}

	if cache != nil {
		cache[id] = v
	}
	return v
}
