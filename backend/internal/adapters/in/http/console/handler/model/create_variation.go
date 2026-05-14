// backend/internal/adapters/in/http/console/handler/model/create_variation.go
package model

import (
	"encoding/json"
	"net/http"
)

// ------------------------------------------------------------
// POST /models/{productBlueprintID}/variations
// ------------------------------------------------------------
func (h *ModelHandler) createVariation(w http.ResponseWriter, r *http.Request, productBlueprintID string) {
	ctx := r.Context()

	if productBlueprintID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid productBlueprintID"})
		return
	}

	var req createModelVariationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	newVar := toNewModelVariation(productBlueprintID, req)
	if err := newVar.Validate(); err != nil {
		writeModelErr(w, err)
		return
	}

	mv, err := h.uc.CreateModelVariation(ctx, newVar)
	if err != nil {
		writeModelErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(mv)
}
