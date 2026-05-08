// backend/internal/adapters/in/http/console/handler/model/mapper.go
package model

import (
	modeldom "narratives/internal/domain/model"
)

func toMeasurements(in map[string]float64) modeldom.Measurements {
	ms := make(modeldom.Measurements)
	for k, v := range in {
		key := k
		if key == "" {
			continue
		}
		ms[key] = int(v)
	}
	return ms
}

// rgb 必須化方針に従い、req.RGB は常に使用する（省略分岐なし）
func toNewModelVariation(productBlueprintID string, req createModelVariationRequest) modeldom.NewModelVariation {
	return modeldom.NewModelVariation{
		ProductBlueprintID: productBlueprintID,
		ModelNumber:        req.ModelNumber,
		Size:               req.Size,
		Color: modeldom.Color{
			Name: req.Color,
			RGB:  req.RGB, // ✅ 必須（0=黒も正）
		},
		Measurements: toMeasurements(req.Measurements),
	}
}

// rgb 必須化方針に従い、Update でも Color.RGB は常に設定する（省略分岐なし）
func toModelVariationUpdate(req createModelVariationRequest) modeldom.ModelVariationUpdate {
	modelNumber := req.ModelNumber
	size := req.Size
	color := modeldom.Color{
		Name: req.Color,
		RGB:  req.RGB, // ✅ 必須（0=黒も正）
	}

	return modeldom.ModelVariationUpdate{
		ModelNumber:  &modelNumber,
		Size:         &size,
		Color:        &color,
		Measurements: toMeasurements(req.Measurements),
	}
}
