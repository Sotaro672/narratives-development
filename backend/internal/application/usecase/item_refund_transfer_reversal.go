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

// completeTransferReversal completes seller-side Stripe Transfer Reversal for
// one primary List-sale item Refund.
//
// Consumer resale never reaches this function. Resale seller proceeds are
// represented by SalesReceivable and must have:
//
//	SalesReceivableID != ""
//	SettlementID == ""
//	TransferReversalAmount == 0
//
// before purchaser Stripe Refund processing.
//
// Primary List sale:
//
//	Refund
//		-> Settlement
//		-> Stripe Transfer Reversal when the seller Transfer already completed
func (u *ItemRefundUsecase) completeTransferReversal(
	ctx context.Context,
	payment paymentdom.Payment,
	settlement settlementdom.Settlement,
	refund refunddom.Refund,
) (refunddom.Refund, error) {
	if err := refund.Validate(); err != nil {
		return refund, err
	}
	if payment.PaymentID == "" ||
		refund.PaymentID != payment.PaymentID ||
		refund.OrderID != payment.PaymentID ||
		settlement.PaymentID != payment.PaymentID ||
		settlement.OrderID != payment.PaymentID ||
		refund.SellerType != refunddom.SellerTypeAccount ||
		refund.SettlementID == "" ||
		refund.SettlementID != settlement.ID ||
		refund.SalesReceivableID != "" {
		return refund, ErrItemRefundSettlementMismatch
	}

	settlementSeller := settlement.SellerIdentity()
	if err := settlementSeller.Validate(); err != nil {
		return refund, ErrItemRefundSettlementMismatch
	}
	if settlementSeller.Type != settlementdom.SellerTypeAccount {
		return refund, ErrItemRefundSettlementMismatch
	}

	refundSeller := refund.SellerIdentity()
	if err := refundSeller.Validate(); err != nil {
		return refund, ErrItemRefundSettlementMismatch
	}
	if refundSeller.Type != refunddom.SellerTypeAccount ||
		refundSeller.CompanyID != settlementSeller.CompanyID ||
		refundSeller.AccountID != settlementSeller.AccountID ||
		refundSeller.StripeAccountID != settlementSeller.StripeAccountID ||
		refundSeller.AvatarID != "" ||
		refundSeller.UserID != "" ||
		refundSeller.PayoutAccountID != "" {
		return refund, ErrItemRefundSettlementMismatch
	}

	if settlement.StripeChargeID == "" ||
		payment.StripeChargeID == "" ||
		settlement.StripeChargeID != payment.StripeChargeID {
		return refund, ErrItemRefundSettlementMismatch
	}

	if refund.TransferReversalAmount < 0 ||
		refund.TransferReversalAmount > settlement.TransferAmount {
		return refund, ErrItemRefundTransferReversalAmountExceeded
	}

	if refund.TransferReversalAmount == 0 {
		if refund.TransferReversalStatus != refunddom.TransferReversalStatusNotRequired ||
			refund.StripeTransferReversalID != "" ||
			refund.TransferReversedAt != nil {
			return refund, refunddom.ErrInvalidTransferReversalStatus
		}

		switch settlement.Status {
		case settlementdom.StatusTransferred:
			if settlement.StripeTransferID == "" ||
				!strings.HasPrefix(settlement.StripeTransferID, "tr_") ||
				settlement.TransferredAt == nil ||
				settlement.TransferredAt.IsZero() {
				return refund, ErrItemRefundSettlementMismatch
			}
			return refund, nil

		case settlementdom.StatusFailed,
			settlementdom.StatusCanceled:
			if settlement.StripeTransferID != "" ||
				settlement.TransferredAt != nil {
				return refund, ErrItemRefundSettlementMismatch
			}
			return refund, nil

		case settlementdom.StatusPending,
			settlementdom.StatusReady,
			settlementdom.StatusTransferring,
			settlementdom.StatusFailedRetryable:
			return refund, ErrItemRefundSettlementNotTransferred

		case settlementdom.StatusReversed:
			return refund, ErrItemRefundSettlementUnavailable

		default:
			return refund, ErrItemRefundSettlementUnavailable
		}
	}

	if settlement.Status != settlementdom.StatusTransferred {
		return refund, ErrItemRefundSettlementNotTransferred
	}
	if settlement.StripeTransferID == "" ||
		!strings.HasPrefix(settlement.StripeTransferID, "tr_") ||
		settlement.TransferredAt == nil ||
		settlement.TransferredAt.IsZero() {
		return refund, ErrItemRefundSettlementMismatch
	}

	switch refund.TransferReversalStatus {
	case refunddom.TransferReversalStatusSucceeded:
		if refund.StripeTransferReversalID == "" ||
			!strings.HasPrefix(refund.StripeTransferReversalID, "trr_") ||
			refund.TransferReversedAt == nil ||
			refund.TransferReversedAt.IsZero() {
			return refund, refunddom.ErrInvalidTransferReversalStatus
		}
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

	if refund.SellerType != refunddom.SellerTypeAccount ||
		refund.SalesReceivableID != "" ||
		refund.SettlementID != settlement.ID ||
		refund.TransferReversalAmount <= 0 {
		return refund, ErrItemRefundSettlementMismatch
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
			Seller:       settlementSeller,
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
	if updated.SellerType != refunddom.SellerTypeAccount ||
		updated.SettlementID != settlement.ID ||
		updated.SalesReceivableID != "" ||
		updated.TransferReversalStatus != refunddom.TransferReversalStatusSucceeded ||
		updated.StripeTransferReversalID != result.StripeTransferReversalID ||
		updated.TransferReversedAt == nil ||
		updated.TransferReversedAt.IsZero() {
		return refund, refunddom.ErrConflict
	}

	return *updated, nil
}

func (u *ItemRefundUsecase) persistTransferReversalFailure(
	ctx context.Context,
	refund refunddom.Refund,
	reversalErr error,
) (refunddom.Refund, error) {
	if refund.SellerType != refunddom.SellerTypeAccount ||
		refund.SettlementID == "" ||
		refund.SalesReceivableID != "" ||
		refund.TransferReversalAmount <= 0 {
		return refund, ErrItemRefundSettlementMismatch
	}

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
