package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	pbdom "narratives/internal/domain/productBlueprint"
)

// ProductBlueprintHandler は /product-blueprints 関連のエンドポイントを担当します（単一取得のみ）。
type ProductBlueprintHandler struct {
	uc *usecase.ProductBlueprintUsecase
}

// NewProductBlueprintHandler はHTTPハンドラを初期化します。
func NewProductBlueprintHandler(uc *usecase.ProductBlueprintUsecase) http.Handler {
	return &ProductBlueprintHandler{uc: uc}
}

// ServeHTTP はHTTPルーティングの入口です。
func (h *ProductBlueprintHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/product-blueprints/"):
		id := strings.TrimPrefix(r.URL.Path, "/product-blueprints/")
		h.get(w, r, id)
	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// GET /product-blueprints/{id}
func (h *ProductBlueprintHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	pb, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeProductBlueprintErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(pb)
}

// エラーハンドリング
func writeProductBlueprintErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	// NotFound の有無が不明なため InvalidID のみ特別扱い
	if err == pbdom.ErrInvalidID {
		code = http.StatusBadRequest
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
