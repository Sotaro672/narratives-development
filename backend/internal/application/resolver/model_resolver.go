// backend/internal/application/resolver/model_resolver.go
package resolver

import (
	"context"

	modeldom "narratives/internal/domain/model"
)

// ------------------------------------------------------------
// ModelVariation (modelId → modelNumber/size/color/rgb/volume/measurements)
// - Firestore の保存 label は repository / mapper 側で domain に変換済みとする
// - resolver は domain の正規 field だけを読む
//
// apparel:
//   - kind:         "apparel"
//   - modelNumber:  apparelMV.ModelNumber
//   - size:         apparelMV.Size
//   - color:        apparelMV.Color.Name
//   - rgb:          apparelMV.Color.RGB
//   - measurements: apparelMV.Measurements
//
// alcohol:
//   - kind:        "alcohol"
//   - modelNumber: alcoholMV.ModelNumber
//   - volumeValue: alcoholMV.Volume.Value
//   - volumeUnit:  alcoholMV.Volume.Unit
// ------------------------------------------------------------

type ModelResolved struct {
	Kind        string
	ModelNumber string

	// apparel
	Size         string
	Color        string
	RGB          *int
	Measurements modeldom.Measurements

	// alcohol
	VolumeValue *int
	VolumeUnit  string
}

// ResolveModelResolved は modelId から model 表示情報を解決する。
// 取得できなかった場合はゼロ値を返す。
func (r *NameResolver) ResolveModelResolved(ctx context.Context, variationID string) ModelResolved {
	if r == nil || r.modelNumberRepo == nil {
		return ModelResolved{}
	}

	if variationID == "" {
		return ModelResolved{}
	}

	mv, err := r.modelNumberRepo.GetByID(ctx, variationID)
	if err != nil || mv == nil {
		return ModelResolved{}
	}

	if apparelMV, ok := mv.(modeldom.ApparelModelVariation); ok {
		colorName, rgb := extractColorNameAndRGBFromApparelModelVariation(apparelMV)

		return ModelResolved{
			Kind:         string(modeldom.ModelVariationKindApparel),
			ModelNumber:  apparelMV.ModelNumber,
			Size:         apparelMV.Size,
			Color:        colorName,
			RGB:          rgb,
			Measurements: apparelMV.Measurements,
		}
	}

	if alcoholMV, ok := mv.(modeldom.AlcoholModelVariation); ok {
		volumeValue, volumeUnit := extractVolumeValueAndUnitFromAlcoholModelVariation(alcoholMV)

		return ModelResolved{
			Kind:        string(modeldom.ModelVariationKindAlcohol),
			ModelNumber: alcoholMV.ModelNumber,
			VolumeValue: volumeValue,
			VolumeUnit:  volumeUnit,
		}
	}

	return ModelResolved{}
}

// domain.ApparelModelVariation.Color の正規 field を直接読む。
// Firestore の color.name / color.rgb は repository mapper 側で
// modeldom.Color{Name, RGB} に変換済みであることを前提にする。
func extractColorNameAndRGBFromApparelModelVariation(mv modeldom.ApparelModelVariation) (string, *int) {
	rgb := mv.Color.RGB

	return mv.Color.Name, &rgb
}

// domain.AlcoholModelVariation.Volume の正規 field を直接読む。
// Firestore の volume.value / volume.unit は repository mapper 側で
// modeldom.Volume{Value, Unit} に変換済みであることを前提にする。
func extractVolumeValueAndUnitFromAlcoholModelVariation(mv modeldom.AlcoholModelVariation) (*int, string) {
	value := mv.Volume.Value

	return &value, mv.Volume.Unit
}
