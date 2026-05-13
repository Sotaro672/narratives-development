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

func toCatalogProductBlueprintDTO(pb *pbdom.ProductBlueprint) dto.CatalogProductBlueprintDTO {
	if pb == nil {
		return dto.CatalogProductBlueprintDTO{}
	}

	out := dto.CatalogProductBlueprintDTO{
		ID:          pb.ID,
		ProductName: pb.ProductName,
		BrandID:     pb.BrandID,
		CompanyID:   pb.CompanyID,

		// fit / material / weight / qualityAssurance は ProductBlueprint 直下ではなく
		// CategoryFields に集約する。
		Fit:              categoryFieldString(pb.CategoryFields, "fit"),
		Material:         categoryFieldString(pb.CategoryFields, "material"),
		Weight:           categoryFieldFloat64(pb.CategoryFields, "weight"),
		QualityAssurance: categoryFieldStringSlice(pb.CategoryFields, "qualityAssurance"),

		Printed: pb.Printed,

		ProductIDTagType: pb.ProductIdTag.Type,

		ModelRefs: nil,
	}

	if len(pb.ModelRefs) > 0 {
		refs := make([]dto.CatalogProductBlueprintModelRefDTO, 0, len(pb.ModelRefs))
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

func categoryFieldString(fields pbdom.CategoryFields, key string) string {
	if len(fields) == 0 || key == "" {
		return ""
	}

	v, ok := fields[key]
	if !ok || v == nil {
		return ""
	}

	switch x := v.(type) {
	case string:
		return x
	default:
		return ""
	}
}

func categoryFieldFloat64(fields pbdom.CategoryFields, key string) float64 {
	if len(fields) == 0 || key == "" {
		return 0
	}

	v, ok := fields[key]
	if !ok || v == nil {
		return 0
	}

	switch x := v.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case int:
		return float64(x)
	case int8:
		return float64(x)
	case int16:
		return float64(x)
	case int32:
		return float64(x)
	case int64:
		return float64(x)
	case uint:
		return float64(x)
	case uint8:
		return float64(x)
	case uint16:
		return float64(x)
	case uint32:
		return float64(x)
	case uint64:
		return float64(x)
	default:
		return 0
	}
}

func categoryFieldStringSlice(fields pbdom.CategoryFields, key string) []string {
	if len(fields) == 0 || key == "" {
		return nil
	}

	v, ok := fields[key]
	if !ok || v == nil {
		return nil
	}

	switch x := v.(type) {
	case []string:
		return append([]string(nil), x...)

	case []any:
		out := make([]string, 0, len(x))
		for _, item := range x {
			s, ok := item.(string)
			if !ok || s == "" {
				continue
			}
			out = append(out, s)
		}
		if len(out) == 0 {
			return nil
		}
		return out

	default:
		return nil
	}
}

// Mint -> CatalogInventoryDTO（Firestore 正: productBlueprintId / tokenBlueprintId / modelIds / stock.*.accumulation / stock.*.reservedCount）
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
