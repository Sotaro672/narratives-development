// backend/internal/domain/salesReceivable/entity.go
package salesReceivable

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

// Status represents the lifecycle of resale proceeds owed by AMOL to a resale seller.
//
// pending:
//
//	Payment has succeeded, but the resale fulfillment boundary has not yet been completed.
//
// available:
//
//	Fulfillment has completed and the receivable may be included in a bank payout.
//
// reserved:
//
//	The receivable has been assigned to a bank payout and must not be selected by another payout.
//
// paid:
//
//	The associated bank payout has completed.
//
// canceled:
//
//	The receivable was canceled before being paid, for example because the underlying resale item was refunded before payout.
type Status string

const (
	StatusPending   Status = "pending"
	StatusAvailable Status = "available"
	StatusReserved  Status = "reserved"
	StatusPaid      Status = "paid"
	StatusCanceled  Status = "canceled"
)

var AllowedStatuses = map[Status]struct{}{
	StatusPending:   {},
	StatusAvailable: {},
	StatusReserved:  {},
	StatusPaid:      {},
	StatusCanceled:  {},
}

const CurrencyJPY = "JPY"

var DefaultStatus = StatusPending

var (
	ErrInvalidID                  = errors.New("salesReceivable: invalid id")
	ErrInvalidOrderID             = errors.New("salesReceivable: invalid orderId")
	ErrInvalidPaymentID           = errors.New("salesReceivable: invalid paymentId")
	ErrPaymentOrderMismatch       = errors.New("salesReceivable: paymentId does not match orderId")
	ErrInvalidOrderItemIndex      = errors.New("salesReceivable: invalid orderItemIndex")
	ErrInvalidResaleID            = errors.New("salesReceivable: invalid resaleId")
	ErrInvalidAvatarID            = errors.New("salesReceivable: invalid avatarId")
	ErrInvalidUserID              = errors.New("salesReceivable: invalid userId")
	ErrInvalidPayoutAccountID     = errors.New("salesReceivable: invalid payoutAccountId")
	ErrPayoutAccountOwnerMismatch = errors.New("salesReceivable: payoutAccountId does not match userId")
	ErrInvalidGrossAmount         = errors.New("salesReceivable: invalid grossAmount")
	ErrInvalidPlatformFeeAmount   = errors.New("salesReceivable: invalid platformFeeAmount")
	ErrInvalidBrandFeeAmount      = errors.New("salesReceivable: invalid brandFeeAmount")
	ErrInvalidReceivableAmount    = errors.New("salesReceivable: invalid receivableAmount")
	ErrAmountMismatch             = errors.New("salesReceivable: grossAmount does not equal platformFeeAmount + brandFeeAmount + receivableAmount")
	ErrInvalidCurrency            = errors.New("salesReceivable: invalid currency")
	ErrInvalidStatus              = errors.New("salesReceivable: invalid status")
	ErrInvalidStatusTransition    = errors.New("salesReceivable: invalid status transition")
	ErrInvalidBankPayoutID        = errors.New("salesReceivable: invalid bankPayoutId")
	ErrInvalidCreatedAt           = errors.New("salesReceivable: invalid createdAt")
	ErrInvalidUpdatedAt           = errors.New("salesReceivable: invalid updatedAt")
	ErrInvalidAvailableAt         = errors.New("salesReceivable: invalid availableAt")
	ErrInvalidReservedAt          = errors.New("salesReceivable: invalid reservedAt")
	ErrInvalidPaidAt              = errors.New("salesReceivable: invalid paidAt")
	ErrInvalidCanceledAt          = errors.New("salesReceivable: invalid canceledAt")
)

// SalesReceivable represents resale proceeds that AMOL owes to one resale seller for one successfully paid resale Order item.
//
// Persistence:
//   - collection: salesReceivables
//   - one active resale Order item creates one SalesReceivable
//   - document ID is deterministic by PaymentID and OrderItemIndex
//   - SalesReceivables are never aggregated across multiple Order items
//
// GrossAmount represents the resale distribution base after shipping cost has
// been deducted from the merchandise amount.
//
// Amount invariant:
//
//	GrossAmount = PlatformFeeAmount + BrandFeeAmount + ReceivableAmount
//
// PlatformFeeAmount is AMOL's share of the resale distribution base.
// BrandFeeAmount is the productBlueprint Brand's share.
// ReceivableAmount is the amount owed to the resale seller.
//
// OrderItemIndex and ResaleID identify the immutable resale item represented by this receivable.
//
// PayoutAccountID identifies the seller's registered payout account at the time the Order was created. It is not a bank-destination snapshot.
//
// Bank account fields and plaintext/ciphertext account numbers must never be persisted in SalesReceivable. The actual bank destination is snapshotted only when a BankPayout is created.
type SalesReceivable struct {
	ID string `json:"id" firestore:"id"`

	OrderID        string `json:"orderId" firestore:"orderId"`
	PaymentID      string `json:"paymentId" firestore:"paymentId"`
	OrderItemIndex int    `json:"orderItemIndex" firestore:"orderItemIndex"`
	ResaleID       string `json:"resaleId" firestore:"resaleId"`

	AvatarID        string `json:"avatarId" firestore:"avatarId"`
	UserID          string `json:"userId" firestore:"userId"`
	PayoutAccountID string `json:"payoutAccountId" firestore:"payoutAccountId"`

	GrossAmount       int `json:"grossAmount" firestore:"grossAmount"`
	PlatformFeeAmount int `json:"platformFeeAmount" firestore:"platformFeeAmount"`
	BrandFeeAmount    int `json:"brandFeeAmount" firestore:"brandFeeAmount"`
	ReceivableAmount  int `json:"receivableAmount" firestore:"receivableAmount"`

	Currency string `json:"currency" firestore:"currency"`
	Status   Status `json:"status" firestore:"status"`

	BankPayoutID string `json:"bankPayoutId,omitempty" firestore:"bankPayoutId,omitempty"`

	CreatedAt time.Time `json:"createdAt" firestore:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt" firestore:"updatedAt"`

	AvailableAt *time.Time `json:"availableAt,omitempty" firestore:"availableAt,omitempty"`
	ReservedAt  *time.Time `json:"reservedAt,omitempty" firestore:"reservedAt,omitempty"`
	PaidAt      *time.Time `json:"paidAt,omitempty" firestore:"paidAt,omitempty"`
	CanceledAt  *time.Time `json:"canceledAt,omitempty" firestore:"canceledAt,omitempty"`
}

// NewID creates the deterministic Firestore document ID.
//
//	PaymentID + "_resale_item_" + OrderItemIndex
//
// OrderItemIndex is immutable within the Order and allows one Payment to contain multiple independent resale receivables, including multiple items belonging to the same seller.
func NewID(paymentID string, orderItemIndex int) (string, error) {
	if paymentID == "" {
		return "", ErrInvalidPaymentID
	}
	if strings.Contains(paymentID, "/") {
		return "", ErrInvalidID
	}
	if orderItemIndex < 0 {
		return "", ErrInvalidOrderItemIndex
	}

	return paymentID + "_resale_item_" + strconv.Itoa(orderItemIndex), nil
}

// New creates a pending SalesReceivable after a successful resale payment.
//
// GrossAmount is the resale distribution base after shipping cost has been deducted.
// No bank payout is assigned at creation. The receivable becomes available only after this exact resale Order item has crossed the fulfillment boundary.
func New(
	id string,
	orderID string,
	paymentID string,
	orderItemIndex int,
	resaleID string,
	avatarID string,
	userID string,
	payoutAccountID string,
	grossAmount int,
	platformFeeAmount int,
	brandFeeAmount int,
	receivableAmount int,
	currency string,
	createdAt time.Time,
) (SalesReceivable, error) {
	createdAt = createdAt.UTC()

	receivable := SalesReceivable{
		ID: id,

		OrderID:        orderID,
		PaymentID:      paymentID,
		OrderItemIndex: orderItemIndex,
		ResaleID:       resaleID,

		AvatarID:        avatarID,
		UserID:          userID,
		PayoutAccountID: payoutAccountID,

		GrossAmount:       grossAmount,
		PlatformFeeAmount: platformFeeAmount,
		BrandFeeAmount:    brandFeeAmount,
		ReceivableAmount:  receivableAmount,

		Currency: currency,
		Status:   DefaultStatus,

		BankPayoutID: "",

		CreatedAt: createdAt,
		UpdatedAt: createdAt,

		AvailableAt: nil,
		ReservedAt:  nil,
		PaidAt:      nil,
		CanceledAt:  nil,
	}

	if err := receivable.Validate(); err != nil {
		return SalesReceivable{}, err
	}

	return receivable, nil
}

// MarkAvailable makes the receivable eligible for a future bank payout.
func (r *SalesReceivable) MarkAvailable(now time.Time) error {
	if r == nil || r.Status != StatusPending {
		return ErrInvalidStatusTransition
	}
	if now.IsZero() {
		return ErrInvalidAvailableAt
	}

	availableAt := now.UTC()
	if availableAt.Before(r.CreatedAt) {
		return ErrInvalidAvailableAt
	}

	r.Status = StatusAvailable
	r.AvailableAt = &availableAt
	r.UpdatedAt = availableAt

	return r.Validate()
}

// Reserve assigns an available receivable to one BankPayout.
//
// Once reserved, the receivable must not be selected by another payout.
func (r *SalesReceivable) Reserve(bankPayoutID string, now time.Time) error {
	if r == nil || r.Status != StatusAvailable {
		return ErrInvalidStatusTransition
	}
	if bankPayoutID == "" || strings.Contains(bankPayoutID, "/") {
		return ErrInvalidBankPayoutID
	}
	if now.IsZero() {
		return ErrInvalidReservedAt
	}

	reservedAt := now.UTC()
	if r.AvailableAt == nil || reservedAt.Before(*r.AvailableAt) {
		return ErrInvalidReservedAt
	}

	r.Status = StatusReserved
	r.BankPayoutID = bankPayoutID
	r.ReservedAt = &reservedAt
	r.UpdatedAt = reservedAt

	return r.Validate()
}

// ReleaseReservation returns a reserved receivable to available.
//
// This is used when a bank payout is abandoned before money movement has completed. A paid receivable cannot be released.
func (r *SalesReceivable) ReleaseReservation(now time.Time) error {
	if r == nil || r.Status != StatusReserved {
		return ErrInvalidStatusTransition
	}
	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}

	updatedAt := now.UTC()
	if r.ReservedAt == nil || updatedAt.Before(*r.ReservedAt) {
		return ErrInvalidUpdatedAt
	}

	r.Status = StatusAvailable
	r.BankPayoutID = ""
	r.ReservedAt = nil
	r.UpdatedAt = updatedAt

	return r.Validate()
}

// MarkPaid records completion of the BankPayout currently reserving this receivable.
func (r *SalesReceivable) MarkPaid(now time.Time) error {
	if r == nil || r.Status != StatusReserved {
		return ErrInvalidStatusTransition
	}
	if r.BankPayoutID == "" {
		return ErrInvalidBankPayoutID
	}
	if now.IsZero() {
		return ErrInvalidPaidAt
	}

	paidAt := now.UTC()
	if r.ReservedAt == nil || paidAt.Before(*r.ReservedAt) {
		return ErrInvalidPaidAt
	}

	r.Status = StatusPaid
	r.PaidAt = &paidAt
	r.UpdatedAt = paidAt

	return r.Validate()
}

// Cancel cancels an unpaid receivable.
//
// pending and available receivables may be canceled. A reserved receivable must first be released from its BankPayout. A paid receivable cannot be canceled; post-payout recovery must be represented separately rather than rewriting payout history.
func (r *SalesReceivable) Cancel(now time.Time) error {
	if r == nil {
		return ErrInvalidStatusTransition
	}

	switch r.Status {
	case StatusPending, StatusAvailable:
	default:
		return ErrInvalidStatusTransition
	}

	if now.IsZero() {
		return ErrInvalidCanceledAt
	}

	canceledAt := now.UTC()
	if canceledAt.Before(r.CreatedAt) {
		return ErrInvalidCanceledAt
	}
	if r.AvailableAt != nil && canceledAt.Before(*r.AvailableAt) {
		return ErrInvalidCanceledAt
	}

	r.Status = StatusCanceled
	r.BankPayoutID = ""
	r.ReservedAt = nil
	r.CanceledAt = &canceledAt
	r.UpdatedAt = canceledAt

	return r.Validate()
}

func IsValidStatus(status Status) bool {
	if status == "" {
		return false
	}

	_, ok := AllowedStatuses[status]
	return ok
}

// Validate verifies all persistence invariants.
//
// No input normalization is performed. Values supplied by the application layer are validated exactly as received.
func (r SalesReceivable) Validate() error {
	if r.ID == "" || strings.Contains(r.ID, "/") {
		return ErrInvalidID
	}
	if r.OrderID == "" || strings.Contains(r.OrderID, "/") {
		return ErrInvalidOrderID
	}
	if r.PaymentID == "" || strings.Contains(r.PaymentID, "/") {
		return ErrInvalidPaymentID
	}

	// Current AMOL payments use the Order ID as PaymentID.
	if r.PaymentID != r.OrderID {
		return ErrPaymentOrderMismatch
	}

	if r.OrderItemIndex < 0 {
		return ErrInvalidOrderItemIndex
	}
	if r.ResaleID == "" || strings.Contains(r.ResaleID, "/") {
		return ErrInvalidResaleID
	}

	expectedID, err := NewID(r.PaymentID, r.OrderItemIndex)
	if err != nil {
		return err
	}
	if r.ID != expectedID {
		return ErrInvalidID
	}

	if r.AvatarID == "" {
		return ErrInvalidAvatarID
	}
	if r.UserID == "" {
		return ErrInvalidUserID
	}
	if r.PayoutAccountID == "" {
		return ErrInvalidPayoutAccountID
	}
	if r.PayoutAccountID != r.UserID {
		return ErrPayoutAccountOwnerMismatch
	}

	if r.GrossAmount <= 0 {
		return ErrInvalidGrossAmount
	}
	if r.PlatformFeeAmount < 0 || r.PlatformFeeAmount > r.GrossAmount {
		return ErrInvalidPlatformFeeAmount
	}
	if r.BrandFeeAmount < 0 || r.BrandFeeAmount > r.GrossAmount {
		return ErrInvalidBrandFeeAmount
	}
	if r.ReceivableAmount <= 0 || r.ReceivableAmount > r.GrossAmount {
		return ErrInvalidReceivableAmount
	}
	if r.PlatformFeeAmount > r.GrossAmount-r.BrandFeeAmount {
		return ErrAmountMismatch
	}
	if r.GrossAmount-r.PlatformFeeAmount-r.BrandFeeAmount != r.ReceivableAmount {
		return ErrAmountMismatch
	}
	if r.Currency != CurrencyJPY {
		return ErrInvalidCurrency
	}
	if !IsValidStatus(r.Status) {
		return ErrInvalidStatus
	}
	if r.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if r.UpdatedAt.IsZero() || r.UpdatedAt.Before(r.CreatedAt) {
		return ErrInvalidUpdatedAt
	}

	switch r.Status {
	case StatusPending:
		if r.BankPayoutID != "" ||
			r.AvailableAt != nil ||
			r.ReservedAt != nil ||
			r.PaidAt != nil ||
			r.CanceledAt != nil {
			return ErrInvalidStatus
		}

	case StatusAvailable:
		if r.BankPayoutID != "" ||
			r.AvailableAt == nil ||
			r.AvailableAt.IsZero() ||
			r.ReservedAt != nil ||
			r.PaidAt != nil ||
			r.CanceledAt != nil {
			return ErrInvalidStatus
		}
		if r.AvailableAt.Before(r.CreatedAt) {
			return ErrInvalidAvailableAt
		}
		if r.UpdatedAt.Before(*r.AvailableAt) {
			return ErrInvalidUpdatedAt
		}

	case StatusReserved:
		if r.BankPayoutID == "" ||
			strings.Contains(r.BankPayoutID, "/") ||
			r.AvailableAt == nil ||
			r.AvailableAt.IsZero() ||
			r.ReservedAt == nil ||
			r.ReservedAt.IsZero() ||
			r.PaidAt != nil ||
			r.CanceledAt != nil {
			return ErrInvalidStatus
		}
		if r.AvailableAt.Before(r.CreatedAt) {
			return ErrInvalidAvailableAt
		}
		if r.ReservedAt.Before(*r.AvailableAt) {
			return ErrInvalidReservedAt
		}
		if !r.UpdatedAt.Equal(*r.ReservedAt) {
			return ErrInvalidUpdatedAt
		}

	case StatusPaid:
		if r.BankPayoutID == "" ||
			strings.Contains(r.BankPayoutID, "/") ||
			r.AvailableAt == nil ||
			r.AvailableAt.IsZero() ||
			r.ReservedAt == nil ||
			r.ReservedAt.IsZero() ||
			r.PaidAt == nil ||
			r.PaidAt.IsZero() ||
			r.CanceledAt != nil {
			return ErrInvalidStatus
		}
		if r.AvailableAt.Before(r.CreatedAt) {
			return ErrInvalidAvailableAt
		}
		if r.ReservedAt.Before(*r.AvailableAt) {
			return ErrInvalidReservedAt
		}
		if r.PaidAt.Before(*r.ReservedAt) {
			return ErrInvalidPaidAt
		}
		if !r.UpdatedAt.Equal(*r.PaidAt) {
			return ErrInvalidUpdatedAt
		}

	case StatusCanceled:
		if r.BankPayoutID != "" ||
			r.ReservedAt != nil ||
			r.PaidAt != nil ||
			r.CanceledAt == nil ||
			r.CanceledAt.IsZero() {
			return ErrInvalidStatus
		}
		if r.CanceledAt.Before(r.CreatedAt) {
			return ErrInvalidCanceledAt
		}
		if r.AvailableAt != nil {
			if r.AvailableAt.IsZero() || r.AvailableAt.Before(r.CreatedAt) {
				return ErrInvalidAvailableAt
			}
			if r.CanceledAt.Before(*r.AvailableAt) {
				return ErrInvalidCanceledAt
			}
		}
		if !r.UpdatedAt.Equal(*r.CanceledAt) {
			return ErrInvalidUpdatedAt
		}
	}

	return nil
}
