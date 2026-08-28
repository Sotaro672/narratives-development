// backend/internal/application/usecase/item_refund_amount.go
package usecase

import (
	"context"
	"strings"

	orderdom "narratives/internal/domain/order"
	refunddom "narratives/internal/domain/refund"
	settlementdom "narratives/internal/domain/settlement"
)

// ============================================================
// Amount Calculation
// ============================================================

func calculateItemRefundAmount(
	order orderdom.Order,
	itemIndex int,
	policy refunddom.OpenedReturnRefundPolicy,
) (itemRefundAmountSummary, error) {
	if policy == "" {
		summary, err := orderdom.CalculateOrderItemRefundAmount(
			order,
			itemIndex,
		)
		if err != nil {
			return itemRefundAmountSummary{}, err
		}

		return itemRefundAmountSummary{
			MerchandiseAmount:    summary.MerchandiseAmount,
			MerchandiseTaxAmount: summary.MerchandiseTaxAmount,
			RefundAmount:         summary.RefundAmount,
		}, nil
	}

	if err := refunddom.ValidateOpenedReturnRefundPolicy(policy); err != nil {
		return itemRefundAmountSummary{}, err
	}

	summary, err := orderdom.CalculateOpenedReturnRefundAmount(
		order,
		itemIndex,
		policy,
	)
	if err != nil {
		return itemRefundAmountSummary{}, err
	}

	return itemRefundAmountSummary{
		Policy:                    summary.Policy,
		MerchandiseAmount:         summary.MerchandiseAmount,
		MerchandiseTaxAmount:      summary.MerchandiseTaxAmount,
		OutboundShippingAmount:    summary.OutboundShippingAmount,
		OutboundShippingTaxAmount: summary.OutboundShippingTaxAmount,
		ReturnShippingAmount:      summary.ReturnShippingAmount,
		ReturnShippingTaxAmount:   summary.ReturnShippingTaxAmount,
		RefundAmount:              summary.StripeRefundAmount,
	}, nil
}

func (u *ItemRefundUsecase) resolveTransferReversalAmount(
	ctx context.Context,
	settlement settlementdom.Settlement,
	targetItem orderdom.OrderItemSnapshot,
	amountSummary itemRefundAmountSummary,
) (int, error) {
	switch settlement.Status {
	case settlementdom.StatusTransferred:
		if settlement.StripeTransferID == "" ||
			!strings.HasPrefix(settlement.StripeTransferID, "tr_") {
			return 0, ErrItemRefundSettlementMismatch
		}

		return u.calculateTransferReversalAmount(
			ctx,
			targetItem,
			amountSummary,
		)

	case settlementdom.StatusFailed,
		settlementdom.StatusCanceled:
		if settlement.StripeTransferID != "" ||
			settlement.TransferredAt != nil {
			return 0, ErrItemRefundSettlementMismatch
		}

		// Seller Transfer never completed, so purchaser refund does not need
		// a seller-side Transfer Reversal.
		return 0, nil

	case settlementdom.StatusPending,
		settlementdom.StatusReady,
		settlementdom.StatusTransferring,
		settlementdom.StatusFailedRetryable:
		// These states may still result in a future seller Transfer.
		// Refunding the purchaser now would risk paying the seller afterward.
		return 0, ErrItemRefundSettlementNotTransferred

	case settlementdom.StatusReversed:
		return 0, ErrItemRefundSettlementUnavailable

	default:
		return 0, ErrItemRefundSettlementUnavailable
	}
}

func (u *ItemRefundUsecase) calculateTransferReversalAmount(
	ctx context.Context,
	targetItem orderdom.OrderItemSnapshot,
	amountSummary itemRefundAmountSummary,
) (int, error) {
	platformFeeAmount, err := u.platformFeeCalculator.CalculatePlatformFee(
		ctx,
		settlementdom.PlatformFeeInput{
			CompanyID:            targetItem.SellerSnapshot.CompanyID,
			AccountID:            targetItem.SellerSnapshot.AccountID,
			MerchandiseAmount:    amountSummary.MerchandiseAmount,
			MerchandiseTaxAmount: amountSummary.MerchandiseTaxAmount,
			ShippingAmount:       amountSummary.OutboundShippingAmount,
			ShippingTaxAmount:    amountSummary.OutboundShippingTaxAmount,
			GrossAmount:          amountSummary.RefundAmount,
		},
	)
	if err != nil {
		return 0, err
	}

	if platformFeeAmount < 0 ||
		platformFeeAmount > amountSummary.RefundAmount {
		return 0, ErrItemRefundInvalidPlatformFee
	}

	return amountSummary.RefundAmount - platformFeeAmount, nil
}
