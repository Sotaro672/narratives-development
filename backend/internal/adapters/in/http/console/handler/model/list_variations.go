// backend/internal/adapters/in/http/console/handler/model/list_variations.go
package model

import (
	"encoding/json"
	"net/http"
	"strings"
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

	productBlueprintID = strings.TrimSpace(productBlueprintID)
	if productBlueprintID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid productBlueprintID"})
		return
	}

	vars, err := h.uc.ListModelVariationsByProductBlueprintID(ctx, productBlueprintID)
	if err != nil {
		writeModelErr(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(toModelVariationDTOs(vars))
}
