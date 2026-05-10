// backend/internal/domain/payment/entity.go
package payment

import (
	"errors"
	"time"
)

// PaymentStatus represents the application-side payment state mirrored from Stripe lifecycle.
type PaymentStatus string

const (
	StatusPending        PaymentStatus = "pending"
	StatusRequiresAction PaymentStatus = "requires_action"
	StatusProcessing     PaymentStatus = "processing"
	StatusSucceeded      PaymentStatus = "succeeded"
	StatusFailed         PaymentStatus = "failed"
	StatusCanceled       PaymentStatus = "canceled"
)

var AllowedStatuses = map[PaymentStatus]struct{}{
	StatusPending:        {},
	StatusRequiresAction: {},
	StatusProcessing:     {},
	StatusSucceeded:      {},
	StatusFailed:         {},
	StatusCanceled:       {},
}

var DefaultStatus = StatusPending

func IsValidStatus(s PaymentStatus) bool {
	if s == "" {
		return false
	}
	_, ok := AllowedStatuses[s]
	return ok
}

// Payment is the application-side representation of a payment attempt/result.
//
// Firestore rule:
// - payment document ID is the same value as order document ID.
// - PaymentID represents that document ID.
// - paymentId itself is NOT stored as a field in the payment document.
//
// 正の Firestore payment record schema:
//
//	amount
//	createdAt
//	paymentMethodId
//	status
//	stripeCustomerId
//	stripePaymentIntentId
//	stripePaymentMethodId
type Payment struct {
	// PaymentID is the Firestore payment document ID.
	// It must be the same value as order.ID.
	PaymentID string

	PaymentMethodID string

	StripeCustomerID      string
	StripePaymentMethodID string
	StripePaymentIntentID string

	Amount int
	Status PaymentStatus

	ErrorType *string
	ErrorCode *string
	ErrorMsg  *string

	CreatedAt time.Time
}

// Errors
var (
	ErrInvalidPaymentID           = errors.New("payment: invalid paymentId")
	ErrInvalidPaymentMethodID     = errors.New("payment: invalid paymentMethodId")
	ErrInvalidStripeCustomerID    = errors.New("payment: invalid stripeCustomerId")
	ErrInvalidStripePaymentMethod = errors.New("payment: invalid stripePaymentMethodId")
	ErrInvalidStripePaymentIntent = errors.New("payment: invalid stripePaymentIntentId")
	ErrInvalidAmount              = errors.New("payment: invalid amount")
	ErrInvalidStatus              = errors.New("payment: invalid status")
	ErrInvalidErrorType           = errors.New("payment: invalid errorType")
	ErrInvalidErrorCode           = errors.New("payment: invalid errorCode")
	ErrInvalidErrorMsg            = errors.New("payment: invalid errorMsg")
	ErrInvalidCreatedAt           = errors.New("payment: invalid createdAt")
)

// Policy
var (
	MinAmount = 0 // inclusive; set to 1 if required
	MaxAmount = 0 // 0 disables upper bound
)

// Constructors

// New creates a Payment.
//
// paymentID must be the same value as order.ID.
// The value is used as the Firestore payment document ID.
func New(
	paymentID string,
	paymentMethodID string,
	stripeCustomerID string,
	stripePaymentMethodID string,
	stripePaymentIntentID string,
	amount int,
	status PaymentStatus,
	errorType *string,
	errorCode *string,
	errorMsg *string,
	createdAt time.Time,
) (Payment, error) {
	st := status
	if string(st) == "" {
		st = DefaultStatus
	}

	p := Payment{
		PaymentID:             paymentID,
		PaymentMethodID:       paymentMethodID,
		StripeCustomerID:      stripeCustomerID,
		StripePaymentMethodID: stripePaymentMethodID,
		StripePaymentIntentID: stripePaymentIntentID,
		Amount:                amount,
		Status:                st,
		ErrorType:             errorType,
		ErrorCode:             errorCode,
		ErrorMsg:              errorMsg,
		CreatedAt:             createdAt.UTC(),
	}
	if err := p.validate(); err != nil {
		return Payment{}, err
	}
	return p, nil
}

// NewWithNow creates a Payment with the provided current time.
//
// paymentID must be the same value as order.ID.
func NewWithNow(
	paymentID string,
	paymentMethodID string,
	stripeCustomerID string,
	stripePaymentMethodID string,
	stripePaymentIntentID string,
	amount int,
	status PaymentStatus,
	errorType *string,
	errorCode *string,
	errorMsg *string,
	now time.Time,
) (Payment, error) {
	now = now.UTC()
	return New(
		paymentID,
		paymentMethodID,
		stripeCustomerID,
		stripePaymentMethodID,
		stripePaymentIntentID,
		amount,
		status,
		errorType,
		errorCode,
		errorMsg,
		now,
	)
}

// Behavior

func (p *Payment) SetPaymentID(paymentID string) error {
	if paymentID == "" {
		return ErrInvalidPaymentID
	}
	p.PaymentID = paymentID
	return nil
}

func (p *Payment) SetStatus(next PaymentStatus) error {
	if !IsValidStatus(next) {
		return ErrInvalidStatus
	}
	p.Status = next
	return nil
}

func (p *Payment) SetPaymentMethodID(paymentMethodID string) error {
	if paymentMethodID == "" {
		return ErrInvalidPaymentMethodID
	}
	p.PaymentMethodID = paymentMethodID
	return nil
}

func (p *Payment) SetStripeCustomerID(stripeCustomerID string) error {
	if stripeCustomerID == "" {
		return ErrInvalidStripeCustomerID
	}
	p.StripeCustomerID = stripeCustomerID
	return nil
}

func (p *Payment) SetStripePaymentMethodID(stripePaymentMethodID string) error {
	if stripePaymentMethodID == "" {
		return ErrInvalidStripePaymentMethod
	}
	p.StripePaymentMethodID = stripePaymentMethodID
	return nil
}

func (p *Payment) SetStripePaymentIntentID(stripePaymentIntentID string) error {
	if stripePaymentIntentID == "" {
		return ErrInvalidStripePaymentIntent
	}
	p.StripePaymentIntentID = stripePaymentIntentID
	return nil
}

func (p *Payment) SetAmount(amount int) error {
	if amount < MinAmount || (MaxAmount > 0 && amount > MaxAmount) {
		return ErrInvalidAmount
	}
	p.Amount = amount
	return nil
}

func (p *Payment) SetErrorType(errType *string) error {
	if errType != nil && *errType == "" {
		return ErrInvalidErrorType
	}
	p.ErrorType = errType
	return nil
}

func (p *Payment) SetErrorCode(errCode *string) error {
	if errCode != nil && *errCode == "" {
		return ErrInvalidErrorCode
	}
	p.ErrorCode = errCode
	return nil
}

func (p *Payment) SetErrorMsg(errMsg *string) error {
	if errMsg != nil && *errMsg == "" {
		return ErrInvalidErrorMsg
	}
	p.ErrorMsg = errMsg
	return nil
}

// Validation

func (p Payment) validate() error {
	if p.PaymentID == "" {
		return ErrInvalidPaymentID
	}

	if p.PaymentMethodID == "" {
		return ErrInvalidPaymentMethodID
	}

	if p.StripeCustomerID == "" {
		return ErrInvalidStripeCustomerID
	}

	if p.StripePaymentMethodID == "" {
		return ErrInvalidStripePaymentMethod
	}

	if p.Amount < MinAmount || (MaxAmount > 0 && p.Amount > MaxAmount) {
		return ErrInvalidAmount
	}

	if !IsValidStatus(p.Status) {
		return ErrInvalidStatus
	}

	if p.ErrorType != nil && *p.ErrorType == "" {
		return ErrInvalidErrorType
	}
	if p.ErrorCode != nil && *p.ErrorCode == "" {
		return ErrInvalidErrorCode
	}
	if p.ErrorMsg != nil && *p.ErrorMsg == "" {
		return ErrInvalidErrorMsg
	}

	if p.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if p.StripePaymentIntentID == "" {
		return ErrInvalidStripePaymentIntent
	}
	return nil
}
