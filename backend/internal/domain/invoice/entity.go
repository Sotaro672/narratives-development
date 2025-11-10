// backend\internal\domain\invoice\entity.go
package invoice

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Invoice mirrors web-app/src/shared/types/invoice.ts
// all amounts are integers. currency is a 3-letter code (e.g., JPY).
type Invoice struct {
	OrderID           string             `json:"orderId"`
	OrderItemInvoices []OrderItemInvoice `json:"orderItemInvoices"`
	Subtotal          int                `json:"subtotal"`
	DiscountAmount    int                `json:"discountAmount"`
	TaxAmount         int                `json:"taxAmount"`
	ShippingCost      int                `json:"shippingCost"`
	TotalAmount       int                `json:"totalAmount"`
	Currency          string             `json:"currency"`
	CreatedAt         time.Time          `json:"createdAt"`
	UpdatedAt         time.Time          `json:"updatedAt"`
	BillingAddressID  string             `json:"billingAddressId"`
}

// OrderItemInvoice mirrors web-app/src/shared/types/invoice.ts (OrderItemInvoice)
type OrderItemInvoice struct {
	ID          string    `json:"id"`
	OrderItemID string    `json:"orderItemId"`
	UnitPrice   int       `json:"unitPrice"`
	TotalPrice  int       `json:"totalPrice"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// InvoiceStatus mirrors TS: 'unpaid' | 'paid' | 'refunded'
type InvoiceStatus string

const (
	StatusUnpaid   InvoiceStatus = "unpaid"
	StatusPaid     InvoiceStatus = "paid"
	StatusRefunded InvoiceStatus = "refunded"
)

func IsValidStatus(s InvoiceStatus) bool {
	switch s {
	case StatusUnpaid, StatusPaid, StatusRefunded:
		return true
	default:
		return false
	}
}

// Common domain errors
var ErrPriceNotFound = errors.New("price not found")

// Errors
var (
	ErrInvalidAmount           = errors.New("invoice: invalid amount")
	ErrInvalidTotal            = errors.New("invoice: invalid totalAmount (does not match components)")
	ErrInvalidOrderID          = errors.New("invoice: invalid orderId")
	ErrInvalidBillingAddressID = errors.New("invoice: invalid billingAddressId")
	ErrInvalidCurrency         = errors.New("invoice: invalid currency")
	ErrInvalidID               = errors.New("invoice: invalid id")
	ErrInvalidOrderItemID      = errors.New("invoice: invalid orderItemId")
	ErrInvalidCreatedAt        = errors.New("invoice: invalid createdAt")
	ErrInvalidUpdatedAt        = errors.New("invoice: invalid updatedAt")
)

// Policy (align with invoiceConstants.ts as needed)
var (
	// Amount bounds; set Max* to 0 to disable upper-checks.
	MinMoney = 0
	MaxMoney = 0

	// Enforce that TotalAmount == Subtotal - DiscountAmount + TaxAmount + ShippingCost
	EnforceTotalEquality = true

	// Currency code policy (nil disables strict check)
	currencyRe = regexp.MustCompile(`^[A-Z]{3}$`)
)

// ================================
// Invoice constructors and methods
// ================================

// NewInvoice constructs a full Invoice aligned with the TS interface.
func NewInvoice(
	orderID string,
	orderItemInvoices []OrderItemInvoice,
	subtotal, discount, tax, shipping, total int,
	currency string,
	createdAt, updatedAt time.Time,
	billingAddressID string,
) (Invoice, error) {
	inv := Invoice{
		OrderID:           strings.TrimSpace(orderID),
		OrderItemInvoices: append([]OrderItemInvoice(nil), orderItemInvoices...),
		Subtotal:          subtotal,
		DiscountAmount:    discount,
		TaxAmount:         tax,
		ShippingCost:      shipping,
		TotalAmount:       total,
		Currency:          strings.TrimSpace(strings.ToUpper(currency)),
		CreatedAt:         createdAt.UTC(),
		UpdatedAt:         updatedAt.UTC(),
		BillingAddressID:  strings.TrimSpace(billingAddressID),
	}
	if err := inv.validate(); err != nil {
		return Invoice{}, err
	}
	return inv, nil
}

func (i *Invoice) ComputeTotal() int {
	return i.Subtotal - i.DiscountAmount + i.TaxAmount + i.ShippingCost
}

func (i *Invoice) RecalculateTotal() {
	i.TotalAmount = i.ComputeTotal()
}

func (i *Invoice) UpdateComponents(subtotal, discount, tax, shipping int, recalc bool) error {
	i.Subtotal, i.DiscountAmount, i.TaxAmount, i.ShippingCost = subtotal, discount, tax, shipping
	if recalc {
		i.RecalculateTotal()
	}
	return i.validate()
}

func (i *Invoice) Touch(now time.Time) error {
	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}
	now = now.UTC()
	if !i.CreatedAt.IsZero() && now.Before(i.CreatedAt) {
		return ErrInvalidUpdatedAt
	}
	i.UpdatedAt = now
	return nil
}

func (i Invoice) validate() error {
	// required strings
	if strings.TrimSpace(i.OrderID) == "" {
		return ErrInvalidOrderID
	}
	if strings.TrimSpace(i.BillingAddressID) == "" {
		return ErrInvalidBillingAddressID
	}
	if strings.TrimSpace(i.Currency) == "" || (currencyRe != nil && !currencyRe.MatchString(i.Currency)) {
		return ErrInvalidCurrency
	}

	// times
	if i.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if i.UpdatedAt.IsZero() || i.UpdatedAt.Before(i.CreatedAt) {
		return ErrInvalidUpdatedAt
	}

	// money components
	if !moneyOK(i.Subtotal) || !moneyOK(i.DiscountAmount) || !moneyOK(i.TaxAmount) || !moneyOK(i.ShippingCost) || !moneyOK(i.TotalAmount) {
		return ErrInvalidAmount
	}
	if EnforceTotalEquality && i.TotalAmount != i.ComputeTotal() {
		return ErrInvalidTotal
	}

	// validate order item invoices
	for _, oi := range i.OrderItemInvoices {
		if err := oi.validate(); err != nil {
			return err
		}
	}

	return nil
}

// =======================================
// OrderItemInvoice constructors/mutators
// =======================================

func NewOrderItemInvoice(
	id, orderItemID string,
	unitPrice, totalPrice int,
	createdAt, updatedAt time.Time,
) (OrderItemInvoice, error) {
	oi := OrderItemInvoice{
		ID:          strings.TrimSpace(id),
		OrderItemID: strings.TrimSpace(orderItemID),
		UnitPrice:   unitPrice,
		TotalPrice:  totalPrice,
		CreatedAt:   createdAt.UTC(),
		UpdatedAt:   updatedAt.UTC(),
	}
	if err := oi.validate(); err != nil {
		return OrderItemInvoice{}, err
	}
	return oi, nil
}

func NewOrderItemInvoiceFromStringTimes(
	id, orderItemID string,
	unitPrice, totalPrice int,
	createdAt, updatedAt string,
) (OrderItemInvoice, error) {
	ct, err := parseTime(createdAt, ErrInvalidCreatedAt)
	if err != nil {
		return OrderItemInvoice{}, err
	}
	ut, err := parseTime(updatedAt, ErrInvalidUpdatedAt)
	if err != nil {
		return OrderItemInvoice{}, err
	}
	return NewOrderItemInvoice(id, orderItemID, unitPrice, totalPrice, ct, ut)
}

func (o *OrderItemInvoice) SetUnitPrice(v int) error {
	if !moneyOK(v) {
		return ErrInvalidAmount
	}
	o.UnitPrice = v
	return o.touch(time.Now().UTC())
}

func (o *OrderItemInvoice) SetTotalPrice(v int) error {
	if !moneyOK(v) {
		return ErrInvalidAmount
	}
	o.TotalPrice = v
	return o.touch(time.Now().UTC())
}

func (o *OrderItemInvoice) ReassignOrderItem(orderItemID string) error {
	orderItemID = strings.TrimSpace(orderItemID)
	if orderItemID == "" {
		return ErrInvalidOrderItemID
	}
	o.OrderItemID = orderItemID
	return o.touch(time.Now().UTC())
}

func (o *OrderItemInvoice) touch(now time.Time) error {
	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}
	o.UpdatedAt = now.UTC()
	return nil
}

func (o OrderItemInvoice) validate() error {
	if o.ID == "" {
		return ErrInvalidID
	}
	if o.OrderItemID == "" {
		return ErrInvalidOrderItemID
	}
	if !moneyOK(o.UnitPrice) || !moneyOK(o.TotalPrice) {
		return ErrInvalidAmount
	}
	if o.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if o.UpdatedAt.IsZero() || o.UpdatedAt.Before(o.CreatedAt) {
		return ErrInvalidUpdatedAt
	}
	return nil
}

// ============
// Helpers
// ============

func moneyOK(v int) bool {
	if v < MinMoney {
		return false
	}
	if MaxMoney > 0 && v > MaxMoney {
		return false
	}
	return true
}

func parseTime(s string, classify error) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, classify
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), nil
	}
	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("%w: cannot parse %q", classify, s)
}

// =======================================
// Backward-compat helpers (optional)
// =======================================

// ZeroInvoice returns an Invoice with zero amounts; other required fields are empty.
// Note: Not validated; meant for scaffolding or tests.
func ZeroInvoice() Invoice {
	return Invoice{
		OrderID:           "",
		OrderItemInvoices: nil,
		Subtotal:          0,
		DiscountAmount:    0,
		TaxAmount:         0,
		ShippingCost:      0,
		TotalAmount:       0,
		Currency:          "",
		CreatedAt:         time.Time{},
		UpdatedAt:         time.Time{},
		BillingAddressID:  "",
	}
}

// ========================================
// SQL DDL
// ========================================
const InvoicesTableDDL = `
-- Migration: Initialize Invoice domain
-- Mirrors backend/internal/domain/invoice/entity.go

BEGIN;

CREATE TABLE IF NOT EXISTS invoices (
  order_id           TEXT        PRIMARY KEY,
  subtotal           INTEGER     NOT NULL CHECK (subtotal >= 0),
  discount_amount    INTEGER     NOT NULL CHECK (discount_amount >= 0),
  tax_amount         INTEGER     NOT NULL CHECK (tax_amount >= 0),
  shipping_cost      INTEGER     NOT NULL CHECK (shipping_cost >= 0),
  total_amount       INTEGER     NOT NULL CHECK (total_amount >= 0),
  currency           TEXT        NOT NULL,
  created_at         TIMESTAMPTZ NOT NULL,
  updated_at         TIMESTAMPTZ NOT NULL,
  billing_address_id TEXT        NOT NULL,

  -- Basic validations
  CONSTRAINT chk_invoices_currency_len CHECK (char_length(currency) = 3),
  CONSTRAINT chk_invoices_time_order CHECK (updated_at >= created_at),

  -- Keep total consistency with domain rule:
  CONSTRAINT chk_invoices_total_consistency
    CHECK (total_amount = subtotal - discount_amount + tax_amount + shipping_cost)
);

-- Order item level invoices
CREATE TABLE IF NOT EXISTS order_item_invoices (
  id            TEXT        PRIMARY KEY,
  order_item_id TEXT        NOT NULL,
  unit_price    INTEGER     NOT NULL CHECK (unit_price >= 0),
  total_price   INTEGER     NOT NULL CHECK (total_price >= 0),
  created_at    TIMESTAMPTZ NOT NULL,
  updated_at    TIMESTAMPTZ NOT NULL,

  -- Optional linkage to invoices (nullable to avoid strict domain coupling)
  order_id      TEXT        NULL REFERENCES invoices(order_id) ON DELETE CASCADE,

  CONSTRAINT chk_order_item_invoices_time_order CHECK (updated_at >= created_at)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_invoices_currency            ON invoices(currency);
CREATE INDEX IF NOT EXISTS idx_invoices_created_at          ON invoices(created_at);
CREATE INDEX IF NOT EXISTS idx_invoices_updated_at          ON invoices(updated_at);
CREATE INDEX IF NOT EXISTS idx_invoices_billing_address_id  ON invoices(billing_address_id);

CREATE INDEX IF NOT EXISTS idx_order_item_invoices_item_id  ON order_item_invoices(order_item_id);
CREATE INDEX IF NOT EXISTS idx_order_item_invoices_order_id ON order_item_invoices(order_id);

COMMIT;
`
