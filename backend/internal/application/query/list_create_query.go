// backend/internal/application/query/list_create_query.go
package query

import (
	"context"
	"errors"
	"strings"

	querydto "narratives/internal/application/query/dto"
	resolver "narratives/internal/application/resolver"
)

// ============================================================
// ListCreateQuery
// - listCreate 画面に必要な最小情報を組み立てる（1出品 = 1 inventory）
// - pbId から: productName / brandName
// - tbId から: tokenName / brandName
//
// NOTE:
// - productBlueprintPatchReader / tokenBlueprintPatchReader は
//   inventory_query.go 側の定義を正として「重複定義しない」
// ============================================================

type ListCreateQuery struct {
	pbPatchRepo  productBlueprintPatchReader // defined in inventory_query.go
	tbPatchRepo  tokenBlueprintPatchReader   // defined in inventory_query.go
	nameResolver *resolver.NameResolver
}

func NewListCreateQuery(
	pbPatchRepo productBlueprintPatchReader,
	tbPatchRepo tokenBlueprintPatchReader,
	nameResolver *resolver.NameResolver,
) *ListCreateQuery {
	return &ListCreateQuery{
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

	dto := &querydto.ListCreateDTO{
		InventoryID:        buildInventoryID(pbID, tbID),
		ProductBlueprintID: pbID,
		TokenBlueprintID:   tbID,

		ProductBrandName: productBrandName,
		ProductName:      productName,

		TokenBrandName: tokenBrandName,
		TokenName:      tokenName,
	}

	return dto, nil
}

// inventoryId = "{pbId}__{tbId}"
func buildInventoryID(productBlueprintID, tokenBlueprintID string) string {
	return strings.TrimSpace(productBlueprintID) + "__" + strings.TrimSpace(tokenBlueprintID)
}
