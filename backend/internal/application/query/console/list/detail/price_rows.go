// backend/internal/application/query/console/list/detail/price_rows.go
//
// 機能: ListDetailDTO の priceRows / stock / model attributes を生成する
// 責任:
// - listdom.List の価格行を抽出し、DTO(ListDetailPriceRowDTO)へ変換する
// - 在庫情報は InventoryDetailGetter から優先的に取得し、なければ stock=0 とする
// - displayOrder は productBlueprintPatch(ModelRefs) から付与する（取得できない場合は nil）
// - model 情報は resolver.ModelResolved を使って解決する
//   - apparel: kind / modelNumber / size / color / rgb
//   - alcohol: kind / modelNumber / volumeValue / volumeUnit
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
	rows := it.Prices
	if len(rows) == 0 {
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
					Kind:        r.Kind,
					ModelNumber: r.ModelNumber,

					Size:  r.Size,
					Color: r.Color,
					RGB:   r.RGB,

					VolumeValue: r.VolumeValue,
					VolumeUnit:  r.VolumeUnit,
				}
			}
		}
	}

	modelResolvedCache := map[string]resolver.ModelResolved{}

	out := make([]querydto.ListDetailPriceRowDTO, 0, len(rows))
	total := 0

	resolvedNonEmpty := 0
	resolvedEmpty := 0
	stockFromInv := 0
	stockFromList := 0
	displayOrderHit := 0

	for _, r := range rows {
		modelID := r.ModelID
		if modelID == "" {
			continue
		}

		price := r.Price
		pricePtr := &price

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
			// ListPriceRow の正規形は ModelID / Price のみ。
			// Stock は list ではなく inventory 側から解決する。
			stock = 0
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
		}

		mr, ok := attrByModel[modelID]
		if !ok {
			mr = q.resolveModelResolvedCached(ctx, modelID, modelResolvedCache)
		}

		applyModelResolvedToListDetailPriceRow(&dtoRow, modelID, mr)

		if dtoRow.ModelNumber != "" ||
			dtoRow.Size != "" ||
			dtoRow.Color != "" ||
			dtoRow.RGB != nil ||
			dtoRow.VolumeValue != nil ||
			dtoRow.VolumeUnit != "" {
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

func applyModelResolvedToListDetailPriceRow(
	row *querydto.ListDetailPriceRowDTO,
	modelID string,
	mr resolver.ModelResolved,
) {
	if row == nil {
		return
	}

	mn := mr.ModelNumber
	if mn == "" {
		mn = modelID
	}
	if mn == "" {
		mn = "-"
	}

	row.Kind = mr.Kind
	row.ModelNumber = mn

	if mr.Kind == "alcohol" {
		row.VolumeValue = mr.VolumeValue
		row.VolumeUnit = mr.VolumeUnit
		return
	}

	sz := mr.Size
	cl := mr.Color

	if sz == "" {
		sz = "-"
	}
	if cl == "" {
		cl = "-"
	}

	row.Size = sz
	row.Color = cl
	row.RGB = mr.RGB
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
