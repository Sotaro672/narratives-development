// backend/internal/application/usecase/item_refund_validation.go
package usecase

import (
	orderdom "narratives/internal/domain/order"
	refunddom "narratives/internal/domain/refund"
	salesreceivabledom "narratives/internal/domain/salesReceivable"
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
	settlement *settlementdom.Settlement,
	salesReceivable *salesreceivabledom.SalesReceivable,
	amountSummary itemRefundAmountSummary,
) error {
	if err := refund.Validate(); err != nil {
		return err
	}
	if refund.InquiryID != in.InquiryID ||
		refund.OrderID != order.ID ||
		refund.PaymentID != order.ID ||
		refund.OrderItemIndex != in.ItemIndex {
		return ErrItemRefundExistingRefundMismatch
	}
	if refund.Policy != amountSummary.Policy ||
		refund.MerchandiseAmount != amountSummary.MerchandiseAmount ||
		refund.MerchandiseTaxAmount != amountSummary.MerchandiseTaxAmount ||
		refund.OutboundShippingAmount != amountSummary.OutboundShippingAmount ||
		refund.OutboundShippingTaxAmount != amountSummary.OutboundShippingTaxAmount ||
		refund.ReturnShippingAmount != amountSummary.ReturnShippingAmount ||
		refund.ReturnShippingTaxAmount != amountSummary.ReturnShippingTaxAmount ||
		refund.RefundAmount != amountSummary.RefundAmount ||
		refund.Currency != refunddom.CurrencyJPY {
		return ErrItemRefundExistingRefundMismatch
	}

	switch targetItem.Type {
	case orderdom.OrderItemTypeList:
		return validateExistingListItemRefund(
			refund,
			in,
			targetItem,
			settlement,
			salesReceivable,
		)

	case orderdom.OrderItemTypeResale:
		return validateExistingResaleItemRefund(
			refund,
			order,
			targetItem,
			settlement,
			salesReceivable,
		)

	default:
		return ErrItemRefundExistingRefundMismatch
	}
}

func validateExistingListItemRefund(
	refund refunddom.Refund,
	in itemRefundRequest,
	targetItem orderdom.OrderItemSnapshot,
	settlement *settlementdom.Settlement,
	salesReceivable *salesreceivabledom.SalesReceivable,
) error {
	if settlement == nil || salesReceivable != nil {
		return ErrItemRefundExistingRefundMismatch
	}
	if in.CompanyID == "" ||
		targetItem.SellerSnapshot.CompanyID != in.CompanyID {
		return ErrItemRefundExistingRefundMismatch
	}

	targetSeller, err := itemRefundSellerIdentity(targetItem)
	if err != nil {
		return ErrItemRefundExistingRefundMismatch
	}
	if targetSeller.Type != settlementdom.SellerTypeAccount {
		return ErrItemRefundExistingRefundMismatch
	}

	settlementSeller := settlement.SellerIdentity()
	if err := settlementSeller.Validate(); err != nil {
		return ErrItemRefundExistingRefundMismatch
	}
	if settlementSeller.Type != settlementdom.SellerTypeAccount ||
		settlementSeller != targetSeller {
		return ErrItemRefundExistingRefundMismatch
	}

	expectedRefundSeller := refunddom.SellerIdentity{
		Type:            refunddom.SellerTypeAccount,
		CompanyID:       targetSeller.CompanyID,
		AccountID:       targetSeller.AccountID,
		StripeAccountID: targetSeller.StripeAccountID,
	}
	if err := expectedRefundSeller.Validate(); err != nil {
		return ErrItemRefundExistingRefundMismatch
	}

	refundSeller := refund.SellerIdentity()
	if err := refundSeller.Validate(); err != nil {
		return ErrItemRefundExistingRefundMismatch
	}
	if refundSeller != expectedRefundSeller {
		return ErrItemRefundExistingRefundMismatch
	}

	if settlement.ID == "" ||
		settlement.PaymentID != refund.PaymentID ||
		settlement.OrderID != refund.OrderID ||
		settlement.Currency != settlementdom.CurrencyJPY ||
		refund.SettlementID != settlement.ID ||
		refund.SalesReceivableID != "" {
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

func validateExistingResaleItemRefund(
	refund refunddom.Refund,
	order orderdom.Order,
	targetItem orderdom.OrderItemSnapshot,
	settlement *settlementdom.Settlement,
	salesReceivable *salesreceivabledom.SalesReceivable,
) error {
	if settlement != nil || salesReceivable == nil {
		return ErrItemRefundExistingRefundMismatch
	}
	if targetItem.Type != orderdom.OrderItemTypeResale ||
		targetItem.ResaleID == "" ||
		targetItem.Qty != 1 ||
		targetItem.Price <= 0 {
		return ErrItemRefundExistingRefundMismatch
	}

	expectedRefundSeller, err := itemRefundResaleSellerIdentity(targetItem)
	if err != nil {
		return ErrItemRefundExistingRefundMismatch
	}

	refundSeller := refund.SellerIdentity()
	if err := refundSeller.Validate(); err != nil {
		return ErrItemRefundExistingRefundMismatch
	}
	if refundSeller != expectedRefundSeller ||
		refund.SellerType != refunddom.SellerTypeResale {
		return ErrItemRefundExistingRefundMismatch
	}

	if err := salesReceivable.Validate(); err != nil {
		return ErrItemRefundExistingRefundMismatch
	}

	expectedReceivableID, err := salesreceivabledom.NewID(
		order.ID,
		refund.OrderItemIndex,
	)
	if err != nil {
		return ErrItemRefundExistingRefundMismatch
	}

	snapshot := targetItem.SellerSnapshot
	if salesReceivable.ID != expectedReceivableID ||
		salesReceivable.OrderID != order.ID ||
		salesReceivable.PaymentID != order.ID ||
		salesReceivable.OrderItemIndex != refund.OrderItemIndex ||
		salesReceivable.ResaleID != targetItem.ResaleID ||
		salesReceivable.AvatarID != snapshot.AvatarID ||
		salesReceivable.UserID != snapshot.UserID ||
		salesReceivable.PayoutAccountID != snapshot.PayoutAccountID ||
		salesReceivable.PayoutAccountID != salesReceivable.UserID ||
		salesReceivable.MerchandiseAmount != targetItem.Price ||
		salesReceivable.Currency != salesreceivabledom.CurrencyJPY {
		return ErrItemRefundExistingRefundMismatch
	}

	if salesReceivable.Status != salesreceivabledom.StatusCanceled ||
		salesReceivable.BankPayoutID != "" {
		return ErrItemRefundExistingRefundMismatch
	}

	if refund.SettlementID != "" ||
		refund.SalesReceivableID != salesReceivable.ID ||
		refund.TransferReversalAmount != 0 ||
		refund.TransferReversalStatus != refunddom.TransferReversalStatusNotRequired ||
		refund.StripeTransferReversalID != "" ||
		refund.TransferReversedAt != nil {
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
		u.salesReceivableService == nil ||
		u.refundRepo == nil ||
		u.platformFeeCalculator == nil ||
		u.stripeRefundGateway == nil ||
		u.stripeTransferReversalGateway == nil ||
		u.now == nil {
		return ErrItemRefundNotConfigured
	}

	return nil
}
