// backend/internal/application/resolver/model_resolver.go
package resolver

import (
	"context"

	modeldom "narratives/internal/domain/model"
)

// ------------------------------------------------------------
// ModelVariation (modelId → modelNumber/size/color/rgb)
// - Firestore の保存 label は repository / mapper 側で domain に変換済みとする
// - resolver は domain の正規 field だけを読む
//   - modelNumber: apparelMV.ModelNumber
//   - size:        apparelMV.Size
//   - color:       apparelMV.Color.Name
//   - rgb:         apparelMV.Color.RGB
// ------------------------------------------------------------

type ModelResolved struct {
	ModelNumber string
	Size        string
	Color       string
	RGB         *int
}

// ResolveModelResolved は modelId から modelNumber/size/color/rgb を解決する。
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

	apparelMV, ok := toResolverApparelModelVariation(*mv)
	if !ok {
		return ModelResolved{}
	}

	colorName, rgb := extractColorNameAndRGBFromApparelModelVariation(apparelMV)

	return ModelResolved{
		ModelNumber: apparelMV.ModelNumber,
		Size:        apparelMV.Size,
		Color:       colorName,
		RGB:         rgb,
	}
}

// domain.ApparelModelVariation.Color の正規 field を直接読む。
// Firestore の color.name / color.rgb は repository mapper 側で
// modeldom.Color{Name, RGB} に変換済みであることを前提にする。
func extractColorNameAndRGBFromApparelModelVariation(mv modeldom.ApparelModelVariation) (string, *int) {
	rgb := mv.Color.RGB
	return mv.Color.Name, &rgb
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
