// backend/internal/application/query/mall/history_model_resolver.go
package mall

import (
	"context"
	"encoding/json"
	"errors"

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
	in HistoryResolveModelInput,
) (HistoryResolvedModel, error) {
	if r == nil || r.modelRepo == nil {
		return HistoryResolvedModel{}, ErrHistoryModelResolverNotConfigured
	}

	modelID := in.ModelID
	if modelID == "" {
		return HistoryResolvedModel{}, ErrHistoryModelIDEmpty
	}

	variation, err := r.modelRepo.GetModelVariationByID(ctx, modelID)
	if err != nil {
		return HistoryResolvedModel{}, err
	}

	out := HistoryResolvedModel{
		ModelID:            modelID,
		InventoryID:        in.InventoryID,
		ProductBlueprintID: in.ProductBlueprintID,
		TokenBlueprintID:   in.TokenBlueprintID,
	}

	if variation == nil || *variation == nil {
		return out, nil
	}

	apparelVariation, ok := toHistoryApparelModelVariation(*variation)
	if !ok {
		return out, nil
	}

	out.Size = apparelVariation.Size
	out.ModelNumber = apparelVariation.ModelNumber
	out.Measurements = cloneHistoryModelMeasurements(apparelVariation.Measurements)
	out.Color = historyColorFromModelColor(apparelVariation.Color)

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

func historyColorFromModelColor(color modeldom.Color) *HistoryColor {
	body, err := json.Marshal(color)
	if err != nil {
		return nil
	}

	var out HistoryColor
	if err := json.Unmarshal(body, &out); err != nil {
		return nil
	}

	if out.Name == "" && out.Hex == "" {
		return nil
	}

	return &out
}
