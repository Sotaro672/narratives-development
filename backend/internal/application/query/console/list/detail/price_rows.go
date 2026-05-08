// backend/internal/application/query/console/list/detail/price_rows.go
//
// 機能: ListDetailDTO の priceRows / stock / model attributes を生成する
// 責任:
// - listdom.List の価格行を抽出し、DTO(ListDetailPriceRowDTO)へ変換する
// - 在庫情報は InventoryDetailGetter から優先的に取得し、なければ List の値を利用する
// - displayOrder は productBlueprintPatch(ModelRefs) から付与する（取得できない場合は nil）
package detail

import (
	"context"

	querydto "narratives/internal/application/query/console/dto"
	listq "narratives/internal/application/query/console/list"
	resolver "narratives/internal/application/resolver"
	listdom "narratives/internal/domain/list"
)

func (q *ListDetailQuery) buildDetailPriceRows(
	ctx context.Context,
	it listdom.List,
	inventoryID string,
	productBlueprintID string,
) ([]querydto.ListDetailPriceRowDTO, int, string) {
	rowsAny := listq.ExtractPriceRowsFromList(it)
	if len(rowsAny) == 0 {
		return []querydto.ListDetailPriceRowDTO{}, 0, "rows=0"
	}

	displayOrderByModel := q.buildDisplayOrderByModelID(ctx, productBlueprintID)

	stockByModel := map[string]int{}
	attrByModel := map[string]resolver.ModelResolved{}
	invUsed := false
	invErrMsg := ""

	invID := inventoryID
	if invID != "" && q != nil && q.invGetter != nil {
		if invDTO, err := q.invGetter.GetDetailByID(ctx, invID); err != nil {
			invErrMsg = "invErr=" + err.Error()
		} else if invDTO != nil {
			invUsed = true
			for _, r := range invDTO.Rows {
				mid := r.ModelID
				if mid == "" {
					continue
				}
				stockByModel[mid] = r.Stock
				attrByModel[mid] = resolver.ModelResolved{
					Size:  r.Size,
					Color: r.Color,
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
	displayOrderHit := 0

	for _, r := range rowsAny {
		modelID := listq.ReadStringField(r, "ModelID", "ModelId", "ID", "Id")
		if modelID == "" {
			continue
		}

		pricePtr := listq.ReadIntPtrField(r, "Price", "price")

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
			stock = listq.ReadIntField(r, "Stock", "stock")
			stockFromList++
		}

		var dispPtr *int
		if displayOrderByModel != nil {
			if v, ok := displayOrderByModel[modelID]; ok {
				dispPtr = v
				if v != nil {
					displayOrderHit++
				}
			}
		}

		dtoRow := querydto.ListDetailPriceRowDTO{
			ModelID:      modelID,
			DisplayOrder: dispPtr,
			Stock:        stock,
			Price:        pricePtr,
			Size:         "",
			Color:        "",
			RGB:          nil,
		}

		if mr, ok := attrByModel[modelID]; ok {
			dtoRow.Size = mr.Size
			dtoRow.Color = mr.Color
			dtoRow.RGB = mr.RGB
		} else {
			mr := q.resolveModelResolvedCached(ctx, modelID, modelResolvedCache)
			dtoRow.Size = mr.Size
			dtoRow.Color = mr.Color
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

	meta := "rows=" + listq.Itoa(len(out)) +
		" resolvedNonEmpty=" + listq.Itoa(resolvedNonEmpty) +
		" resolvedEmpty=" + listq.Itoa(resolvedEmpty) +
		" invUsed=" + listq.Bool01(invUsed) +
		" stockFromInv=" + listq.Itoa(stockFromInv) +
		" stockFromList=" + listq.Itoa(stockFromList) +
		" displayOrderHit=" + listq.Itoa(displayOrderHit)
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
	id := variationID
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
