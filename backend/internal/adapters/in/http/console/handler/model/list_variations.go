// backend/internal/adapters/in/http/console/handler/model/list_variations.go
package model

import (
	"encoding/json"
	"net/http"

	modeldom "narratives/internal/domain/model"
)

// ------------------------------------------------------------
// GET /models/by-blueprint/{productBlueprintID}/variations
// ------------------------------------------------------------
func (h *ModelHandler) listVariationsByProductBlueprintID(
	w http.ResponseWriter,
	r *http.Request,
	productBlueprintID string,
) {
	ctx := r.Context()

	if productBlueprintID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid productBlueprintID"})
		return
	}

	vars, err := h.uc.GetModelVariations(ctx, productBlueprintID)
	if err != nil {
		writeModelErr(w, err)
		return
	}

	apparelVars := make([]modeldom.ApparelModelVariation, 0, len(vars))
	for _, v := range vars {
		apparel, ok := toApparelModelVariation(v)
		if !ok {
			continue
		}
		apparelVars = append(apparelVars, apparel)
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(toModelVariationDTOs(apparelVars))
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
