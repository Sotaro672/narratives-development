// backend/internal/adapters/in/http/mall/handler/invoice_handler.go
package mallHandler

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

// InvoiceHandler handles /mall/me/invoices endpoints.
//
// ✅ 責務分離（採用方針）
// - POST /mall/me/invoices : invoice テーブル起票のみ（checkout/webhook はしない）
// - GET  /mall/me/invoices/{orderId} : get invoice by orderId
// - GET  /mall/me/invoices?orderId=... : get invoice by orderId
//
// NOTE:
// - payment 起票は /mall/me/payment 側の責務
// - SELF_BASE_URL / CheckoutUsecase は不要
type InvoiceHandler struct {
	invoiceUC *usecase.InvoiceUsecase
}

func NewInvoiceHandler(invoiceUC *usecase.InvoiceUsecase) http.Handler {
	return &InvoiceHandler{invoiceUC: invoiceUC}
}

func (h *InvoiceHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Preflight
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if h == nil || h.invoiceUC == nil {
		_mInvWriteJSONError(w, http.StatusInternalServerError, "invoice usecase is not configured")
		return
	}

	// normalize path (drop trailing slash)
	path0 := strings.TrimSpace(r.URL.Path)
	path0 = strings.TrimSuffix(path0, "/")
	if path0 == "" {
		path0 = "/"
	}

	// ✅ support /mall/* mounts:
	// - /mall/me/invoices -> /me/invoices
	// - if router already stripped "/mall", it may already be "/me/invoices"
	if strings.HasPrefix(path0, "/mall/") {
		path0 = strings.TrimPrefix(path0, "/mall")
		if path0 == "" {
			path0 = "/"
		}
	}

	const base = "/me/invoices"

	// /me/invoices
	if path0 == base {
		switch r.Method {
		case http.MethodPost:
			h.post(w, r)
			return

		case http.MethodGet:
			orderID := strings.TrimSpace(r.URL.Query().Get("orderId"))
			if orderID == "" {
				_mInvWriteJSONError(w, http.StatusBadRequest, "orderId is required")
				return
			}
			h.getByOrderID(w, r, orderID)
			return

		default:
			_mInvWriteJSONError(w, http.StatusNotFound, "not_found")
			return
		}
	}

	// /me/invoices/{orderId}
	if strings.HasPrefix(path0, base+"/") {
		orderID := strings.TrimSpace(strings.TrimPrefix(path0, base+"/"))
		if orderID == "" || strings.Contains(orderID, "/") {
			_mInvWriteJSONError(w, http.StatusBadRequest, "invalid orderId")
			return
		}

		switch r.Method {
		case http.MethodGet:
			h.getByOrderID(w, r, orderID)
			return
		default:
			_mInvWriteJSONError(w, http.StatusNotFound, "not_found")
			return
		}
	}

	_mInvWriteJSONError(w, http.StatusNotFound, "not_found")
}

// ------------------------------------------------------------
// Requests
// ------------------------------------------------------------

type createInvoiceRequest struct {
	OrderID string `json:"orderId"`
	Prices  []int  `json:"prices"`

	Tax         int `json:"tax"`
	ShippingFee int `json:"shippingFee"`

	// ✅ 互換のため受け取ってもよいが、invoice 起票の責務では使わない（payment 側で使う）
	BillingAddressID string `json:"billingAddressId"`

	// optional (dev/test)
	Amount *int `json:"amount"`

	// NOTE:
	// paid は将来互換のため受け取るが、起票では常に paid=false。
	Paid *bool `json:"paid"`
}

func (h *InvoiceHandler) post(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	reqID := _mInvPickReqID(r)

	const maxBodyLog = 4096
	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		log.Printf("[mall/invoice] reqId=%s read_body_failed err=%v", reqID, err)
		_mInvWriteJSONError(w, http.StatusBadRequest, "invalid_body")
		return
	}
	_ = r.Body.Close()

	var req createInvoiceRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		log.Printf(
			"[mall/invoice] reqId=%s invalid_json err=%v bodyLen=%d bodySample=%q",
			reqID, err, len(bodyBytes), _mInvSampleBytes(bodyBytes, maxBodyLog),
		)
		_mInvWriteJSONError(w, http.StatusBadRequest, "invalid_json")
		return
	}

	orderID := strings.TrimSpace(req.OrderID)
	if orderID == "" {
		log.Printf("[mall/invoice] reqId=%s bad_request reason=missing_orderId", reqID)
		_mInvWriteJSONError(w, http.StatusBadRequest, "orderId is required")
		return
	}

	// ✅ invoice 起票には prices が必須（order から再計算したい場合は InvoiceUsecase 側で吸収）
	if len(req.Prices) == 0 {
		log.Printf("[mall/invoice] reqId=%s bad_request reason=missing_prices orderId=%q", reqID, _mInvMaskID(orderID))
		_mInvWriteJSONError(w, http.StatusBadRequest, "prices is required")
		return
	}

	// ✅ paid は起票では無視（常に false）
	if req.Paid != nil {
		log.Printf("[mall/invoice] reqId=%s note=paid_ignored_on_create orderId=%q paid_in=%t",
			reqID, _mInvMaskID(orderID), *req.Paid,
		)
	}

	// ✅ billingAddressId は invoice 起票の責務では使わない（payment 側に渡す）
	if strings.TrimSpace(req.BillingAddressID) != "" {
		log.Printf("[mall/invoice] reqId=%s note=billingAddressId_ignored_on_invoice_create orderId=%q",
			reqID, _mInvMaskID(orderID),
		)
	}

	out, cErr := h.invoiceUC.Create(ctx, usecase.CreateInvoiceInput{
		OrderID:     orderID,
		Prices:      req.Prices,
		Tax:         req.Tax,
		ShippingFee: req.ShippingFee,
	})
	if cErr != nil {
		log.Printf("[mall/invoice] reqId=%s create_failed err=%v orderId=%q prices=%d", reqID, cErr, _mInvMaskID(orderID), len(req.Prices))
		_mInvWriteInvoiceErr(w, cErr)
		return
	}

	log.Printf("[mall/invoice] reqId=%s created ok orderId=%q paid=%t", reqID, _mInvMaskID(out.OrderID), out.Paid)
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(out)
}

func (h *InvoiceHandler) getByOrderID(w http.ResponseWriter, r *http.Request, orderID string) {
	ctx := r.Context()
	reqID := _mInvPickReqID(r)

	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		_mInvWriteJSONError(w, http.StatusBadRequest, "orderId is required")
		return
	}

	out, err := h.invoiceUC.GetByOrderID(ctx, orderID)
	if err != nil {
		log.Printf("[mall/invoice] reqId=%s get_failed orderId=%q err=%v", reqID, _mInvMaskID(orderID), err)
		_mInvWriteInvoiceErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(out)
}

// ------------------------------------------------------------
// Error mapping
// ------------------------------------------------------------

func _mInvWriteInvoiceErr(w http.ResponseWriter, err error) {
	if err == nil {
		_mInvWriteJSONError(w, http.StatusInternalServerError, "unknown_error")
		return
	}

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
		// keep 500
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func _mInvWriteJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// ------------------------------------------------------------
// helpers (avoid name collisions in handlers package)
// ------------------------------------------------------------

func _mInvPickReqID(r *http.Request) string {
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

func _mInvSampleBytes(b []byte, limit int) string {
	if limit <= 0 || len(b) == 0 {
		return ""
	}
	if len(b) <= limit {
		return string(b)
	}
	return string(b[:limit]) + "...(truncated)"
}

func _mInvMaskID(s string) string {
	t := strings.TrimSpace(s)
	if t == "" {
		return ""
	}
	if len(t) <= 10 {
		return t
	}
	return t[:4] + "***" + t[len(t)-4:]
}
