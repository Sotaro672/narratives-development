// backend/internal/application/query/mall/catalog/catalog_query_mappers.go
package catalogQuery

import (
	dto "narratives/internal/application/query/mall/dto"

	invdom "narratives/internal/domain/inventory"
	ldom "narratives/internal/domain/list"
	pbdom "narratives/internal/domain/productBlueprint"
	productBlueprintReview "narratives/internal/domain/productBlueprintReview"
)

func toCatalogListDTO(l ldom.List) dto.CatalogListDTO {
	return dto.CatalogListDTO{
		ID:          l.ID,
		Title:       l.Title,
		Description: l.Description,
		Image:       l.ImageID, // primary image docID (not URL)
		Prices:      l.Prices,

		InventoryID: l.InventoryID,
	}
}

func toCatalogProductBlueprintDTO(
	pb *pbdom.ProductBlueprint,
) dto.CatalogProductBlueprintDTO {
	if pb == nil {
		return dto.CatalogProductBlueprintDTO{}
	}

	category := pb.ProductBlueprintCategory

	out := dto.CatalogProductBlueprintDTO{
		ID:          pb.ID,
		ProductName: pb.ProductName,
		BrandID:     pb.BrandID,
		CompanyID:   pb.CompanyID,

		Printed:          pb.Printed,
		ProductIDTagType: pb.ProductIdTag.Type,

		ProductBlueprintCategoryID:     category.ID,
		ProductBlueprintCategoryCode:   category.Code,
		ProductBlueprintCategoryKind:   string(category.Kind),
		ProductBlueprintCategoryNameEn: category.NameEn,
		ProductBlueprintCategoryNameJa: category.NameJa,
		ProductBlueprintCategoryPath:   append([]string(nil), category.Path...),

		CategoryFields: cloneCatalogCategoryFields(pb.CategoryFields),

		ModelRefs: nil,
	}

	if len(pb.ModelRefs) > 0 {
		refs := make(
			[]dto.CatalogProductBlueprintModelRefDTO,
			0,
			len(pb.ModelRefs),
		)

		for _, r := range pb.ModelRefs {
			if r.ModelID == "" {
				continue
			}

			refs = append(refs, dto.CatalogProductBlueprintModelRefDTO{
				ModelID:      r.ModelID,
				DisplayOrder: r.DisplayOrder,
			})
		}

		if len(refs) > 0 {
			out.ModelRefs = refs
		}
	}

	return out
}

func cloneCatalogCategoryFields(fields pbdom.CategoryFields) map[string]any {
	if len(fields) == 0 {
		return nil
	}

	out := make(map[string]any, len(fields))

	for key, value := range fields {
		if key == "" || value == nil {
			continue
		}

		out[key] = value
	}

	if len(out) == 0 {
		return nil
	}

	return out
}

// Mint -> CatalogInventoryDTO
// Firestore 正:
// productBlueprintId / tokenBlueprintId / modelIds / stock.*.accumulation / stock.*.reservedCount
func toCatalogInventoryDTOFromMint(m invdom.Mint) *dto.CatalogInventoryDTO {
	out := &dto.CatalogInventoryDTO{
		ID:                 m.ID,
		ProductBlueprintID: m.ProductBlueprintID,
		TokenBlueprintID:   m.TokenBlueprintID,
		ModelIDs:           append([]string{}, m.ModelIDs...),
		Stock:              map[string]dto.CatalogInventoryModelStockDTO{},
	}

	if m.Stock == nil {
		return out
	}

	for modelID, ms := range m.Stock {
		if modelID == "" {
			continue
		}

		out.Stock[modelID] = dto.CatalogInventoryModelStockDTO{
			Accumulation:  ms.Accumulation,
			ReservedCount: ms.ReservedCount,
		}
	}

	return out
}

// ProductBlueprintReview summary -> CatalogProductReviewSummaryDTO
func toCatalogProductReviewSummaryDTO(
	s productBlueprintReview.ProductReviewSummary,
) *dto.CatalogProductReviewSummaryDTO {
	return &dto.CatalogProductReviewSummaryDTO{
		ProductBlueprintID: s.ProductBlueprintID,
		Status:             s.Status,
		TotalCount:         s.TotalCount,
		AverageRating:      s.AverageRating,
		Rating5Count:       s.Rating5Count,
		Rating4Count:       s.Rating4Count,
		Rating3Count:       s.Rating3Count,
		Rating2Count:       s.Rating2Count,
		Rating1Count:       s.Rating1Count,
	}
}
