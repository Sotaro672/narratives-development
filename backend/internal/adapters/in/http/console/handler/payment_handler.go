// backend/internal/adapters/in/http/console/handler/payment_handler.go
package consoleHandler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	// buyer auth context (uid)
	"narratives/internal/adapters/in/http/middleware"

	// uid -> avatarId + addresses
	mallquery "narratives/internal/application/query/mall"

	usecase "narratives/internal/application/usecase"
	paymentdom "narratives/internal/domain/payment"
)

// PaymentHandler handles:
// - GET /payments/{id}
// - GET /mall/payment
type PaymentHandler struct {
	uc     *usecase.PaymentUsecase
	orderQ *mallquery.OrderQuery
}

// NewPaymentHandler initializes handler.
func NewPaymentHandler(uc *usecase.PaymentUsecase) http.Handler {
	return &PaymentHandler{uc: uc, orderQ: nil}
}

// NewPaymentHandlerWithOrderQuery injects order query for /mall/payment.
func NewPaymentHandlerWithOrderQuery(
	uc *usecase.PaymentUsecase,
	orderQ *mallquery.OrderQuery,
) http.Handler {
	return &PaymentHandler{uc: uc, orderQ: orderQ}
}

// ServeHTTP routes requests.
func (h *PaymentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	path0 := strings.TrimSuffix(r.URL.Path, "/")
	if path0 == "" {
		path0 = "/"
	}

	if strings.HasPrefix(path0, "/mall/") {
		path0 = strings.TrimPrefix(path0, "/mall")
		if path0 == "" {
			path0 = "/"
		}
	}

	switch {
	case r.Method == http.MethodGet && path0 == "/payment":
		h.getPaymentContext(w, r)

	case r.Method == http.MethodGet && strings.HasPrefix(path0, "/payments/"):
		id := strings.TrimPrefix(path0, "/payments/")
		h.get(w, r, id)

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// ------------------------------------------------------------
// GET /mall/payment
// ------------------------------------------------------------

func (h *PaymentHandler) getPaymentContext(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.orderQ == nil {
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "order_query_not_initialized"})
		return
	}

	uid, ok := middleware.CurrentUserUID(r)
	if !ok || uid == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	out, err := h.orderQ.ResolveByUID(r.Context(), uid)
	if err != nil {
		if errors.Is(err, mallquery.ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
			return
		}

		log.Printf("[payment_handler] GET /mall/payment error uid=%q err=%v", uid, err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal_error"})
		return
	}

	_ = json.NewEncoder(w).Encode(out)
}

// ------------------------------------------------------------
// GET /payments/{id}
//
// id is paymentId.
// paymentId must be the same value as order.ID.
// ------------------------------------------------------------

func (h *PaymentHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "payment_usecase_not_initialized"})
		return
	}

	paymentID := strings.Trim(id, " \t\r\n/")
	if paymentID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid paymentId"})
		return
	}

	p, err := h.uc.GetByID(r.Context(), paymentID)
	if err != nil {
		writePaymentErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(p)
}

func writePaymentErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	switch {
	case errors.Is(err, paymentdom.ErrInvalidPaymentID):
		code = http.StatusBadRequest
	case errors.Is(err, paymentdom.ErrNotFound):
		code = http.StatusNotFound
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
