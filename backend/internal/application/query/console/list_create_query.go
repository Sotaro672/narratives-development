// backend/internal/application/query/console/list_create_query.go
package query

import (
	"context"
	"errors"
	"log"

	querydto "narratives/internal/application/query/console/dto"
	resolver "narratives/internal/application/resolver"
	invdom "narratives/internal/domain/inventory"
)

// ============================================================
// ListCreateQuery
// - listCreate 画面に必要な最小情報を組み立てる（1出品 = 1 inventory）
//
// 方針:
// - PriceRows の母集団を「productBlueprintPatch.ModelRefs」に統一する。
// - displayOrder は「取得して返すのみ」。
// - 並べ替え（displayOrder 昇順 / size,color 等）は一切しない。
// - inventoryId の build/split は廃止（inventoryId は inventory テーブルから拾う）
// - PriceRows には productBlueprintCategory / model kind に応じた model 情報を含める。
//   - apparel: modelNumber / size / color / rgb
//   - alcohol: modelNumber / volumeValue / volumeUnit
// ============================================================

type ListCreateQuery struct {
	// inventory から stock / inventoryId(pb/tb取得含む) を引くため
	// ※ GetByInventoryID を使うなら必須
	invRepo inventoryReader // defined in inventory_query.go

	pbPatchRepo  productBlueprintPatchReader // defined in inventory_query.go
	tbPatchRepo  tokenBlueprintPatchReader   // defined in inventory_query.go
	nameResolver *resolver.NameResolver
}

// GetByInventoryID を使うなら invRepo が必要になる
func NewListCreateQueryWithInventory(
	invRepo inventoryReader,
	pbPatchRepo productBlueprintPatchReader,
	tbPatchRepo tokenBlueprintPatchReader,
	nameResolver *resolver.NameResolver,
) *ListCreateQuery {
	return &ListCreateQuery{
		invRepo:      invRepo,
		pbPatchRepo:  pbPatchRepo,
		tbPatchRepo:  tbPatchRepo,
		nameResolver: nameResolver,
	}
}

// ============================================================
// inventoryId -> ListCreateDTO
// - inventoryId を split しない
// - pbId/tbId は inventory テーブルから拾うのみ
// ============================================================

func (q *ListCreateQuery) GetByInventoryID(ctx context.Context, inventoryID string) (*querydto.ListCreateDTO, error) {
	if q == nil {
		return nil, errors.New("list create query is nil")
	}
	if q.invRepo == nil {
		return nil, errors.New("list create query: invRepo is not configured (GetByInventoryID requires inventory repository)")
	}

	id := inventoryID
	if id == "" {
		return nil, errors.New("inventoryId is required")
	}

	// inventory テーブルから pbId/tbId を拾う（split禁止）
	inv, err := q.invRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	pbID := inv.ProductBlueprintID
	tbID := inv.TokenBlueprintID
	if pbID == "" || tbID == "" {
		return nil, errors.New("productBlueprintId/tokenBlueprintId is empty in inventory")
	}

	dto, err := q.buildByIDs(ctx, pbID, tbID)
	if err != nil {
		return nil, err
	}

	// inventoryId は「組み立てず」取得した docId をそのまま返す
	dto.InventoryID = id
	return dto, nil
}

// ============================================================
// internal: pbId/tbId -> ListCreateDTO
// - inventoryId は build しない
// - invRepo があれば「inventory テーブルから拾った ID」を返す
// ============================================================

func (q *ListCreateQuery) buildByIDs(
	ctx context.Context,
	productBlueprintID string,
	tokenBlueprintID string,
) (*querydto.ListCreateDTO, error) {
	if q == nil {
		return nil, errors.New("list create query is nil")
	}

	pbID := productBlueprintID
	tbID := tokenBlueprintID
	if pbID == "" || tbID == "" {
		return nil, errors.New("productBlueprintId and tokenBlueprintId are required")
	}

	// ------------------------------------------------------------
	// ProductBlueprint: productName / brandName
	// ------------------------------------------------------------
	productName := ""
	productBrandName := ""

	if q.nameResolver != nil {
		productName = q.nameResolver.ResolveProductName(ctx, pbID)
	}

	if q.pbPatchRepo != nil {
		if patch, err := q.pbPatchRepo.GetPatchByID(ctx, pbID); err == nil {
			brandID := getStringFieldAny(patch, "BrandID", "BrandId", "brandId")
			if brandID != "" && q.nameResolver != nil {
				productBrandName = q.nameResolver.ResolveBrandName(ctx, brandID)
			}
			if productBrandName == "" {
				productBrandName = getStringFieldAny(patch, "BrandName", "brandName")
			}
		}
	}

	// ------------------------------------------------------------
	// TokenBlueprint: tokenName / brandName
	// ------------------------------------------------------------
	tokenName := ""
	tokenBrandName := ""

	if q.nameResolver != nil {
		tokenName = q.nameResolver.ResolveTokenName(ctx, tbID)
	}

	if q.tbPatchRepo != nil {
		if patch, err := q.tbPatchRepo.GetPatchByID(ctx, tbID); err == nil {
			brandID := getStringFieldAny(patch, "BrandID", "BrandId", "brandId")
			if brandID != "" && q.nameResolver != nil {
				tokenBrandName = q.nameResolver.ResolveBrandName(ctx, brandID)
			}
			if tokenBrandName == "" {
				tokenBrandName = getStringFieldAny(patch, "BrandName", "brandName")
			}
		}
	}

	// ------------------------------------------------------------
	// ModelRefs / PriceRows（並べ替えしない）
	// ------------------------------------------------------------
	modelRefs := q.listModelRefs(ctx, pbID)
	priceRows, totalStock, invID := q.buildPriceRowsByIDs(ctx, pbID, tbID, modelRefs)

	dto := &querydto.ListCreateDTO{
		// inventoryId は build しない。拾えた場合のみ入る。
		InventoryID:        invID,
		ProductBlueprintID: pbID,
		TokenBlueprintID:   tbID,

		ProductBrandName: productBrandName,
		ProductName:      productName,

		TokenBrandName: tokenBrandName,
		TokenName:      tokenName,

		ModelRefs:  modelRefs,
		PriceRows:  priceRows,
		TotalStock: totalStock,
	}

	return dto, nil
}

// ============================================================
// internal: build PriceRows
// - 母集団: productBlueprintPatch.ModelRefs（順序はそのまま）
// - stock: inventory が取れれば picked.Stock[modelId] を反映、無ければ 0
// - stock==0 でも行を出す（価格入力のため）
// - 並べ替えはしない
// - model 情報は resolver.ModelResolved を正として詰める
//   - apparel: kind / modelNumber / size / color / rgb
//   - alcohol: kind / modelNumber / volumeValue / volumeUnit
// ============================================================

func (q *ListCreateQuery) buildPriceRowsByIDs(
	ctx context.Context,
	productBlueprintID string,
	tokenBlueprintID string,
	modelRefs []querydto.ListCreateModelRefDTO,
) ([]querydto.ListCreatePriceRowDTO, int, string) {
	if q == nil {
		return nil, 0, ""
	}

	pbID := productBlueprintID
	tbID := tokenBlueprintID
	if pbID == "" || tbID == "" {
		return nil, 0, ""
	}
	if len(modelRefs) == 0 {
		return nil, 0, ""
	}

	// inventoryId を build しない。invRepo から「該当Mint」を拾い、その ID を使う。
	var picked *invdom.Mint
	if q.invRepo != nil {
		invs, err := q.invRepo.ListByProductBlueprintID(ctx, pbID)
		if err == nil && len(invs) > 0 {
			for i := range invs {
				if invs[i].TokenBlueprintID == tbID {
					picked = &invs[i]
					break
				}
			}
		}
	}

	rows := make([]querydto.ListCreatePriceRowDTO, 0, len(modelRefs))
	total := 0

	for _, ref := range modelRefs {
		mid := ref.ModelID
		if mid == "" {
			continue
		}

		stock := 0
		if picked != nil && picked.Stock != nil {
			if ms, ok := picked.Stock[mid]; ok {
				// domain contract（ModelStock.Validate）前提の素直な計算
				available := ms.Accumulation - ms.ReservedCount
				if available < 0 {
					// 契約上は起きない想定だが、画面を壊さない保険
					log.Printf("[list_create_query][stock] WARN availableStock negative accumulation=%d reserved=%d -> clamp to 0", ms.Accumulation, ms.ReservedCount)
					available = 0
				}
				stock = available
			}
		}

		attr := resolver.ModelResolved{}
		if q.nameResolver != nil {
			attr = q.nameResolver.ResolveModelResolved(ctx, mid)
		}

		mn := attr.ModelNumber
		if mn == "" {
			mn = mid
		}
		if mn == "" {
			mn = "-"
		}

		row := querydto.ListCreatePriceRowDTO{
			ModelID:      mid,
			Kind:         attr.Kind,
			ModelNumber:  mn,
			DisplayOrder: ref.DisplayOrder,
			Stock:        stock,
			Price:        nil,
		}

		if attr.Kind == "alcohol" {
			row.VolumeValue = attr.VolumeValue
			row.VolumeUnit = attr.VolumeUnit

			if row.VolumeValue == nil || row.VolumeUnit == "" {
				log.Printf(
					"[list_create_query][modelResolved] alcohol model missing volume pbId=%q tbId=%q modelId=%q kind=%q modelNumber=%q volumeValue=%v volumeUnit=%q stock=%d",
					pbID, tbID, mid, attr.Kind, mn, row.VolumeValue, row.VolumeUnit, stock,
				)
			}
		} else {
			sz := attr.Size
			cl := attr.Color

			if sz == "" {
				sz = "-"
			}
			if cl == "" {
				cl = "-"
			}

			row.Size = sz
			row.Color = cl
			row.RGB = attr.RGB
		}

		rows = append(rows, row)
		total += stock
	}

	invID := ""
	if picked != nil {
		invID = picked.ID
	}

	return rows, total, invID
}

func toDisplayOrderPtr(v int) *int {
	if v == 0 {
		return nil
	}
	x := v
	return &x
}

// 母集団: productBlueprintPatch.ModelRefs（順序は patch のまま）
func (q *ListCreateQuery) listModelRefs(ctx context.Context, productBlueprintID string) []querydto.ListCreateModelRefDTO {
	if q == nil || q.pbPatchRepo == nil {
		return nil
	}
	pbID := productBlueprintID
	if pbID == "" {
		return nil
	}

	patch, err := q.pbPatchRepo.GetPatchByID(ctx, pbID)
	if err != nil {
		return nil
	}
	if patch.ModelRefs == nil || len(*patch.ModelRefs) == 0 {
		return nil
	}

	refs := *patch.ModelRefs
	seen := map[string]struct{}{}
	out := make([]querydto.ListCreateModelRefDTO, 0, len(refs))

	// 並べ替えしない：入力順のまま
	for _, r := range refs {
		mid := r.ModelID
		if mid == "" {
			continue
		}
		if _, ok := seen[mid]; ok {
			continue
		}
		seen[mid] = struct{}{}

		out = append(out, querydto.ListCreateModelRefDTO{
			ModelID:      mid,
			DisplayOrder: toDisplayOrderPtr(r.DisplayOrder),
		})
	}
	return out
}
