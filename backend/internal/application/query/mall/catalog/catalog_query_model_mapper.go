// backend/internal/application/query/mall/catalog/catalog_query_model_mapper.go
package catalogQuery

import (
	dto "narratives/internal/application/query/mall/dto"

	modeldom "narratives/internal/domain/model"
)

func toCatalogModelVariationDTOAny(v any) (dto.CatalogModelVariationDTO, bool) {
	switch x := v.(type) {
	case modeldom.ModelVariation:
		return toCatalogModelVariationDTO(x)
	case *modeldom.ModelVariation:
		if x == nil {
			return dto.CatalogModelVariationDTO{}, false
		}
		return toCatalogModelVariationDTO(*x)
	default:
		return dto.CatalogModelVariationDTO{}, false
	}
}

func toCatalogModelVariationDTO(mv modeldom.ModelVariation) (dto.CatalogModelVariationDTO, bool) {
	if mv.ID == "" {
		return dto.CatalogModelVariationDTO{}, false
	}

	measurements := map[string]int{}
	for k, v := range mv.Measurements {
		if k == "" {
			continue
		}
		measurements[k] = v
	}

	return dto.CatalogModelVariationDTO{
		ID:                 mv.ID,
		ProductBlueprintID: mv.ProductBlueprintID,
		ModelNumber:        mv.ModelNumber,
		Size:               mv.Size,

		ColorName: mv.Color.Name,
		ColorRGB:  mv.Color.RGB,

		Measurements: measurements,

		StockKeys: 0,
	}, true
}
