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
	paymentdom "narratives/internal/domain/payment"
)

type PaymentHandler struct {
	uc     *usecase.PaymentUsecase
	flowUC *usecase.PaymentFlowUsecase
	orderQ OrderQuery
}

// OrderQuery is the typed contract PaymentHandler needs.
type OrderQuery interface {
	GetOrderContextByUID(
		ctx context.Context,
		uid string,
	) (dto.OrderContextDTO, error)
}

func NewPaymentHandler(
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

func (h *PaymentHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
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
	case r.Method == http.MethodGet &&
		path0 == "/me/payments":
		h.getPaymentsContext(w, r)
		return

	case r.Method == http.MethodPost &&
		path0 == "/me/payments":
		h.postPayments(w, r)
		return

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(
			map[string]string{
				"error": "not_found",
			},
		)
		return
	}
}

// ------------------------------------------------------------
// GET /me/payments
// ------------------------------------------------------------

func (h *PaymentHandler) getPaymentsContext(
	w http.ResponseWriter,
	r *http.Request,
) {
	if h == nil || h.orderQ == nil {
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(
			map[string]string{
				"error": "order_query_not_initialized",
			},
		)
		return
	}

	uid, ok := middleware.CurrentUserUID(r)
	if !ok || strings.TrimSpace(uid) == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(
			map[string]string{
				"error": "unauthorized",
			},
		)
		return
	}

	uid = strings.TrimSpace(uid)

	out, err := h.orderQ.GetOrderContextByUID(
		r.Context(),
		uid,
	)
	if err != nil {
		if errors.Is(err, mallquery.ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(
				map[string]string{
					"error": "not_found",
				},
			)
			return
		}

		log.Printf(
			"[mall/payments] GET /me/payments failed uid=%q err=%v",
			uid,
			err,
		)

		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(
			map[string]any{
				"error":  "internal_error",
				"detail": err.Error(),
			},
		)
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

	// Amount is a pointer so that an omitted value can be distinguished
	// from zero. Zero is valid because payment.MinAmount is zero.
	Amount *int `json:"amount"`

	PaymentMethodID string `json:"paymentMethodId"`

	StripeCustomerID      string `json:"stripeCustomerId"`
	StripePaymentMethodID string `json:"stripePaymentMethodId"`

	// Server-managed value.
	// Create request must not set this field.
	StripePaymentIntentID string `json:"stripePaymentIntentId"`
}

func (h *PaymentHandler) postPayments(
	w http.ResponseWriter,
	r *http.Request,
) {
	if h == nil || h.flowUC == nil {
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(
			map[string]string{
				"error": "payment_flow_usecase_not_initialized",
			},
		)
		return
	}

	uid, ok := middleware.CurrentUserUID(r)
	if !ok || strings.TrimSpace(uid) == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(
			map[string]string{
				"error": "unauthorized",
			},
		)
		return
	}

	uid = strings.TrimSpace(uid)

	var req payCreateReq

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(
			map[string]any{
				"error":  "invalid_json",
				"detail": err.Error(),
			},
		)
		return
	}

	req.PaymentID = strings.TrimSpace(
		req.PaymentID,
	)
	req.PaymentMethodID = strings.TrimSpace(
		req.PaymentMethodID,
	)
	req.StripeCustomerID = strings.TrimSpace(
		req.StripeCustomerID,
	)
	req.StripePaymentMethodID = strings.TrimSpace(
		req.StripePaymentMethodID,
	)
	req.StripePaymentIntentID = strings.TrimSpace(
		req.StripePaymentIntentID,
	)

	if req.PaymentID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(
			map[string]string{
				"error": "paymentId_required",
			},
		)
		return
	}

	// Amount is required, but zero is valid.
	if req.Amount == nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(
			map[string]string{
				"error": "amount_required",
			},
		)
		return
	}

	if *req.Amount < paymentdom.MinAmount {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(
			map[string]string{
				"error": "amount_invalid",
			},
		)
		return
	}

	if paymentdom.MaxAmount > 0 &&
		*req.Amount > paymentdom.MaxAmount {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(
			map[string]string{
				"error": "amount_invalid",
			},
		)
		return
	}

	if req.PaymentMethodID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(
			map[string]string{
				"error": "paymentMethodId_required",
			},
		)
		return
	}

	if req.StripeCustomerID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(
			map[string]string{
				"error": "stripeCustomerId_required",
			},
		)
		return
	}

	if req.StripePaymentMethodID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(
			map[string]string{
				"error": "stripePaymentMethodId_required",
			},
		)
		return
	}

	if req.StripePaymentIntentID != "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(
			map[string]string{
				"error": "stripePaymentIntentId_server_managed",
			},
		)
		return
	}

	amount := *req.Amount

	result, err := h.flowUC.CreatePaymentAndStartWithResult(
		r.Context(),
		usecase.CreatePaymentAndStartInput{
			// UserIDはrequest bodyから受け取らず、
			// 認証middlewareが確定したUIDを使用する。
			UserID: uid,

			PaymentID: req.PaymentID,

			PaymentMethodID: req.PaymentMethodID,

			StripeCustomerID: req.StripeCustomerID,
			StripePaymentMethodID: req.
				StripePaymentMethodID,

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
			amount,
			err,
		)

		status := paymentFlowHTTPStatus(err)

		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(
			map[string]any{
				"error":  "payment_flow_failed",
				"detail": err.Error(),
			},
		)
		return
	}

	if result == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(
			map[string]string{
				"error": "payment_flow_result_empty",
			},
		)
		return
	}

	response := map[string]any{
		"payment": result.Payment,

		"paymentId": result.PaymentID,
		"status":    string(result.Status),

		"stripePaymentIntentId": result.
			StripePaymentIntentID,
		"clientSecret":   result.ClientSecret,
		"requiresAction": result.RequiresAction,
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

// paymentFlowHTTPStatus classifies errors only by their typed or
// sentinel identity. Error-message text must not affect the HTTP status.
//
// Wrapped errors remain classifiable when each layer wraps them with %w.
func paymentFlowHTTPStatus(err error) int {
	if err == nil {
		return http.StatusInternalServerError
	}

	switch {
	case errors.Is(
		err,
		usecase.ErrPaymentFlowOrderAlreadyPaid,
	),
		errors.Is(
			err,
			paymentdom.ErrConflict,
		):
		return http.StatusConflict

	case errors.Is(
		err,
		mallquery.ErrNotFound,
	),
		errors.Is(
			err,
			usecase.ErrPaymentFlowOrderNotFound,
		),
		errors.Is(
			err,
			paymentdom.ErrNotFound,
		):
		return http.StatusNotFound

	case errors.Is(
		err,
		usecase.ErrPaymentFlowUserIDEmpty,
	),
		errors.Is(
			err,
			usecase.ErrPaymentFlowPaymentIDEmpty,
		),
		errors.Is(
			err,
			usecase.ErrPaymentFlowPaymentMethodEmpty,
		),
		errors.Is(
			err,
			usecase.ErrPaymentFlowAmountInvalid,
		),
		errors.Is(
			err,
			usecase.ErrPaymentFlowOrderIDMismatch,
		),
		errors.Is(
			err,
			usecase.ErrPaymentFlowOrderOwnerMismatch,
		),
		errors.Is(
			err,
			usecase.ErrPaymentFlowOrderAmountInvalid,
		),
		errors.Is(
			err,
			usecase.ErrPaymentFlowOrderAmountMismatch,
		),
		errors.Is(
			err,
			usecase.ErrPaymentFlowStripeCustomerIDEmpty,
		),
		errors.Is(
			err,
			usecase.ErrPaymentFlowStripePaymentMethodIDEmpty,
		),
		errors.Is(
			err,
			usecase.ErrPaymentFlowStripePaymentIntentIDEmpty,
		),
		errors.Is(
			err,
			paymentdom.ErrInvalidPaymentID,
		),
		errors.Is(
			err,
			paymentdom.ErrInvalidPaymentMethodID,
		),
		errors.Is(
			err,
			paymentdom.ErrInvalidStripeCustomerID,
		),
		errors.Is(
			err,
			paymentdom.ErrInvalidStripePaymentMethod,
		),
		errors.Is(
			err,
			paymentdom.ErrInvalidStripePaymentIntent,
		),
		errors.Is(
			err,
			paymentdom.ErrInvalidAmount,
		),
		errors.Is(
			err,
			paymentdom.ErrInvalidStatus,
		),
		errors.Is(
			err,
			paymentdom.ErrInvalidErrorType,
		),
		errors.Is(
			err,
			paymentdom.ErrInvalidErrorCode,
		),
		errors.Is(
			err,
			paymentdom.ErrInvalidErrorMsg,
		):
		return http.StatusBadRequest

	default:
		return http.StatusInternalServerError
	}
}
