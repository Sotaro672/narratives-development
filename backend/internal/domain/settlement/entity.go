// backend/internal/domain/settlement/entity.go
package settlement

import (
	"errors"
	"strings"
	"time"
)

// SettlementStatus represents the seller-side Stripe Connect settlement state.
//
// Settlement is separate from:
//
// - Payment: purchaser -> AMOL Stripe Platform payment
// - Order item Transferred: Solana token ownership transfer
//
// Settlement represents:
//
// AMOL Stripe Platform -> Stripe Connected Account
type SettlementStatus string

const (
	StatusPending         SettlementStatus = "pending"
	StatusReady           SettlementStatus = "ready"
	StatusTransferring    SettlementStatus = "transferring"
	StatusTransferred     SettlementStatus = "transferred"
	StatusFailedRetryable SettlementStatus = "failed_retryable"
	StatusFailed          SettlementStatus = "failed"
	StatusCanceled        SettlementStatus = "canceled"
	StatusReversed        SettlementStatus = "reversed"
)

var AllowedStatuses = map[SettlementStatus]struct{}{
	StatusPending:         {},
	StatusReady:           {},
	StatusTransferring:    {},
	StatusTransferred:     {},
	StatusFailedRetryable: {},
	StatusFailed:          {},
	StatusCanceled:        {},
	StatusReversed:        {},
}

const (
	CurrencyJPY = "JPY"
)

var DefaultStatus = StatusPending

// Settlement represents the amount attributable to one seller Stripe Account
// inside one Order.
//
// One Order may therefore have multiple Settlement records.
//
// Example:
//
// Order
//   - Account A -> Settlement A
//   - Account B -> Settlement B
//
// Multiple Brands that use the same AccountID must be aggregated into the same
// Settlement before this entity is created.
//
// Amount invariant:
//
//	GrossAmount = PlatformFeeAmount + TransferAmount
//
// StripeTransferID is the Stripe Connect Transfer ID and is unrelated to
// Solana token transfer.
type Settlement struct {
	ID string

	OrderID   string
	PaymentID string

	CompanyID string
	AccountID string

	StripeAccountID string

	StripePaymentIntentID string
	StripeChargeID        string
	StripeTransferID      string

	StripeTransferReversalID string

	TransferGroup string

	GrossAmount       int
	PlatformFeeAmount int
	TransferAmount    int

	Currency string

	Status SettlementStatus

	ErrorType *string
	ErrorCode *string
	ErrorMsg  *string

	CreatedAt time.Time
	UpdatedAt time.Time

	TransferredAt *time.Time
	ReversedAt    *time.Time
}

// Errors

var (
	ErrInvalidID = errors.New(
		"settlement: invalid id",
	)
	ErrInvalidOrderID = errors.New(
		"settlement: invalid orderId",
	)
	ErrInvalidPaymentID = errors.New(
		"settlement: invalid paymentId",
	)
	ErrPaymentOrderMismatch = errors.New(
		"settlement: paymentId does not match orderId",
	)
	ErrInvalidCompanyID = errors.New(
		"settlement: invalid companyId",
	)
	ErrInvalidAccountID = errors.New(
		"settlement: invalid accountId",
	)
	ErrInvalidStripeAccountID = errors.New(
		"settlement: invalid stripeAccountId",
	)
	ErrInvalidStripePaymentIntentID = errors.New(
		"settlement: invalid stripePaymentIntentId",
	)
	ErrInvalidStripeChargeID = errors.New(
		"settlement: invalid stripeChargeId",
	)
	ErrInvalidStripeTransferID = errors.New(
		"settlement: invalid stripeTransferId",
	)
	ErrInvalidStripeTransferReversalID = errors.New(
		"settlement: invalid stripeTransferReversalId",
	)
	ErrInvalidTransferGroup = errors.New(
		"settlement: invalid transferGroup",
	)
	ErrInvalidGrossAmount = errors.New(
		"settlement: invalid grossAmount",
	)
	ErrInvalidPlatformFeeAmount = errors.New(
		"settlement: invalid platformFeeAmount",
	)
	ErrInvalidTransferAmount = errors.New(
		"settlement: invalid transferAmount",
	)
	ErrAmountMismatch = errors.New(
		"settlement: grossAmount does not equal platformFeeAmount + transferAmount",
	)
	ErrInvalidCurrency = errors.New(
		"settlement: invalid currency",
	)
	ErrInvalidStatus = errors.New(
		"settlement: invalid status",
	)
	ErrInvalidStatusTransition = errors.New(
		"settlement: invalid status transition",
	)
	ErrInvalidErrorType = errors.New(
		"settlement: invalid errorType",
	)
	ErrInvalidErrorCode = errors.New(
		"settlement: invalid errorCode",
	)
	ErrInvalidErrorMsg = errors.New(
		"settlement: invalid errorMsg",
	)
	ErrFailureReasonRequired = errors.New(
		"settlement: failure reason is required",
	)
	ErrInvalidCreatedAt = errors.New(
		"settlement: invalid createdAt",
	)
	ErrInvalidUpdatedAt = errors.New(
		"settlement: invalid updatedAt",
	)
	ErrInvalidTransferredAt = errors.New(
		"settlement: invalid transferredAt",
	)
	ErrInvalidReversedAt = errors.New(
		"settlement: invalid reversedAt",
	)
)

// NewID creates the deterministic Firestore Settlement document ID.
//
// A Payment document ID is the Order ID in the current payment model.
// Combining PaymentID and AccountID guarantees one Settlement per Account
// inside one payment.
func NewID(
	paymentID string,
	accountID string,
) (string, error) {
	if paymentID == "" {
		return "", ErrInvalidPaymentID
	}

	if accountID == "" {
		return "", ErrInvalidAccountID
	}

	if strings.Contains(paymentID, "/") ||
		strings.Contains(accountID, "/") {
		return "", ErrInvalidID
	}

	return paymentID + "_" + accountID, nil
}

func IsValidStatus(
	status SettlementStatus,
) bool {
	if status == "" {
		return false
	}

	_, ok := AllowedStatuses[status]
	return ok
}

// New creates a Settlement before the Stripe Transfer is executed.
//
// stripePaymentIntentID and stripeChargeID must already be known.
// StripeTransferID is intentionally empty at creation.
//
// status may be:
//
// - pending
// - ready
//
// An empty status defaults to pending.
func New(
	id string,
	orderID string,
	paymentID string,
	companyID string,
	accountID string,
	stripeAccountID string,
	stripePaymentIntentID string,
	stripeChargeID string,
	transferGroup string,
	grossAmount int,
	platformFeeAmount int,
	transferAmount int,
	currency string,
	status SettlementStatus,
	createdAt time.Time,
) (Settlement, error) {
	if status == "" {
		status = DefaultStatus
	}

	createdAt = createdAt.UTC()

	s := Settlement{
		ID:                       id,
		OrderID:                  orderID,
		PaymentID:                paymentID,
		CompanyID:                companyID,
		AccountID:                accountID,
		StripeAccountID:          stripeAccountID,
		StripePaymentIntentID:    stripePaymentIntentID,
		StripeChargeID:           stripeChargeID,
		StripeTransferID:         "",
		StripeTransferReversalID: "",
		TransferGroup:            transferGroup,
		GrossAmount:              grossAmount,
		PlatformFeeAmount:        platformFeeAmount,
		TransferAmount:           transferAmount,
		Currency:                 currency,
		Status:                   status,
		ErrorType:                nil,
		ErrorCode:                nil,
		ErrorMsg:                 nil,
		CreatedAt:                createdAt,
		UpdatedAt:                createdAt,
		TransferredAt:            nil,
		ReversedAt:               nil,
	}

	if status != StatusPending &&
		status != StatusReady {
		return Settlement{},
			ErrInvalidStatus
	}

	if err := s.Validate(); err != nil {
		return Settlement{}, err
	}

	return s, nil
}

// MarkReady marks the Settlement as ready to be sent to the Stripe Transfer
// worker.
func (s *Settlement) MarkReady(
	now time.Time,
) error {
	if s == nil {
		return ErrInvalidStatusTransition
	}

	switch s.Status {
	case StatusPending,
		StatusFailedRetryable:

	default:
		return ErrInvalidStatusTransition
	}

	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}

	s.Status = StatusReady
	s.ErrorType = nil
	s.ErrorCode = nil
	s.ErrorMsg = nil
	s.UpdatedAt = now.UTC()

	return s.Validate()
}

// StartTransfer claims the Settlement for Stripe Transfer processing.
func (s *Settlement) StartTransfer(
	now time.Time,
) error {
	if s == nil {
		return ErrInvalidStatusTransition
	}

	switch s.Status {
	case StatusReady,
		StatusFailedRetryable:

	default:
		return ErrInvalidStatusTransition
	}

	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}

	s.Status = StatusTransferring
	s.ErrorType = nil
	s.ErrorCode = nil
	s.ErrorMsg = nil
	s.UpdatedAt = now.UTC()

	return s.Validate()
}

// MarkTransferred records a successfully created Stripe Connect Transfer.
func (s *Settlement) MarkTransferred(
	stripeTransferID string,
	now time.Time,
) error {
	if s == nil ||
		s.Status != StatusTransferring {
		return ErrInvalidStatusTransition
	}

	if !isStripeTransferID(
		stripeTransferID,
	) {
		return ErrInvalidStripeTransferID
	}

	if now.IsZero() {
		return ErrInvalidTransferredAt
	}

	transferredAt := now.UTC()

	s.StripeTransferID = stripeTransferID
	s.Status = StatusTransferred
	s.ErrorType = nil
	s.ErrorCode = nil
	s.ErrorMsg = nil
	s.TransferredAt = &transferredAt
	s.UpdatedAt = transferredAt

	return s.Validate()
}

// MarkFailedRetryable records a Stripe Transfer failure that may be retried.
func (s *Settlement) MarkFailedRetryable(
	errorType *string,
	errorCode *string,
	errorMsg *string,
	now time.Time,
) error {
	if s == nil ||
		s.Status != StatusTransferring {
		return ErrInvalidStatusTransition
	}

	if err := validateFailureReason(
		errorType,
		errorCode,
		errorMsg,
	); err != nil {
		return err
	}

	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}

	s.Status = StatusFailedRetryable
	s.ErrorType = errorType
	s.ErrorCode = errorCode
	s.ErrorMsg = errorMsg
	s.UpdatedAt = now.UTC()

	return s.Validate()
}

// MarkFailed records a terminal Stripe Transfer failure.
func (s *Settlement) MarkFailed(
	errorType *string,
	errorCode *string,
	errorMsg *string,
	now time.Time,
) error {
	if s == nil ||
		s.Status != StatusTransferring {
		return ErrInvalidStatusTransition
	}

	if err := validateFailureReason(
		errorType,
		errorCode,
		errorMsg,
	); err != nil {
		return err
	}

	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}

	s.Status = StatusFailed
	s.ErrorType = errorType
	s.ErrorCode = errorCode
	s.ErrorMsg = errorMsg
	s.UpdatedAt = now.UTC()

	return s.Validate()
}

// Cancel cancels a Settlement that has not been transferred.
//
// A transferring Settlement cannot be canceled because the result of the
// Stripe request must first be reconciled.
func (s *Settlement) Cancel(
	now time.Time,
) error {
	if s == nil {
		return ErrInvalidStatusTransition
	}

	switch s.Status {
	case StatusPending,
		StatusReady,
		StatusFailedRetryable:

	default:
		return ErrInvalidStatusTransition
	}

	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}

	s.Status = StatusCanceled
	s.ErrorType = nil
	s.ErrorCode = nil
	s.ErrorMsg = nil
	s.UpdatedAt = now.UTC()

	return s.Validate()
}

// MarkReversed records a full Stripe Transfer reversal.
//
// This is separate from a card payment Refund.
// Refund and Transfer Reversal must be coordinated by the application layer.
func (s *Settlement) MarkReversed(
	stripeTransferReversalID string,
	now time.Time,
) error {
	if s == nil ||
		s.Status != StatusTransferred {
		return ErrInvalidStatusTransition
	}

	if !isStripeTransferReversalID(
		stripeTransferReversalID,
	) {
		return ErrInvalidStripeTransferReversalID
	}

	if now.IsZero() {
		return ErrInvalidReversedAt
	}

	reversedAt := now.UTC()

	s.StripeTransferReversalID =
		stripeTransferReversalID

	s.Status = StatusReversed
	s.ErrorType = nil
	s.ErrorCode = nil
	s.ErrorMsg = nil
	s.ReversedAt = &reversedAt
	s.UpdatedAt = reversedAt

	return s.Validate()
}

// Validate verifies all Settlement persistence invariants.
func (s Settlement) Validate() error {
	if s.ID == "" ||
		strings.Contains(s.ID, "/") {
		return ErrInvalidID
	}

	if s.OrderID == "" {
		return ErrInvalidOrderID
	}

	if s.PaymentID == "" {
		return ErrInvalidPaymentID
	}

	// Current AMOL payment documents use the Order ID as PaymentID.
	if s.PaymentID != s.OrderID {
		return ErrPaymentOrderMismatch
	}

	if s.CompanyID == "" {
		return ErrInvalidCompanyID
	}

	if s.AccountID == "" {
		return ErrInvalidAccountID
	}

	if !isStripeAccountID(
		s.StripeAccountID,
	) {
		return ErrInvalidStripeAccountID
	}

	if !isStripePaymentIntentID(
		s.StripePaymentIntentID,
	) {
		return ErrInvalidStripePaymentIntentID
	}

	if !isStripeChargeID(
		s.StripeChargeID,
	) {
		return ErrInvalidStripeChargeID
	}

	if s.TransferGroup == "" {
		return ErrInvalidTransferGroup
	}

	if s.GrossAmount <= 0 {
		return ErrInvalidGrossAmount
	}

	if s.PlatformFeeAmount < 0 ||
		s.PlatformFeeAmount > s.GrossAmount {
		return ErrInvalidPlatformFeeAmount
	}

	if s.TransferAmount <= 0 ||
		s.TransferAmount > s.GrossAmount {
		return ErrInvalidTransferAmount
	}

	if s.GrossAmount-
		s.PlatformFeeAmount !=
		s.TransferAmount {
		return ErrAmountMismatch
	}

	if s.Currency != CurrencyJPY {
		return ErrInvalidCurrency
	}

	if !IsValidStatus(s.Status) {
		return ErrInvalidStatus
	}

	if s.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}

	if s.UpdatedAt.IsZero() ||
		s.UpdatedAt.Before(
			s.CreatedAt,
		) {
		return ErrInvalidUpdatedAt
	}

	if err :=
		validateOptionalErrorString(
			s.ErrorType,
			ErrInvalidErrorType,
		); err != nil {
		return err
	}

	if err :=
		validateOptionalErrorString(
			s.ErrorCode,
			ErrInvalidErrorCode,
		); err != nil {
		return err
	}

	if err :=
		validateOptionalErrorString(
			s.ErrorMsg,
			ErrInvalidErrorMsg,
		); err != nil {
		return err
	}

	switch s.Status {
	case StatusPending,
		StatusReady,
		StatusTransferring,
		StatusCanceled:
		if s.StripeTransferID != "" ||
			s.StripeTransferReversalID != "" ||
			s.TransferredAt != nil ||
			s.ReversedAt != nil {
			return ErrInvalidStatus
		}

		if s.ErrorType != nil ||
			s.ErrorCode != nil ||
			s.ErrorMsg != nil {
			return ErrInvalidStatus
		}

	case StatusFailedRetryable,
		StatusFailed:
		if s.StripeTransferID != "" ||
			s.StripeTransferReversalID != "" ||
			s.TransferredAt != nil ||
			s.ReversedAt != nil {
			return ErrInvalidStatus
		}

		if err := validateFailureReason(
			s.ErrorType,
			s.ErrorCode,
			s.ErrorMsg,
		); err != nil {
			return err
		}

	case StatusTransferred:
		if !isStripeTransferID(
			s.StripeTransferID,
		) {
			return ErrInvalidStripeTransferID
		}

		if s.StripeTransferReversalID != "" {
			return ErrInvalidStripeTransferReversalID
		}

		if s.TransferredAt == nil ||
			s.TransferredAt.IsZero() {
			return ErrInvalidTransferredAt
		}

		if s.TransferredAt.Before(
			s.CreatedAt,
		) {
			return ErrInvalidTransferredAt
		}

		if s.ReversedAt != nil {
			return ErrInvalidReversedAt
		}

		if s.ErrorType != nil ||
			s.ErrorCode != nil ||
			s.ErrorMsg != nil {
			return ErrInvalidStatus
		}

	case StatusReversed:
		if !isStripeTransferID(
			s.StripeTransferID,
		) {
			return ErrInvalidStripeTransferID
		}

		if !isStripeTransferReversalID(
			s.StripeTransferReversalID,
		) {
			return ErrInvalidStripeTransferReversalID
		}

		if s.TransferredAt == nil ||
			s.TransferredAt.IsZero() {
			return ErrInvalidTransferredAt
		}

		if s.ReversedAt == nil ||
			s.ReversedAt.IsZero() {
			return ErrInvalidReversedAt
		}

		if s.ReversedAt.Before(
			*s.TransferredAt,
		) {
			return ErrInvalidReversedAt
		}

		if s.ErrorType != nil ||
			s.ErrorCode != nil ||
			s.ErrorMsg != nil {
			return ErrInvalidStatus
		}
	}

	return nil
}

func validateFailureReason(
	errorType *string,
	errorCode *string,
	errorMsg *string,
) error {
	if err := validateOptionalErrorString(
		errorType,
		ErrInvalidErrorType,
	); err != nil {
		return err
	}

	if err := validateOptionalErrorString(
		errorCode,
		ErrInvalidErrorCode,
	); err != nil {
		return err
	}

	if err := validateOptionalErrorString(
		errorMsg,
		ErrInvalidErrorMsg,
	); err != nil {
		return err
	}

	if errorType == nil &&
		errorCode == nil &&
		errorMsg == nil {
		return ErrFailureReasonRequired
	}

	return nil
}

func validateOptionalErrorString(
	value *string,
	invalidError error,
) error {
	if value != nil &&
		*value == "" {
		return invalidError
	}

	return nil
}

func isStripeAccountID(
	value string,
) bool {
	return strings.HasPrefix(
		value,
		"acct_",
	) &&
		len(value) > len("acct_")
}

func isStripePaymentIntentID(
	value string,
) bool {
	return strings.HasPrefix(
		value,
		"pi_",
	) &&
		len(value) > len("pi_")
}

func isStripeChargeID(
	value string,
) bool {
	return strings.HasPrefix(
		value,
		"ch_",
	) &&
		len(value) > len("ch_")
}

func isStripeTransferID(
	value string,
) bool {
	return strings.HasPrefix(
		value,
		"tr_",
	) &&
		len(value) > len("tr_")
}

func isStripeTransferReversalID(
	value string,
) bool {
	return strings.HasPrefix(
		value,
		"trr_",
	) &&
		len(value) > len("trr_")
}
