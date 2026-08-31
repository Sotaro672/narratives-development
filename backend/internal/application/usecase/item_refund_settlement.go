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

// resolveSettlement resolves the primary-sale Settlement corresponding to one
// List Order item.
//
// Consumer resale items must never use Settlement. Their seller-side financial
// state is represented by the item-level SalesReceivable.
//
// One Payment may contain multiple primary-sale sellers, while items belonging
// to the same Account seller share one Settlement. The target Settlement is
// therefore resolved by the immutable Account seller identity captured in the
// Order item snapshot.
func (u *ItemRefundUsecase) resolveSettlement(
	ctx context.Context,
	payment paymentdom.Payment,
	targetItem orderdom.OrderItemSnapshot,
) (settlementdom.Settlement, error) {
	if u == nil || u.settlementRepo == nil {
		return settlementdom.Settlement{}, ErrItemRefundNotConfigured
	}
	if targetItem.Type != orderdom.OrderItemTypeList {
		return settlementdom.Settlement{}, ErrItemRefundSettlementMismatch
	}
	if payment.PaymentID == "" {
		return settlementdom.Settlement{}, ErrItemRefundSettlementMismatch
	}

	targetSeller, err := itemRefundSellerIdentity(targetItem)
	if err != nil {
		return settlementdom.Settlement{}, err
	}
	if targetSeller.Type != settlementdom.SellerTypeAccount {
		return settlementdom.Settlement{}, ErrItemRefundSettlementMismatch
	}

	expectedSettlementID, err := settlementdom.NewID(
		payment.PaymentID,
		targetSeller,
	)
	if err != nil {
		return settlementdom.Settlement{}, ErrItemRefundSettlementMismatch
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
		current := settlements[index]
		currentSeller := current.SellerIdentity()

		if err := currentSeller.Validate(); err != nil {
			return settlementdom.Settlement{}, ErrItemRefundSettlementMismatch
		}
		if currentSeller.Type != settlementdom.SellerTypeAccount {
			continue
		}
		if currentSeller != targetSeller {
			continue
		}

		if matched != nil {
			return settlementdom.Settlement{}, ErrItemRefundSettlementDuplicate
		}

		copy := current
		matched = &copy
	}

	if matched == nil {
		return settlementdom.Settlement{}, ErrItemRefundSettlementNotFound
	}

	matchedSeller := matched.SellerIdentity()
	if err := matchedSeller.Validate(); err != nil {
		return settlementdom.Settlement{}, ErrItemRefundSettlementMismatch
	}

	if matchedSeller.Type != settlementdom.SellerTypeAccount ||
		matchedSeller != targetSeller ||
		matched.ID != expectedSettlementID ||
		matched.PaymentID != payment.PaymentID ||
		matched.OrderID != payment.PaymentID ||
		matched.Currency != settlementdom.CurrencyJPY ||
		matched.StripeChargeID == "" ||
		matched.StripeChargeID != payment.StripeChargeID {
		return settlementdom.Settlement{}, ErrItemRefundSettlementMismatch
	}

	switch matched.Status {
	case settlementdom.StatusTransferred:
		if matched.StripeTransferID == "" ||
			matched.TransferredAt == nil ||
			matched.TransferredAt.IsZero() {
			return settlementdom.Settlement{}, ErrItemRefundSettlementMismatch
		}
		return *matched, nil

	case settlementdom.StatusPending,
		settlementdom.StatusReady,
		settlementdom.StatusTransferring,
		settlementdom.StatusFailedRetryable:
		if matched.TransferredAt != nil {
			return settlementdom.Settlement{}, ErrItemRefundSettlementMismatch
		}
		return *matched, nil

	case settlementdom.StatusFailed,
		settlementdom.StatusCanceled:
		if matched.StripeTransferID != "" ||
			matched.TransferredAt != nil {
			return settlementdom.Settlement{}, ErrItemRefundSettlementMismatch
		}
		return *matched, nil

	case settlementdom.StatusReversed:
		return settlementdom.Settlement{}, ErrItemRefundSettlementUnavailable

	default:
		return settlementdom.Settlement{}, ErrItemRefundSettlementUnavailable
	}
}
