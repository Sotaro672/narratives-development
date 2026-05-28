// backend/internal/adapters/in/http/mall/handler/payment_handler.go
package mallHandler

import (
	"context"
	"encoding/json"
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
	uc     *usecase.PaymentUsecase
	flowUC *usecase.PaymentFlowUsecase
	orderQ OrderQuery
}

// OrderQuery is the typed contract PaymentHandler needs.
type OrderQuery interface {
	GetOrderContextByUID(ctx context.Context, uid string) (dto.OrderContextDTO, error)
}

func NewPaymentHandler(uc *usecase.PaymentUsecase) http.Handler {
	return &PaymentHandler{
		uc:     uc,
		flowUC: nil,
		orderQ: nil,
	}
}

func NewPaymentHandlerWithOrderQuery(
	uc *usecase.PaymentUsecase,
	orderQ OrderQuery,
) http.Handler {
	return &PaymentHandler{
		uc:     uc,
		flowUC: nil,
		orderQ: orderQ,
	}
}

// NewPaymentHandlerWithPaymentFlow is the preferred constructor for
// Stripe PaymentIntent based payment flow.
func NewPaymentHandlerWithPaymentFlow(
	uc *usecase.PaymentUsecase,
	flowUC *usecase.PaymentFlowUsecase,
) http.Handler {
	return &PaymentHandler{
		uc:     uc,
		flowUC: flowUC,
		orderQ: nil,
	}
}

// NewPaymentHandlerWithOrderQueryAndPaymentFlow is the preferred constructor
// when both GET /mall/me/payments and POST /mall/me/payments are enabled.
func NewPaymentHandlerWithOrderQueryAndPaymentFlow(
	uc *usecase.PaymentUsecase,
	orderQ OrderQuery,
	flowUC *usecase.PaymentFlowUsecase,
) http.Handler {
	return &PaymentHandler{
		uc:     uc,
		flowUC: flowUC,
		orderQ: orderQ,
	}
}

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
	case r.Method == http.MethodGet && path0 == "/me/payments":
		h.getPaymentsContext(w, r)
		return

	case r.Method == http.MethodPost && path0 == "/me/payments":
		h.postPayments(w, r)
		return

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}
}

// ------------------------------------------------------------
// GET /me/payments
// ------------------------------------------------------------

func (h *PaymentHandler) getPaymentsContext(w http.ResponseWriter, r *http.Request) {
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

	out, err := h.orderQ.GetOrderContextByUID(r.Context(), uid)
	if err != nil {
		if errors.Is(err, mallquery.ErrNotFound) || payIsNotFoundLike(err) {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
			return
		}

		log.Printf("[mall/payments] GET /me/payments failed uid=%q err=%v", uid, err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error":  "internal_error",
			"detail": err.Error(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(out)
}

// ------------------------------------------------------------
// POST /me/payments
// ------------------------------------------------------------

type payCreateReq struct {
	// PaymentID is the payment document ID.
	// It must be the same value as order.ID.
	PaymentID string `json:"paymentId"`

	Amount int `json:"amount"`

	PaymentMethodID string `json:"paymentMethodId"`

	StripeCustomerID      string `json:"stripeCustomerId"`
	StripePaymentMethodID string `json:"stripePaymentMethodId"`

	// Server-managed value.
	// Create request must not set this field.
	StripePaymentIntentID string `json:"stripePaymentIntentId"`
}

func (h *PaymentHandler) postPayments(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.flowUC == nil {
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "payment_flow_usecase_not_initialized"})
		return
	}

	uid, ok := middleware.CurrentUserUID(r)
	if !ok || uid == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	var req payCreateReq
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error":  "invalid_json",
			"detail": err.Error(),
		})
		return
	}

	req.PaymentID = strings.TrimSpace(req.PaymentID)
	req.PaymentMethodID = strings.TrimSpace(req.PaymentMethodID)
	req.StripeCustomerID = strings.TrimSpace(req.StripeCustomerID)
	req.StripePaymentMethodID = strings.TrimSpace(req.StripePaymentMethodID)
	req.StripePaymentIntentID = strings.TrimSpace(req.StripePaymentIntentID)

	if req.PaymentID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "paymentId_required"})
		return
	}

	if req.Amount <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "amount_invalid"})
		return
	}

	if req.PaymentMethodID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "paymentMethodId_required"})
		return
	}

	if req.StripeCustomerID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "stripeCustomerId_required"})
		return
	}

	if req.StripePaymentMethodID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "stripePaymentMethodId_required"})
		return
	}

	if req.StripePaymentIntentID != "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "stripePaymentIntentId_server_managed"})
		return
	}

	amount := req.Amount

	result, err := h.flowUC.CreatePaymentAndStartWithResult(
		r.Context(),
		usecase.CreatePaymentAndStartInput{
			PaymentID: req.PaymentID,

			PaymentMethodID: req.PaymentMethodID,

			StripeCustomerID:      req.StripeCustomerID,
			StripePaymentMethodID: req.StripePaymentMethodID,

			Amount: &amount,
		},
	)
	if err != nil {
		log.Printf(
			"[mall/payments] POST /me/payments failed uid=%q paymentId=%q paymentMethodId=%q stripeCustomerId=%q stripePaymentMethodId=%q amount=%d err=%v",
			uid,
			req.PaymentID,
			req.PaymentMethodID,
			req.StripeCustomerID,
			req.StripePaymentMethodID,
			req.Amount,
			err,
		)

		status := http.StatusInternalServerError
		if payIsBadRequestLike(err) {
			status = http.StatusBadRequest
		} else if errors.Is(err, mallquery.ErrNotFound) || payIsNotFoundLike(err) {
			status = http.StatusNotFound
		}

		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error":  "payment_flow_failed",
			"detail": err.Error(),
		})
		return
	}

	if result == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "payment_flow_result_empty"})
		return
	}

	response := map[string]any{
		"payment": result.Payment,

		"paymentId": result.PaymentID,
		"status":    string(result.Status),

		"stripePaymentIntentId": result.StripePaymentIntentID,
		"clientSecret":          result.ClientSecret,
		"requiresAction":        result.RequiresAction,
	}

	if result.ErrorType != nil {
		response["errorType"] = *result.ErrorType
	}
	if result.ErrorCode != nil {
		response["errorCode"] = *result.ErrorCode
	}
	if result.ErrorMessage != nil {
		response["errorMessage"] = *result.ErrorMessage
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(response)
}

// ------------------------------------------------------------
// helpers
// ------------------------------------------------------------

func payIsNotFoundLike(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())

	return strings.Contains(msg, "not found") ||
		strings.Contains(msg, "not_found") ||
		strings.Contains(msg, "404")
}

func payIsBadRequestLike(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())

	return strings.Contains(msg, "invalid") ||
		strings.Contains(msg, "required") ||
		strings.Contains(msg, "missing") ||
		strings.Contains(msg, "empty")
}
