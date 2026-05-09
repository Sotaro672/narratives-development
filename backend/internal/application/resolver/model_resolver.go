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
//   - modelNumber: mv.ModelNumber
//   - size:        mv.Size
//   - color:       mv.Color.Name
//   - rgb:         mv.Color.RGB
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
	if err != nil || mv == nil {
		return ModelResolved{}
	}

	colorName, rgb := extractColorNameAndRGBFromModelVariation(mv)

	return ModelResolved{
		ModelNumber: mv.ModelNumber,
		Size:        mv.Size,
		Color:       colorName,
		RGB:         rgb,
	}
}

// domain.ModelVariation.Color の正規 field を直接読む。
// Firestore の color.name / color.rgb は repository mapper 側で
// modeldom.Color{Name, RGB} に変換済みであることを前提にする。
func extractColorNameAndRGBFromModelVariation(mv *modeldom.ModelVariation) (string, *int) {
	if mv == nil {
		return "", nil
	}

	rgb := mv.Color.RGB
	return mv.Color.Name, &rgb
}
