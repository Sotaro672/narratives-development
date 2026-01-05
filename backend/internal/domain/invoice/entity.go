// backend/internal/domain/invoice/entity.go
package invoice

import (
	"errors"
	"strings"
	"time"
)

type Invoice struct {
	OrderID     string
	Prices      []int
	Tax         int
	ShippingFee int
	Paid        bool

	// ✅ paid が false -> true になった瞬間（=支払い確定時刻）
	// paid が false のままなら nil のまま
	UpdatedAt *time.Time
}

// New creates a new invoice.
// paid defaults to false.
func New(orderID string, prices []int, tax int, shippingFee int) (Invoice, error) {
	inv := Invoice{
		OrderID:     strings.TrimSpace(orderID),
		Prices:      prices,
		Tax:         tax,
		ShippingFee: shippingFee,
		Paid:        false, // ✅ default
		UpdatedAt:   nil,
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
	if i.UpdatedAt != nil && i.UpdatedAt.IsZero() {
		return errors.New("invoice: invalid updatedAt")
	}
	return nil
}

// SetPaid updates paid and records UpdatedAt only when false -> true.
func (i *Invoice) SetPaid(paid bool, at time.Time) error {
	prev := i.Paid
	i.Paid = paid

	// ✅ false -> true の瞬間だけ updatedAt を入れる
	if !prev && paid {
		t := at.UTC()
		i.UpdatedAt = &t
	}

	return i.Validate()
}
