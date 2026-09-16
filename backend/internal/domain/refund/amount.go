// backend/internal/domain/refund/amount.go
package refund

// TotalSellerBurdenAmount returns the total seller-side burden represented by
// this Refund.
//
// Return shipping is excluded from Stripe RefundAmount because it was not part
// of the purchaser's original Charge.
func (r Refund) TotalSellerBurdenAmount() (int, error) {
	total, err := safeAddRefundAmount(
		r.RefundAmount,
		r.ReturnShippingAmount,
	)
	if err != nil {
		return 0, err
	}

	return safeAddRefundAmount(
		total,
		r.ReturnShippingTaxAmount,
	)
}

func calculateRefundAmount(
	merchandiseAmount int,
	merchandiseTaxAmount int,
	outboundShippingAmount int,
	outboundShippingTaxAmount int,
) (int, error) {
	if merchandiseAmount < 0 {
		return 0, ErrInvalidMerchandiseAmount
	}
	if merchandiseTaxAmount < 0 {
		return 0, ErrInvalidMerchandiseTaxAmount
	}
	if outboundShippingAmount < 0 {
		return 0, ErrInvalidOutboundShippingAmount
	}
	if outboundShippingTaxAmount < 0 {
		return 0, ErrInvalidOutboundShippingTaxAmount
	}

	total, err := safeAddRefundAmount(
		merchandiseAmount,
		merchandiseTaxAmount,
	)
	if err != nil {
		return 0, err
	}

	total, err = safeAddRefundAmount(
		total,
		outboundShippingAmount,
	)
	if err != nil {
		return 0, err
	}

	total, err = safeAddRefundAmount(
		total,
		outboundShippingTaxAmount,
	)
	if err != nil {
		return 0, err
	}

	if total <= 0 {
		return 0, ErrInvalidRefundAmount
	}

	return total, nil
}

func safeAddRefundAmount(
	left int,
	right int,
) (int, error) {
	if left < 0 || right < 0 {
		return 0, ErrInvalidRefundAmount
	}

	maxInt := int(^uint(0) >> 1)
	if left > maxInt-right {
		return 0, ErrInvalidRefundAmount
	}

	return left + right, nil
}
