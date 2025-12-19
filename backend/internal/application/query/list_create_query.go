// backend/internal/application/query/list_create_query.go
package query

import (
	"context"
	"errors"
	"sort"
	"strings"

	querydto "narratives/internal/application/query/dto"
	resolver "narratives/internal/application/resolver"
	invdom "narratives/internal/domain/inventory"
)

// ============================================================
// ListCreateQuery
// - listCreate 画面に必要な最小情報を組み立てる（1出品 = 1 inventory）
// - pbId から: productName / brandName
// - tbId から: tokenName / brandName
// - inventory から: modelId ごとの model metadata (size/color/rgb) + stock
//
// NOTE:
// - productBlueprintPatchReader / tokenBlueprintPatchReader / inventoryReader / modelStockLen は
//   inventory_query.go 側の定義を正として「重複定義しない」
// ============================================================

type ListCreateQuery struct {
	// ✅ inventory から rows を作るため
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
		invRepo:      nil, // optional (backward compatible)
		pbPatchRepo:  pbPatchRepo,
		tbPatchRepo:  tbPatchRepo,
		nameResolver: nameResolver,
	}
}

// ✅ NEW: inventory reader も注入できるコンストラクタ（PriceRows/TotalStock を埋める）
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

// GetByIDs assembles ListCreateDTO from pbId/tbId.
// inventoryId は "{pbId}__{tbId}" 前提で生成する（1出品=1inventory）。
func (q *ListCreateQuery) GetByIDs(
	ctx context.Context,
	productBlueprintID string,
	tokenBlueprintID string,
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

	// productName は resolver（pbRepo:GetProductNameByID）から取るのが正
	if q.nameResolver != nil {
		productName = strings.TrimSpace(q.nameResolver.ResolveProductName(ctx, pbID))
	}

	// brandName は pbPatch.BrandID -> resolver.ResolveBrandName
	if q.pbPatchRepo != nil {
		if patch, err := q.pbPatchRepo.GetPatchByID(ctx, pbID); err == nil {
			brandID := ""
			if patch.BrandID != nil {
				brandID = strings.TrimSpace(*patch.BrandID)
			}
			if brandID != "" && q.nameResolver != nil {
				productBrandName = strings.TrimSpace(q.nameResolver.ResolveBrandName(ctx, brandID))
			}
			// fallback: Patch に BrandName が入っていれば使う
			if productBrandName == "" && patch.BrandName != nil {
				productBrandName = strings.TrimSpace(*patch.BrandName)
			}
		}
	}

	// ------------------------------------------------------------
	// TokenBlueprint: tokenName / brandName
	// ------------------------------------------------------------
	tokenName := ""
	tokenBrandName := ""

	// tokenName は resolver（tokenBlueprintRepo:GetByID の Name/Symbol）から取るのが正
	if q.nameResolver != nil {
		tokenName = strings.TrimSpace(q.nameResolver.ResolveTokenName(ctx, tbID))
	}

	// brandName は tbPatch.BrandID -> resolver.ResolveBrandName
	if q.tbPatchRepo != nil {
		if patch, err := q.tbPatchRepo.GetPatchByID(ctx, tbID); err == nil {
			brandID := ""
			if patch.BrandID != nil {
				brandID = strings.TrimSpace(*patch.BrandID)
			}
			if brandID != "" && q.nameResolver != nil {
				tokenBrandName = strings.TrimSpace(q.nameResolver.ResolveBrandName(ctx, brandID))
			}
			// fallback: Patch に BrandName が入っていれば使う
			if tokenBrandName == "" && patch.BrandName != nil {
				tokenBrandName = strings.TrimSpace(*patch.BrandName)
			}
		}
	}

	// ------------------------------------------------------------
	// ✅ inventory: modelId ごとの metadata + stock（PriceCard 用）
	// ------------------------------------------------------------
	priceRows, totalStock := q.buildPriceRowsFromInventory(ctx, pbID, tbID)

	dto := &querydto.ListCreateDTO{
		InventoryID:        buildInventoryID(pbID, tbID),
		ProductBlueprintID: pbID,
		TokenBlueprintID:   tbID,

		ProductBrandName: productBrandName,
		ProductName:      productName,

		TokenBrandName: tokenBrandName,
		TokenName:      tokenName,

		// ✅ PriceCard 用
		PriceRows:  priceRows,
		TotalStock: totalStock,
	}

	return dto, nil
}

// inventoryId = "{pbId}__{tbId}"
func buildInventoryID(productBlueprintID, tokenBlueprintID string) string {
	return strings.TrimSpace(productBlueprintID) + "__" + strings.TrimSpace(tokenBlueprintID)
}

// ============================================================
// internal: inventory -> priceRows
// - inventory.Stock から modelId ごとの stock を集計
// - NameResolver.ResolveModelResolved で size/color/rgb を解決
//
// Note:
// - invRepo が nil の場合は空で返す（画面が壊れないように）
// ============================================================

func (q *ListCreateQuery) buildPriceRowsFromInventory(
	ctx context.Context,
	productBlueprintID string,
	tokenBlueprintID string,
) ([]querydto.ListCreatePriceRowDTO, int) {
	if q == nil || q.invRepo == nil {
		return nil, 0
	}

	pbID := strings.TrimSpace(productBlueprintID)
	tbID := strings.TrimSpace(tokenBlueprintID)
	if pbID == "" || tbID == "" {
		return nil, 0
	}

	invs, err := q.invRepo.ListByProductBlueprintID(ctx, pbID)
	if err != nil || len(invs) == 0 {
		return nil, 0
	}

	// 1出品=1inventory 前提:
	// - まずは inventoryId が一致するものを優先
	// - なければ tokenBlueprintId が一致するものを採用
	wantInvID := buildInventoryID(pbID, tbID)

	var picked *invdom.Mint
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
	if picked == nil || picked.Stock == nil {
		return nil, 0
	}

	rows := make([]querydto.ListCreatePriceRowDTO, 0, len(picked.Stock))
	total := 0

	for modelID, ms := range picked.Stock {
		mid := strings.TrimSpace(modelID)
		if mid == "" {
			continue
		}

		stock := modelStockLen(ms) // defined in inventory_query.go
		if stock <= 0 {
			continue
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

		// price は現状 inventory から取れない前提なので nil（= 未入力）
		rows = append(rows, querydto.ListCreatePriceRowDTO{
			ModelID: mid,
			Stock:   stock,
			Size:    sz,
			Color:   cl,
			RGB:     attr.RGB,
			Price:   nil,
		})

		total += stock
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Size != rows[j].Size {
			return rows[i].Size < rows[j].Size
		}
		if rows[i].Color != rows[j].Color {
			return rows[i].Color < rows[j].Color
		}
		return rows[i].ModelID < rows[j].ModelID
	})

	return rows, total
}
