package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	badom "narratives/internal/domain/billingAddress"
)

// BillingAddressHandler は /billing-addresses 関連のエンドポイントを担当します（単一取得のみ）。
type BillingAddressHandler struct {
	getUC *usecase.BillingAddressUsecase
}

// NewBillingAddressHandler はHTTPハンドラを初期化します。
func NewBillingAddressHandler(getUC *usecase.BillingAddressUsecase) http.Handler {
	return &BillingAddressHandler{getUC: getUC}
}

// ServeHTTP はHTTPルーティングの入口です。
func (h *BillingAddressHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/billing-addresses/"):
		id := strings.TrimPrefix(r.URL.Path, "/billing-addresses/")
		h.get(w, r, id)
	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// GET /billing-addresses/{id}
func (h *BillingAddressHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	addr, err := h.getUC.GetByID(ctx, id)
	if err != nil {
		writeBillingAddressErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(addr)
}

// エラーハンドリング
func writeBillingAddressErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	switch err {
	case badom.ErrInvalidID:
		code = http.StatusBadRequest
	case badom.ErrNotFound:
		code = http.StatusNotFound
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
