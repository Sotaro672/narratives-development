// backend/internal/application/query/mall/history_model_resolver.go
package mall

import (
	"context"
	"encoding/json"
	"errors"

	historydto "narratives/internal/application/query/mall/dto"

	modeldom "narratives/internal/domain/model"
)

var (
	ErrHistoryModelResolverNotConfigured = errors.New("mall history model resolver: not configured")
)

type HistoryModelVariationRepository interface {
	GetModelVariationByID(ctx context.Context, variationID string) (*modeldom.ModelVariation, error)
}

type HistoryModelResolverImpl struct {
	modelRepo HistoryModelVariationRepository
}

func NewHistoryModelResolver(
	modelRepo HistoryModelVariationRepository,
) *HistoryModelResolverImpl {
	return &HistoryModelResolverImpl{
		modelRepo: modelRepo,
	}
}

func (r *HistoryModelResolverImpl) ResolveHistoryModelByID(
	ctx context.Context,
	in historydto.HistoryResolveModelInput,
) (historydto.HistoryResolvedModel, error) {
	if r == nil || r.modelRepo == nil {
		return historydto.HistoryResolvedModel{}, ErrHistoryModelResolverNotConfigured
	}

	modelID := in.ModelID
	if modelID == "" {
		return historydto.HistoryResolvedModel{}, ErrHistoryModelIDEmpty
	}

	variation, err := r.modelRepo.GetModelVariationByID(ctx, modelID)
	if err != nil {
		return historydto.HistoryResolvedModel{}, err
	}

	out := historydto.HistoryResolvedModel{
		ModelID:            modelID,
		InventoryID:        in.InventoryID,
		ProductBlueprintID: in.ProductBlueprintID,
		TokenBlueprintID:   in.TokenBlueprintID,
	}

	if variation == nil || *variation == nil {
		return out, nil
	}

	if apparelVariation, ok := toHistoryApparelModelVariation(*variation); ok {
		out.Kind = "apparel"
		out.Size = apparelVariation.Size
		out.ModelNumber = apparelVariation.ModelNumber
		out.Measurements = cloneHistoryModelMeasurements(apparelVariation.Measurements)
		out.Color = historyColorFromModelColor(apparelVariation.Color)

		return out, nil
	}

	if alcoholVariation, ok := toHistoryAlcoholModelVariation(*variation); ok {
		out.Kind = "alcohol"
		out.ModelNumber = alcoholVariation.ModelNumber
		out.VolumeValue = historyVolumeValueFromAlcoholModelVariation(alcoholVariation)
		out.VolumeUnit = alcoholVariation.Volume.Unit

		return out, nil
	}

	return out, nil
}

func toHistoryApparelModelVariation(v modeldom.ModelVariation) (modeldom.ApparelModelVariation, bool) {
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

func toHistoryAlcoholModelVariation(v modeldom.ModelVariation) (modeldom.AlcoholModelVariation, bool) {
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

func cloneHistoryModelMeasurements(in map[string]int) map[string]int {
	if len(in) == 0 {
		return nil
	}

	out := make(map[string]int, len(in))
	for key, value := range in {
		if key == "" {
			continue
		}

		out[key] = value
	}

	if len(out) == 0 {
		return nil
	}

	return out
}

func historyColorFromModelColor(color modeldom.Color) *historydto.HistoryColor {
	body, err := json.Marshal(color)
	if err != nil {
		return nil
	}

	var out historydto.HistoryColor
	if err := json.Unmarshal(body, &out); err != nil {
		return nil
	}

	if out.Name == "" && out.Hex == "" && out.RGB == nil {
		return nil
	}

	return &out
}

func historyVolumeValueFromAlcoholModelVariation(
	mv modeldom.AlcoholModelVariation,
) *int {
	value := mv.Volume.Value
	return &value
}
