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

// InvoiceHandler handles /mall/me/invoices endpoints (A案).
//
// Routes:
// - POST /mall/me/invoices              : creates invoice + triggers mock payment webhook (A flow)
// - GET  /mall/me/invoices/{orderId}    : get invoice by orderId
// - GET  /mall/me/invoices?orderId=...  : get invoice by orderId
//
// A flow:
//
//	invoice -> (self) /mall/webhooks/stripe -> payment -> invoice.paid=true
//	- POST uses CheckoutUsecase (orchestration)
//	- GET uses InvoiceUsecase (read)
type InvoiceHandler struct {
	invoiceUC  *usecase.InvoiceUsecase
	checkoutUC *usecase.CheckoutUsecase
}

func NewInvoiceHandler(invoiceUC *usecase.InvoiceUsecase, checkoutUC *usecase.CheckoutUsecase) http.Handler {
	return &InvoiceHandler{
		invoiceUC:  invoiceUC,
		checkoutUC: checkoutUC,
	}
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
	// checkoutUC は POST でのみ必須。GET は invoiceUC のみで動く。

	path := strings.TrimSpace(r.URL.Path)
	path = strings.TrimSuffix(path, "/")

	const base = "/mall/me/invoices"

	// /mall/me/invoices
	if path == base {
		switch r.Method {
		case http.MethodPost:
			if h.checkoutUC == nil {
				// A: SELF_BASE_URL 未設定などで CheckoutUC が無効化されているケース
				_mInvWriteJSONError(w, http.StatusServiceUnavailable, "checkout is disabled (SELF_BASE_URL not configured)")
				return
			}
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

	// /mall/me/invoices/{orderId}
	if strings.HasPrefix(path, base+"/") {
		orderID := strings.TrimSpace(strings.TrimPrefix(path, base+"/"))
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

	// ✅ A: webhook を叩くために必須
	BillingAddressID string `json:"billingAddressId"`

	// optional (dev/test)
	Amount *int `json:"amount"`

	// NOTE:
	// paid は将来互換のため受け取るが、起票では常に paid=false。
	// 支払い確定（paid=true）は webhook -> payment 起票 -> PaymentUsecase が invoiceRepo 経由で立てる。
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
	if len(req.Prices) == 0 {
		log.Printf("[mall/invoice] reqId=%s bad_request reason=missing_prices orderId=%q", reqID, _mInvMaskID(orderID))
		_mInvWriteJSONError(w, http.StatusBadRequest, "prices is required")
		return
	}

	billingAddrID := strings.TrimSpace(req.BillingAddressID)
	if billingAddrID == "" {
		log.Printf("[mall/invoice] reqId=%s bad_request reason=missing_billingAddressId orderId=%q", reqID, _mInvMaskID(orderID))
		_mInvWriteJSONError(w, http.StatusBadRequest, "billingAddressId is required")
		return
	}

	// ✅ paid は起票では無視（常に false）
	if req.Paid != nil {
		log.Printf("[mall/invoice] reqId=%s note=paid_ignored_on_create orderId=%q paid_in=%t",
			reqID, _mInvMaskID(orderID), *req.Paid,
		)
	}

	out, cErr := h.checkoutUC.CreateInvoiceAndTriggerPayment(ctx, usecase.CreateInvoiceAndTriggerPaymentInput{
		OrderID:          orderID,
		Prices:           req.Prices,
		Tax:              req.Tax,
		ShippingFee:      req.ShippingFee,
		BillingAddressID: billingAddrID,
		Amount:           req.Amount,
	})
	if cErr != nil {
		// 重要：invoice は作れている可能性がある（Aの設計）
		// その場合は 201 で返しつつ warning を付ける（クライアント運用が楽）
		if strings.TrimSpace(out.OrderID) != "" {
			log.Printf("[mall/invoice] reqId=%s created_with_warning orderId=%q warn=%v",
				reqID, _mInvMaskID(out.OrderID), cErr,
			)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"invoice": out,
				"warning": cErr.Error(),
			})
			return
		}

		log.Printf("[mall/invoice] reqId=%s create_failed err=%v orderId=%q prices=%d", reqID, cErr, _mInvMaskID(orderID), len(req.Prices))
		_mInvWriteInvoiceErr(w, cErr)
		return
	}

	log.Printf("[mall/invoice] reqId=%s created ok orderId=%q paid=%t (payment will mark paid via webhook)",
		reqID, _mInvMaskID(out.OrderID), out.Paid,
	)
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
		// checkout: webhook trigger failed -> 500
	}

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
