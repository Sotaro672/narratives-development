package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	shadom "narratives/internal/domain/shippingAddress"
)

// ShippingAddressHandler は /shipping-addresses 関連のエンドポイントを担当します（単一取得のみ）。
type ShippingAddressHandler struct {
	uc *usecase.ShippingAddressUsecase
}

// NewShippingAddressHandler はHTTPハンドラを初期化します。
func NewShippingAddressHandler(uc *usecase.ShippingAddressUsecase) http.Handler {
	return &ShippingAddressHandler{uc: uc}
}

// ServeHTTP はHTTPルーティングの入口です。
func (h *ShippingAddressHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/shipping-addresses/"):
		id := strings.TrimPrefix(r.URL.Path, "/shipping-addresses/")
		h.get(w, r, id)
	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// GET /shipping-addresses/{id}
func (h *ShippingAddressHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	addr, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeShippingAddressErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(addr)
}

// エラーハンドリング
func writeShippingAddressErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	switch err {
	case shadom.ErrInvalidID:
		code = http.StatusBadRequest
	case shadom.ErrNotFound:
		code = http.StatusNotFound
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
