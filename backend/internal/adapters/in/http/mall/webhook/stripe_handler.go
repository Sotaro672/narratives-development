// backend/internal/adapters/in/http/mall/webhook/stripe_handler.go
package mallHandler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	usecase "narratives/internal/application/usecase"
	refunddom "narratives/internal/domain/refund"
)

const stripeWebhookMaxBodyBytes int64 = 1 << 20 // 1 MiB

type StripeWebhookHandler struct {
	paymentUC                      *usecase.PaymentUsecase
	orderUC                        *usecase.OrderUsecase
	settlementUC                   *usecase.SettlementUsecase
	refundUC                       *usecase.RefundUsecase
	itemRefundUC                   *usecase.ItemRefundUsecase
	refundRepo                     refunddom.RepositoryPort
	refundCompletionNotificationUC usecase.RefundCompletionNotificationUsecasePort
	signingSecret                  string
	tolerance                      time.Duration
	now                            func() time.Time
}

func NewStripeWebhookHandler(
	paymentUC *usecase.PaymentUsecase,
	orderUC *usecase.OrderUsecase,
	settlementUC *usecase.SettlementUsecase,
	refundUC *usecase.RefundUsecase,
	itemRefundUC *usecase.ItemRefundUsecase,
	refundRepo refunddom.RepositoryPort,
	refundCompletionNotificationUC usecase.RefundCompletionNotificationUsecasePort,
	signingSecret string,
) http.Handler {
	return &StripeWebhookHandler{
		paymentUC:                      paymentUC,
		orderUC:                        orderUC,
		settlementUC:                   settlementUC,
		refundUC:                       refundUC,
		itemRefundUC:                   itemRefundUC,
		refundRepo:                     refundRepo,
		refundCompletionNotificationUC: refundCompletionNotificationUC,
		signingSecret:                  signingSecret,
		tolerance:                      5 * time.Minute,
		now:                            time.Now,
	}
}

func (h *StripeWebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		writeStripeWebhookJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
		return
	}
	if h == nil || h.paymentUC == nil {
		writeStripeWebhookJSON(w, http.StatusInternalServerError, map[string]string{"error": "payment_usecase_not_initialized"})
		return
	}

	secret := strings.TrimSpace(h.signingSecret)
	if secret == "" {
		writeStripeWebhookJSON(w, http.StatusNotImplemented, map[string]string{"error": "stripe_webhook_secret_not_configured"})
		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, stripeWebhookMaxBodyBytes))
	if err != nil {
		writeStripeWebhookJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_body"})
		return
	}

	signatureHeader := strings.TrimSpace(r.Header.Get("Stripe-Signature"))
	if signatureHeader == "" {
		writeStripeWebhookJSON(w, http.StatusBadRequest, map[string]string{"error": "missing_stripe_signature"})
		return
	}

	now := time.Now().UTC()
	if h.now != nil {
		now = h.now().UTC()
	}
	if err := verifyStripeSignature(signatureHeader, body, secret, now, h.tolerance); err != nil {
		writeStripeWebhookJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_signature"})
		return
	}

	var event stripeEvent
	if err := json.Unmarshal(body, &event); err != nil {
		writeStripeWebhookJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	refundInput, refundSupported, err := extractStripeRefundEventInput(event)
	if err != nil {
		writeStripeWebhookJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_stripe_refund_event"})
		return
	}
	if refundSupported {
		if err := h.handleStripeRefundEvent(r.Context(), refundInput); err != nil {
			writeStripeWebhookJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
			return
		}
		writeStripeWebhookJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}

	input, supported, err := extractStripePaymentEventInput(event)
	if err != nil {
		writeStripeWebhookJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_stripe_event"})
		return
	}
	if !supported {
		writeStripeWebhookJSON(w, http.StatusOK, map[string]string{"status": "ignored"})
		return
	}

	if err := h.handleStripePaymentEvent(r.Context(), input); err != nil {
		switch {
		case errors.Is(err, errStripeWebhookOrderUsecaseNotInitialized):
			writeStripeWebhookJSON(w, http.StatusInternalServerError, map[string]string{"error": "order_usecase_not_initialized"})
		case errors.Is(err, errStripeWebhookSettlementUsecaseNotInitialized):
			writeStripeWebhookJSON(w, http.StatusInternalServerError, map[string]string{"error": "settlement_usecase_not_initialized"})
		default:
			writeStripeWebhookJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		}
		return
	}

	writeStripeWebhookJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *StripeWebhookHandler) handleStripeRefundEvent(ctx context.Context, in stripeRefundEventInput) error {
	if in.RefundID != "" {
		if err := h.applyStripeItemRefundEvent(ctx, in); err != nil {
			return err
		}
		return h.completeSucceededItemRefund(ctx, in.RefundID)
	}

	if err := h.applyStripeRefundEvent(ctx, in); err != nil {
		return err
	}
	return h.completeSucceededRefund(ctx, in.PaymentID)
}

func writeStripeWebhookJSON(w http.ResponseWriter, statusCode int, value any) {
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(value)
}
