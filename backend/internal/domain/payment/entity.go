// backend/internal/domain/payment/entity.go
package payment

import (
	"errors"
	"fmt"
	"strings"
	"time"

	domcommon "narratives/internal/domain/common"
)

// PaymentStatus (mirror TS)
type PaymentStatus string

// Optional policy: if empty, any non-empty status is accepted.
var AllowedStatuses = map[PaymentStatus]struct{}{}

// ✅ Front に「status を必須にしない」方針に合わせて、未指定時はこの値に寄せる
const (
	StatusPending PaymentStatus = "pending"
)

var DefaultStatus = StatusPending

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
//
// ✅ Front に合わせる変更点:
// - BillingAddressID は「チェックアウト開始前は未確定」になり得るため任意（空を許可）
type Payment struct {
	InvoiceID        string
	BillingAddressID string // optional
	Amount           int
	Status           PaymentStatus
	ErrorType        *string
	CreatedAt        time.Time
}

// Errors
var (
	ErrInvalidInvoiceID = errors.New("payment: invalid invoiceId")
	ErrInvalidAmount    = errors.New("payment: invalid amount")
	ErrInvalidStatus    = errors.New("payment: invalid status")
	ErrInvalidErrorType = errors.New("payment: invalid errorType")
	ErrInvalidCreatedAt = errors.New("payment: invalid createdAt")
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
	invoiceID = strings.TrimSpace(invoiceID)
	billingAddressID = strings.TrimSpace(billingAddressID)

	// ✅ status はフロント必須にしない：未指定ならデフォルトを入れる
	st := status
	if strings.TrimSpace(string(st)) == "" {
		st = DefaultStatus
	}

	p := Payment{
		InvoiceID:        invoiceID,
		BillingAddressID: billingAddressID, // empty allowed
		Amount:           amount,
		Status:           st,
		ErrorType:        domcommon.NormalizeStringPtr(errorType),
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
	ct, err := domcommon.ParseTime(createdAtStr)
	if err != nil {
		return Payment{}, fmt.Errorf("%w: %v", ErrInvalidCreatedAt, err)
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

// ✅ checkout 後に確定できるように setter を用意（空も許可）
func (p *Payment) SetBillingAddressID(billingAddressID string) {
	p.BillingAddressID = strings.TrimSpace(billingAddressID)
}

func (p *Payment) SetAmount(amount int) error {
	if amount < MinAmount || (MaxAmount > 0 && amount > MaxAmount) {
		return ErrInvalidAmount
	}
	p.Amount = amount
	return nil
}

func (p *Payment) SetErrorType(errType *string) error {
	et := domcommon.NormalizeStringPtr(errType)
	// if explicitly provided empty string, it becomes nil (cleared)
	p.ErrorType = et
	return nil
}

// Validation

func (p Payment) validate() error {
	if p.InvoiceID == "" {
		return ErrInvalidInvoiceID
	}

	// ✅ BillingAddressID は空を許可（フロント必須にしない）
	// if p.BillingAddressID == "" { return ErrInvalidBillingAddressID }

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
