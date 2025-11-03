package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	paymentdom "narratives/internal/domain/payment"
)

// PaymentHandler は /payments 関連のエンドポイントを担当します（単一取得のみ）。
type PaymentHandler struct {
	uc *usecase.PaymentUsecase
}

// NewPaymentHandler はHTTPハンドラを初期化します。
func NewPaymentHandler(uc *usecase.PaymentUsecase) http.Handler {
	return &PaymentHandler{uc: uc}
}

// ServeHTTP はHTTPルーティングの入口です。
func (h *PaymentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/payments/"):
		id := strings.TrimPrefix(r.URL.Path, "/payments/")
		h.get(w, r, id)
	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// GET /payments/{id}
func (h *PaymentHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	p, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writePaymentErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(p)
}

// エラーハンドリング
func writePaymentErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	switch err {
	case paymentdom.ErrInvalidID:
		code = http.StatusBadRequest
	case paymentdom.ErrNotFound:
		code = http.StatusNotFound
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
