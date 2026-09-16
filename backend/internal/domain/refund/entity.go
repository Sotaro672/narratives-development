// backend/internal/domain/refund/entity.go
package refund

import "time"

// Refund represents one item-level purchaser refund.
//
// RequestedMerchandiseRefundAmount stores the tax-inclusive merchandise refund
// amount selected by the seller.
//
// RefundOutboundShipping and CoverReturnShipping preserve the seller's original
// refund decision independently from the calculated monetary components.
//
// Monetary components themselves are authoritative values calculated from the
// persisted Order snapshot by the backend.
type Refund struct {
	ID string

	InquiryID string

	OrderID        string
	PaymentID      string
	OrderItemIndex int

	SellerType SellerType

	CompanyID string
	AccountID string

	AvatarID        string
	UserID          string
	PayoutAccountID string

	StripeAccountID string

	SettlementID      string
	SalesReceivableID string

	RequestedMerchandiseRefundAmount int
	RefundOutboundShipping           bool
	CoverReturnShipping              bool

	MerchandiseAmount    int
	MerchandiseTaxAmount int

	OutboundShippingAmount    int
	OutboundShippingTaxAmount int

	ReturnShippingAmount    int
	ReturnShippingTaxAmount int

	RefundAmount int
	Currency     string

	StripeRefundID string
	Status         RefundStatus
	RefundedAt     *time.Time

	TransferReversalAmount int

	StripeTransferReversalID string
	TransferReversalStatus   TransferReversalStatus
	TransferReversedAt       *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}
