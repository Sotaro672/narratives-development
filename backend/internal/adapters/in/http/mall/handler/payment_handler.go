// backend/internal/adapters/in/http/mall/handler/payment_handler.go
package mallHandler

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"narratives/internal/adapters/in/http/middleware"
	mallquery "narratives/internal/application/query/mall"
	dto "narratives/internal/application/query/mall/dto"
	usecase "narratives/internal/application/usecase"
)

type PaymentHandler struct {
	flowUC *usecase.PaymentFlowUsecase
	orderQ OrderQuery
}

// OrderQuery is the typed contract PaymentHandler needs.
type OrderQuery interface {
	GetOrderContextByUID(ctx context.Context, uid string) (dto.OrderContextDTO, error)
}

func NewPaymentHandler(
	orderQ OrderQuery,
	flowUC *usecase.PaymentFlowUsecase,
) http.Handler {
	return &PaymentHandler{
		flowUC: flowUC,
		orderQ: orderQ,
	}
}

func (h *PaymentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodOptions {
		w.Header().Set("Allow", "GET, OPTIONS")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/")
	if path == "" {
		path = "/"
	}

	if strings.HasPrefix(path, "/mall/") {
		path = strings.TrimPrefix(path, "/mall")
		if path == "" {
			path = "/"
		}
	}

	switch {
	case r.Method == http.MethodGet && path == "/me/payments":
		h.getPaymentsContext(w, r)
		return

	case r.Method == http.MethodPost && path == "/me/payments":
		w.Header().Set("Allow", "GET, OPTIONS")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
			"error": "payment_creation_disabled",
		})
		return

	default:
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "not_found",
		})
		return
	}
}

// ------------------------------------------------------------
// GET /me/payments
// ------------------------------------------------------------

func (h *PaymentHandler) getPaymentsContext(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.orderQ == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]string{
			"error": "order_query_not_initialized",
		})
		return
	}

	uid, ok := middleware.CurrentUserUID(r)
	if !ok || strings.TrimSpace(uid) == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "unauthorized",
		})
		return
	}

	uid = strings.TrimSpace(uid)

	out, err := h.orderQ.GetOrderContextByUID(r.Context(), uid)
	if err != nil {
		if errors.Is(err, mallquery.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "not_found",
			})
			return
		}

		log.Printf(
			"[mall/payments] GET /me/payments failed uid=%q err=%v",
			uid,
			err,
		)

		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":  "internal_error",
			"detail": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, out)
}
