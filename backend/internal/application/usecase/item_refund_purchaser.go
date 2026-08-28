// backend/internal/application/usecase/item_refund_purchaser.go
package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	applicationport "narratives/internal/application/port"
	paymentdom "narratives/internal/domain/payment"
	refunddom "narratives/internal/domain/refund"
)

// ============================================================
// Purchaser Refund
// ============================================================

func (u *ItemRefundUsecase) createPurchaserRefund(
	ctx context.Context,
	payment paymentdom.Payment,
	refund refunddom.Refund,
) (refunddom.Refund, error) {
	result, err := u.stripeRefundGateway.CreateRefund(
		ctx,
		applicationport.CreateStripeRefundInput{
			StripeChargeID: payment.StripeChargeID,
			Amount:         refund.RefundAmount,
			IdempotencyKey: itemRefundIdempotencyKey(
				"refund",
				refund.ID,
			),
			PaymentID: payment.PaymentID,
			RefundID:  refund.ID,
		},
	)
	if err != nil {
		return refund, fmt.Errorf(
			"item refund: create Stripe refund: %w",
			err,
		)
	}

	if result == nil {
		return refund, ErrItemRefundStripeRefundResultEmpty
	}

	if result.StripeRefundID == "" ||
		!strings.HasPrefix(result.StripeRefundID, "re_") {
		return refund, refunddom.ErrInvalidStripeRefundID
	}

	if result.CreatedAt.IsZero() {
		return refund, refunddom.ErrInvalidRefundedAt
	}

	status, refundedAt, err := mapStripeRefundResult(result)
	if err != nil {
		return refund, err
	}

	if status == refunddom.StatusSucceeded &&
		refundedAt != nil &&
		refundedAt.Before(refund.CreatedAt) {
		delta := refund.CreatedAt.Sub(*refundedAt)

		if delta >= time.Second {
			return refund, refunddom.ErrInvalidRefundedAt
		}

		normalized := refund.CreatedAt.UTC()
		refundedAt = &normalized
	}

	updated, err := u.refundRepo.UpdateByID(
		ctx,
		refund.ID,
		refunddom.UpdateRefundInput{
			Operation:      refunddom.UpdateOperationApplyStripeRefund,
			StripeRefundID: result.StripeRefundID,
			RefundStatus:   status,
			RefundedAt:     refundedAt,
			UpdatedAt:      u.nowUTC(),
		},
	)
	if err != nil {
		return refund, fmt.Errorf(
			"item refund: persist Stripe refund result: %w",
			err,
		)
	}

	if updated == nil {
		return refund, refunddom.ErrConflict
	}

	switch updated.Status {
	case refunddom.StatusPending,
		refunddom.StatusRequiresAction,
		refunddom.StatusSucceeded:
		return *updated, nil

	case refunddom.StatusFailed,
		refunddom.StatusCanceled:
		return *updated, ErrItemRefundStripeRefundTerminal

	default:
		return *updated, ErrItemRefundStripeRefundStatusInvalid
	}
}

func mapStripeRefundResult(
	result *applicationport.CreateStripeRefundResult,
) (
	refunddom.RefundStatus,
	*time.Time,
	error,
) {
	if result == nil {
		return "", nil, ErrItemRefundStripeRefundResultEmpty
	}

	switch result.Status {
	case paymentdom.RefundStatusPending:
		return refunddom.StatusPending, nil, nil

	case paymentdom.RefundStatusRequiresAction:
		return refunddom.StatusRequiresAction, nil, nil

	case paymentdom.RefundStatusSucceeded:
		if result.CreatedAt.IsZero() {
			return "", nil, refunddom.ErrInvalidRefundedAt
		}

		refundedAt := result.CreatedAt.UTC()
		return refunddom.StatusSucceeded, &refundedAt, nil

	case paymentdom.RefundStatusFailed:
		return refunddom.StatusFailed, nil, nil

	case paymentdom.RefundStatusCanceled:
		return refunddom.StatusCanceled, nil, nil

	default:
		return "", nil, ErrItemRefundStripeRefundStatusInvalid
	}
}
