// backend\internal\domain\payment\entity.go
package payment

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// PaymentStatus (mirror TS)
type PaymentStatus string

// Optional policy: if empty, any non-empty status is accepted.
var AllowedStatuses = map[PaymentStatus]struct{}{}

func IsValidStatus(s PaymentStatus) bool {
	if s == "" {
		return false
	}
	if len(AllowedStatuses) == 0 {
		return true
	}
	_, ok := AllowedStatuses[s]
	return ok
}

// Entity (mirror TS Payment)
type Payment struct {
	ID               string
	InvoiceID        string
	BillingAddressID string
	Amount           int
	Status           PaymentStatus
	ErrorType        *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        *time.Time
}

// Errors
var (
	ErrInvalidID               = errors.New("payment: invalid id")
	ErrInvalidInvoiceID        = errors.New("payment: invalid invoiceId")
	ErrInvalidBillingAddressID = errors.New("payment: invalid billingAddressId")
	ErrInvalidAmount           = errors.New("payment: invalid amount")
	ErrInvalidStatus           = errors.New("payment: invalid status")
	ErrInvalidErrorType        = errors.New("payment: invalid errorType")
	ErrInvalidCreatedAt        = errors.New("payment: invalid createdAt")
	ErrInvalidUpdatedAt        = errors.New("payment: invalid updatedAt")
	ErrInvalidDeletedAt        = errors.New("payment: invalid deletedAt")
)

// Policy
var (
	MinAmount = 0 // inclusive; set to 1 if required
	MaxAmount = 0 // 0 disables upper bound
)

// Constructors

func New(
	id, invoiceID, billingAddressID string,
	amount int,
	status PaymentStatus,
	errorType *string,
	createdAt, updatedAt time.Time,
	deletedAt *time.Time,
) (Payment, error) {
	p := Payment{
		ID:               strings.TrimSpace(id),
		InvoiceID:        strings.TrimSpace(invoiceID),
		BillingAddressID: strings.TrimSpace(billingAddressID),
		Amount:           amount,
		Status:           status,
		ErrorType:        normalizePtr(errorType),
		CreatedAt:        createdAt.UTC(),
		UpdatedAt:        updatedAt.UTC(),
		DeletedAt:        normalizeTimePtr(deletedAt),
	}
	if err := p.validate(); err != nil {
		return Payment{}, err
	}
	return p, nil
}

func NewWithNow(
	id, invoiceID, billingAddressID string,
	amount int,
	status PaymentStatus,
	errorType *string,
	now time.Time,
) (Payment, error) {
	now = now.UTC()
	return New(id, invoiceID, billingAddressID, amount, status, errorType, now, now, nil)
}

func NewFromStringTimes(
	id, invoiceID, billingAddressID string,
	amount int,
	status PaymentStatus,
	errorType *string,
	createdAtStr, updatedAtStr string,
	deletedAtStr *string,
) (Payment, error) {
	ct, err := parseTime(createdAtStr, ErrInvalidCreatedAt)
	if err != nil {
		return Payment{}, err
	}
	ut, err := parseTime(updatedAtStr, ErrInvalidUpdatedAt)
	if err != nil {
		return Payment{}, err
	}
	var dt *time.Time
	if deletedAtStr != nil && strings.TrimSpace(*deletedAtStr) != "" {
		t, err := parseTime(*deletedAtStr, ErrInvalidDeletedAt)
		if err != nil {
			return Payment{}, err
		}
		dt = &t
	}
	return New(id, invoiceID, billingAddressID, amount, status, errorType, ct, ut, dt)
}

// Behavior

func (p *Payment) Touch(now time.Time) error {
	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}
	p.UpdatedAt = now.UTC()
	return nil
}

func (p *Payment) SetStatus(next PaymentStatus, now time.Time) error {
	if !IsValidStatus(next) {
		return ErrInvalidStatus
	}
	p.Status = next
	return p.Touch(now)
}

func (p *Payment) SetErrorType(errType *string, now time.Time) error {
	et := normalizePtr(errType)
	// if explicitly provided empty string, it becomes nil (cleared)
	p.ErrorType = et
	return p.Touch(now)
}

func (p *Payment) MarkDeleted(at time.Time, now time.Time) error {
	if at.IsZero() {
		return ErrInvalidDeletedAt
	}
	utc := at.UTC()
	p.DeletedAt = &utc
	return p.Touch(now)
}

// Validation

func (p Payment) validate() error {
	if p.ID == "" {
		return ErrInvalidID
	}
	if p.InvoiceID == "" {
		return ErrInvalidInvoiceID
	}
	if p.BillingAddressID == "" {
		return ErrInvalidBillingAddressID
	}
	if p.Amount < MinAmount || (MaxAmount > 0 && p.Amount > MaxAmount) {
		return ErrInvalidAmount
	}
	if !IsValidStatus(p.Status) {
		return ErrInvalidStatus
	}
	if p.ErrorType != nil && strings.TrimSpace(*p.ErrorType) == "" {
		return ErrInvalidErrorType
	}
	if p.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if p.UpdatedAt.IsZero() || p.UpdatedAt.Before(p.CreatedAt) {
		return ErrInvalidUpdatedAt
	}
	if p.DeletedAt != nil && (p.DeletedAt.IsZero() || p.DeletedAt.Before(p.CreatedAt)) {
		return ErrInvalidDeletedAt
	}
	return nil
}

// Helpers

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
const PaymentsTableDDL = `
-- Migration: Initialize payments table (mirrors domain/payment/entity.go)

BEGIN;

CREATE TABLE IF NOT EXISTS payments (
  id                 TEXT        PRIMARY KEY,
  invoice_id         TEXT        NOT NULL,
  billing_address_id TEXT        NOT NULL,
  amount             INTEGER     NOT NULL,
  status             TEXT        NOT NULL,
  error_type         TEXT        NULL,
  created_at         TIMESTAMPTZ NOT NULL,
  updated_at         TIMESTAMPTZ NOT NULL,
  deleted_at         TIMESTAMPTZ NULL,

  -- Basic non-empty checks
  CONSTRAINT chk_payments_non_empty CHECK (
    char_length(trim(id)) > 0
    AND char_length(trim(invoice_id)) > 0
    AND char_length(trim(billing_address_id)) > 0
  ),

  -- Amount policy (MinAmount = 0)
  CONSTRAINT chk_payments_amount CHECK (amount >= 0),

  -- Status must be non-empty (enum is open in domain)
  CONSTRAINT chk_payments_status_non_empty CHECK (char_length(trim(status)) > 0),

  -- error_type optional but if present must be non-empty
  CONSTRAINT chk_payments_error_type_non_empty CHECK (
    error_type IS NULL OR char_length(trim(error_type)) > 0
  ),

  -- Time order coherence
  CONSTRAINT chk_payments_time_order CHECK (
    updated_at >= created_at
    AND (deleted_at IS NULL OR deleted_at >= created_at)
  )
);

-- Useful indexes
CREATE INDEX IF NOT EXISTS idx_payments_invoice_id   ON payments(invoice_id);
CREATE INDEX IF NOT EXISTS idx_payments_status       ON payments(status);
CREATE INDEX IF NOT EXISTS idx_payments_billing_id   ON payments(billing_address_id);
CREATE INDEX IF NOT EXISTS idx_payments_amount       ON payments(amount);
CREATE INDEX IF NOT EXISTS idx_payments_created_at   ON payments(created_at);
CREATE INDEX IF NOT EXISTS idx_payments_updated_at   ON payments(updated_at);
CREATE INDEX IF NOT EXISTS idx_payments_deleted_at   ON payments(deleted_at);

COMMIT;
`
