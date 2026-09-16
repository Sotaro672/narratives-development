// backend/internal/domain/refund/constructor.go
package refund

import "time"

// NewRefundInput contains authoritative values required to construct one Refund.
//
// Selection represents the seller's original decision.
//
// Merchandise/tax/shipping values must already have been calculated from the
// authoritative Order snapshot before this constructor is called.
type NewRefundInput struct {
	InquiryID string

	OrderID        string
	PaymentID      string
	OrderItemIndex int

	Seller SellerIdentity

	SettlementID      string
	SalesReceivableID string

	Selection ReturnRefundSelection

	MerchandiseAmount    int
	MerchandiseTaxAmount int

	OutboundShippingAmount    int
	OutboundShippingTaxAmount int

	ReturnShippingAmount    int
	ReturnShippingTaxAmount int

	TransferReversalAmount int

	Currency  string
	CreatedAt time.Time
}

// New creates one Refund using an explicitly supplied deterministic ID.
func New(
	id string,
	in NewRefundInput,
) (Refund, error) {
	if err := in.Seller.Validate(); err != nil {
		return Refund{}, err
	}

	if err := ValidateReturnRefundSelection(
		in.Selection,
	); err != nil {
		return Refund{}, err
	}

	createdAt := in.CreatedAt.UTC()

	refundAmount, err := calculateRefundAmount(
		in.MerchandiseAmount,
		in.MerchandiseTaxAmount,
		in.OutboundShippingAmount,
		in.OutboundShippingTaxAmount,
	)
	if err != nil {
		return Refund{}, err
	}

	transferReversalStatus :=
		TransferReversalStatusNotRequired

	if in.TransferReversalAmount > 0 {
		transferReversalStatus =
			TransferReversalStatusPending
	}

	r := Refund{
		ID: id,

		InquiryID: in.InquiryID,

		OrderID:        in.OrderID,
		PaymentID:      in.PaymentID,
		OrderItemIndex: in.OrderItemIndex,

		SellerType: in.Seller.Type,

		CompanyID: in.Seller.CompanyID,
		AccountID: in.Seller.AccountID,

		AvatarID:        in.Seller.AvatarID,
		UserID:          in.Seller.UserID,
		PayoutAccountID: in.Seller.PayoutAccountID,

		StripeAccountID: in.Seller.StripeAccountID,

		SettlementID:      in.SettlementID,
		SalesReceivableID: in.SalesReceivableID,

		RequestedMerchandiseRefundAmount: in.Selection.MerchandiseRefundAmount,
		RefundOutboundShipping:           in.Selection.RefundOutboundShipping,
		CoverReturnShipping:              in.Selection.CoverReturnShipping,

		MerchandiseAmount:    in.MerchandiseAmount,
		MerchandiseTaxAmount: in.MerchandiseTaxAmount,

		OutboundShippingAmount:    in.OutboundShippingAmount,
		OutboundShippingTaxAmount: in.OutboundShippingTaxAmount,

		ReturnShippingAmount:    in.ReturnShippingAmount,
		ReturnShippingTaxAmount: in.ReturnShippingTaxAmount,

		RefundAmount: refundAmount,

		Currency: in.Currency,

		StripeRefundID: "",
		Status:         DefaultStatus,
		RefundedAt:     nil,

		TransferReversalAmount: in.TransferReversalAmount,

		StripeTransferReversalID: "",
		TransferReversalStatus:   transferReversalStatus,
		TransferReversedAt:       nil,

		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}

	if err := r.Validate(); err != nil {
		return Refund{}, err
	}

	return r, nil
}

// NewForOrderItem creates a Refund using the deterministic Order-item Refund ID.
func NewForOrderItem(
	in NewRefundInput,
) (Refund, error) {
	id, err := NewID(
		in.OrderID,
		in.OrderItemIndex,
	)
	if err != nil {
		return Refund{}, err
	}

	return New(
		id,
		in,
	)
}
