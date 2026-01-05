// backend\internal\adapters\in\http\sns\payment_handler.go
package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	// ✅ buyer auth context (uid)
	"narratives/internal/adapters/in/http/middleware"

	// ✅ uid -> avatarId + addresses
	snsquery "narratives/internal/application/query/mall"

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

	// ✅ DEBUG: entry log (confirms handler is reached)
	// NOTE: keep it short; auth token is truncated
	{
		auth := strings.TrimSpace(r.Header.Get("Authorization"))
		if len(auth) > 24 {
			auth = auth[:24] + "..."
		}
		log.Printf(
			"[sns_payment_handler] entry method=%s path=%q rawQuery=%q hasUC=%v hasOrderQ=%v auth=%q",
			r.Method,
			r.URL.Path,
			r.URL.RawQuery,
			h != nil && h.uc != nil,
			h != nil && h.orderQ != nil,
			auth,
		)
	}

	// ✅ Allow CORS preflight
	if r.Method == http.MethodOptions {
		log.Printf("[sns_payment_handler] preflight ok path=%q", r.URL.Path)
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

	// ✅ DEBUG: routing decision
	log.Printf("[sns_payment_handler] route_decision normalized=%q", path0)

	switch {
	// ✅ NEW: GET /sns/payment  (normalized to /payment)
	case r.Method == http.MethodGet && path0 == "/payment":
		log.Printf("[sns_payment_handler] hit GET /sns/payment (normalized=%q)", path0)
		h.getPaymentContext(w, r)
		return

	// existing: GET /payments/{id}
	case r.Method == http.MethodGet && strings.HasPrefix(path0, "/payments/"):
		id := strings.TrimPrefix(path0, "/payments/")
		log.Printf("[sns_payment_handler] hit GET /payments/{id} id=%q", strings.TrimSpace(id))
		h.get(w, r, id)
		return

	default:
		log.Printf("[sns_payment_handler] not_found method=%s normalized=%q", r.Method, path0)
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}
}

// ------------------------------------------------------------
// NEW: GET /sns/payment  (uid -> avatarId + shipping/billing)
// ------------------------------------------------------------
func (h *PaymentHandler) getPaymentContext(w http.ResponseWriter, r *http.Request) {
	log.Printf("[sns_payment_handler] getPaymentContext start")

	if h == nil || h.orderQ == nil {
		log.Printf("[sns_payment_handler] orderQ is nil -> 501")
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "order_query_not_initialized"})
		return
	}

	uid, ok := middleware.CurrentUserUID(r)
	log.Printf("[sns_payment_handler] uid_from_context ok=%v uid=%q", ok, maskUID(uid))

	if !ok || strings.TrimSpace(uid) == "" {
		log.Printf("[sns_payment_handler] unauthorized (no uid) -> 401")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	ctx := r.Context()

	out, err := h.orderQ.ResolveByUID(ctx, uid)
	if err != nil {
		// uid に紐づく avatar が無い / まだ onboarding 未完了
		if errors.Is(err, snsquery.ErrNotFound) {
			log.Printf("[sns_payment_handler] ResolveByUID not found uid=%q -> 404", maskUID(uid))
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
			return
		}

		log.Printf("[sns_payment_handler] ResolveByUID error uid=%q err=%v -> 500", maskUID(uid), err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal_error"})
		return
	}

	log.Printf("[sns_payment_handler] ResolveByUID ok uid=%q -> 200", maskUID(uid))
	_ = json.NewEncoder(w).Encode(out)
}

// ------------------------------------------------------------
// existing: GET /payments/{id}
// ------------------------------------------------------------
func (h *PaymentHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	if h == nil || h.uc == nil {
		log.Printf("[sns_payment_handler] payment usecase nil -> 503")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "payment_usecase_not_initialized"})
		return
	}

	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		log.Printf("[sns_payment_handler] invalid id -> 400")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	p, err := h.uc.GetByID(ctx, id)
	if err != nil {
		log.Printf("[sns_payment_handler] GetByID error id=%q err=%v", id, err)
		writePaymentErr(w, err)
		return
	}
	log.Printf("[sns_payment_handler] GetByID ok id=%q", id)
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
