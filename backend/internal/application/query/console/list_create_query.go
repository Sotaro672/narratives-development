// backend/internal/application/query/console/list_create_query.go
package query

import (
	"context"
	"errors"
	"strings"

	querydto "narratives/internal/application/query/console/dto"
	resolver "narratives/internal/application/resolver"
	invdom "narratives/internal/domain/inventory"
)

// ============================================================
// ListCreateQuery
// - listCreate 画面に必要な最小情報を組み立てる（1出品 = 1 inventory）
//
// ✅ 方針:
// - PriceRows の母集団を「productBlueprintPatch.ModelRefs」に統一する。
// - displayOrder は「取得して返すのみ」。
// - ✅ 並べ替え（displayOrder 昇順 / size,color 等）は一切しない。
//   -> ModelRefs の順序は patch の順序をそのまま返す。
// ============================================================

type ListCreateQuery struct {
	// inventory から stock を引くため（任意）
	invRepo inventoryReader // defined in inventory_query.go

	pbPatchRepo  productBlueprintPatchReader // defined in inventory_query.go
	tbPatchRepo  tokenBlueprintPatchReader   // defined in inventory_query.go
	nameResolver *resolver.NameResolver
}

// 互換: 既存 DI を壊さない（invRepo なしでも DTO の基本情報は返す）
func NewListCreateQuery(
	pbPatchRepo productBlueprintPatchReader,
	tbPatchRepo tokenBlueprintPatchReader,
	nameResolver *resolver.NameResolver,
) *ListCreateQuery {
	return &ListCreateQuery{
		invRepo:      nil, // optional
		pbPatchRepo:  pbPatchRepo,
		tbPatchRepo:  tbPatchRepo,
		nameResolver: nameResolver,
	}
}

// 互換: inventory reader も注入できるコンストラクタ（TotalStock を埋める）
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
// inventoryId から ListCreateDTO を組み立てる（互換: listImage 入力無し）
// ============================================================

func (q *ListCreateQuery) GetByInventoryID(
	ctx context.Context,
	inventoryID string,
) (*querydto.ListCreateDTO, error) {
	return q.GetByInventoryIDWithListImage(ctx, inventoryID, "")
}

// inventoryId から ListCreateDTO を組み立てる（listImage 入力あり）
func (q *ListCreateQuery) GetByInventoryIDWithListImage(
	ctx context.Context,
	inventoryID string,
	listImageURL string,
) (*querydto.ListCreateDTO, error) {
	if q == nil {
		return nil, errors.New("list create query is nil")
	}

	invID := strings.TrimSpace(inventoryID)
	if invID == "" {
		return nil, errors.New("inventoryId is required")
	}

	// inventoryId = "{pbId}__{tbId}" を正として parse（※この仕様は維持）
	pbID, tbID, ok := parseInventoryID(invID)
	if !ok || pbID == "" || tbID == "" {
		return nil, errors.New("invalid inventoryId format (expected {pbId}__{tbId})")
	}

	// ------------------------------------------------------------
	// ProductBlueprint: productName / brandName
	// ------------------------------------------------------------
	productName := ""
	productBrandName := ""

	if q.nameResolver != nil {
		productName = strings.TrimSpace(q.nameResolver.ResolveProductName(ctx, pbID))
	}

	if q.pbPatchRepo != nil {
		if patch, err := q.pbPatchRepo.GetPatchByID(ctx, pbID); err == nil {
			brandID := strings.TrimSpace(getStringFieldAny(patch, "BrandID", "BrandId", "brandId"))
			if brandID != "" && q.nameResolver != nil {
				productBrandName = strings.TrimSpace(q.nameResolver.ResolveBrandName(ctx, brandID))
			}
			if productBrandName == "" {
				productBrandName = strings.TrimSpace(getStringFieldAny(patch, "BrandName", "brandName"))
			}
		}
	}

	// ------------------------------------------------------------
	// TokenBlueprint: tokenName / brandName
	// ------------------------------------------------------------
	tokenName := ""
	tokenBrandName := ""

	if q.nameResolver != nil {
		tokenName = strings.TrimSpace(q.nameResolver.ResolveTokenName(ctx, tbID))
	}

	if q.tbPatchRepo != nil {
		if patch, err := q.tbPatchRepo.GetPatchByID(ctx, tbID); err == nil {
			brandID := strings.TrimSpace(getStringFieldAny(patch, "BrandID", "BrandId", "brandId"))
			if brandID != "" && q.nameResolver != nil {
				tokenBrandName = strings.TrimSpace(q.nameResolver.ResolveBrandName(ctx, brandID))
			}
			if tokenBrandName == "" {
				tokenBrandName = strings.TrimSpace(getStringFieldAny(patch, "BrandName", "brandName"))
			}
		}
	}

	// ------------------------------------------------------------
	// ✅ ModelRefs / PriceRows（displayOrder は取得して渡すのみ。並べ替えしない）
	// ------------------------------------------------------------
	modelRefs := q.listModelRefs(ctx, pbID)
	priceRows, totalStock := q.buildPriceRowsByInventoryID(ctx, invID, modelRefs)

	dto := &querydto.ListCreateDTO{
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

	dto.ListImageURL = strings.TrimSpace(listImageURL)
	return dto, nil
}

// GetByIDs assembles ListCreateDTO from pbId/tbId.（※残しているが、今後消すなら別途）
// NOTE: ここも「並べ替えしない」方針に揃える
func (q *ListCreateQuery) GetByIDs(
	ctx context.Context,
	productBlueprintID string,
	tokenBlueprintID string,
) (*querydto.ListCreateDTO, error) {
	return q.GetByIDsWithListImage(ctx, productBlueprintID, tokenBlueprintID, "")
}

func (q *ListCreateQuery) GetByIDsWithListImage(
	ctx context.Context,
	productBlueprintID string,
	tokenBlueprintID string,
	listImageURL string,
) (*querydto.ListCreateDTO, error) {
	if q == nil {
		return nil, errors.New("list create query is nil")
	}

	pbID := strings.TrimSpace(productBlueprintID)
	tbID := strings.TrimSpace(tokenBlueprintID)
	if pbID == "" || tbID == "" {
		return nil, errors.New("productBlueprintId and tokenBlueprintId are required")
	}

	// ------------------------------------------------------------
	// ProductBlueprint: productName / brandName
	// ------------------------------------------------------------
	productName := ""
	productBrandName := ""

	if q.nameResolver != nil {
		productName = strings.TrimSpace(q.nameResolver.ResolveProductName(ctx, pbID))
	}

	if q.pbPatchRepo != nil {
		if patch, err := q.pbPatchRepo.GetPatchByID(ctx, pbID); err == nil {
			brandID := strings.TrimSpace(getStringFieldAny(patch, "BrandID", "BrandId", "brandId"))
			if brandID != "" && q.nameResolver != nil {
				productBrandName = strings.TrimSpace(q.nameResolver.ResolveBrandName(ctx, brandID))
			}
			if productBrandName == "" {
				productBrandName = strings.TrimSpace(getStringFieldAny(patch, "BrandName", "brandName"))
			}
		}
	}

	// ------------------------------------------------------------
	// TokenBlueprint: tokenName / brandName
	// ------------------------------------------------------------
	tokenName := ""
	tokenBrandName := ""

	if q.nameResolver != nil {
		tokenName = strings.TrimSpace(q.nameResolver.ResolveTokenName(ctx, tbID))
	}

	if q.tbPatchRepo != nil {
		if patch, err := q.tbPatchRepo.GetPatchByID(ctx, tbID); err == nil {
			brandID := strings.TrimSpace(getStringFieldAny(patch, "BrandID", "BrandId", "brandId"))
			if brandID != "" && q.nameResolver != nil {
				tokenBrandName = strings.TrimSpace(q.nameResolver.ResolveBrandName(ctx, brandID))
			}
			if tokenBrandName == "" {
				tokenBrandName = strings.TrimSpace(getStringFieldAny(patch, "BrandName", "brandName"))
			}
		}
	}

	// ------------------------------------------------------------
	// ✅ ModelRefs / PriceRows（並べ替えしない）
	// ------------------------------------------------------------
	modelRefs := q.listModelRefs(ctx, pbID)
	priceRows, totalStock := q.buildPriceRowsByIDs(ctx, pbID, tbID, modelRefs)

	dto := &querydto.ListCreateDTO{
		InventoryID:        buildInventoryID(pbID, tbID),
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

	dto.ListImageURL = strings.TrimSpace(listImageURL)
	return dto, nil
}

// inventoryId = "{pbId}__{tbId}"
func buildInventoryID(productBlueprintID, tokenBlueprintID string) string {
	return strings.TrimSpace(productBlueprintID) + "__" + strings.TrimSpace(tokenBlueprintID)
}

func parseInventoryID(inventoryID string) (pbID string, tbID string, ok bool) {
	id := strings.TrimSpace(inventoryID)
	if id == "" {
		return "", "", false
	}
	parts := strings.SplitN(id, "__", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	pb := strings.TrimSpace(parts[0])
	tb := strings.TrimSpace(parts[1])
	if pb == "" || tb == "" {
		return "", "", false
	}
	return pb, tb, true
}

func toDisplayOrderPtr(v int) *int {
	// 互換: 0 は未設定として null に寄せる
	if v == 0 {
		return nil
	}
	x := v
	return &x
}

// ============================================================
// internal: build PriceRows
// - 母集団: productBlueprintPatch.ModelRefs（順序はそのまま）
// - stock: inventory が取れれば picked.Stock[modelId] を反映、無ければ 0
// - 重要: stock==0 でも行を出す（価格入力のため）
// - ✅ 並べ替えはしない（displayOrder/size/color/modelId で sort しない）
// ============================================================

func (q *ListCreateQuery) buildPriceRowsByIDs(
	ctx context.Context,
	productBlueprintID string,
	tokenBlueprintID string,
	modelRefs []querydto.ListCreateModelRefDTO,
) ([]querydto.ListCreatePriceRowDTO, int) {
	if q == nil {
		return nil, 0
	}

	pbID := strings.TrimSpace(productBlueprintID)
	tbID := strings.TrimSpace(tokenBlueprintID)
	if pbID == "" || tbID == "" {
		return nil, 0
	}

	if len(modelRefs) == 0 {
		return nil, 0
	}

	// inventory を拾えれば stock 参照に使う（拾えなくても PriceRows は返す）
	var picked *invdom.Mint
	if q.invRepo != nil {
		invs, err := q.invRepo.ListByProductBlueprintID(ctx, pbID)
		if err == nil && len(invs) > 0 {
			wantInvID := buildInventoryID(pbID, tbID)

			for i := range invs {
				if strings.TrimSpace(invs[i].ID) == wantInvID {
					picked = &invs[i]
					break
				}
			}
			if picked == nil {
				for i := range invs {
					if strings.TrimSpace(invs[i].TokenBlueprintID) == tbID {
						picked = &invs[i]
						break
					}
				}
			}
		}
	}

	rows := make([]querydto.ListCreatePriceRowDTO, 0, len(modelRefs))
	total := 0

	for _, ref := range modelRefs {
		mid := strings.TrimSpace(ref.ModelID)
		if mid == "" {
			continue
		}

		stock := 0
		if picked != nil && picked.Stock != nil {
			if ms, ok := picked.Stock[mid]; ok {
				// ✅ availableStock を採用（accumulation - reservedCount）
				_, _, available := modelStockNumbers(ms) // defined in inventory_query.go
				stock = available
			}
		}

		attr := resolver.ModelResolved{}
		if q.nameResolver != nil {
			attr = q.nameResolver.ResolveModelResolved(ctx, mid)
		}

		sz := strings.TrimSpace(attr.Size)
		cl := strings.TrimSpace(attr.Color)
		if sz == "" {
			sz = "-"
		}
		if cl == "" {
			cl = "-"
		}

		rows = append(rows, querydto.ListCreatePriceRowDTO{
			ModelID:      mid,
			DisplayOrder: ref.DisplayOrder, // ✅ 取得して渡すのみ
			Stock:        stock,
			Size:         sz,
			Color:        cl,
			RGB:          attr.RGB,
			Price:        nil,
		})

		total += stock
	}

	return rows, total
}

func (q *ListCreateQuery) buildPriceRowsByInventoryID(
	ctx context.Context,
	inventoryID string,
	modelRefs []querydto.ListCreateModelRefDTO,
) ([]querydto.ListCreatePriceRowDTO, int) {
	if q == nil {
		return nil, 0
	}

	invID := strings.TrimSpace(inventoryID)
	if invID == "" {
		return nil, 0
	}

	pbID, tbID, ok := parseInventoryID(invID)
	if !ok || pbID == "" || tbID == "" {
		return nil, 0
	}

	if len(modelRefs) == 0 {
		return nil, 0
	}

	// inventory を拾えれば stock 参照に使う（拾えなくても PriceRows は返す）
	var picked *invdom.Mint
	if q.invRepo != nil {
		invs, err := q.invRepo.ListByProductBlueprintID(ctx, pbID)
		if err == nil && len(invs) > 0 {
			for i := range invs {
				if strings.TrimSpace(invs[i].ID) == invID {
					picked = &invs[i]
					break
				}
			}
			if picked == nil {
				for i := range invs {
					if strings.TrimSpace(invs[i].TokenBlueprintID) == tbID {
						picked = &invs[i]
						break
					}
				}
			}
		}
	}

	rows := make([]querydto.ListCreatePriceRowDTO, 0, len(modelRefs))
	total := 0

	for _, ref := range modelRefs {
		mid := strings.TrimSpace(ref.ModelID)
		if mid == "" {
			continue
		}

		stock := 0
		if picked != nil && picked.Stock != nil {
			if ms, ok := picked.Stock[mid]; ok {
				// ✅ availableStock を採用（accumulation - reservedCount）
				_, _, available := modelStockNumbers(ms) // defined in inventory_query.go
				stock = available
			}
		}

		attr := resolver.ModelResolved{}
		if q.nameResolver != nil {
			attr = q.nameResolver.ResolveModelResolved(ctx, mid)
		}

		sz := strings.TrimSpace(attr.Size)
		cl := strings.TrimSpace(attr.Color)
		if sz == "" {
			sz = "-"
		}
		if cl == "" {
			cl = "-"
		}

		rows = append(rows, querydto.ListCreatePriceRowDTO{
			ModelID:      mid,
			DisplayOrder: ref.DisplayOrder, // ✅ 取得して渡すのみ
			Stock:        stock,
			Size:         sz,
			Color:        cl,
			RGB:          attr.RGB,
			Price:        nil,
		})

		total += stock
	}

	return rows, total
}

// 母集団: productBlueprintPatch.ModelRefs（順序は patch のまま）
func (q *ListCreateQuery) listModelRefs(ctx context.Context, productBlueprintID string) []querydto.ListCreateModelRefDTO {
	if q == nil || q.pbPatchRepo == nil {
		return nil
	}
	pbID := strings.TrimSpace(productBlueprintID)
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

	// ✅ 並べ替えしない：入力順のまま
	for _, r := range refs {
		mid := strings.TrimSpace(r.ModelID)
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
