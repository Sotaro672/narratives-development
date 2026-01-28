// backend/internal/adapters/in/http/console/handler/model/get_variation.go
package model

import (
	"encoding/json"
	"net/http"
	"strings"
)

// ------------------------------------------------------------
// GET /models/variations/{variationId}
// ------------------------------------------------------------
func (h *ModelHandler) getVariationByID(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	// ★ 期待値: modelId (= variationId) から modelNumber/size/color を解決する
	mv, err := h.uc.GetModelVariationByID(ctx, id)
	if err != nil {
		writeModelErr(w, err)
		return
	}

	// mv は *modeldom.ModelVariation なので、DTO 変換関数へは値で渡す
	_ = json.NewEncoder(w).Encode(toModelVariationDTO(*mv))
}
