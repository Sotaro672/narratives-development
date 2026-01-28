// backend/internal/adapters/in/http/console/handler/model/update_variation.go
package model

import (
	"encoding/json"
	"net/http"
	"strings"
)

// ------------------------------------------------------------
// PUT /models/{id}
// ------------------------------------------------------------
func (h *ModelHandler) updateVariation(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	var req createModelVariationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	updates := toModelVariationUpdate(req)

	mv, err := h.uc.UpdateModelVariation(ctx, id, updates)
	if err != nil {
		writeModelErr(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(mv)
}
