// backend/internal/domain/order/entity.go
package order

import (
	"errors"
	"time"
)

// ========================================
// Snapshot structs (stored in Order)
// ========================================

type ShippingSnapshot struct {
	ZipCode string `json:"zipCode"`
	State   string `json:"state"`
	City    string `json:"city"`
	Street  string `json:"street"`
	Street2 string `json:"street2"`
	Country string `json:"country"`
}

type PaymentMethodSnapshot struct {
	CustomerID     string `json:"customerId"`
	Brand          string `json:"brand"`
	Last4          string `json:"last4"`
	ExpMonth       int    `json:"expMonth"`
	ExpYear        int    `json:"expYear"`
	CardholderName string `json:"cardholderName"`
	IsDefault      bool   `json:"isDefault"`
}

// OrderItemType identifies what kind of item is stored in Order.Items.
type OrderItemType string

const (
	OrderItemTypeList   OrderItemType = "list"
	OrderItemTypeResale OrderItemType = "resale"
)

// OrderItemSnapshot is stored inside Order.Items.
//
// List item:
//   - type: "list"
//   - modelId, inventoryId, listId
//   - productBlueprintId, tokenBlueprintId
//   - qty, price
//
// Resale item:
//   - type: "resale"
//   - resaleId, productId
//   - productBlueprintId, tokenBlueprintId, brandId
//   - qty=1, price
//
// Transfer, cancellation, and dispatch state is maintained per item.
type OrderItemSnapshot struct {
	Type OrderItemType `json:"type"`

	// List item identifiers.
	ModelID     string `json:"modelId,omitempty"`
	InventoryID string `json:"inventoryId,omitempty"`
	ListID      string `json:"listId,omitempty"`

	// Resale item identifier.
	ResaleID string `json:"resaleId,omitempty"`

	// Product identifiers.
	ProductID          string `json:"productId,omitempty"`
	ProductBlueprintID string `json:"productBlueprintId,omitempty"`
	TokenBlueprintID   string `json:"tokenBlueprintId,omitempty"`
	BrandID            string `json:"brandId,omitempty"`

	Qty   int `json:"qty"`
	Price int `json:"price"`

	IsCanceled   bool `json:"isCanceled"`
	IsDispatched bool `json:"isDispatched"`

	Transferred   bool       `json:"transferred"`
	TransferredAt *time.Time `json:"transferredAt,omitempty"`
}

// ========================================
// Entity
// ========================================

type Order struct {
	ID       string `json:"id"`
	UserID   string `json:"userId"`
	AvatarID string `json:"avatarId"`
	CartID   string `json:"cartId"`

	ShippingSnapshot      ShippingSnapshot      `json:"shippingSnapshot"`
	PaymentMethodSnapshot PaymentMethodSnapshot `json:"paymentMethodSnapshot"`

	// Paid is maintained at the Order aggregate level.
	Paid bool `json:"paid"`

	Items     []OrderItemSnapshot `json:"items"`
	CreatedAt time.Time           `json:"createdAt"`
}

// OrderPatch represents partial updates to Order fields.
// A nil field means "no change".
type OrderPatch struct {
	UserID   *string
	AvatarID *string
	CartID   *string

	ShippingSnapshot      *ShippingSnapshot
	PaymentMethodSnapshot *PaymentMethodSnapshot

	Paid *bool

	Items *[]OrderItemSnapshot
}

// ========================================
// Errors
// ========================================

var (
	ErrInvalidID               = errors.New("order: invalid id")
	ErrInvalidUserID           = errors.New("order: invalid userId")
	ErrInvalidAvatarID         = errors.New("order: invalid avatarId")
	ErrInvalidCartID           = errors.New("order: invalid cartId")
	ErrInvalidShippingSnapshot = errors.New("order: invalid shippingSnapshot")
	ErrInvalidPaymentMethod    = errors.New("order: invalid paymentMethodSnapshot")
	ErrInvalidItems            = errors.New("order: invalid items")
	ErrInvalidCreatedAt        = errors.New("order: invalid createdAt")

	ErrInvalidItemSnapshot = errors.New("order: invalid item snapshot")
)

// ========================================
// Policy
// ========================================

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
// Behavior (mutators)
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

func (o *Order) UpdateItemCanceled(
	index int,
	isCanceled bool,
) error {
	if o == nil {
		return ErrInvalidItems
	}
	if index < 0 || index >= len(o.Items) {
		return ErrInvalidItems
	}

	o.Items[index].IsCanceled = isCanceled
	return nil
}

func (o *Order) UpdateItemDispatched(
	index int,
	isDispatched bool,
) error {
	if o == nil {
		return ErrInvalidItems
	}
	if index < 0 || index >= len(o.Items) {
		return ErrInvalidItems
	}

	o.Items[index].IsDispatched = isDispatched
	return nil
}

// UpdateItemTransferred updates item-level transfer state.
// transferred=true requires a non-zero transferredAt value.
func (o *Order) UpdateItemTransferred(
	index int,
	transferred bool,
	at time.Time,
) error {
	if o == nil {
		return ErrInvalidItems
	}
	if index < 0 || index >= len(o.Items) {
		return ErrInvalidItems
	}

	if transferred {
		if at.IsZero() {
			return ErrInvalidItemSnapshot
		}

		transferredAt := at.UTC()
		o.Items[index].Transferred = true
		o.Items[index].TransferredAt = &transferredAt
		return nil
	}

	o.Items[index].Transferred = false
	o.Items[index].TransferredAt = nil
	return nil
}

// ========================================
// Validation
// ========================================

// Validate verifies all invariants required for persisting an Order.
//
// Repository implementations must call Validate immediately before Create or
// Update so callers cannot bypass domain invariants by invoking the Repository
// directly.
func (o Order) Validate() error {
	if o.ID == "" {
		return ErrInvalidID
	}
	if o.UserID == "" {
		return ErrInvalidUserID
	}
	if o.AvatarID == "" {
		return ErrInvalidAvatarID
	}
	if o.CartID == "" {
		return ErrInvalidCartID
	}
	if err := validateShippingSnapshot(
		o.ShippingSnapshot,
	); err != nil {
		return err
	}
	if err := validatePaymentMethodSnapshot(
		o.PaymentMethodSnapshot,
	); err != nil {
		return err
	}
	if err := validateItems(o.Items); err != nil {
		return err
	}
	if o.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}

	return nil
}

func validateShippingSnapshot(
	s ShippingSnapshot,
) error {
	if s.State == "" {
		return ErrInvalidShippingSnapshot
	}
	if s.City == "" {
		return ErrInvalidShippingSnapshot
	}
	if s.Street == "" {
		return ErrInvalidShippingSnapshot
	}
	if s.Country == "" {
		return ErrInvalidShippingSnapshot
	}

	return nil
}

func validatePaymentMethodSnapshot(
	p PaymentMethodSnapshot,
) error {
	if p.CustomerID == "" {
		return ErrInvalidPaymentMethod
	}
	if p.Brand == "" {
		return ErrInvalidPaymentMethod
	}
	if p.Last4 == "" {
		return ErrInvalidPaymentMethod
	}
	if p.ExpMonth < 1 || p.ExpMonth > 12 {
		return ErrInvalidPaymentMethod
	}
	if p.ExpYear < 2000 || p.ExpYear > 9999 {
		return ErrInvalidPaymentMethod
	}
	if p.CardholderName == "" {
		return ErrInvalidPaymentMethod
	}

	return nil
}

func validateItems(items []OrderItemSnapshot) error {
	if len(items) < MinItemsRequired {
		return ErrInvalidItems
	}

	for _, item := range items {
		if err := validateItemSnapshot(item); err != nil {
			return err
		}
	}

	return nil
}

func validateItemSnapshot(
	item OrderItemSnapshot,
) error {
	switch item.Type {
	case OrderItemTypeList:
		return validateListItemSnapshot(item)

	case OrderItemTypeResale:
		return validateResaleItemSnapshot(item)

	default:
		return ErrInvalidItemSnapshot
	}
}

func validateListItemSnapshot(
	item OrderItemSnapshot,
) error {
	if item.ModelID == "" {
		return ErrInvalidItemSnapshot
	}
	if item.InventoryID == "" {
		return ErrInvalidItemSnapshot
	}
	if item.ListID == "" {
		return ErrInvalidItemSnapshot
	}
	if item.ProductBlueprintID == "" {
		return ErrInvalidItemSnapshot
	}
	if item.TokenBlueprintID == "" {
		return ErrInvalidItemSnapshot
	}

	// Resale-only identifiers must not be mixed into a list item.
	if item.ResaleID != "" ||
		item.ProductID != "" ||
		item.BrandID != "" {
		return ErrInvalidItemSnapshot
	}

	if item.Qty <= 0 {
		return ErrInvalidItemSnapshot
	}
	if item.Price < 0 {
		return ErrInvalidItemSnapshot
	}

	return validateItemTransferState(item)
}

func validateResaleItemSnapshot(
	item OrderItemSnapshot,
) error {
	if item.ResaleID == "" {
		return ErrInvalidItemSnapshot
	}
	if item.ProductID == "" {
		return ErrInvalidItemSnapshot
	}
	if item.ProductBlueprintID == "" {
		return ErrInvalidItemSnapshot
	}
	if item.TokenBlueprintID == "" {
		return ErrInvalidItemSnapshot
	}
	if item.BrandID == "" {
		return ErrInvalidItemSnapshot
	}

	// List-only identifiers must not be mixed into a resale item.
	if item.ModelID != "" ||
		item.InventoryID != "" ||
		item.ListID != "" {
		return ErrInvalidItemSnapshot
	}

	if item.Qty != 1 {
		return ErrInvalidItemSnapshot
	}
	if item.Price < 0 {
		return ErrInvalidItemSnapshot
	}

	return validateItemTransferState(item)
}

func validateItemTransferState(
	item OrderItemSnapshot,
) error {
	if item.Transferred {
		if item.TransferredAt == nil ||
			item.TransferredAt.IsZero() {
			return ErrInvalidItemSnapshot
		}

		return nil
	}

	if item.TransferredAt != nil {
		return ErrInvalidItemSnapshot
	}

	return nil
}
