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
		summary, err := refunddom.CalculateOrderItemRefundAmount(order, itemIndex)
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

	summary, err := refunddom.CalculateOpenedReturnRefundAmount(order, itemIndex, policy)
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

// resolveTransferReversalAmount determines the seller-side Stripe Transfer
// Reversal required for one primary List-sale item refund.
//
// Consumer resale never reaches this function. Resale seller proceeds are
// represented by SalesReceivable and TransferReversalAmount is always zero.
func (u *ItemRefundUsecase) resolveTransferReversalAmount(
	ctx context.Context,
	settlement settlementdom.Settlement,
	targetItem orderdom.OrderItemSnapshot,
	amountSummary itemRefundAmountSummary,
) (int, error) {
	if targetItem.Type != orderdom.OrderItemTypeList {
		return 0, ErrItemRefundSettlementMismatch
	}

	targetSeller, err := itemRefundSellerIdentity(targetItem)
	if err != nil {
		return 0, err
	}

	settlementSeller := settlement.SellerIdentity()
	if err := settlementSeller.Validate(); err != nil {
		return 0, ErrItemRefundSettlementMismatch
	}
	if settlementSeller.Type != settlementdom.SellerTypeAccount ||
		settlementSeller != targetSeller {
		return 0, ErrItemRefundSettlementMismatch
	}

	switch settlement.Status {
	case settlementdom.StatusTransferred:
		if settlement.StripeTransferID == "" ||
			!strings.HasPrefix(settlement.StripeTransferID, "tr_") ||
			settlement.TransferredAt == nil ||
			settlement.TransferredAt.IsZero() {
			return 0, ErrItemRefundSettlementMismatch
		}

		return u.calculateTransferReversalAmount(ctx, targetItem, amountSummary)

	case settlementdom.StatusFailed,
		settlementdom.StatusCanceled:
		if settlement.StripeTransferID != "" ||
			settlement.TransferredAt != nil {
			return 0, ErrItemRefundSettlementMismatch
		}

		// Seller Transfer never completed, therefore the purchaser refund does
		// not require a seller-side Stripe Transfer Reversal.
		return 0, nil

	case settlementdom.StatusPending,
		settlementdom.StatusReady,
		settlementdom.StatusTransferring,
		settlementdom.StatusFailedRetryable:
		// These states may still produce a seller Transfer. Refunding the buyer
		// before that state is canceled would risk paying the seller afterward.
		return 0, ErrItemRefundSettlementNotTransferred

	case settlementdom.StatusReversed:
		return 0, ErrItemRefundSettlementUnavailable

	default:
		return 0, ErrItemRefundSettlementUnavailable
	}
}

// calculateTransferReversalAmount calculates the partial Stripe Transfer
// Reversal for one primary List-sale item.
//
// ReversalAmount:
//
//	RefundAmount - platform fee attributable to refunded components
//
// This calculation is intentionally unavailable for consumer resale.
func (u *ItemRefundUsecase) calculateTransferReversalAmount(
	ctx context.Context,
	targetItem orderdom.OrderItemSnapshot,
	amountSummary itemRefundAmountSummary,
) (int, error) {
	if u == nil || u.platformFeeCalculator == nil {
		return 0, ErrItemRefundNotConfigured
	}
	if targetItem.Type != orderdom.OrderItemTypeList {
		return 0, ErrItemRefundSettlementMismatch
	}
	if amountSummary.RefundAmount <= 0 {
		return 0, refunddom.ErrInvalidRefundAmount
	}

	seller, err := itemRefundSellerIdentity(targetItem)
	if err != nil {
		return 0, err
	}

	platformFeeAmount, err := u.platformFeeCalculator.CalculatePlatformFee(
		ctx,
		settlementdom.PlatformFeeInput{
			Seller:               seller,
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

// itemRefundSellerIdentity resolves the Settlement seller identity for a primary
// List-sale Order item.
//
// Settlement is no longer used for consumer resale. A resale item must be
// resolved through its item-level SalesReceivable instead.
func itemRefundSellerIdentity(
	targetItem orderdom.OrderItemSnapshot,
) (settlementdom.SellerIdentity, error) {
	if targetItem.Type != orderdom.OrderItemTypeList {
		return settlementdom.SellerIdentity{}, ErrItemRefundSettlementMismatch
	}

	snapshot := targetItem.SellerSnapshot
	if snapshot.CompanyID == "" ||
		snapshot.AccountID == "" ||
		snapshot.StripeAccountID == "" ||
		snapshot.AvatarID != "" ||
		snapshot.UserID != "" ||
		snapshot.PayoutAccountID != "" {
		return settlementdom.SellerIdentity{}, ErrItemRefundSettlementMismatch
	}

	seller := settlementdom.SellerIdentity{
		Type:            settlementdom.SellerTypeAccount,
		CompanyID:       snapshot.CompanyID,
		AccountID:       snapshot.AccountID,
		StripeAccountID: snapshot.StripeAccountID,
	}

	if err := seller.Validate(); err != nil {
		return settlementdom.SellerIdentity{}, ErrItemRefundSettlementMismatch
	}

	return seller, nil
}
