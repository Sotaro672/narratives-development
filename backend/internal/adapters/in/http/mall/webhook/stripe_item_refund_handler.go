// backend/internal/adapters/in/http/mall/webhook/stripe_item_refund_handler.go
package mallHandler

import (
	"context"
	"errors"
	"strings"
	"time"

	usecase "narratives/internal/application/usecase"
	paymentdom "narratives/internal/domain/payment"
	refunddom "narratives/internal/domain/refund"
)

func (h *StripeWebhookHandler) applyStripeItemRefundEvent(ctx context.Context, in stripeRefundEventInput) error {
	if h == nil || h.paymentUC == nil {
		return errors.New("stripe webhook: payment usecase is not initialized")
	}
	if h.refundRepo == nil {
		return errors.New("stripe webhook: refund repository is not initialized")
	}

	refundID := strings.TrimSpace(in.RefundID)
	if refundID == "" || strings.Contains(refundID, "/") {
		return refunddom.ErrInvalidID
	}

	current, err := h.refundRepo.GetByID(ctx, refundID)
	if err != nil {
		return err
	}
	if current == nil || current.ID != refundID {
		return refunddom.ErrNotFound
	}
	if err := current.Validate(); err != nil {
		return err
	}
	if current.PaymentID != in.PaymentID || current.OrderID != in.PaymentID {
		return refunddom.ErrPaymentOrderMismatch
	}
	if current.RefundAmount != in.Amount {
		return refunddom.ErrInvalidRefundAmount
	}

	payment, err := h.paymentUC.GetByPaymentID(ctx, in.PaymentID)
	if err != nil {
		return err
	}
	if payment == nil || payment.PaymentID != in.PaymentID {
		return paymentdom.ErrNotFound
	}
	if payment.Status != paymentdom.StatusSucceeded {
		return paymentdom.ErrRefundRequiresSucceeded
	}
	if payment.StripeChargeID == "" || payment.StripeChargeID != in.StripeChargeID {
		return paymentdom.ErrInvalidStripeChargeID
	}
	if payment.StripePaymentIntentID == "" || payment.StripePaymentIntentID != in.StripePaymentIntentID {
		return paymentdom.ErrInvalidStripePaymentIntent
	}
	if current.StripeRefundID != "" && current.StripeRefundID != in.StripeRefundID {
		return refunddom.ErrConflict
	}

	nextStatus, ok := itemRefundStatusFromPaymentRefundStatus(in.Status)
	if !ok {
		return refunddom.ErrInvalidStatus
	}
	if current.Status == nextStatus || isTerminalItemRefundStatus(current.Status) {
		return nil
	}

	occurredAt := in.OccurredAt.UTC()
	if occurredAt.IsZero() {
		return refunddom.ErrInvalidUpdatedAt
	}
	if occurredAt.Before(current.CreatedAt) {
		delta := current.CreatedAt.Sub(occurredAt)
		if delta >= time.Second {
			return refunddom.ErrInvalidUpdatedAt
		}
		occurredAt = current.CreatedAt.UTC()
	}

	var refundedAt *time.Time
	if nextStatus == refunddom.StatusSucceeded {
		value := occurredAt
		refundedAt = &value
	}

	updated, err := h.refundRepo.UpdateByID(ctx, refundID, refunddom.UpdateRefundInput{
		Operation:      refunddom.UpdateOperationApplyStripeRefund,
		StripeRefundID: in.StripeRefundID,
		RefundStatus:   nextStatus,
		RefundedAt:     refundedAt,
		UpdatedAt:      occurredAt,
	})
	if err != nil {
		return err
	}
	if updated == nil || updated.ID != refundID {
		return refunddom.ErrConflict
	}

	return nil
}

func (h *StripeWebhookHandler) completeSucceededItemRefund(ctx context.Context, refundID string) error {
	if h == nil || h.refundRepo == nil {
		return errors.New("stripe webhook: refund repository is not initialized")
	}

	refundID = strings.TrimSpace(refundID)
	if refundID == "" || strings.Contains(refundID, "/") {
		return refunddom.ErrInvalidID
	}

	refund, err := h.refundRepo.GetByID(ctx, refundID)
	if err != nil {
		return err
	}
	if refund == nil || refund.ID != refundID {
		return refunddom.ErrNotFound
	}
	if err := refund.Validate(); err != nil {
		return err
	}
	if refund.Status != refunddom.StatusSucceeded {
		return nil
	}
	if h.itemRefundUC == nil {
		return errors.New("stripe webhook: item refund usecase is not initialized")
	}

	selection := refunddom.ReturnRefundSelection{
		MerchandiseRefundAmount: refund.RequestedMerchandiseRefundAmount,
		RefundOutboundShipping:  refund.RefundOutboundShipping,
		CoverReturnShipping:     refund.CoverReturnShipping,
	}
	if err := refunddom.ValidateReturnRefundSelection(selection); err != nil {
		return err
	}

	completed, err := h.itemRefundUC.RefundOrderItem(ctx, usecase.RefundOrderItemInput{
		InquiryID: refund.InquiryID,
		OrderID:   refund.OrderID,
		ItemIndex: refund.OrderItemIndex,
		CompanyID: refund.CompanyID,
		Selection: selection,
	})
	if err != nil {
		return err
	}
	if !completed.IsFinanciallyCompleted() {
		return nil
	}
	if h.orderUC == nil {
		return errors.New("stripe webhook: order usecase is not initialized")
	}
	if h.refundCompletionNotificationUC == nil {
		return errors.New("stripe webhook: refund completion notification usecase is not initialized")
	}

	order, err := h.orderUC.GetByID(ctx, completed.OrderID)
	if err != nil {
		return err
	}

	orderID := strings.TrimSpace(order.ID)
	if orderID == "" || orderID != completed.OrderID || orderID != completed.PaymentID {
		return errors.New("stripe webhook: item refund order does not match refund")
	}

	userID := strings.TrimSpace(order.UserID)
	if userID == "" {
		return errors.New("stripe webhook: item refund order userId is empty")
	}

	stripeRefundID := strings.TrimSpace(completed.StripeRefundID)
	if stripeRefundID == "" {
		return refunddom.ErrInvalidStripeRefundID
	}

	_, err = h.refundCompletionNotificationUC.EnsureDelivery(ctx, usecase.EnsureRefundCompletionNotificationInput{
		PaymentID:      completed.PaymentID,
		OrderID:        orderID,
		UserID:         userID,
		StripeRefundID: stripeRefundID,
		RefundedAmount: completed.RefundAmount,
	})
	return err
}

func itemRefundStatusFromPaymentRefundStatus(status paymentdom.RefundStatus) (refunddom.RefundStatus, bool) {
	switch status {
	case paymentdom.RefundStatusPending:
		return refunddom.StatusPending, true
	case paymentdom.RefundStatusRequiresAction:
		return refunddom.StatusRequiresAction, true
	case paymentdom.RefundStatusSucceeded:
		return refunddom.StatusSucceeded, true
	case paymentdom.RefundStatusFailed:
		return refunddom.StatusFailed, true
	case paymentdom.RefundStatusCanceled:
		return refunddom.StatusCanceled, true
	default:
		return "", false
	}
}

func isTerminalItemRefundStatus(status refunddom.RefundStatus) bool {
	switch status {
	case refunddom.StatusSucceeded, refunddom.StatusFailed, refunddom.StatusCanceled:
		return true
	default:
		return false
	}
}
