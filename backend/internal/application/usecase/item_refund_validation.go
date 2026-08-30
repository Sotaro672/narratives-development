// backend/internal/application/usecase/item_refund_validation.go
package usecase

import (
	orderdom "narratives/internal/domain/order"
	refunddom "narratives/internal/domain/refund"
	settlementdom "narratives/internal/domain/settlement"
)

// ============================================================
// Existing Refund Validation
// ============================================================

func validateExistingItemRefund(
	refund refunddom.Refund,
	in itemRefundRequest,
	order orderdom.Order,
	targetItem orderdom.OrderItemSnapshot,
	settlement settlementdom.Settlement,
	amountSummary itemRefundAmountSummary,
) error {
	if err := refund.Validate(); err != nil {
		return err
	}

	targetSeller, err := itemRefundSellerIdentity(targetItem)
	if err != nil {
		return ErrItemRefundExistingRefundMismatch
	}

	settlementSeller := settlement.SellerIdentity()
	if err := settlementSeller.Validate(); err != nil {
		return ErrItemRefundExistingRefundMismatch
	}

	if settlementSeller != targetSeller {
		return ErrItemRefundExistingRefundMismatch
	}

	if refund.InquiryID != in.InquiryID ||
		refund.OrderID != order.ID ||
		refund.PaymentID != order.ID ||
		refund.OrderItemIndex != in.ItemIndex ||
		refund.SettlementID != settlement.ID {
		return ErrItemRefundExistingRefundMismatch
	}

	switch targetSeller.Type {
	case settlementdom.SellerTypeAccount:
		if refund.CompanyID != targetSeller.CompanyID ||
			refund.AccountID != targetSeller.AccountID {
			return ErrItemRefundExistingRefundMismatch
		}

	case settlementdom.SellerTypeAvatar:
		// The current Refund aggregate still requires CompanyID and AccountID
		// and cannot persist an Avatar seller identity. Resale Refund support
		// must be enabled only after refund.Refund is generalized to SellerIdentity.
		return ErrItemRefundExistingRefundMismatch

	default:
		return ErrItemRefundExistingRefundMismatch
	}

	if refund.Policy != amountSummary.Policy ||
		refund.MerchandiseAmount != amountSummary.MerchandiseAmount ||
		refund.MerchandiseTaxAmount != amountSummary.MerchandiseTaxAmount ||
		refund.OutboundShippingAmount != amountSummary.OutboundShippingAmount ||
		refund.OutboundShippingTaxAmount != amountSummary.OutboundShippingTaxAmount ||
		refund.ReturnShippingAmount != amountSummary.ReturnShippingAmount ||
		refund.ReturnShippingTaxAmount != amountSummary.ReturnShippingTaxAmount ||
		refund.RefundAmount != amountSummary.RefundAmount {
		return ErrItemRefundExistingRefundMismatch
	}

	if refund.Currency != refunddom.CurrencyJPY {
		return ErrItemRefundExistingRefundMismatch
	}

	if refund.TransferReversalAmount < 0 ||
		refund.TransferReversalAmount > settlement.TransferAmount {
		return ErrItemRefundExistingRefundMismatch
	}

	if (settlement.Status == settlementdom.StatusFailed ||
		settlement.Status == settlementdom.StatusCanceled) &&
		refund.TransferReversalAmount != 0 {
		return ErrItemRefundExistingRefundMismatch
	}

	return nil
}

// ============================================================
// Validation
// ============================================================

func (u *ItemRefundUsecase) validateConfigured() error {
	if u == nil ||
		u.orderReader == nil ||
		u.paymentReader == nil ||
		u.settlementRepo == nil ||
		u.refundRepo == nil ||
		u.platformFeeCalculator == nil ||
		u.stripeRefundGateway == nil ||
		u.stripeTransferReversalGateway == nil ||
		u.now == nil {
		return ErrItemRefundNotConfigured
	}

	return nil
}
