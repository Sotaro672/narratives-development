// backend/internal/adapters/in/http/console/handler/model/delete_variation.go
package model

import (
	"encoding/json"
	"net/http"
	"strings"
)

// ------------------------------------------------------------
// DELETE /models/{id}
// ------------------------------------------------------------
func (h *ModelHandler) deleteVariation(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	mv, err := h.uc.DeleteModelVariation(ctx, id)
	if err != nil {
		writeModelErr(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(mv)
}
