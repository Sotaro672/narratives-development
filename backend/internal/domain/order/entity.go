// backend/internal/domain/order/entity.go
package order

import (
	"errors"
	"time"
)

// ========================================
// Entity
// ========================================

type Order struct {
	ID       string `json:"id"`
	UserID   string `json:"userId"`
	AvatarID string `json:"avatarId"`
	CartID   string `json:"cartId"`

	ShippingSnapshot ShippingSnapshot `json:"shippingSnapshot"`

	ShippingQuoteSnapshot ShippingQuoteSnapshot `json:"shippingQuoteSnapshot"`

	PaymentMethodSnapshot PaymentMethodSnapshot `json:"paymentMethodSnapshot"`

	// Paid is maintained at the Order aggregate level.
	Paid bool `json:"paid"`

	Items     []OrderItemSnapshot `json:"items"`
	CreatedAt time.Time           `json:"createdAt"`
}

// ========================================
// Errors
// ========================================

var (
	ErrInvalidID                = errors.New("order: invalid id")
	ErrInvalidUserID            = errors.New("order: invalid userId")
	ErrInvalidAvatarID          = errors.New("order: invalid avatarId")
	ErrInvalidCartID            = errors.New("order: invalid cartId")
	ErrInvalidShippingSnapshot  = errors.New("order: invalid shippingSnapshot")
	ErrInvalidShippingQuote     = errors.New("order: invalid shippingQuoteSnapshot")
	ErrInvalidShippingQuoteItem = errors.New(
		"order: invalid shippingQuoteItemSnapshot",
	)
	ErrInvalidPaymentMethod = errors.New(
		"order: invalid paymentMethodSnapshot",
	)
	ErrInvalidItems     = errors.New("order: invalid items")
	ErrInvalidCreatedAt = errors.New("order: invalid createdAt")

	ErrInvalidItemSnapshot   = errors.New("order: invalid item snapshot")
	ErrInvalidSellerSnapshot = errors.New("order: invalid sellerSnapshot")
)

// ========================================
// Policy
// ========================================

const (
	ShippingQuoteCurrencyJPY = "JPY"

	ConsumptionTaxRateReduced  = 8
	ConsumptionTaxRateStandard = 10
)

var (
	MinItemsRequired = 1
)

// ========================================
// Constructors
// ========================================

func New(
	id string,
	userID string,
	avatarID string,
	cartID string,
	shippingSnapshot ShippingSnapshot,
	shippingQuoteSnapshot ShippingQuoteSnapshot,
	paymentMethodSnapshot PaymentMethodSnapshot,
	items []OrderItemSnapshot,
	createdAt time.Time,
) (Order, error) {
	o := Order{
		ID:                    id,
		UserID:                userID,
		AvatarID:              avatarID,
		CartID:                cartID,
		ShippingSnapshot:      shippingSnapshot,
		ShippingQuoteSnapshot: shippingQuoteSnapshot,
		PaymentMethodSnapshot: paymentMethodSnapshot,
		Paid:                  false,
		Items:                 items,
		CreatedAt:             createdAt.UTC(),
	}

	if err := o.Validate(); err != nil {
		return Order{}, err
	}

	return o, nil
}

// ========================================
// Aggregate behavior
// ========================================

func (o *Order) ReplaceItems(items []OrderItemSnapshot) error {
	if err := validateItems(items); err != nil {
		return err
	}

	o.Items = items
	return nil
}

func (o *Order) UpdateShippingSnapshot(
	s ShippingSnapshot,
) error {
	if err := validateShippingSnapshot(s); err != nil {
		return err
	}

	o.ShippingSnapshot = s
	return nil
}

func (o *Order) UpdateShippingQuoteSnapshot(
	s ShippingQuoteSnapshot,
) error {
	if err := validateShippingQuoteSnapshot(s); err != nil {
		return err
	}

	o.ShippingQuoteSnapshot =
		ShippingQuoteSnapshot{
			Items: append(
				[]ShippingQuoteItemSnapshot(nil),
				s.Items...,
			),
			Amount:   s.Amount,
			Currency: s.Currency,
		}

	return nil
}

func (o *Order) UpdatePaymentMethodSnapshot(
	p PaymentMethodSnapshot,
) error {
	if err := validatePaymentMethodSnapshot(p); err != nil {
		return err
	}

	o.PaymentMethodSnapshot = p
	return nil
}

func (o *Order) UpdateAvatarID(avatarID string) error {
	if avatarID == "" {
		return ErrInvalidAvatarID
	}

	o.AvatarID = avatarID
	return nil
}

func (o *Order) UpdatePaid(paid bool) {
	o.Paid = paid
}
