package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	invoicedom "narratives/internal/domain/invoice"
)

// InvoiceHandler は /invoices 関連のエンドポイントを担当します（単一取得のみ）。
type InvoiceHandler struct {
	uc *usecase.InvoiceUsecase
}

// NewInvoiceHandler はHTTPハンドラを初期化します。
func NewInvoiceHandler(uc *usecase.InvoiceUsecase) http.Handler {
	return &InvoiceHandler{uc: uc}
}

// ServeHTTP はHTTPルーティングの入口です。
func (h *InvoiceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/invoices/"):
		id := strings.TrimPrefix(r.URL.Path, "/invoices/")
		h.get(w, r, id)
	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// GET /invoices/{id}
func (h *InvoiceHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	invoice, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeInvoiceErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(invoice)
}

// エラーハンドリング
func writeInvoiceErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	switch err {
	case invoicedom.ErrInvalidID:
		code = http.StatusBadRequest
	case invoicedom.ErrNotFound:
		code = http.StatusNotFound
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
