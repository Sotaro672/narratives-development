// backend/internal/application/usecase/item_refund_settlement.go
package usecase

import (
	"context"

	orderdom "narratives/internal/domain/order"
	paymentdom "narratives/internal/domain/payment"
	settlementdom "narratives/internal/domain/settlement"
)

// ============================================================
// Settlement Resolution
// ============================================================

func (u *ItemRefundUsecase) resolveSettlement(
	ctx context.Context,
	payment paymentdom.Payment,
	targetItem orderdom.OrderItemSnapshot,
) (settlementdom.Settlement, error) {
	targetSeller, err := itemRefundSellerIdentity(targetItem)
	if err != nil {
		return settlementdom.Settlement{}, err
	}

	settlements, err := u.settlementRepo.ListByPaymentID(
		ctx,
		payment.PaymentID,
	)
	if err != nil {
		return settlementdom.Settlement{}, err
	}

	var matched *settlementdom.Settlement

	for index := range settlements {
		settlement := settlements[index]
		settlementSeller := settlement.SellerIdentity()

		if settlementSeller != targetSeller {
			continue
		}

		if matched != nil {
			return settlementdom.Settlement{},
				ErrItemRefundSettlementDuplicate
		}

		copy := settlement
		matched = &copy
	}

	if matched == nil {
		return settlementdom.Settlement{},
			ErrItemRefundSettlementNotFound
	}

	matchedSeller := matched.SellerIdentity()
	if err := matchedSeller.Validate(); err != nil {
		return settlementdom.Settlement{},
			ErrItemRefundSettlementMismatch
	}

	if matched.PaymentID != payment.PaymentID ||
		matched.OrderID != payment.PaymentID ||
		matchedSeller != targetSeller {
		return settlementdom.Settlement{},
			ErrItemRefundSettlementMismatch
	}

	if matched.Currency != settlementdom.CurrencyJPY {
		return settlementdom.Settlement{},
			ErrItemRefundSettlementMismatch
	}

	if matched.StripeChargeID != payment.StripeChargeID {
		return settlementdom.Settlement{},
			ErrItemRefundSettlementMismatch
	}

	switch matched.Status {
	case settlementdom.StatusTransferred:
		return *matched, nil

	case settlementdom.StatusPending,
		settlementdom.StatusReady,
		settlementdom.StatusTransferring,
		settlementdom.StatusFailedRetryable:
		return *matched, nil

	case settlementdom.StatusFailed,
		settlementdom.StatusCanceled:
		if matched.StripeTransferID != "" ||
			matched.TransferredAt != nil {
			return settlementdom.Settlement{},
				ErrItemRefundSettlementMismatch
		}

		return *matched, nil

	case settlementdom.StatusReversed:
		return settlementdom.Settlement{},
			ErrItemRefundSettlementUnavailable

	default:
		return settlementdom.Settlement{},
			ErrItemRefundSettlementUnavailable
	}
}
