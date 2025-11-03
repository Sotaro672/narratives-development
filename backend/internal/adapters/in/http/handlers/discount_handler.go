package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	discountdom "narratives/internal/domain/discount"
)

// DiscountHandler は /discounts 関連のエンドポイントを担当します（単一取得のみ）。
type DiscountHandler struct {
	uc *usecase.DiscountUsecase
}

// NewDiscountHandler はHTTPハンドラを初期化します。
func NewDiscountHandler(uc *usecase.DiscountUsecase) http.Handler {
	return &DiscountHandler{uc: uc}
}

// ServeHTTP はHTTPルーティングの入口です。
func (h *DiscountHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/discounts/"):
		id := strings.TrimPrefix(r.URL.Path, "/discounts/")
		h.get(w, r, id)
	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// GET /discounts/{id}
func (h *DiscountHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	d, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeDiscountErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(d)
}

// エラーハンドリング
func writeDiscountErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	switch err {
	case discountdom.ErrInvalidID:
		code = http.StatusBadRequest
	case discountdom.ErrNotFound:
		code = http.StatusNotFound
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
