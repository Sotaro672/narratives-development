// backend/internal/domain/payment/entity.go
package payment

import (
	"errors"
	"strings"
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

// RefundStatus represents the Stripe Refund lifecycle separately from
// PaymentIntent lifecycle.
//
// Payment.Status remains succeeded after a refund because the original
// PaymentIntent succeeded. RefundStatus records the subsequent refund state.
type RefundStatus string

const (
	RefundStatusNone           RefundStatus = "none"
	RefundStatusPending        RefundStatus = "pending"
	RefundStatusRequiresAction RefundStatus = "requires_action"
	RefundStatusSucceeded      RefundStatus = "succeeded"
	RefundStatusFailed         RefundStatus = "failed"
	RefundStatusCanceled       RefundStatus = "canceled"
)

var AllowedRefundStatuses = map[RefundStatus]struct{}{
	RefundStatusNone:           {},
	RefundStatusPending:        {},
	RefundStatusRequiresAction: {},
	RefundStatusSucceeded:      {},
	RefundStatusFailed:         {},
	RefundStatusCanceled:       {},
}

var DefaultRefundStatus = RefundStatusNone

func IsValidRefundStatus(s RefundStatus) bool {
	if s == "" {
		return false
	}
	_, ok := AllowedRefundStatuses[s]
	return ok
}

// Payment is the application-side representation of a payment attempt/result.
//
// Firestore rule:
// - payment document ID is the same value as order document ID.
// - PaymentID represents that document ID.
// - paymentId itself is NOT stored as a field in the payment document.
//
// 正規のFirestore payment record schema:
//
//	amount
//	createdAt
//	paymentMethodId
//	status
//	stripeChargeId
//	stripeCustomerId
//	stripePaymentIntentId
//	stripePaymentMethodId
//	transferGroup
//	stripeRefundId
//	refundStatus
//	refundedAmount
//	refundedAt
//
// stripePaymentIntentId and transferGroup are required for every payment status,
// including pending.
//
// stripeChargeId is optional until Stripe has created a Charge.
// When available, it identifies the source transaction used by the later
// Stripe Connect settlement transfer.
//
// Refund state is independent from PaymentStatus. A successful refund keeps
// PaymentStatus=succeeded and records RefundStatus=succeeded instead.
//
// The current AMOL refund flow supports full refunds only. Therefore a
// successful refund must have RefundedAmount equal to Amount.
type Payment struct {
	// PaymentID is the Firestore payment document ID.
	// It must be the same value as order.ID.
	PaymentID string

	PaymentMethodID string

	StripeCustomerID      string
	StripePaymentMethodID string
	StripePaymentIntentID string
	StripeChargeID        string

	TransferGroup string

	Amount int
	Status PaymentStatus

	StripeRefundID string
	RefundStatus   RefundStatus
	RefundedAmount int
	RefundedAt     *time.Time

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
	ErrInvalidStripeChargeID      = errors.New("payment: invalid stripeChargeId")
	ErrInvalidStripeRefundID      = errors.New("payment: invalid stripeRefundId")
	ErrInvalidTransferGroup       = errors.New("payment: invalid transferGroup")
	ErrInvalidAmount              = errors.New("payment: invalid amount")
	ErrInvalidStatus              = errors.New("payment: invalid status")
	ErrInvalidRefundStatus        = errors.New("payment: invalid refund status")
	ErrInvalidRefundedAmount      = errors.New("payment: invalid refundedAmount")
	ErrInvalidRefundedAt          = errors.New("payment: invalid refundedAt")
	ErrRefundRequiresSucceeded    = errors.New("payment: refund requires succeeded payment")
	ErrInvalidRefundState         = errors.New("payment: invalid refund state")
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
//
// stripePaymentIntentID and transferGroup are required for every status,
// including pending.
//
// stripeChargeID may be empty until Stripe has created a Charge.
//
// A newly created Payment starts with RefundStatusNone. Refund state is applied
// later with SetRefundState after Stripe creates a Refund object.
func New(
	paymentID string,
	paymentMethodID string,
	stripeCustomerID string,
	stripePaymentMethodID string,
	stripePaymentIntentID string,
	stripeChargeID string,
	transferGroup string,
	amount int,
	status PaymentStatus,
	errorType *string,
	errorCode *string,
	errorMsg *string,
	createdAt time.Time,
) (Payment, error) {
	st := status
	if st == "" {
		st = DefaultStatus
	}

	p := Payment{
		PaymentID:             paymentID,
		PaymentMethodID:       paymentMethodID,
		StripeCustomerID:      stripeCustomerID,
		StripePaymentMethodID: stripePaymentMethodID,
		StripePaymentIntentID: stripePaymentIntentID,
		StripeChargeID:        stripeChargeID,
		TransferGroup:         transferGroup,
		Amount:                amount,
		Status:                st,
		StripeRefundID:        "",
		RefundStatus:          DefaultRefundStatus,
		RefundedAmount:        0,
		RefundedAt:            nil,
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
// stripePaymentIntentID and transferGroup are required for every status.
// stripeChargeID may be empty until Stripe has created a Charge.
func NewWithNow(
	paymentID string,
	paymentMethodID string,
	stripeCustomerID string,
	stripePaymentMethodID string,
	stripePaymentIntentID string,
	stripeChargeID string,
	transferGroup string,
	amount int,
	status PaymentStatus,
	errorType *string,
	errorCode *string,
	errorMsg *string,
	now time.Time,
) (Payment, error) {
	return New(
		paymentID,
		paymentMethodID,
		stripeCustomerID,
		stripePaymentMethodID,
		stripePaymentIntentID,
		stripeChargeID,
		transferGroup,
		amount,
		status,
		errorType,
		errorCode,
		errorMsg,
		now.UTC(),
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

	if p == nil {
		return ErrInvalidStatus
	}

	if p.RefundStatus != RefundStatusNone &&
		next != StatusSucceeded {
		return ErrRefundRequiresSucceeded
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

// SetStripeChargeID sets the Stripe Charge used as the source transaction
// for later Stripe Connect settlement transfers.
//
// An empty Charge ID is not accepted by this setter.
// The Payment field itself may remain empty until Stripe has created a Charge.
func (p *Payment) SetStripeChargeID(stripeChargeID string) error {
	if stripeChargeID == "" {
		return ErrInvalidStripeChargeID
	}
	p.StripeChargeID = stripeChargeID
	return nil
}

func (p *Payment) SetTransferGroup(transferGroup string) error {
	if transferGroup == "" {
		return ErrInvalidTransferGroup
	}
	p.TransferGroup = transferGroup
	return nil
}

func (p *Payment) SetAmount(amount int) error {
	if amount < MinAmount || (MaxAmount > 0 && amount > MaxAmount) {
		return ErrInvalidAmount
	}

	if p == nil {
		return ErrInvalidAmount
	}

	next := *p
	next.Amount = amount

	if err := next.validateRefundState(); err != nil {
		return err
	}

	p.Amount = amount
	return nil
}

// SetRefundState records the Stripe Refund lifecycle without changing the
// original PaymentIntent lifecycle represented by Payment.Status.
//
// The current AMOL flow supports full refunds only.
//
// State invariants:
//
// - none:
//
//	StripeRefundID="", RefundedAmount=0, RefundedAt=nil
//
// - pending / requires_action / failed / canceled:
//
//	StripeRefundID=re_..., RefundedAmount=0, RefundedAt=nil
//
// - succeeded:
//
//	StripeRefundID=re_..., RefundedAmount=Payment.Amount,
//	RefundedAt must be non-nil
//
// Any refund state other than none requires Payment.Status=succeeded.
func (p *Payment) SetRefundState(
	stripeRefundID string,
	refundStatus RefundStatus,
	refundedAmount int,
	refundedAt *time.Time,
) error {
	if p == nil {
		return ErrInvalidRefundState
	}

	next := *p

	next.StripeRefundID = stripeRefundID
	next.RefundStatus = refundStatus
	next.RefundedAmount = refundedAmount

	if refundedAt == nil {
		next.RefundedAt = nil
	} else {
		value := refundedAt.UTC()
		next.RefundedAt = &value
	}

	if err := next.validateRefundState(); err != nil {
		return err
	}

	p.StripeRefundID = next.StripeRefundID
	p.RefundStatus = next.RefundStatus
	p.RefundedAmount = next.RefundedAmount
	p.RefundedAt = next.RefundedAt

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

	// Stripe PaymentIntent must already exist before the payment entity
	// or Firestore payment document is created.
	//
	// This field is required for every status, including pending.
	if p.StripePaymentIntentID == "" {
		return ErrInvalidStripePaymentIntent
	}

	// StripeChargeID is intentionally optional here.
	// A Charge may not exist yet while the PaymentIntent is pending,
	// processing, or requires additional customer action.
	//
	// Once Stripe reports a Charge, SetStripeChargeID must be used to
	// persist the non-empty Charge ID.

	// transferGroup associates the platform PaymentIntent with the later
	// Stripe Connect Transfers created for this Order.
	if p.TransferGroup == "" {
		return ErrInvalidTransferGroup
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

	if err := p.validateRefundState(); err != nil {
		return err
	}

	return nil
}

func (p Payment) validateRefundState() error {
	if !IsValidRefundStatus(
		p.RefundStatus,
	) {
		return ErrInvalidRefundStatus
	}

	switch p.RefundStatus {
	case RefundStatusNone:
		if p.StripeRefundID != "" ||
			p.RefundedAmount != 0 ||
			p.RefundedAt != nil {
			return ErrInvalidRefundState
		}

		return nil

	case RefundStatusPending,
		RefundStatusRequiresAction,
		RefundStatusFailed,
		RefundStatusCanceled:
		if p.Status != StatusSucceeded {
			return ErrRefundRequiresSucceeded
		}

		if !isStripeRefundID(
			p.StripeRefundID,
		) {
			return ErrInvalidStripeRefundID
		}

		if p.RefundedAmount != 0 {
			return ErrInvalidRefundedAmount
		}

		if p.RefundedAt != nil {
			return ErrInvalidRefundedAt
		}

		return nil

	case RefundStatusSucceeded:
		if p.Status != StatusSucceeded {
			return ErrRefundRequiresSucceeded
		}

		if !isStripeRefundID(
			p.StripeRefundID,
		) {
			return ErrInvalidStripeRefundID
		}

		if p.Amount <= 0 ||
			p.RefundedAmount != p.Amount {
			return ErrInvalidRefundedAmount
		}

		if p.RefundedAt == nil ||
			p.RefundedAt.IsZero() {
			return ErrInvalidRefundedAt
		}

		if !p.CreatedAt.IsZero() &&
			p.RefundedAt.Before(
				p.CreatedAt,
			) {
			return ErrInvalidRefundedAt
		}

		return nil

	default:
		return ErrInvalidRefundStatus
	}
}

func isStripeRefundID(
	value string,
) bool {
	return strings.HasPrefix(
		value,
		"re_",
	) &&
		len(value) > len("re_")
}
