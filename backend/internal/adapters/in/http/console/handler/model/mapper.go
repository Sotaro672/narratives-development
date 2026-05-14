// backend/internal/adapters/in/http/console/handler/model/mapper.go
package model

import (
	modeldom "narratives/internal/domain/model"
)

func toMeasurements(in map[string]float64) modeldom.Measurements {
	if len(in) == 0 {
		return nil
	}

	ms := make(modeldom.Measurements, len(in))
	for k, v := range in {
		key := k
		if key == "" {
			continue
		}
		ms[key] = int(v)
	}

	if len(ms) == 0 {
		return nil
	}

	return ms
}

// toNewModelVariation は request を category-specific な domain input に変換する。
//
// NOTE:
//   - req.Kind == "alcohol" の場合は alcohol variation を作る。
//   - それ以外、または kind 未指定の場合は既存互換として apparel variation を作る。
//   - rgb 必須化方針に従い、apparel では req.RGB を常に使用する。
//   - req.RGB == 0 は黒として正なので falsy 扱いしない。
func toNewModelVariation(
	productBlueprintID string,
	req createModelVariationRequest,
) modeldom.NewModelVariation {
	if req.Kind == string(modeldom.ModelVariationKindAlcohol) {
		return modeldom.NewModelVariationFromAlcohol(modeldom.NewAlcoholModelVariation{
			ProductBlueprintID: productBlueprintID,
			ModelNumber:        req.ModelNumber,
			Volume:             req.Volume,
		})
	}

	return modeldom.NewModelVariationFromApparel(modeldom.NewApparelModelVariation{
		ProductBlueprintID: productBlueprintID,
		ModelNumber:        req.ModelNumber,
		Size:               req.Size,
		Color: modeldom.Color{
			Name: req.Color,
			RGB:  req.RGB,
		},
		Measurements: toMeasurements(req.Measurements),
	})
}

// toModelVariationUpdate は request を category-specific な update DTO に変換する。
//
// NOTE:
//   - req.Kind == "alcohol" の場合は volume 更新を返す。
//   - それ以外、または kind 未指定の場合は既存互換として apparel 更新を返す。
//   - rgb 必須化方針に従い、apparel では req.RGB を常に使用する。
//   - req.RGB == 0 は黒として正なので falsy 扱いしない。
func toModelVariationUpdate(req createModelVariationRequest) modeldom.ModelVariationUpdate {
	modelNumber := req.ModelNumber

	if req.Kind == string(modeldom.ModelVariationKindAlcohol) {
		volume := req.Volume

		return modeldom.ModelVariationUpdate{
			ModelNumber: &modelNumber,
			Volume:      &volume,
		}
	}

	size := req.Size
	color := modeldom.Color{
		Name: req.Color,
		RGB:  req.RGB,
	}

	return modeldom.ModelVariationUpdate{
		ModelNumber:  &modelNumber,
		Size:         &size,
		Color:        &color,
		Measurements: toMeasurements(req.Measurements),
	}
}
