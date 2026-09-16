// backend/internal/domain/refund/validation.go
package refund

import "strings"

// Validate verifies all Refund persistence invariants.
func (r Refund) Validate() error {
	if r.InquiryID == "" ||
		strings.Contains(r.InquiryID, "/") {
		return ErrInvalidInquiryID
	}

	if r.OrderID == "" ||
		strings.Contains(r.OrderID, "/") {
		return ErrInvalidOrderID
	}

	if r.PaymentID == "" ||
		strings.Contains(r.PaymentID, "/") {
		return ErrInvalidPaymentID
	}

	if r.PaymentID != r.OrderID {
		return ErrPaymentOrderMismatch
	}

	if r.OrderItemIndex < 0 {
		return ErrInvalidOrderItemIndex
	}

	expectedID, err := NewID(
		r.OrderID,
		r.OrderItemIndex,
	)
	if err != nil ||
		r.ID != expectedID {
		return ErrInvalidID
	}

	seller := r.SellerIdentity()

	if err := seller.Validate(); err != nil {
		return err
	}

	if err :=
		r.validateSellerFinancialReference(); err != nil {
		return err
	}

	if r.MerchandiseAmount < 0 {
		return ErrInvalidMerchandiseAmount
	}

	if r.MerchandiseTaxAmount < 0 {
		return ErrInvalidMerchandiseTaxAmount
	}

	if r.OutboundShippingAmount < 0 {
		return ErrInvalidOutboundShippingAmount
	}

	if r.OutboundShippingTaxAmount < 0 {
		return ErrInvalidOutboundShippingTaxAmount
	}

	if r.ReturnShippingAmount < 0 {
		return ErrInvalidReturnShippingAmount
	}

	if r.ReturnShippingTaxAmount < 0 {
		return ErrInvalidReturnShippingTaxAmount
	}

	if err :=
		r.validateReturnRefundAmounts(); err != nil {
		return err
	}

	if r.RefundAmount <= 0 {
		return ErrInvalidRefundAmount
	}

	expectedRefundAmount, err :=
		calculateRefundAmount(
			r.MerchandiseAmount,
			r.MerchandiseTaxAmount,
			r.OutboundShippingAmount,
			r.OutboundShippingTaxAmount,
		)
	if err != nil {
		return err
	}

	if r.RefundAmount !=
		expectedRefundAmount {
		return ErrRefundAmountMismatch
	}

	if _, err :=
		r.TotalSellerBurdenAmount(); err != nil {
		return err
	}

	if r.Currency != CurrencyJPY {
		return ErrInvalidCurrency
	}

	if !IsValidStatus(r.Status) {
		return ErrInvalidStatus
	}

	if err :=
		r.validateStripeRefundState(); err != nil {
		return err
	}

	if r.TransferReversalAmount < 0 ||
		r.TransferReversalAmount >
			r.RefundAmount {
		return ErrInvalidTransferReversalAmount
	}

	if r.SellerType == SellerTypeResale &&
		r.TransferReversalAmount != 0 {
		return ErrInvalidTransferReversalAmount
	}

	if !IsValidTransferReversalStatus(
		r.TransferReversalStatus,
	) {
		return ErrInvalidTransferReversalStatus
	}

	if err :=
		r.validateTransferReversalState(); err != nil {
		return err
	}

	if r.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}

	if r.UpdatedAt.IsZero() {
		return ErrInvalidUpdatedAt
	}

	if r.UpdatedAt.Before(
		r.CreatedAt,
	) {
		return ErrInvalidUpdatedAt
	}

	if r.RefundedAt != nil &&
		r.RefundedAt.Before(
			r.CreatedAt,
		) {
		return ErrInvalidRefundedAt
	}

	if r.TransferReversedAt != nil &&
		r.TransferReversedAt.Before(
			r.CreatedAt,
		) {
		return ErrInvalidTransferReversedAt
	}

	return nil
}

// validateReturnRefundAmounts validates the persisted seller refund decision
// against the calculated Refund monetary components.
//
// Exact shipping values and merchandise maximums are validated against the
// authoritative Order snapshot in the amount calculator / application layer.
func (r Refund) validateReturnRefundAmounts() error {
	selection :=
		r.ReturnRefundSelection()

	if err :=
		ValidateReturnRefundSelection(
			selection,
		); err != nil {
		return err
	}

	merchandiseRefundAmount, err :=
		safeAddRefundAmount(
			r.MerchandiseAmount,
			r.MerchandiseTaxAmount,
		)
	if err != nil {
		return err
	}

	if merchandiseRefundAmount !=
		r.RequestedMerchandiseRefundAmount {
		return ErrInvalidReturnRefundAmounts
	}

	if !r.RefundOutboundShipping {
		if r.OutboundShippingAmount != 0 ||
			r.OutboundShippingTaxAmount != 0 {
			return ErrInvalidReturnRefundAmounts
		}
	}

	if !r.CoverReturnShipping {
		if r.ReturnShippingAmount != 0 ||
			r.ReturnShippingTaxAmount != 0 {
			return ErrInvalidReturnRefundAmounts
		}
	}

	return nil
}
