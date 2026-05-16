// backend/internal/application/query/mall/catalog/catalog_query_model_mapper.go
package catalogQuery

import (
	dto "narratives/internal/application/query/mall/dto"

	modeldom "narratives/internal/domain/model"
)

func toCatalogModelVariationDTOAny(v any) (dto.CatalogModelVariationDTO, bool) {
	switch x := v.(type) {
	case modeldom.ApparelModelVariation:
		return toCatalogApparelModelVariationDTO(x)
	case *modeldom.ApparelModelVariation:
		if x == nil {
			return dto.CatalogModelVariationDTO{}, false
		}
		return toCatalogApparelModelVariationDTO(*x)

	case modeldom.AlcoholModelVariation:
		return toCatalogAlcoholModelVariationDTO(x)
	case *modeldom.AlcoholModelVariation:
		if x == nil {
			return dto.CatalogModelVariationDTO{}, false
		}
		return toCatalogAlcoholModelVariationDTO(*x)

	case modeldom.ModelVariation:
		return toCatalogModelVariationDTO(x)
	case *modeldom.ModelVariation:
		if x == nil || *x == nil {
			return dto.CatalogModelVariationDTO{}, false
		}
		return toCatalogModelVariationDTO(*x)

	default:
		return dto.CatalogModelVariationDTO{}, false
	}
}

func toCatalogModelVariationDTO(mv modeldom.ModelVariation) (dto.CatalogModelVariationDTO, bool) {
	if mv == nil {
		return dto.CatalogModelVariationDTO{}, false
	}

	if apparel, ok := toApparelModelVariation(mv); ok {
		return toCatalogApparelModelVariationDTO(apparel)
	}

	if alcohol, ok := toAlcoholModelVariation(mv); ok {
		return toCatalogAlcoholModelVariationDTO(alcohol)
	}

	return dto.CatalogModelVariationDTO{}, false
}

func toCatalogApparelModelVariationDTO(mv modeldom.ApparelModelVariation) (dto.CatalogModelVariationDTO, bool) {
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
		Kind:               "apparel",
		ModelNumber:        mv.ModelNumber,

		Size: mv.Size,

		ColorName: mv.Color.Name,
		ColorRGB:  mv.Color.RGB,

		Measurements: measurements,

		StockKeys: 0,
	}, true
}

func toCatalogAlcoholModelVariationDTO(mv modeldom.AlcoholModelVariation) (dto.CatalogModelVariationDTO, bool) {
	if mv.ID == "" {
		return dto.CatalogModelVariationDTO{}, false
	}

	value := mv.Volume.Value

	return dto.CatalogModelVariationDTO{
		ID:                 mv.ID,
		ProductBlueprintID: mv.ProductBlueprintID,
		Kind:               "alcohol",
		ModelNumber:        mv.ModelNumber,

		VolumeValue: &value,
		VolumeUnit:  mv.Volume.Unit,

		Measurements: map[string]int{},

		StockKeys: 0,
	}, true
}

func toApparelModelVariation(v modeldom.ModelVariation) (modeldom.ApparelModelVariation, bool) {
	if v == nil {
		return modeldom.ApparelModelVariation{}, false
	}

	switch x := v.(type) {
	case modeldom.ApparelModelVariation:
		return x, true
	case *modeldom.ApparelModelVariation:
		if x == nil {
			return modeldom.ApparelModelVariation{}, false
		}
		return *x, true
	default:
		return modeldom.ApparelModelVariation{}, false
	}
}

func toAlcoholModelVariation(v modeldom.ModelVariation) (modeldom.AlcoholModelVariation, bool) {
	if v == nil {
		return modeldom.AlcoholModelVariation{}, false
	}

	switch x := v.(type) {
	case modeldom.AlcoholModelVariation:
		return x, true
	case *modeldom.AlcoholModelVariation:
		if x == nil {
			return modeldom.AlcoholModelVariation{}, false
		}
		return *x, true
	default:
		return modeldom.AlcoholModelVariation{}, false
	}
}
