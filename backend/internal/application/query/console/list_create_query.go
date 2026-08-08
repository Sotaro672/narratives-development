// backend/internal/application/query/console/list_create_query.go
package query

import (
	"context"
	"errors"

	querydto "narratives/internal/application/query/console/dto"
	resolver "narratives/internal/application/resolver"
	invdom "narratives/internal/domain/inventory"
)

// ============================================================
// ListCreateQuery
// - listCreate 画面に必要な最小情報を組み立てる（1出品 = 1 inventory）
//
// 方針:
// - PriceRows の母集団を「productBlueprint.ModelRefs」に統一する。
// - productBlueprint は GetByID で取得する。
// - tokenBlueprint は GetByID で取得する。
// - displayOrder は「取得して返すのみ」。
// - 並べ替え（displayOrder 昇順 / size,color 等）は一切しない。
// - inventoryId の build/split は廃止（inventoryId は inventory テーブルから拾う）
// - PriceRows には productBlueprintCategory / model kind に応じた model 情報を含める。
//   - apparel: size / color / rgb
//   - alcohol: volumeValue / volumeUnit
// ============================================================

type ListCreateQuery struct {
	// inventory から stock / pb/tb を引くため
	// ※ GetByInventoryID を使うなら必須
	invRepo inventoryReader // defined in inventory_query.go

	pbRepo       inventoryProductBlueprintReader // defined in inventory_query.go
	tbRepo       inventoryTokenBlueprintReader   // defined in inventory_query.go
	nameResolver *resolver.NameResolver
}

// GetByInventoryID を使うなら invRepo が必要になる
func NewListCreateQueryWithInventory(
	invRepo inventoryReader,
	pbRepo inventoryProductBlueprintReader,
	tbRepo inventoryTokenBlueprintReader,
	nameResolver *resolver.NameResolver,
) *ListCreateQuery {
	return &ListCreateQuery{
		invRepo:      invRepo,
		pbRepo:       pbRepo,
		tbRepo:       tbRepo,
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

	return q.buildByIDs(ctx, pbID, tbID)
}

// ============================================================
// internal: pbId/tbId -> ListCreateDTO
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

	if q.pbRepo != nil {
		if pb, err := q.pbRepo.GetByID(ctx, pbID); err == nil {
			if productName == "" {
				productName = pb.ProductName
			}

			brandID := pb.BrandID
			if brandID != "" && q.nameResolver != nil {
				productBrandName = q.nameResolver.ResolveBrandName(ctx, brandID)
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

	if q.tbRepo != nil {
		if tb, err := q.tbRepo.GetByID(ctx, tbID); err == nil && tb != nil {
			if tokenName == "" {
				tokenName = tb.Name
			}

			brandID := tb.BrandID
			if brandID != "" && q.nameResolver != nil {
				tokenBrandName = q.nameResolver.ResolveBrandName(ctx, brandID)
			}
		}
	}

	// ------------------------------------------------------------
	// PriceRows（並べ替えしない）
	// ------------------------------------------------------------
	modelRefs := q.listModelRefs(ctx, pbID)
	priceRows := q.buildPriceRowsByIDs(ctx, pbID, tbID, modelRefs)

	dto := &querydto.ListCreateDTO{
		ProductBrandName: productBrandName,
		ProductName:      productName,

		TokenBrandName: tokenBrandName,
		TokenName:      tokenName,

		PriceRows: priceRows,
	}

	return dto, nil
}

// ============================================================
// internal: build PriceRows
// - 母集団: productBlueprint.ModelRefs（順序はそのまま）
// - stock: inventory が取れれば picked.Stock[modelId] を反映、無ければ 0
// - stock==0 でも行を出す（価格入力のため）
// - 並べ替えはしない
// - model 情報は resolver.ModelResolved を正として詰める
//   - apparel: kind / size / color / rgb
//   - alcohol: kind / volumeValue / volumeUnit
// ============================================================

func (q *ListCreateQuery) buildPriceRowsByIDs(
	ctx context.Context,
	productBlueprintID string,
	tokenBlueprintID string,
	modelRefs []querydto.ListCreateModelRefDTO,
) []querydto.ListCreatePriceRowDTO {
	if q == nil {
		return nil
	}

	pbID := productBlueprintID
	tbID := tokenBlueprintID
	if pbID == "" || tbID == "" {
		return nil
	}
	if len(modelRefs) == 0 {
		return nil
	}

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
					available = 0
				}
				stock = available
			}
		}

		attr := resolver.ModelResolved{}
		if q.nameResolver != nil {
			attr = q.nameResolver.ResolveModelResolved(ctx, mid)
		}

		row := querydto.ListCreatePriceRowDTO{
			ModelID:      mid,
			Kind:         attr.Kind,
			DisplayOrder: ref.DisplayOrder,
			Stock:        stock,
			Price:        nil,
		}

		if attr.Kind == "alcohol" {
			row.VolumeValue = attr.VolumeValue
			row.VolumeUnit = attr.VolumeUnit
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
	}

	return rows
}

func toDisplayOrderPtr(v int) *int {
	if v == 0 {
		return nil
	}
	x := v
	return &x
}

// 母集団: productBlueprint.ModelRefs（順序は productBlueprint のまま）
func (q *ListCreateQuery) listModelRefs(ctx context.Context, productBlueprintID string) []querydto.ListCreateModelRefDTO {
	if q == nil || q.pbRepo == nil {
		return nil
	}

	pbID := productBlueprintID
	if pbID == "" {
		return nil
	}

	pb, err := q.pbRepo.GetByID(ctx, pbID)
	if err != nil {
		return nil
	}
	if len(pb.ModelRefs) == 0 {
		return nil
	}

	refs := pb.ModelRefs
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
