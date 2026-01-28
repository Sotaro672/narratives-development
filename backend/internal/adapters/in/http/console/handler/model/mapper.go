// backend/internal/adapters/in/http/console/handler/model/mapper.go
package model

import (
	"strings"

	modeldom "narratives/internal/domain/model"
)

func toMeasurements(in map[string]float64) modeldom.Measurements {
	ms := make(modeldom.Measurements)
	for k, v := range in {
		key := strings.TrimSpace(k)
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
		ProductBlueprintID: strings.TrimSpace(productBlueprintID),
		ModelNumber:        strings.TrimSpace(req.ModelNumber),
		Size:               strings.TrimSpace(req.Size),
		Color: modeldom.Color{
			Name: strings.TrimSpace(req.Color),
			RGB:  req.RGB, // ✅ 必須（0=黒も正）
		},
		Measurements: toMeasurements(req.Measurements),
	}
}

// rgb 必須化方針に従い、Update でも Color.RGB は常に設定する（省略分岐なし）
func toModelVariationUpdate(req createModelVariationRequest) modeldom.ModelVariationUpdate {
	modelNumber := strings.TrimSpace(req.ModelNumber)
	size := strings.TrimSpace(req.Size)
	color := modeldom.Color{
		Name: strings.TrimSpace(req.Color),
		RGB:  req.RGB, // ✅ 必須（0=黒も正）
	}

	return modeldom.ModelVariationUpdate{
		ModelNumber:  &modelNumber,
		Size:         &size,
		Color:        &color,
		Measurements: toMeasurements(req.Measurements),
	}
}
