// backend/internal/adapters/in/http/console/handler/model/routes.go
package model

import (
	"encoding/json"
	"net/http"
	"strings"
)

// ServeHTTP はHTTPルーティングの入口です。
func (h *ModelHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {

	// ------------------------------------------------------------
	// GET /models/variations/{variationId}
	//   → ModelUsecase.GetModelVariationByID
	// ※ mintRequest の「モデル別検査結果」(modelId=variationId) 用に追加
	// ------------------------------------------------------------
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/models/variations/"):
		if id, ok := extractSingleID(r.URL.Path, "/models/variations/"); ok {
			h.getVariationByID(w, r, id)
			return
		}
		writeNotFound(w)
		return

	// ------------------------------------------------------------
	// PUT /models/variations/{id}
	//   → 既存の PUT /models/{id} と同じ処理（互換エイリアス）
	// ------------------------------------------------------------
	case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/models/variations/"):
		if id, ok := extractSingleID(r.URL.Path, "/models/variations/"); ok {
			h.updateVariation(w, r, id)
			return
		}
		writeNotFound(w)
		return

	// ------------------------------------------------------------
	// DELETE /models/variations/{id}
	//   → 既存の DELETE /models/{id} と同じ処理（互換エイリアス）
	// ------------------------------------------------------------
	case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/models/variations/"):
		if id, ok := extractSingleID(r.URL.Path, "/models/variations/"); ok {
			h.deleteVariation(w, r, id)
			return
		}
		writeNotFound(w)
		return

	// ------------------------------------------------------------
	// GET /models/by-blueprint/{productBlueprintID}/variations
	//   → ModelUsecase.ListModelVariationsByProductBlueprintID
	// ------------------------------------------------------------
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/models/by-blueprint/"):
		if productBlueprintID, ok := extractBlueprintIDForList(r.URL.Path); ok {
			h.listVariationsByProductBlueprintID(w, r, productBlueprintID)
			return
		}
		writeNotFound(w)
		return

	// ------------------------------------------------------------
	// POST /models/{productBlueprintID}/variations
	//   → ModelUsecase.CreateModelVariation
	// ------------------------------------------------------------
	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/models/"):
		if productBlueprintID, ok := extractBlueprintIDForCreate(r.URL.Path); ok {
			h.createVariation(w, r, productBlueprintID)
			return
		}
		writeNotFound(w)
		return

	// ------------------------------------------------------------
	// PUT /models/{id}
	//   → ModelUsecase.UpdateModelVariation
	// ------------------------------------------------------------
	case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/models/"):
		if id, ok := extractModelID(r.URL.Path); ok {
			h.updateVariation(w, r, id)
			return
		}
		writeNotFound(w)
		return

	// ------------------------------------------------------------
	// DELETE /models/{id}
	//   → ModelUsecase.DeleteModelVariation
	// ------------------------------------------------------------
	case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/models/"):
		if id, ok := extractModelID(r.URL.Path); ok {
			h.deleteVariation(w, r, id)
			return
		}
		writeNotFound(w)
		return

	// ------------------------------------------------------------
	// GET /models/{id}
	//   → ModelUsecase.GetByID
	// ------------------------------------------------------------
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/models/"):
		// ✅ /models/variations/... は上の case で処理済みの想定だが、
		//   念のため誤ルーティングを防ぐ
		if id, ok := extractModelID(r.URL.Path); ok {
			h.get(w, r, id)
			return
		}
		writeNotFound(w)
		return

	default:
		writeNotFound(w)
		return
	}
}

func writeNotFound(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
}
