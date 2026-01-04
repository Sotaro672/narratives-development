// backend/internal/domain/invoice/entity.go
package invoice

import (
	"errors"
	"strings"
)

type Invoice struct {
	OrderID     string
	Prices      []int
	Tax         int
	ShippingFee int
	Paid        bool
}

// New creates a new invoice.
// paid defaults to false (caller may set Paid after New if needed).
func New(orderID string, prices []int, tax int, shippingFee int) (Invoice, error) {
	inv := Invoice{
		OrderID:     strings.TrimSpace(orderID),
		Prices:      prices,
		Tax:         tax,
		ShippingFee: shippingFee,
		Paid:        false, // ✅ default
	}
	if err := inv.Validate(); err != nil {
		return Invoice{}, err
	}
	return inv, nil
}

func (i Invoice) Validate() error {
	if strings.TrimSpace(i.OrderID) == "" {
		return errors.New("invoice: invalid orderId")
	}
	if len(i.Prices) == 0 {
		return errors.New("invoice: prices is required")
	}
	for _, p := range i.Prices {
		// order item price: 0 OK / negative NG
		if p < 0 {
			return errors.New("invoice: invalid price")
		}
	}
	if i.Tax < 0 {
		return errors.New("invoice: invalid tax")
	}
	if i.ShippingFee < 0 {
		return errors.New("invoice: invalid shippingFee")
	}
	return nil
}

func (i *Invoice) SetPaid(paid bool) error {
	i.Paid = paid
	return i.Validate()
}
