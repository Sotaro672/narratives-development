// backend/internal/adapters/in/http/handlers/payment_handler.go
package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	// ✅ buyer auth context (uid)
	"narratives/internal/adapters/in/http/middleware"

	// ✅ uid -> avatarId + addresses
	snsquery "narratives/internal/application/query/sns"

	usecase "narratives/internal/application/usecase"
	paymentdom "narratives/internal/domain/payment"
)

// PaymentHandler handles:
// - GET /payments/{id} (existing)
// - GET /sns/payment   (NEW: resolve uid -> avatarId + addresses)
type PaymentHandler struct {
	uc     *usecase.PaymentUsecase
	orderQ *snsquery.SNSOrderQuery
}

// NewPaymentHandler initializes handler (existing behavior only).
func NewPaymentHandler(uc *usecase.PaymentUsecase) http.Handler {
	return &PaymentHandler{uc: uc, orderQ: nil}
}

// ✅ NEW: inject order query (for /sns/payment).
func NewPaymentHandlerWithOrderQuery(uc *usecase.PaymentUsecase, orderQ *snsquery.SNSOrderQuery) http.Handler {
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

	// ✅ support /sns/*
	// - /sns/payment     -> /payment
	// - /sns/payments/xx -> /payments/xx
	if strings.HasPrefix(path0, "/sns/") {
		path0 = strings.TrimPrefix(path0, "/sns")
		if path0 == "" {
			path0 = "/"
		}
	}

	switch {
	// ✅ NEW: GET /sns/payment  (normalized to /payment)
	case r.Method == http.MethodGet && path0 == "/payment":
		h.getPaymentContext(w, r)

	// existing: GET /payments/{id}
	case r.Method == http.MethodGet && strings.HasPrefix(path0, "/payments/"):
		id := strings.TrimPrefix(path0, "/payments/")
		h.get(w, r, id)

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// ------------------------------------------------------------
// NEW: GET /sns/payment  (uid -> avatarId + shipping/billing)
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

	out, err := h.orderQ.ResolveByUID(ctx, uid)
	if err != nil {
		// uid に紐づく avatar が無い / まだ onboarding 未完了
		if errors.Is(err, snsquery.ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
			return
		}

		log.Printf("[payment_handler] GET /sns/payment error uid=%q err=%v", maskUID(uid), err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal_error"})
		return
	}

	_ = json.NewEncoder(w).Encode(out)
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
	case paymentdom.ErrInvalidID:
		code = http.StatusBadRequest
	case paymentdom.ErrNotFound:
		code = http.StatusNotFound
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

// avoid logging raw uid
func maskUID(uid string) string {
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return ""
	}
	if len(uid) <= 6 {
		return "***"
	}
	return uid[:3] + "***" + uid[len(uid)-3:]
}
