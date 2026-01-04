// backend/internal/domain/order/entity.go
package order

import (
	"errors"
	"strings"
	"time"
)

// ========================================
// Snapshot structs (stored in Order)
// ========================================

type ShippingSnapshot struct {
	ZipCode string
	State   string
	City    string
	Street  string
	Street2 string
	Country string
}

type BillingSnapshot struct {
	Last4          string
	CardHolderName string
}

// OrderItemSnapshot is stored inside Order.Items.
// Expectation: items are NOT split by listId, and each item is
// [modelId, inventoryId, qty, price].
type OrderItemSnapshot struct {
	ModelID     string `json:"modelId"`
	InventoryID string `json:"inventoryId"`
	Qty         int    `json:"qty"`
	Price       int    `json:"price"`
}

// ========================================
// Entity
// ========================================

type Order struct {
	ID     string
	UserID string
	CartID string

	ShippingSnapshot ShippingSnapshot
	BillingSnapshot  BillingSnapshot // last4 + cardHolderName only

	Items          []OrderItemSnapshot `json:"items"`
	InvoiceID      string
	PaymentID      string
	TransferedDate *time.Time // note: mirrors TS: transferedDate
	CreatedAt      time.Time
	UpdatedAt      time.Time
	UpdatedBy      *string
}

// OrderPatch represents partial updates to Order fields.
// A nil field means "no change".
type OrderPatch struct {
	UserID *string
	CartID *string

	ShippingSnapshot *ShippingSnapshot
	BillingSnapshot  *BillingSnapshot

	Items          *[]OrderItemSnapshot
	InvoiceID      *string
	PaymentID      *string
	TransferedDate *time.Time
	UpdatedBy      *string
}

// ========================================
// Errors
// ========================================

var (
	ErrInvalidID              = errors.New("order: invalid id")
	ErrInvalidUserID          = errors.New("order: invalid userId")
	ErrInvalidCartID          = errors.New("order: invalid cartId")
	ErrInvalidShippingAddress = errors.New("order: invalid shippingSnapshot")
	ErrInvalidBillingAddress  = errors.New("order: invalid billingSnapshot")
	ErrInvalidItems           = errors.New("order: invalid items")
	ErrInvalidInvoiceID       = errors.New("order: invalid invoiceId")
	ErrInvalidPaymentID       = errors.New("order: invalid paymentId")
	ErrInvalidTransferredDate = errors.New("order: invalid transferredDate")
	ErrInvalidCreatedAt       = errors.New("order: invalid createdAt")
	ErrInvalidUpdatedAt       = errors.New("order: invalid updatedAt")
	ErrInvalidUpdatedBy       = errors.New("order: invalid updatedBy")

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
	cartID string,
	shippingSnapshot ShippingSnapshot,
	billingSnapshot BillingSnapshot,
	items []OrderItemSnapshot,
	invoiceID string,
	paymentID string,
	transferedDate *time.Time,
	createdAt time.Time,
	updatedAt time.Time,
	updatedBy *string,
) (Order, error) {
	o := Order{
		ID:     strings.TrimSpace(id),
		UserID: strings.TrimSpace(userID),
		CartID: strings.TrimSpace(cartID),

		ShippingSnapshot: normalizeShippingSnapshot(shippingSnapshot),
		BillingSnapshot:  normalizeBillingSnapshot(billingSnapshot),

		Items:          normalizeItems(items),
		InvoiceID:      strings.TrimSpace(invoiceID),
		PaymentID:      strings.TrimSpace(paymentID),
		TransferedDate: normalizeTimePtr(transferedDate),
		CreatedAt:      createdAt.UTC(),
		UpdatedAt:      updatedAt.UTC(),
		UpdatedBy:      normalizePtr(updatedBy),
	}
	if err := o.validate(); err != nil {
		return Order{}, err
	}
	return o, nil
}

// ========================================
// Behavior (mutators)
// ========================================

func (o *Order) Touch(now time.Time) error {
	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}
	o.UpdatedAt = now.UTC()
	return nil
}

// New name to mirror TS field "transferedDate"
func (o *Order) SetTransfered(at time.Time, now time.Time) error {
	if at.IsZero() {
		return ErrInvalidTransferredDate
	}
	utc := at.UTC()
	o.TransferedDate = &utc
	return o.Touch(now)
}

// Backward-compat method name removed from app layer, but keep as alias if referenced.
func (o *Order) SetTransferred(at time.Time, now time.Time) error {
	return o.SetTransfered(at, now)
}

func (o *Order) ReplaceItems(items []OrderItemSnapshot, now time.Time) error {
	ns := normalizeItems(items)
	if err := validateItems(ns); err != nil {
		return err
	}
	o.Items = ns
	return o.Touch(now)
}

// ✅ Replace AddressID update with Snapshot update
func (o *Order) UpdateShippingSnapshot(s ShippingSnapshot, now time.Time) error {
	s = normalizeShippingSnapshot(s)
	if err := validateShippingSnapshot(s); err != nil {
		return err
	}
	o.ShippingSnapshot = s
	return o.Touch(now)
}

func (o *Order) UpdateBillingSnapshot(b BillingSnapshot, now time.Time) error {
	b = normalizeBillingSnapshot(b)
	if err := validateBillingSnapshot(b); err != nil {
		return err
	}
	o.BillingSnapshot = b
	return o.Touch(now)
}

func (o *Order) UpdateInvoice(invoiceID string, now time.Time) error {
	invoiceID = strings.TrimSpace(invoiceID)
	if invoiceID == "" {
		return ErrInvalidInvoiceID
	}
	o.InvoiceID = invoiceID
	return o.Touch(now)
}

func (o *Order) UpdatePayment(paymentID string, now time.Time) error {
	paymentID = strings.TrimSpace(paymentID)
	if paymentID == "" {
		return ErrInvalidPaymentID
	}
	o.PaymentID = paymentID
	return o.Touch(now)
}

// ========================================
// Validation
// ========================================

func (o Order) validate() error {
	if o.ID == "" {
		return ErrInvalidID
	}
	if o.UserID == "" {
		return ErrInvalidUserID
	}
	if o.CartID == "" {
		return ErrInvalidCartID
	}
	if err := validateShippingSnapshot(o.ShippingSnapshot); err != nil {
		return err
	}
	if err := validateBillingSnapshot(o.BillingSnapshot); err != nil {
		return err
	}
	if err := validateItems(o.Items); err != nil {
		return err
	}
	if o.InvoiceID == "" {
		return ErrInvalidInvoiceID
	}
	if o.PaymentID == "" {
		return ErrInvalidPaymentID
	}
	if o.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if o.UpdatedAt.IsZero() {
		return ErrInvalidUpdatedAt
	}
	if o.UpdatedAt.Before(o.CreatedAt) {
		return ErrInvalidUpdatedAt
	}
	if o.TransferedDate != nil && (o.TransferedDate.IsZero() || o.TransferedDate.Before(o.CreatedAt)) {
		return ErrInvalidTransferredDate
	}
	if o.UpdatedBy != nil && strings.TrimSpace(*o.UpdatedBy) == "" {
		return ErrInvalidUpdatedBy
	}
	return nil
}

func validateShippingSnapshot(s ShippingSnapshot) error {
	if strings.TrimSpace(s.State) == "" {
		return ErrInvalidShippingAddress
	}
	if strings.TrimSpace(s.City) == "" {
		return ErrInvalidShippingAddress
	}
	if strings.TrimSpace(s.Street) == "" {
		return ErrInvalidShippingAddress
	}
	if strings.TrimSpace(s.Country) == "" {
		return ErrInvalidShippingAddress
	}
	return nil
}

func validateBillingSnapshot(b BillingSnapshot) error {
	last4 := strings.TrimSpace(b.Last4)
	if last4 == "" {
		return ErrInvalidBillingAddress
	}
	// cardHolderName は任意（空でもOK）
	return nil
}

func validateItems(items []OrderItemSnapshot) error {
	if len(items) < MinItemsRequired {
		return ErrInvalidItems
	}
	for _, it := range items {
		if strings.TrimSpace(it.ModelID) == "" {
			return ErrInvalidItemSnapshot
		}
		if strings.TrimSpace(it.InventoryID) == "" {
			return ErrInvalidItemSnapshot
		}
		if it.Qty <= 0 {
			return ErrInvalidItemSnapshot
		}
		if it.Price < 0 {
			return ErrInvalidItemSnapshot
		}
	}
	return nil
}

// ========================================
// Helpers
// ========================================

func normalizeShippingSnapshot(s ShippingSnapshot) ShippingSnapshot {
	s.ZipCode = strings.TrimSpace(s.ZipCode)
	s.State = strings.TrimSpace(s.State)
	s.City = strings.TrimSpace(s.City)
	s.Street = strings.TrimSpace(s.Street)
	s.Street2 = strings.TrimSpace(s.Street2)
	s.Country = strings.TrimSpace(s.Country)
	return s
}

func normalizeBillingSnapshot(b BillingSnapshot) BillingSnapshot {
	b.Last4 = strings.TrimSpace(b.Last4)
	b.CardHolderName = strings.TrimSpace(b.CardHolderName)
	return b
}

func normalizeItems(items []OrderItemSnapshot) []OrderItemSnapshot {
	out := make([]OrderItemSnapshot, 0, len(items))
	for _, it := range items {
		n := OrderItemSnapshot{
			ModelID:     strings.TrimSpace(it.ModelID),
			InventoryID: strings.TrimSpace(it.InventoryID),
			Qty:         it.Qty,
			Price:       it.Price,
		}
		// 空は validateItems で弾くのでここでは落とさない
		out = append(out, n)
	}
	return out
}

func normalizePtr(p *string) *string {
	if p == nil {
		return nil
	}
	v := strings.TrimSpace(*p)
	if v == "" {
		return nil
	}
	return &v
}

func normalizeTimePtr(p *time.Time) *time.Time {
	if p == nil {
		return nil
	}
	if p.IsZero() {
		return nil
	}
	utc := p.UTC()
	return &utc
}
