// backend\internal\adapters\in\http\console\handler\invoice_handler.go
package consoleHandler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	invoicedom "narratives/internal/domain/invoice"
)

// InvoiceHandler handles /invoices endpoints.
// - POST /invoices
// - GET  /invoices/{orderId}
// - GET  /invoices?orderId=...
type InvoiceHandler struct {
	uc *usecase.InvoiceUsecase
}

func NewInvoiceHandler(uc *usecase.InvoiceUsecase) http.Handler {
	return &InvoiceHandler{uc: uc}
}

func (h *InvoiceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	path := strings.TrimSpace(r.URL.Path)

	// /invoices or /invoices/
	if path == "/invoices" || path == "/invoices/" {
		switch r.Method {
		case http.MethodPost:
			h.post(w, r)
			return
		case http.MethodGet:
			orderID := strings.TrimSpace(r.URL.Query().Get("orderId"))
			if orderID == "" {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "orderId is required"})
				return
			}
			h.getByOrderID(w, r, orderID)
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
			return
		}
	}

	// GET /invoices/{orderId}
	if strings.HasPrefix(path, "/invoices/") {
		orderID := strings.TrimPrefix(path, "/invoices/")
		orderID = strings.TrimSpace(orderID)
		if orderID == "" || strings.Contains(orderID, "/") {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid orderId"})
			return
		}

		switch r.Method {
		case http.MethodGet:
			h.getByOrderID(w, r, orderID)
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
}

// ------------------------------------------------------------
// Requests
// ------------------------------------------------------------

type createInvoiceRequest struct {
	OrderID string `json:"orderId"`
	Prices  []int  `json:"prices"`

	Tax         int `json:"tax"`
	ShippingFee int `json:"shippingFee"`

	// NOTE:
	// paid は将来互換のため受け取るが、起票では常に paid=false とする。
	// 支払い確定（paid=true）は payment_handler.go から UpdatePaid を呼ぶ。
	Paid *bool `json:"paid"`
}

func (h *InvoiceHandler) post(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqID := _invPickReqID(r)

	const maxBodyLog = 4096
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[invoice] reqId=%s read_body_failed err=%v", reqID, err)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_body"})
		return
	}

	var req createInvoiceRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		log.Printf(
			"[invoice] reqId=%s invalid_json err=%v bodyLen=%d bodySample=%q",
			reqID, err, len(bodyBytes), _invSampleBytes(bodyBytes, maxBodyLog),
		)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_json"})
		return
	}

	orderID := strings.TrimSpace(req.OrderID)
	if orderID == "" {
		log.Printf("[invoice] reqId=%s bad_request reason=missing_orderId", reqID)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "orderId is required"})
		return
	}
	if len(req.Prices) == 0 {
		log.Printf("[invoice] reqId=%s bad_request reason=missing_prices orderId=%q", reqID, _invMaskID(orderID))
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "prices is required"})
		return
	}

	// ✅ paid は起票では無視（常に false）
	if req.Paid != nil {
		log.Printf("[invoice] reqId=%s note=paid_ignored_on_create orderId=%q paid_in=%t",
			reqID, _invMaskID(orderID), *req.Paid,
		)
	}

	in := usecase.CreateInvoiceInput{
		OrderID:     orderID,
		Prices:      req.Prices,
		Tax:         req.Tax,
		ShippingFee: req.ShippingFee,
	}

	out, err := h.uc.Create(ctx, in)
	if err != nil {
		log.Printf("[invoice] reqId=%s create_failed err=%v orderId=%q prices=%d", reqID, err, _invMaskID(orderID), len(req.Prices))
		writeInvoiceErr(w, err)
		return
	}

	log.Printf("[invoice] reqId=%s created ok orderId=%q paid=%t", reqID, _invMaskID(out.OrderID), out.Paid)
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(out)
}

func (h *InvoiceHandler) getByOrderID(w http.ResponseWriter, r *http.Request, orderID string) {
	ctx := r.Context()
	reqID := _invPickReqID(r)

	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "orderId is required"})
		return
	}

	out, err := h.uc.GetByOrderID(ctx, orderID)
	if err != nil {
		log.Printf("[invoice] reqId=%s get_failed orderId=%q err=%v", reqID, _invMaskID(orderID), err)
		writeInvoiceErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(out)
}

// ------------------------------------------------------------
// Error mapping
// ------------------------------------------------------------

func writeInvoiceErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	msg := strings.ToLower(strings.TrimSpace(err.Error()))

	switch {
	case errors.Is(err, context.Canceled):
		code = 499
	case errors.Is(err, invoicedom.ErrNotFound) || msg == "not_found" || strings.Contains(msg, "not found") || strings.Contains(msg, "not_found"):
		code = http.StatusNotFound
	case errors.Is(err, invoicedom.ErrConflict) || strings.Contains(msg, "conflict") || strings.Contains(msg, "already exists"):
		code = http.StatusConflict
	case strings.Contains(msg, "invalid") || strings.Contains(msg, "required") || strings.Contains(msg, "missing"):
		code = http.StatusBadRequest
	default:
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

// ------------------------------------------------------------
// helpers (avoid name collisions in handlers package)
// ------------------------------------------------------------

func _invPickReqID(r *http.Request) string {
	if r == nil {
		return "-"
	}
	for _, k := range []string{"X-Request-Id", "X-Cloud-Trace-Context", "X-Amzn-Trace-Id"} {
		if v := strings.TrimSpace(r.Header.Get(k)); v != "" {
			return v
		}
	}
	return "-"
}

func _invSampleBytes(b []byte, limit int) string {
	if limit <= 0 || len(b) == 0 {
		return ""
	}
	if len(b) <= limit {
		return string(b)
	}
	return string(b[:limit]) + "...(truncated)"
}

func _invMaskID(s string) string {
	t := strings.TrimSpace(s)
	if t == "" {
		return ""
	}
	if len(t) <= 10 {
		return t
	}
	return t[:4] + "***" + t[len(t)-4:]
}
