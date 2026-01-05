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
//
// ✅ docId = invoiceId を採用するため:
// - ID(=docId用) / UpdatedAt / DeletedAt を削除
type Payment struct {
	InvoiceID        string
	BillingAddressID string
	Amount           int
	Status           PaymentStatus
	ErrorType        *string
	CreatedAt        time.Time
}

// Errors
var (
	ErrInvalidInvoiceID        = errors.New("payment: invalid invoiceId")
	ErrInvalidBillingAddressID = errors.New("payment: invalid billingAddressId")
	ErrInvalidAmount           = errors.New("payment: invalid amount")
	ErrInvalidStatus           = errors.New("payment: invalid status")
	ErrInvalidErrorType        = errors.New("payment: invalid errorType")
	ErrInvalidCreatedAt        = errors.New("payment: invalid createdAt")
)

// Policy
var (
	MinAmount = 0 // inclusive; set to 1 if required
	MaxAmount = 0 // 0 disables upper bound
)

// Constructors

func New(
	invoiceID, billingAddressID string,
	amount int,
	status PaymentStatus,
	errorType *string,
	createdAt time.Time,
) (Payment, error) {
	p := Payment{
		InvoiceID:        strings.TrimSpace(invoiceID),
		BillingAddressID: strings.TrimSpace(billingAddressID),
		Amount:           amount,
		Status:           status,
		ErrorType:        normalizePtr(errorType),
		CreatedAt:        createdAt.UTC(),
	}
	if err := p.validate(); err != nil {
		return Payment{}, err
	}
	return p, nil
}

func NewWithNow(
	invoiceID, billingAddressID string,
	amount int,
	status PaymentStatus,
	errorType *string,
	now time.Time,
) (Payment, error) {
	now = now.UTC()
	return New(invoiceID, billingAddressID, amount, status, errorType, now)
}

func NewFromStringTimes(
	invoiceID, billingAddressID string,
	amount int,
	status PaymentStatus,
	errorType *string,
	createdAtStr string,
) (Payment, error) {
	ct, err := parseTime(createdAtStr, ErrInvalidCreatedAt)
	if err != nil {
		return Payment{}, err
	}
	return New(invoiceID, billingAddressID, amount, status, errorType, ct)
}

// Behavior

func (p *Payment) SetStatus(next PaymentStatus) error {
	if !IsValidStatus(next) {
		return ErrInvalidStatus
	}
	p.Status = next
	return nil
}

func (p *Payment) SetErrorType(errType *string) error {
	et := normalizePtr(errType)
	// if explicitly provided empty string, it becomes nil (cleared)
	p.ErrorType = et
	return nil
}

// Validation

func (p Payment) validate() error {
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
