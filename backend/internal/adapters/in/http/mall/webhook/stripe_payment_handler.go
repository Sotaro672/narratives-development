// backend/internal/adapters/in/http/mall/webhook/stripe_payment_handler.go
package mallHandler

import (
	"context"
	"errors"

	usecase "narratives/internal/application/usecase"
	paymentdom "narratives/internal/domain/payment"
)

var (
	errStripeWebhookOrderUsecaseNotInitialized      = errors.New("stripe webhook: order usecase is not initialized")
	errStripeWebhookSettlementUsecaseNotInitialized = errors.New("stripe webhook: settlement usecase is not initialized")
)

func (h *StripeWebhookHandler) handleStripePaymentEvent(ctx context.Context, in usecase.ApplyStripePaymentEventInput) error {
	payment, err := h.paymentUC.ApplyStripeEvent(ctx, in)
	if err != nil {
		return err
	}
	if payment == nil || payment.Status != paymentdom.StatusSucceeded {
		return nil
	}
	if h.orderUC == nil {
		return errStripeWebhookOrderUsecaseNotInitialized
	}
	if h.settlementUC == nil {
		return errStripeWebhookSettlementUsecaseNotInitialized
	}

	order, err := h.orderUC.GetByID(ctx, payment.PaymentID)
	if err != nil {
		return err
	}
	_, err = h.settlementUC.EnsureForSucceededPayment(ctx, order, *payment)
	return err
}
