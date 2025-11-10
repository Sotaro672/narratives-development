// backend/internal/domain/order/entity.go
package order

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ========================================
// Enums (mirror TS)
// ========================================

type LegacyOrderStatus string // persisted legacy status

const (
	// Legacy statuses
	LegacyPaid        LegacyOrderStatus = "paid"
	LegacyTransferred LegacyOrderStatus = "transferred"
)

func IsValidLegacyStatus(s LegacyOrderStatus) bool {
	switch s {
	case LegacyPaid, LegacyTransferred:
		return true
	default:
		return false
	}
}

// ========================================
// Entity (mirror TS Order)
// ========================================

type Order struct {
	ID                string
	OrderNumber       string
	Status            LegacyOrderStatus
	UserID            string
	ShippingAddressID string
	BillingAddressID  string
	ListID            string
	Items             []string `json:"items"`
	InvoiceID         string
	PaymentID         string
	FulfillmentID     string
	TrackingID        *string
	TransferedDate    *time.Time // note: mirrors TS: transferedDate
	CreatedAt         time.Time
	UpdatedAt         time.Time
	UpdatedBy         *string
	DeletedAt         *time.Time
	DeletedBy         *string
}

// OrderPatch represents partial updates to Order fields.
// A nil field means "no change".
type OrderPatch struct {
	OrderNumber       *string
	Status            *LegacyOrderStatus
	UserID            *string
	ShippingAddressID *string
	BillingAddressID  *string
	ListID            *string
	Items             *[]string
	InvoiceID         *string
	PaymentID         *string
	FulfillmentID     *string
	TrackingID        *string
	TransferedDate    *time.Time
	UpdatedBy         *string
	DeletedAt         *time.Time
	DeletedBy         *string
}

// ========================================
// Errors
// ========================================

var (
	ErrInvalidID              = errors.New("order: invalid id")
	ErrInvalidOrderNumber     = errors.New("order: invalid orderNumber")
	ErrInvalidStatus          = errors.New("order: invalid status")
	ErrInvalidUserID          = errors.New("order: invalid userId")
	ErrInvalidShippingAddress = errors.New("order: invalid shippingAddressId")
	ErrInvalidBillingAddress  = errors.New("order: invalid billingAddressId")
	ErrInvalidListID          = errors.New("order: invalid listId")
	ErrInvalidItems           = errors.New("order: invalid items")
	ErrInvalidInvoiceID       = errors.New("order: invalid invoiceId")
	ErrInvalidPaymentID       = errors.New("order: invalid paymentId")
	ErrInvalidFulfillmentID   = errors.New("order: invalid fulfillmentId")
	ErrInvalidTrackingID      = errors.New("order: invalid trackingId")
	ErrInvalidTransferredDate = errors.New("order: invalid transferredDate")
	ErrInvalidCreatedAt       = errors.New("order: invalid createdAt")
	ErrInvalidUpdatedAt       = errors.New("order: invalid updatedAt")
	ErrInvalidUpdatedBy       = errors.New("order: invalid updatedBy")
	ErrInvalidDeletedAt       = errors.New("order: invalid deletedAt")
	ErrInvalidDeletedBy       = errors.New("order: invalid deletedBy")
	ErrInvalidItemID          = errors.New("order: invalid item id")
)

// ========================================
// Policy (align with orderConstants.ts if any)
// ========================================

var (
	// Example order number format; loosen as needed. Empty regex disables format check.
	orderNumberRe    = regexp.MustCompile(`^[A-Z0-9\-]{1,32}$`)
	MinItemsRequired = 1
)

// ========================================
// Constructors
// ========================================

func New(
	id string,
	orderNumber string,
	status LegacyOrderStatus,
	userID string,
	shippingAddressID string,
	billingAddressID string,
	listID string,
	items []string,
	invoiceID string,
	paymentID string,
	fulfillmentID string,
	trackingID *string,
	transferedDate *time.Time,
	createdAt time.Time,
	updatedAt time.Time,
	updatedBy *string,
	deletedAt *time.Time,
	deletedBy *string,
) (Order, error) {
	o := Order{
		ID:                strings.TrimSpace(id),
		OrderNumber:       strings.TrimSpace(orderNumber),
		Status:            status,
		UserID:            strings.TrimSpace(userID),
		ShippingAddressID: strings.TrimSpace(shippingAddressID),
		BillingAddressID:  strings.TrimSpace(billingAddressID),
		ListID:            strings.TrimSpace(listID),
		Items:             dedupTrim(items),
		InvoiceID:         strings.TrimSpace(invoiceID),
		PaymentID:         strings.TrimSpace(paymentID),
		FulfillmentID:     strings.TrimSpace(fulfillmentID),
		TrackingID:        normalizePtr(trackingID),
		TransferedDate:    normalizeTimePtr(transferedDate),
		CreatedAt:         createdAt.UTC(),
		UpdatedAt:         updatedAt.UTC(),
		UpdatedBy:         normalizePtr(updatedBy),
		DeletedAt:         normalizeTimePtr(deletedAt),
		DeletedBy:         normalizePtr(deletedBy),
	}
	if err := o.validate(); err != nil {
		return Order{}, err
	}
	return o, nil
}

func NewFromStringTimes(
	id string,
	orderNumber string,
	status LegacyOrderStatus,
	userID string,
	shippingAddressID string,
	billingAddressID string,
	listID string,
	items []string,
	invoiceID string,
	paymentID string,
	fulfillmentID string,
	trackingID *string,
	transferedDateStr *string,
	createdAtStr string,
	updatedAtStr string,
	updatedBy *string,
	deletedAtStr *string,
	deletedBy *string,
) (Order, error) {
	ca, err := parseTime(createdAtStr, ErrInvalidCreatedAt)
	if err != nil {
		return Order{}, err
	}
	ua, err := parseTime(updatedAtStr, ErrInvalidUpdatedAt)
	if err != nil {
		return Order{}, err
	}
	var td, dd *time.Time
	if transferedDateStr != nil && strings.TrimSpace(*transferedDateStr) != "" {
		t, err := parseTime(*transferedDateStr, ErrInvalidTransferredDate)
		if err != nil {
			return Order{}, err
		}
		td = &t
	}
	if deletedAtStr != nil && strings.TrimSpace(*deletedAtStr) != "" {
		t, err := parseTime(*deletedAtStr, ErrInvalidDeletedAt)
		if err != nil {
			return Order{}, err
		}
		dd = &t
	}
	return New(
		id, orderNumber, status,
		userID, shippingAddressID, billingAddressID, listID,
		items, invoiceID, paymentID, fulfillmentID,
		trackingID, td,
		ca, ua,
		updatedBy, dd, deletedBy,
	)
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

func (o *Order) SetLegacyStatus(s LegacyOrderStatus, now time.Time) error {
	if !IsValidLegacyStatus(s) {
		return ErrInvalidStatus
	}
	o.Status = s
	return o.Touch(now)
}

func (o *Order) SetTracking(id *string, now time.Time) error {
	if id != nil {
		ni := normalizePtr(id)
		if ni != nil && *ni == "" {
			return ErrInvalidTrackingID
		}
		o.TrackingID = ni
	}
	return o.Touch(now)
}

func (o *Order) ReplaceItems(items []string, now time.Time) error {
	ds := dedupTrim(items)
	if err := validateItems(ds); err != nil {
		return err
	}
	o.Items = ds
	return o.Touch(now)
}

func (o *Order) AddItem(itemID string, now time.Time) error {
	itemID = strings.TrimSpace(itemID)
	if itemID == "" {
		return ErrInvalidItems
	}
	if !contains(o.Items, itemID) {
		o.Items = append(o.Items, itemID)
	}
	return o.Touch(now)
}

func (o *Order) RemoveItem(itemID string, now time.Time) error {
	itemID = strings.TrimSpace(itemID)
	if itemID == "" {
		return ErrInvalidItems
	}
	out := o.Items[:0]
	for _, it := range o.Items {
		if it != itemID {
			out = append(out, it)
		}
	}
	o.Items = out
	return o.Touch(now)
}

func (o *Order) UpdateShippingAddress(shippingID string, now time.Time) error {
	shippingID = strings.TrimSpace(shippingID)
	if shippingID == "" {
		return ErrInvalidShippingAddress
	}
	o.ShippingAddressID = shippingID
	return o.Touch(now)
}

func (o *Order) UpdateBillingAddress(billingID string, now time.Time) error {
	billingID = strings.TrimSpace(billingID)
	if billingID == "" {
		return ErrInvalidBillingAddress
	}
	o.BillingAddressID = billingID
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

func (o *Order) UpdateFulfillment(fulfillmentID string, now time.Time) error {
	fulfillmentID = strings.TrimSpace(fulfillmentID)
	if fulfillmentID == "" {
		return ErrInvalidFulfillmentID
	}
	o.FulfillmentID = fulfillmentID
	return o.Touch(now)
}

// New name to mirror TS field "edDate"
func (o *Order) SetTransfered(at time.Time, now time.Time) error {
	if at.IsZero() {
		return ErrInvalidTransferredDate
	}
	utc := at.UTC()
	o.TransferedDate = &utc
	o.Status = LegacyTransferred
	return o.Touch(now)
}

// Backward-compat: keep old method name delegating to new one
func (o *Order) SetTransferred(at time.Time, now time.Time) error {
	return o.SetTransfered(at, now)
}

// ========================================
// Validation
// ========================================

func (o Order) validate() error {
	if o.ID == "" {
		return ErrInvalidID
	}
	if o.OrderNumber == "" {
		return ErrInvalidOrderNumber
	}
	if orderNumberRe != nil && !orderNumberRe.MatchString(o.OrderNumber) {
		return ErrInvalidOrderNumber
	}
	if !IsValidLegacyStatus(o.Status) {
		return ErrInvalidStatus
	}
	if o.UserID == "" {
		return ErrInvalidUserID
	}
	if o.ShippingAddressID == "" {
		return ErrInvalidShippingAddress
	}
	if o.BillingAddressID == "" {
		return ErrInvalidBillingAddress
	}
	if o.ListID == "" {
		return ErrInvalidListID
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
	if o.FulfillmentID == "" {
		return ErrInvalidFulfillmentID
	}
	if o.TrackingID != nil && strings.TrimSpace(*o.TrackingID) == "" {
		return ErrInvalidTrackingID
	}
	if o.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if o.UpdatedAt.IsZero() {
		return ErrInvalidUpdatedAt
	}
	// Time order
	if o.UpdatedAt.Before(o.CreatedAt) {
		return ErrInvalidUpdatedAt
	}
	if o.TransferedDate != nil && (o.TransferedDate.IsZero() || o.TransferedDate.Before(o.CreatedAt)) {
		return ErrInvalidTransferredDate
	}
	// UpdatedBy optional but if set must be non-empty
	if o.UpdatedBy != nil && strings.TrimSpace(*o.UpdatedBy) == "" {
		return ErrInvalidUpdatedBy
	}
	// Deleted pair coherence
	if (o.DeletedAt == nil) != (o.DeletedBy == nil) {
		if o.DeletedAt == nil {
			return ErrInvalidDeletedAt
		}
		return ErrInvalidDeletedBy
	}
	if o.DeletedBy != nil && strings.TrimSpace(*o.DeletedBy) == "" {
		return ErrInvalidDeletedBy
	}
	if o.DeletedAt != nil && o.DeletedAt.Before(o.CreatedAt) {
		return ErrInvalidDeletedAt
	}
	// validate で Items が orderItem の主キー(ID)として妥当か検証します（空文字は不可）。
	for _, id := range o.Items {
		if strings.TrimSpace(id) == "" {
			return ErrInvalidItemID
		}
	}
	return nil
}

// ========================================
// Helpers
// ========================================

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

func dedupTrim(xs []string) []string {
	seen := make(map[string]struct{}, len(xs))
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		x = strings.TrimSpace(x)
		if x == "" {
			continue
		}
		if _, ok := seen[x]; ok {
			continue
		}
		seen[x] = struct{}{}
		out = append(out, x)
	}
	return out
}

func contains(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

func validateItems(items []string) error {
	if len(items) < MinItemsRequired {
		return ErrInvalidItems
	}
	for _, it := range items {
		if strings.TrimSpace(it) == "" {
			return ErrInvalidItems
		}
	}
	seen := make(map[string]struct{}, len(items))
	for _, it := range items {
		if _, ok := seen[it]; ok {
			return ErrInvalidItems
		}
		seen[it] = struct{}{}
	}
	return nil
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

// ========================================
// SQL DDL
// ========================================
const OrdersTableDDL = `
-- Migration: Initialize orders table (mirrors domain/order/entity.go)

BEGIN;

CREATE TABLE IF NOT EXISTS orders (
  id                   TEXT        PRIMARY KEY,
  order_number         TEXT        NOT NULL,
  status               TEXT        NOT NULL CHECK (status IN ('paid','transferred')),
  user_id              TEXT        NOT NULL,
  shipping_address_id  TEXT        NOT NULL,
  billing_address_id   TEXT        NOT NULL,
  list_id              TEXT        NOT NULL,
  items                JSONB       NOT NULL DEFAULT '[]'::jsonb,  -- array of item ids
  invoice_id           TEXT        NOT NULL,
  payment_id           TEXT        NOT NULL,
  fulfillment_id       TEXT        NOT NULL,
  tracking_id          TEXT        NULL,
  transfered_date      TIMESTAMPTZ NULL,                          -- note: TS field uses this spelling
  created_at           TIMESTAMPTZ NOT NULL,
  updated_at           TIMESTAMPTZ NOT NULL,
  updated_by           TEXT        NULL,
  deleted_at           TIMESTAMPTZ NULL,
  deleted_by           TEXT        NULL,

  -- Basic non-empty checks
  CONSTRAINT chk_orders_non_empty CHECK (
    char_length(trim(id)) > 0
    AND char_length(trim(order_number)) > 0
    AND char_length(trim(user_id)) > 0
    AND char_length(trim(shipping_address_id)) > 0
    AND char_length(trim(billing_address_id)) > 0
    AND char_length(trim(list_id)) > 0
    AND char_length(trim(invoice_id)) > 0
    AND char_length(trim(payment_id)) > 0
    AND char_length(trim(fulfillment_id)) > 0
  ),

  -- items must be a JSON array with at least one element
  CONSTRAINT chk_orders_items_array CHECK (
    jsonb_typeof(items) = 'array' AND jsonb_array_length(items) >= 1
  ),

  -- Time order coherence
  CONSTRAINT chk_orders_time_order CHECK (
    updated_at >= created_at
    AND (transfered_date IS NULL OR transfered_date >= created_at)
    AND (deleted_at IS NULL OR deleted_at >= created_at)
  ),

  -- UpdatedBy/DeletedBy coherence
  CONSTRAINT chk_orders_updated_by_non_empty CHECK (
    updated_by IS NULL OR char_length(trim(updated_by)) > 0
  ),
  CONSTRAINT chk_orders_deleted_pair CHECK (
    (deleted_at IS NULL AND deleted_by IS NULL)
    OR (deleted_at IS NOT NULL AND deleted_by IS NOT NULL)
  )
);

-- Useful indexes
CREATE UNIQUE INDEX IF NOT EXISTS uq_orders_order_number ON orders(order_number);
CREATE INDEX IF NOT EXISTS idx_orders_status            ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_user_id           ON orders(user_id);
CREATE INDEX IF NOT EXISTS idx_orders_transfered_date  ON orders(transfered_date);
CREATE INDEX IF NOT EXISTS idx_orders_created_at        ON orders(created_at);
CREATE INDEX IF NOT EXISTS idx_orders_updated_at        ON orders(updated_at);
CREATE INDEX IF NOT EXISTS idx_orders_deleted_at        ON orders(deleted_at);

COMMIT;
`
