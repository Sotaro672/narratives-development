// backend\internal\adapters\in\http\handlers\brand_handler.go
package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	branddom "narratives/internal/domain/brand"
)

// BrandHandler は /brands 関連のエンドポイントを担当します（単一取得のみ）。
type BrandHandler struct {
	uc *usecase.BrandUsecase
}

// NewBrandHandler はHTTPハンドラを初期化します。
func NewBrandHandler(uc *usecase.BrandUsecase) http.Handler {
	return &BrandHandler{uc: uc}
}

// ServeHTTP はHTTPルーティングの入口です。
func (h *BrandHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/brands/"):
		id := strings.TrimPrefix(r.URL.Path, "/brands/")
		h.get(w, r, id)
	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// GET /brands/{id}
func (h *BrandHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	brand, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeBrandErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(brand)
}

// エラーハンドリング
func writeBrandErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	switch err {
	case branddom.ErrInvalidID:
		code = http.StatusBadRequest
	case branddom.ErrNotFound:
		code = http.StatusNotFound
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
