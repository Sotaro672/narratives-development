// backend/internal/adapters/in/http/console/handler/model/get_variation.go
package model

import (
	"encoding/json"
	"net/http"

	modeldom "narratives/internal/domain/model"
)

// ------------------------------------------------------------
// GET /models/variations/{variationId}
// ------------------------------------------------------------
func (h *ModelHandler) getVariationByID(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	// 期待値: modelId (= variationId) から modelNumber/size/color を解決する
	mv, err := h.uc.GetModelVariationByID(ctx, id)
	if err != nil {
		writeModelErr(w, err)
		return
	}
	if mv == nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "variation not found"})
		return
	}

	var apparelMV modeldom.ApparelModelVariation

	switch v := (*mv).(type) {
	case modeldom.ApparelModelVariation:
		apparelMV = v
	case *modeldom.ApparelModelVariation:
		if v == nil {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "variation not found"})
			return
		}
		apparelMV = *v
	default:
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unsupported model variation type"})
		return
	}

	_ = json.NewEncoder(w).Encode(toModelVariationDTO(apparelMV))
}
