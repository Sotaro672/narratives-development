// backend\internal\application\query\mall\catalog\catalog_query_mappers.go
package catalogQuery

import (
	"fmt"
	"strings"

	dto "narratives/internal/application/query/mall/dto"

	invdom "narratives/internal/domain/inventory"
	ldom "narratives/internal/domain/list"
	pbdom "narratives/internal/domain/productBlueprint"
)

func toCatalogListDTO(l ldom.List) dto.CatalogListDTO {
	return dto.CatalogListDTO{
		ID:          strings.TrimSpace(l.ID),
		Title:       strings.TrimSpace(l.Title),
		Description: strings.TrimSpace(l.Description),
		Image:       strings.TrimSpace(l.ImageID),
		Prices:      l.Prices,

		InventoryID: strings.TrimSpace(l.InventoryID),

		ProductBlueprintID: pickStringField(l, "ProductBlueprintID", "ProductBlueprintId", "productBlueprintId"),
		TokenBlueprintID:   pickStringField(l, "TokenBlueprintID", "TokenBlueprintId", "tokenBlueprintId"),
	}
}

func toCatalogProductBlueprintDTO(pb *pbdom.ProductBlueprint) dto.CatalogProductBlueprintDTO {
	out := dto.CatalogProductBlueprintDTO{
		ID:          strings.TrimSpace(pb.ID),
		ProductName: strings.TrimSpace(pb.ProductName),
		BrandID:     strings.TrimSpace(pb.BrandID),
		CompanyID:   strings.TrimSpace(pb.CompanyID),

		ItemType: fmt.Sprint(pb.ItemType),
		Fit:      fmt.Sprint(pb.Fit),
		Material: fmt.Sprint(pb.Material),

		Weight:  pb.Weight,
		Printed: pb.Printed,

		QualityAssurance: append([]string{}, pb.QualityAssurance...),

		ProductIDTagType: pickProductIDTagType(pb),

		// ✅ modelRefs (domain: pb.ModelRefs -> dto: out.ModelRefs)
		ModelRefs: nil,
	}

	if len(pb.ModelRefs) > 0 {
		refs := make([]dto.CatalogProductBlueprintModelRefDTO, 0, len(pb.ModelRefs))
		for _, r := range pb.ModelRefs {
			mid := strings.TrimSpace(r.ModelID)
			if mid == "" {
				continue
			}
			refs = append(refs, dto.CatalogProductBlueprintModelRefDTO{
				ModelID:      mid,
				DisplayOrder: r.DisplayOrder,
			})
		}
		if len(refs) > 0 {
			out.ModelRefs = refs
		}
	}

	return out
}

// Mint -> CatalogInventoryDTO（domain を正とする）
func toCatalogInventoryDTOFromMint(m invdom.Mint) *dto.CatalogInventoryDTO {
	out := &dto.CatalogInventoryDTO{
		ID:                 strings.TrimSpace(m.ID),
		ProductBlueprintID: strings.TrimSpace(m.ProductBlueprintID),
		TokenBlueprintID:   strings.TrimSpace(m.TokenBlueprintID),
		ModelIDs:           append([]string{}, m.ModelIDs...),
		Stock:              map[string]dto.CatalogInventoryModelStockDTO{},
	}

	if m.Stock == nil {
		return out
	}

	for modelID, ms := range m.Stock {
		mid := strings.TrimSpace(modelID)
		if mid == "" {
			continue
		}

		a := pickIntField(ms, "Accumulation", "accumulation", "Count", "count")
		r := pickIntField(ms, "ReservedCount", "reservedCount", "Reserved", "reserved")

		out.Stock[mid] = dto.CatalogInventoryModelStockDTO{
			Accumulation:  a,
			ReservedCount: r,
		}
	}

	return out
}
