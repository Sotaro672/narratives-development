// backend/internal/adapters/in/http/mall/webhook/stripe_full_refund_handler.go
package mallHandler

import (
	"context"
	"errors"
	"strings"
	"time"

	usecase "narratives/internal/application/usecase"
	paymentdom "narratives/internal/domain/payment"
)

func (h *StripeWebhookHandler) applyStripeRefundEvent(ctx context.Context, in stripeRefundEventInput) error {
	if h == nil || h.paymentUC == nil {
		return errors.New("stripe webhook: payment usecase is not initialized")
	}

	current, err := h.paymentUC.GetByPaymentID(ctx, in.PaymentID)
	if err != nil {
		return err
	}
	if current == nil || current.PaymentID != in.PaymentID {
		return paymentdom.ErrNotFound
	}
	if current.Status != paymentdom.StatusSucceeded {
		return paymentdom.ErrRefundRequiresSucceeded
	}
	if current.StripeChargeID == "" || current.StripeChargeID != in.StripeChargeID {
		return paymentdom.ErrInvalidStripeChargeID
	}
	if current.StripePaymentIntentID == "" || current.StripePaymentIntentID != in.StripePaymentIntentID {
		return paymentdom.ErrInvalidStripePaymentIntent
	}
	if in.Amount <= 0 || in.Amount != current.Amount {
		return paymentdom.ErrInvalidRefundedAmount
	}

	if current.RefundStatus != "" && current.RefundStatus != paymentdom.RefundStatusNone {
		if current.StripeRefundID == "" {
			return paymentdom.ErrInvalidRefundState
		}
		if current.StripeRefundID != in.StripeRefundID {
			return paymentdom.ErrConflict
		}
		if current.RefundStatus == in.Status || isTerminalRefundStatus(current.RefundStatus) {
			return nil
		}
	}

	refundedAmount := 0
	var refundedAt *time.Time
	if in.Status == paymentdom.RefundStatusSucceeded {
		refundedAmount = in.Amount
		value := in.OccurredAt.UTC()
		refundedAt = &value
	}

	_, err = h.paymentUC.UpdateRefundState(ctx, usecase.UpdatePaymentRefundStateInput{
		PaymentID:      in.PaymentID,
		StripeRefundID: in.StripeRefundID,
		RefundStatus:   in.Status,
		RefundedAmount: refundedAmount,
		RefundedAt:     refundedAt,
	})
	return err
}

func (h *StripeWebhookHandler) completeSucceededRefund(ctx context.Context, paymentID string) error {
	if h == nil || h.paymentUC == nil {
		return errors.New("stripe webhook: payment usecase is not initialized")
	}

	paymentID = strings.TrimSpace(paymentID)
	if paymentID == "" {
		return paymentdom.ErrInvalidPaymentID
	}

	payment, err := h.paymentUC.GetByPaymentID(ctx, paymentID)
	if err != nil {
		return err
	}
	if payment == nil || payment.PaymentID != paymentID {
		return paymentdom.ErrNotFound
	}
	if payment.RefundStatus != paymentdom.RefundStatusSucceeded {
		return nil
	}
	if h.refundUC == nil {
		return errors.New("stripe webhook: refund usecase is not initialized")
	}
	if h.orderUC == nil {
		return errors.New("stripe webhook: order usecase is not initialized")
	}
	if h.refundCompletionNotificationUC == nil {
		return errors.New("stripe webhook: refund completion notification usecase is not initialized")
	}

	stripeRefundID := strings.TrimSpace(payment.StripeRefundID)
	if stripeRefundID == "" || payment.RefundedAmount <= 0 || payment.RefundedAt == nil {
		return paymentdom.ErrInvalidRefundState
	}

	if _, err := h.refundUC.CompleteSucceededRefund(ctx, usecase.CompleteSucceededRefundInput{
		PaymentID:      payment.PaymentID,
		StripeRefundID: stripeRefundID,
	}); err != nil {
		return err
	}

	order, err := h.orderUC.GetByID(ctx, payment.PaymentID)
	if err != nil {
		return err
	}

	orderID := strings.TrimSpace(order.ID)
	if orderID == "" || orderID != payment.PaymentID {
		return errors.New("stripe webhook: refund order does not match payment")
	}

	userID := strings.TrimSpace(order.UserID)
	if userID == "" {
		return errors.New("stripe webhook: refund order userId is empty")
	}

	_, err = h.refundCompletionNotificationUC.EnsureDelivery(ctx, usecase.EnsureRefundCompletionNotificationInput{
		PaymentID:      payment.PaymentID,
		OrderID:        orderID,
		UserID:         userID,
		StripeRefundID: stripeRefundID,
		RefundedAmount: payment.RefundedAmount,
	})
	return err
}

func isTerminalRefundStatus(status paymentdom.RefundStatus) bool {
	switch status {
	case paymentdom.RefundStatusSucceeded, paymentdom.RefundStatusFailed, paymentdom.RefundStatusCanceled:
		return true
	default:
		return false
	}
}
