// backend/internal/application/usecase/item_refund_transfer_reversal.go
package usecase

import (
	"context"
	"fmt"
	"strings"

	applicationport "narratives/internal/application/port"
	paymentdom "narratives/internal/domain/payment"
	refunddom "narratives/internal/domain/refund"
	settlementdom "narratives/internal/domain/settlement"
)

// ============================================================
// Seller Transfer Reversal
// ============================================================

func (u *ItemRefundUsecase) completeTransferReversal(
	ctx context.Context,
	payment paymentdom.Payment,
	settlement settlementdom.Settlement,
	refund refunddom.Refund,
) (refunddom.Refund, error) {
	if refund.TransferReversalAmount == 0 {
		if refund.TransferReversalStatus !=
			refunddom.TransferReversalStatusNotRequired {
			return refund, refunddom.ErrInvalidTransferReversalStatus
		}

		return refund, nil
	}

	if settlement.Status != settlementdom.StatusTransferred {
		return refund, ErrItemRefundSettlementNotTransferred
	}

	if settlement.StripeTransferID == "" ||
		!strings.HasPrefix(settlement.StripeTransferID, "tr_") {
		return refund, ErrItemRefundSettlementMismatch
	}

	switch refund.TransferReversalStatus {
	case refunddom.TransferReversalStatusSucceeded:
		return refund, nil

	case refunddom.TransferReversalStatusNotRequired:
		return refund, refunddom.ErrInvalidTransferReversalStatus

	case refunddom.TransferReversalStatusFailed:
		return refund, ErrItemRefundTransferReversalTerminal

	case refunddom.TransferReversalStatusFailedRetryable:
		updated, err := u.refundRepo.UpdateByID(
			ctx,
			refund.ID,
			refunddom.UpdateRefundInput{
				Operation: refunddom.UpdateOperationMarkTransferReversalPending,
				UpdatedAt: u.nowUTC(),
			},
		)
		if err != nil {
			return refund, err
		}

		if updated == nil {
			return refund, refunddom.ErrConflict
		}

		refund = *updated

	case refunddom.TransferReversalStatusPending:
		// Continue.

	default:
		return refund, refunddom.ErrInvalidTransferReversalStatus
	}

	result, reversalErr := u.stripeTransferReversalGateway.CreateTransferReversal(
		ctx,
		applicationport.CreateStripeTransferReversalInput{
			StripeTransferID: settlement.StripeTransferID,
			Amount:           refund.TransferReversalAmount,
			IdempotencyKey: itemRefundIdempotencyKey(
				"transfer-reversal",
				refund.ID,
			),
			OrderID:      refund.OrderID,
			PaymentID:    payment.PaymentID,
			SettlementID: settlement.ID,
			CompanyID:    refund.CompanyID,
			AccountID:    refund.AccountID,
		},
	)
	if reversalErr != nil {
		return u.persistTransferReversalFailure(
			ctx,
			refund,
			reversalErr,
		)
	}

	if result == nil {
		return u.persistTransferReversalFailure(
			ctx,
			refund,
			ErrItemRefundStripeTransferReversalResultEmpty,
		)
	}

	if result.StripeTransferReversalID == "" ||
		!strings.HasPrefix(result.StripeTransferReversalID, "trr_") {
		return u.persistTransferReversalFailure(
			ctx,
			refund,
			ErrItemRefundStripeTransferReversalIDInvalid,
		)
	}

	now := u.nowUTC()

	updated, err := u.refundRepo.UpdateByID(
		ctx,
		refund.ID,
		refunddom.UpdateRefundInput{
			Operation:                refunddom.UpdateOperationMarkTransferReversalSucceeded,
			StripeTransferReversalID: result.StripeTransferReversalID,
			TransferReversedAt:       &now,
			UpdatedAt:                now,
		},
	)
	if err != nil {
		return refund, fmt.Errorf(
			"item refund: persist Stripe transfer reversal result: %w",
			err,
		)
	}

	if updated == nil {
		return refund, refunddom.ErrConflict
	}

	return *updated, nil
}

func (u *ItemRefundUsecase) persistTransferReversalFailure(
	ctx context.Context,
	refund refunddom.Refund,
	reversalErr error,
) (refunddom.Refund, error) {
	operation := refunddom.UpdateOperationMarkTransferReversalFailed

	if isRetryableItemRefundError(reversalErr) {
		operation = refunddom.UpdateOperationMarkTransferReversalFailedRetryable
	}

	updated, persistErr := u.refundRepo.UpdateByID(
		ctx,
		refund.ID,
		refunddom.UpdateRefundInput{
			Operation: operation,
			UpdatedAt: u.nowUTC(),
		},
	)
	if persistErr != nil {
		return refund, fmt.Errorf(
			"item refund: Stripe transfer reversal failed: %w; persist failure state: %v",
			reversalErr,
			persistErr,
		)
	}

	if updated != nil {
		refund = *updated
	}

	return refund, fmt.Errorf(
		"item refund: create Stripe transfer reversal: %w",
		reversalErr,
	)
}
