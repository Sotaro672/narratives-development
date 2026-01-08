// backend/internal/adapters/in/http/mall/handler/payment_handler.go
package mallHandler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"

	// ✅ buyer auth context (uid)
	"narratives/internal/adapters/in/http/middleware"

	// keep for sentinel check if available
	mallquery "narratives/internal/application/query/mall"

	usecase "narratives/internal/application/usecase"
	paymentdom "narratives/internal/domain/payment"
)

// PaymentHandler handles:
// - GET /payments/{id} (existing)
// - GET /mall/payment  (resolve uid -> avatarId + addresses)
type PaymentHandler struct {
	uc     *usecase.PaymentUsecase
	orderQ any // ✅ accept any (mall) and call ResolveByUID via reflection
}

// NewPaymentHandler initializes handler (existing behavior only).
func NewPaymentHandler(uc *usecase.PaymentUsecase) http.Handler {
	return &PaymentHandler{uc: uc, orderQ: nil}
}

// ✅ NEW: inject order query (for /mall/payment).
func NewPaymentHandlerWithOrderQuery(uc *usecase.PaymentUsecase, orderQ any) http.Handler {
	return &PaymentHandler{uc: uc, orderQ: orderQ}
}

// ServeHTTP routes requests.
func (h *PaymentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// ✅ Allow CORS preflight
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// normalize path (drop trailing slash)
	path0 := strings.TrimSuffix(r.URL.Path, "/")

	// ✅ support /mall/*
	// - /mall/payment     -> /payment
	// - /mall/payments/xx -> /payments/xx
	if strings.HasPrefix(path0, "/mall/") {
		path0 = strings.TrimPrefix(path0, "/mall")
		if path0 == "" {
			path0 = "/"
		}
	}

	switch {
	// ✅ NEW: GET /mall/payment  (normalized to /payment)
	case r.Method == http.MethodGet && path0 == "/payment":
		h.getPaymentContext(w, r)
		return

	// existing: GET /payments/{id}
	case r.Method == http.MethodGet && strings.HasPrefix(path0, "/payments/"):
		id := strings.TrimPrefix(path0, "/payments/")
		h.get(w, r, id)
		return

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}
}

// ------------------------------------------------------------
// NEW: GET /mall/payment  (uid -> avatarId + shipping/billing)
// ------------------------------------------------------------
func (h *PaymentHandler) getPaymentContext(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.orderQ == nil {
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "order_query_not_initialized"})
		return
	}

	uid, ok := middleware.CurrentUserUID(r)
	if !ok || strings.TrimSpace(uid) == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	ctx := r.Context()

	out, err := callResolveByUID(h.orderQ, ctx, uid)
	if err != nil {
		// best-effort not found mapping
		if errors.Is(err, mallquery.ErrNotFound) || isNotFoundLike(err) {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal_error"})
		return
	}

	_ = json.NewEncoder(w).Encode(out)
}

func callResolveByUID(orderQ any, ctx context.Context, uid string) (any, error) {
	if orderQ == nil {
		return nil, errors.New("order_query_not_initialized")
	}

	rv := reflect.ValueOf(orderQ)
	if !rv.IsValid() {
		return nil, errors.New("order_query_not_initialized")
	}

	// ResolveByUID(ctx, uid)
	m := rv.MethodByName("ResolveByUID")
	if !m.IsValid() {
		// try pointer receiver
		if rv.Kind() != reflect.Pointer && rv.CanAddr() {
			m = rv.Addr().MethodByName("ResolveByUID")
		}
	}
	if !m.IsValid() {
		return nil, errors.New("order_query_missing_method_ResolveByUID")
	}

	// arg check
	if m.Type().NumIn() != 2 {
		return nil, errors.New("order_query_invalid_signature")
	}

	outs := m.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(uid)})
	if len(outs) != 2 {
		return nil, errors.New("order_query_invalid_signature")
	}

	var err error
	if !outs[1].IsNil() {
		if e, ok := outs[1].Interface().(error); ok {
			err = e
		} else {
			err = errors.New("order_query_returned_non_error")
		}
	}

	return outs[0].Interface(), err
}

func isNotFoundLike(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(msg, "not_found") ||
		strings.Contains(msg, "not found") ||
		strings.Contains(msg, "404")
}

// ------------------------------------------------------------
// existing: GET /payments/{id}
// ------------------------------------------------------------
func (h *PaymentHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "payment_usecase_not_initialized"})
		return
	}

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

// error handling (existing)
func writePaymentErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	switch err {
	case paymentdom.ErrInvalidInvoiceID:
		code = http.StatusBadRequest
	case paymentdom.ErrNotFound:
		code = http.StatusNotFound
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
