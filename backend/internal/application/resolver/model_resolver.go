// backend/internal/application/resolver/model_resolver.go
package resolver

import (
	"context"

	modeldom "narratives/internal/domain/model"
)

// ------------------------------------------------------------
// ModelVariation (modelId → modelNumber/size/color/rgb/volume)
// - Firestore の保存 label は repository / mapper 側で domain に変換済みとする
// - resolver は domain の正規 field だけを読む
//
// apparel:
//   - kind:        "apparel"
//   - modelNumber: apparelMV.ModelNumber
//   - size:        apparelMV.Size
//   - color:       apparelMV.Color.Name
//   - rgb:         apparelMV.Color.RGB
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
	Size  string
	Color string
	RGB   *int

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

	id := variationID
	if id == "" {
		return ModelResolved{}
	}

	mv, err := r.modelNumberRepo.GetModelVariationByID(ctx, id)
	if err != nil || mv == nil || *mv == nil {
		return ModelResolved{}
	}

	if apparelMV, ok := toResolverApparelModelVariation(*mv); ok {
		colorName, rgb := extractColorNameAndRGBFromApparelModelVariation(apparelMV)

		return ModelResolved{
			Kind:        "apparel",
			ModelNumber: apparelMV.ModelNumber,
			Size:        apparelMV.Size,
			Color:       colorName,
			RGB:         rgb,
		}
	}

	if alcoholMV, ok := toResolverAlcoholModelVariation(*mv); ok {
		volumeValue, volumeUnit := extractVolumeValueAndUnitFromAlcoholModelVariation(alcoholMV)

		return ModelResolved{
			Kind:        "alcohol",
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

func toResolverApparelModelVariation(v modeldom.ModelVariation) (modeldom.ApparelModelVariation, bool) {
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

func toResolverAlcoholModelVariation(v modeldom.ModelVariation) (modeldom.AlcoholModelVariation, bool) {
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
