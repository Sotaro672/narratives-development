// backend/internal/domain/refund/return_refund_selection.go
package refund

// ReturnRefundSelection represents the refund conditions selected by the seller.
//
// MerchandiseRefundAmount is the tax-inclusive merchandise refund amount.
//
// RefundOutboundShipping indicates whether the purchaser's original outbound
// shipping and its consumption tax are also refunded.
//
// CoverReturnShipping indicates whether the seller bears the return-shipping
// cost and its consumption tax.
//
// Tax amounts and actual shipping amounts must never be accepted from the
// frontend. They are calculated from authoritative persisted data.
type ReturnRefundSelection struct {
	MerchandiseRefundAmount int
	RefundOutboundShipping  bool
	CoverReturnShipping     bool
}

// ValidateReturnRefundSelection validates conditions that do not require an
// Order snapshot.
//
// The maximum merchandise refund must be validated separately against the
// authoritative Order snapshot.
func ValidateReturnRefundSelection(
	selection ReturnRefundSelection,
) error {
	if selection.MerchandiseRefundAmount <= 0 {
		return ErrInvalidReturnRefundAmount
	}

	return nil
}

// ReturnRefundSelection reconstructs the original seller decision persisted in
// this Refund.
func (r Refund) ReturnRefundSelection() ReturnRefundSelection {
	return ReturnRefundSelection{
		MerchandiseRefundAmount: r.RequestedMerchandiseRefundAmount,
		RefundOutboundShipping:  r.RefundOutboundShipping,
		CoverReturnShipping:     r.CoverReturnShipping,
	}
}
