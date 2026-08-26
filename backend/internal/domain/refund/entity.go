// backend/internal/domain/refund/entity.go
package refund

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ============================================================
// Refund Status
// ============================================================

// RefundStatus represents the lifecycle of one item-level purchaser refund.
//
// This status is intentionally independent from Payment.Status.
//
// Payment.Status continues to represent the original PaymentIntent lifecycle.
// Refund records the subsequent partial Stripe Refund lifecycle for one
// Order item.
type RefundStatus string

const (
	StatusCreated        RefundStatus = "created"
	StatusPending        RefundStatus = "pending"
	StatusRequiresAction RefundStatus = "requires_action"
	StatusSucceeded      RefundStatus = "succeeded"
	StatusFailed         RefundStatus = "failed"
	StatusCanceled       RefundStatus = "canceled"
)

var AllowedStatuses = map[RefundStatus]struct{}{
	StatusCreated:        {},
	StatusPending:        {},
	StatusRequiresAction: {},
	StatusSucceeded:      {},
	StatusFailed:         {},
	StatusCanceled:       {},
}

var DefaultStatus = StatusCreated

func IsValidStatus(
	status RefundStatus,
) bool {
	if status == "" {
		return false
	}

	_, ok := AllowedStatuses[status]
	return ok
}

// ============================================================
// Transfer Reversal Status
// ============================================================

// TransferReversalStatus represents seller-side Stripe Transfer Reversal
// processing associated with this item-level refund.
//
// AMOL uses Separate Charges and Transfers:
//
//	PaymentIntent
//		-> Charge
//		-> Transfer
//
// Therefore refunding the purchaser Charge does not automatically reclaim
// money already transferred to the seller.
//
// TransferReversalAmount is zero when no seller-side reversal is required.
type TransferReversalStatus string

const (
	TransferReversalStatusNotRequired     TransferReversalStatus = "not_required"
	TransferReversalStatusPending         TransferReversalStatus = "pending"
	TransferReversalStatusSucceeded       TransferReversalStatus = "succeeded"
	TransferReversalStatusFailedRetryable TransferReversalStatus = "failed_retryable"
	TransferReversalStatusFailed          TransferReversalStatus = "failed"
)

var AllowedTransferReversalStatuses = map[TransferReversalStatus]struct{}{
	TransferReversalStatusNotRequired:     {},
	TransferReversalStatusPending:         {},
	TransferReversalStatusSucceeded:       {},
	TransferReversalStatusFailedRetryable: {},
	TransferReversalStatusFailed:          {},
}

func IsValidTransferReversalStatus(
	status TransferReversalStatus,
) bool {
	if status == "" {
		return false
	}

	_, ok :=
		AllowedTransferReversalStatuses[status]

	return ok
}

// ============================================================
// Constants
// ============================================================

const (
	CurrencyJPY = "JPY"
)

// ============================================================
// Refund
// ============================================================

// Refund represents one item-level purchaser refund.
//
// The existing Payment refund state remains reserved for full-payment refunds.
//
// Item-level partial refunds are persisted independently so that:
//
//   - multiple Order items may be refunded independently
//   - each Stripe Refund object is recorded independently
//   - consumption tax attributable to the returned item is recorded explicitly
//   - seller-side partial Transfer Reversal is recorded independently
//   - retry and idempotency can be handled without mutating Payment's
//     full-refund-only invariant
//
// Amount invariant:
//
//	RefundAmount = MerchandiseAmount + MerchandiseTaxAmount
//
// ShippingAmount and ShippingTaxAmount are intentionally not included.
//
// Current return receipt policy:
//
// - merchandise price is refunded
// - consumption tax attributable to the merchandise is refunded
// - shipping charge is not refunded
// - shipping consumption tax is not refunded
//
// TransferReversalAmount represents the seller-side amount that must be
// reclaimed from the Settlement when the seller Transfer has already completed.
//
// TransferReversalAmount may therefore be smaller than RefundAmount because
// AMOL platform fees may remain outside the seller Transfer.
type Refund struct {
	ID string

	InquiryID string

	OrderID        string
	PaymentID      string
	OrderItemIndex int

	CompanyID string
	AccountID string

	SettlementID string

	MerchandiseAmount    int
	MerchandiseTaxAmount int
	RefundAmount         int

	Currency string

	StripeRefundID string
	Status         RefundStatus
	RefundedAt     *time.Time

	TransferReversalAmount int

	StripeTransferReversalID string
	TransferReversalStatus   TransferReversalStatus
	TransferReversedAt       *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

// ============================================================
// Errors
// ============================================================

var (
	ErrInvalidID = errors.New(
		"refund: invalid id",
	)

	ErrInvalidInquiryID = errors.New(
		"refund: invalid inquiryId",
	)

	ErrInvalidOrderID = errors.New(
		"refund: invalid orderId",
	)

	ErrInvalidPaymentID = errors.New(
		"refund: invalid paymentId",
	)

	ErrPaymentOrderMismatch = errors.New(
		"refund: paymentId does not match orderId",
	)

	ErrInvalidOrderItemIndex = errors.New(
		"refund: invalid orderItemIndex",
	)

	ErrInvalidCompanyID = errors.New(
		"refund: invalid companyId",
	)

	ErrInvalidAccountID = errors.New(
		"refund: invalid accountId",
	)

	ErrInvalidSettlementID = errors.New(
		"refund: invalid settlementId",
	)

	ErrInvalidMerchandiseAmount = errors.New(
		"refund: invalid merchandiseAmount",
	)

	ErrInvalidMerchandiseTaxAmount = errors.New(
		"refund: invalid merchandiseTaxAmount",
	)

	ErrInvalidRefundAmount = errors.New(
		"refund: invalid refundAmount",
	)

	ErrRefundAmountMismatch = errors.New(
		"refund: refundAmount does not equal merchandiseAmount + merchandiseTaxAmount",
	)

	ErrInvalidCurrency = errors.New(
		"refund: invalid currency",
	)

	ErrInvalidStripeRefundID = errors.New(
		"refund: invalid stripeRefundId",
	)

	ErrInvalidStatus = errors.New(
		"refund: invalid status",
	)

	ErrInvalidStatusTransition = errors.New(
		"refund: invalid status transition",
	)

	ErrInvalidRefundedAt = errors.New(
		"refund: invalid refundedAt",
	)

	ErrInvalidTransferReversalAmount = errors.New(
		"refund: invalid transferReversalAmount",
	)

	ErrInvalidStripeTransferReversalID = errors.New(
		"refund: invalid stripeTransferReversalId",
	)

	ErrInvalidTransferReversalStatus = errors.New(
		"refund: invalid transfer reversal status",
	)

	ErrInvalidTransferReversalStatusTransition = errors.New(
		"refund: invalid transfer reversal status transition",
	)

	ErrInvalidTransferReversedAt = errors.New(
		"refund: invalid transferReversedAt",
	)

	ErrTransferReversalRequiresSucceededRefund = errors.New(
		"refund: transfer reversal requires succeeded refund",
	)

	ErrInvalidCreatedAt = errors.New(
		"refund: invalid createdAt",
	)

	ErrInvalidUpdatedAt = errors.New(
		"refund: invalid updatedAt",
	)
)

// ============================================================
// ID
// ============================================================

// NewID creates a deterministic Refund document ID.
//
// One Order item may be refunded only once through the return receipt flow.
//
// Deterministic ID:
//
//	{orderId}_{orderItemIndex}
//
// This allows retries of POST /inquiries/{id}/receive-return to resolve to the
// same Refund record.
func NewID(
	orderID string,
	orderItemIndex int,
) (string, error) {
	if orderID == "" ||
		strings.Contains(
			orderID,
			"/",
		) {
		return "",
			ErrInvalidOrderID
	}

	if orderItemIndex < 0 {
		return "",
			ErrInvalidOrderItemIndex
	}

	return orderID +
			"_" +
			strconv.Itoa(
				orderItemIndex,
			),
		nil
}

// ============================================================
// Constructor
// ============================================================

// New creates one item-level Refund before Stripe Refund execution.
//
// Refund begins with:
//
//	StatusCreated
//
// StripeRefundID remains empty until Stripe has created the Refund object.
//
// TransferReversalStatus is:
//
// - not_required when transferReversalAmount == 0
// - pending when transferReversalAmount > 0
//
// transferReversalAmount must be determined by the application layer from the
// authoritative Settlement allocation. The Refund entity does not calculate
// platform fees or Settlement allocation.
func New(
	id string,
	inquiryID string,
	orderID string,
	paymentID string,
	orderItemIndex int,
	companyID string,
	accountID string,
	settlementID string,
	merchandiseAmount int,
	merchandiseTaxAmount int,
	transferReversalAmount int,
	currency string,
	createdAt time.Time,
) (Refund, error) {
	createdAt =
		createdAt.UTC()

	refundAmount :=
		merchandiseAmount +
			merchandiseTaxAmount

	transferReversalStatus :=
		TransferReversalStatusNotRequired

	if transferReversalAmount > 0 {
		transferReversalStatus =
			TransferReversalStatusPending
	}

	r := Refund{
		ID: id,

		InquiryID: inquiryID,

		OrderID:        orderID,
		PaymentID:      paymentID,
		OrderItemIndex: orderItemIndex,

		CompanyID: companyID,
		AccountID: accountID,

		SettlementID: settlementID,

		MerchandiseAmount:    merchandiseAmount,
		MerchandiseTaxAmount: merchandiseTaxAmount,
		RefundAmount:         refundAmount,

		Currency: currency,

		StripeRefundID: "",
		Status:         DefaultStatus,
		RefundedAt:     nil,

		TransferReversalAmount: transferReversalAmount,

		StripeTransferReversalID: "",
		TransferReversalStatus:   transferReversalStatus,
		TransferReversedAt:       nil,

		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}

	if err := r.Validate(); err != nil {
		return Refund{}, err
	}

	return r, nil
}

// NewForOrderItem creates a Refund using the deterministic Refund ID.
func NewForOrderItem(
	inquiryID string,
	orderID string,
	paymentID string,
	orderItemIndex int,
	companyID string,
	accountID string,
	settlementID string,
	merchandiseAmount int,
	merchandiseTaxAmount int,
	transferReversalAmount int,
	currency string,
	createdAt time.Time,
) (Refund, error) {
	id, err :=
		NewID(
			orderID,
			orderItemIndex,
		)
	if err != nil {
		return Refund{}, err
	}

	return New(
		id,
		inquiryID,
		orderID,
		paymentID,
		orderItemIndex,
		companyID,
		accountID,
		settlementID,
		merchandiseAmount,
		merchandiseTaxAmount,
		transferReversalAmount,
		currency,
		createdAt,
	)
}

// ============================================================
// Stripe Refund Behavior
// ============================================================

// ApplyStripeRefund records the lifecycle returned from one Stripe Refund
// object.
//
// stripeRefundID must identify the same Stripe Refund across subsequent webhook
// updates.
//
// succeeded requires refundedAt.
//
// pending / requires_action / failed / canceled require refundedAt=nil.
func (r *Refund) ApplyStripeRefund(
	stripeRefundID string,
	status RefundStatus,
	refundedAt *time.Time,
	now time.Time,
) error {
	if r == nil {
		return ErrInvalidStatusTransition
	}

	if !isStripeRefundID(
		stripeRefundID,
	) {
		return ErrInvalidStripeRefundID
	}

	if !IsValidStatus(
		status,
	) ||
		status == StatusCreated {
		return ErrInvalidStatus
	}

	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}

	if r.StripeRefundID != "" &&
		r.StripeRefundID != stripeRefundID {
		return ErrInvalidStripeRefundID
	}

	if !canTransitionRefundStatus(
		r.Status,
		status,
	) {
		return ErrInvalidStatusTransition
	}

	next := *r

	next.StripeRefundID =
		stripeRefundID

	next.Status =
		status

	switch status {
	case StatusSucceeded:
		if refundedAt == nil ||
			refundedAt.IsZero() {
			return ErrInvalidRefundedAt
		}

		value :=
			refundedAt.UTC()

		next.RefundedAt =
			&value

	case StatusPending,
		StatusRequiresAction,
		StatusFailed,
		StatusCanceled:
		if refundedAt != nil {
			return ErrInvalidRefundedAt
		}

		next.RefundedAt =
			nil

	default:
		return ErrInvalidStatus
	}

	next.UpdatedAt =
		now.UTC()

	if err := next.Validate(); err != nil {
		return err
	}

	*r = next

	return nil
}

// MarkStripeRefundPending records a pending Stripe Refund.
func (r *Refund) MarkStripeRefundPending(
	stripeRefundID string,
	now time.Time,
) error {
	return r.ApplyStripeRefund(
		stripeRefundID,
		StatusPending,
		nil,
		now,
	)
}

// MarkStripeRefundRequiresAction records a Refund requiring additional action.
func (r *Refund) MarkStripeRefundRequiresAction(
	stripeRefundID string,
	now time.Time,
) error {
	return r.ApplyStripeRefund(
		stripeRefundID,
		StatusRequiresAction,
		nil,
		now,
	)
}

// MarkStripeRefundSucceeded records a successfully completed Stripe Refund.
func (r *Refund) MarkStripeRefundSucceeded(
	stripeRefundID string,
	refundedAt time.Time,
	now time.Time,
) error {
	if refundedAt.IsZero() {
		return ErrInvalidRefundedAt
	}

	value :=
		refundedAt.UTC()

	return r.ApplyStripeRefund(
		stripeRefundID,
		StatusSucceeded,
		&value,
		now,
	)
}

// MarkStripeRefundFailed records a failed Stripe Refund object.
func (r *Refund) MarkStripeRefundFailed(
	stripeRefundID string,
	now time.Time,
) error {
	return r.ApplyStripeRefund(
		stripeRefundID,
		StatusFailed,
		nil,
		now,
	)
}

// MarkStripeRefundCanceled records a canceled Stripe Refund object.
func (r *Refund) MarkStripeRefundCanceled(
	stripeRefundID string,
	now time.Time,
) error {
	return r.ApplyStripeRefund(
		stripeRefundID,
		StatusCanceled,
		nil,
		now,
	)
}

// ============================================================
// Transfer Reversal Behavior
// ============================================================

// MarkTransferReversalPending prepares a seller-side Transfer Reversal retry.
//
// This operation is valid only when:
//
// - purchaser Refund has succeeded
// - TransferReversalAmount > 0
//
// failed_retryable may transition back to pending before retrying Stripe.
func (r *Refund) MarkTransferReversalPending(
	now time.Time,
) error {
	if r == nil {
		return ErrInvalidTransferReversalStatusTransition
	}

	if r.Status != StatusSucceeded {
		return ErrTransferReversalRequiresSucceededRefund
	}

	if r.TransferReversalAmount <= 0 {
		return ErrInvalidTransferReversalAmount
	}

	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}

	switch r.TransferReversalStatus {
	case TransferReversalStatusPending,
		TransferReversalStatusFailedRetryable:

	default:
		return ErrInvalidTransferReversalStatusTransition
	}

	next := *r

	next.TransferReversalStatus =
		TransferReversalStatusPending

	next.UpdatedAt =
		now.UTC()

	if err := next.Validate(); err != nil {
		return err
	}

	*r = next

	return nil
}

// MarkTransferReversalSucceeded records one successful partial Stripe Transfer
// Reversal.
func (r *Refund) MarkTransferReversalSucceeded(
	stripeTransferReversalID string,
	reversedAt time.Time,
	now time.Time,
) error {
	if r == nil {
		return ErrInvalidTransferReversalStatusTransition
	}

	if r.Status != StatusSucceeded {
		return ErrTransferReversalRequiresSucceededRefund
	}

	if r.TransferReversalAmount <= 0 {
		return ErrInvalidTransferReversalAmount
	}

	if !isStripeTransferReversalID(
		stripeTransferReversalID,
	) {
		return ErrInvalidStripeTransferReversalID
	}

	if reversedAt.IsZero() {
		return ErrInvalidTransferReversedAt
	}

	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}

	switch r.TransferReversalStatus {
	case TransferReversalStatusPending,
		TransferReversalStatusFailedRetryable:

	case TransferReversalStatusSucceeded:
		if r.StripeTransferReversalID ==
			stripeTransferReversalID {
			return nil
		}

		return ErrInvalidStripeTransferReversalID

	default:
		return ErrInvalidTransferReversalStatusTransition
	}

	next := *r

	reversedAt =
		reversedAt.UTC()

	next.StripeTransferReversalID =
		stripeTransferReversalID

	next.TransferReversalStatus =
		TransferReversalStatusSucceeded

	next.TransferReversedAt =
		&reversedAt

	next.UpdatedAt =
		now.UTC()

	if err := next.Validate(); err != nil {
		return err
	}

	*r = next

	return nil
}

// MarkTransferReversalFailedRetryable records a retryable Stripe Transfer
// Reversal failure.
func (r *Refund) MarkTransferReversalFailedRetryable(
	now time.Time,
) error {
	return r.markTransferReversalFailed(
		TransferReversalStatusFailedRetryable,
		now,
	)
}

// MarkTransferReversalFailed records a terminal Stripe Transfer Reversal
// failure.
func (r *Refund) MarkTransferReversalFailed(
	now time.Time,
) error {
	return r.markTransferReversalFailed(
		TransferReversalStatusFailed,
		now,
	)
}

func (r *Refund) markTransferReversalFailed(
	status TransferReversalStatus,
	now time.Time,
) error {
	if r == nil {
		return ErrInvalidTransferReversalStatusTransition
	}

	if r.Status != StatusSucceeded {
		return ErrTransferReversalRequiresSucceededRefund
	}

	if r.TransferReversalAmount <= 0 {
		return ErrInvalidTransferReversalAmount
	}

	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}

	if status !=
		TransferReversalStatusFailedRetryable &&
		status !=
			TransferReversalStatusFailed {
		return ErrInvalidTransferReversalStatus
	}

	switch r.TransferReversalStatus {
	case TransferReversalStatusPending,
		TransferReversalStatusFailedRetryable:

	default:
		return ErrInvalidTransferReversalStatusTransition
	}

	next := *r

	next.StripeTransferReversalID =
		""

	next.TransferReversalStatus =
		status

	next.TransferReversedAt =
		nil

	next.UpdatedAt =
		now.UTC()

	if err := next.Validate(); err != nil {
		return err
	}

	*r = next

	return nil
}

// ============================================================
// Completion
// ============================================================

// IsFinanciallyCompleted reports whether all financial operations required for
// the item return have completed.
//
// Order.IsReturnCompleted and Inquiry resolved should only be updated after this
// returns true.
func (r Refund) IsFinanciallyCompleted() bool {
	if r.Status != StatusSucceeded {
		return false
	}

	switch r.TransferReversalStatus {
	case TransferReversalStatusNotRequired,
		TransferReversalStatusSucceeded:
		return true

	default:
		return false
	}
}

// ============================================================
// Validation
// ============================================================

// Validate verifies all Refund persistence invariants.
func (r Refund) Validate() error {
	if r.ID == "" ||
		strings.Contains(
			r.ID,
			"/",
		) {
		return ErrInvalidID
	}

	if r.InquiryID == "" ||
		strings.Contains(
			r.InquiryID,
			"/",
		) {
		return ErrInvalidInquiryID
	}

	if r.OrderID == "" {
		return ErrInvalidOrderID
	}

	if r.PaymentID == "" {
		return ErrInvalidPaymentID
	}

	// Current AMOL payment documents use the Order ID as PaymentID.
	if r.PaymentID != r.OrderID {
		return ErrPaymentOrderMismatch
	}

	if r.OrderItemIndex < 0 {
		return ErrInvalidOrderItemIndex
	}

	if r.CompanyID == "" {
		return ErrInvalidCompanyID
	}

	if r.AccountID == "" {
		return ErrInvalidAccountID
	}

	if r.SettlementID == "" {
		return ErrInvalidSettlementID
	}

	if r.MerchandiseAmount < 0 {
		return ErrInvalidMerchandiseAmount
	}

	if r.MerchandiseTaxAmount < 0 {
		return ErrInvalidMerchandiseTaxAmount
	}

	if r.RefundAmount <= 0 {
		return ErrInvalidRefundAmount
	}

	if r.RefundAmount !=
		r.MerchandiseAmount+
			r.MerchandiseTaxAmount {
		return ErrRefundAmountMismatch
	}

	if r.Currency != CurrencyJPY {
		return ErrInvalidCurrency
	}

	if !IsValidStatus(
		r.Status,
	) {
		return ErrInvalidStatus
	}

	if err := r.validateStripeRefundState(); err != nil {
		return err
	}

	if r.TransferReversalAmount < 0 ||
		r.TransferReversalAmount >
			r.RefundAmount {
		return ErrInvalidTransferReversalAmount
	}

	if !IsValidTransferReversalStatus(
		r.TransferReversalStatus,
	) {
		return ErrInvalidTransferReversalStatus
	}

	if err := r.validateTransferReversalState(); err != nil {
		return err
	}

	if r.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}

	if r.UpdatedAt.IsZero() {
		return ErrInvalidUpdatedAt
	}

	if r.UpdatedAt.Before(
		r.CreatedAt,
	) {
		return ErrInvalidUpdatedAt
	}

	if r.RefundedAt != nil &&
		r.RefundedAt.Before(
			r.CreatedAt,
		) {
		return ErrInvalidRefundedAt
	}

	if r.TransferReversedAt != nil &&
		r.TransferReversedAt.Before(
			r.CreatedAt,
		) {
		return ErrInvalidTransferReversedAt
	}

	return nil
}

func (r Refund) validateStripeRefundState() error {
	switch r.Status {
	case StatusCreated:
		if r.StripeRefundID != "" ||
			r.RefundedAt != nil {
			return ErrInvalidStatus
		}

		return nil

	case StatusPending,
		StatusRequiresAction,
		StatusFailed,
		StatusCanceled:
		if !isStripeRefundID(
			r.StripeRefundID,
		) {
			return ErrInvalidStripeRefundID
		}

		if r.RefundedAt != nil {
			return ErrInvalidRefundedAt
		}

		return nil

	case StatusSucceeded:
		if !isStripeRefundID(
			r.StripeRefundID,
		) {
			return ErrInvalidStripeRefundID
		}

		if r.RefundedAt == nil ||
			r.RefundedAt.IsZero() {
			return ErrInvalidRefundedAt
		}

		return nil

	default:
		return ErrInvalidStatus
	}
}

func (r Refund) validateTransferReversalState() error {
	if r.TransferReversalAmount == 0 {
		if r.TransferReversalStatus !=
			TransferReversalStatusNotRequired {
			return ErrInvalidTransferReversalStatus
		}

		if r.StripeTransferReversalID != "" ||
			r.TransferReversedAt != nil {
			return ErrInvalidTransferReversalStatus
		}

		return nil
	}

	if r.TransferReversalStatus ==
		TransferReversalStatusNotRequired {
		return ErrInvalidTransferReversalStatus
	}

	switch r.TransferReversalStatus {
	case TransferReversalStatusPending,
		TransferReversalStatusFailedRetryable,
		TransferReversalStatusFailed:
		if r.StripeTransferReversalID != "" {
			return ErrInvalidStripeTransferReversalID
		}

		if r.TransferReversedAt != nil {
			return ErrInvalidTransferReversedAt
		}

		return nil

	case TransferReversalStatusSucceeded:
		if r.Status != StatusSucceeded {
			return ErrTransferReversalRequiresSucceededRefund
		}

		if !isStripeTransferReversalID(
			r.StripeTransferReversalID,
		) {
			return ErrInvalidStripeTransferReversalID
		}

		if r.TransferReversedAt == nil ||
			r.TransferReversedAt.IsZero() {
			return ErrInvalidTransferReversedAt
		}

		return nil

	default:
		return ErrInvalidTransferReversalStatus
	}
}

// ============================================================
// Status Transition
// ============================================================

func canTransitionRefundStatus(
	current RefundStatus,
	next RefundStatus,
) bool {
	if current == next {
		return true
	}

	switch current {
	case StatusCreated:
		switch next {
		case StatusPending,
			StatusRequiresAction,
			StatusSucceeded,
			StatusFailed,
			StatusCanceled:
			return true

		default:
			return false
		}

	case StatusPending:
		switch next {
		case StatusRequiresAction,
			StatusSucceeded,
			StatusFailed,
			StatusCanceled:
			return true

		default:
			return false
		}

	case StatusRequiresAction:
		switch next {
		case StatusPending,
			StatusSucceeded,
			StatusFailed,
			StatusCanceled:
			return true

		default:
			return false
		}

	case StatusSucceeded,
		StatusFailed,
		StatusCanceled:
		return false

	default:
		return false
	}
}

// ============================================================
// Stripe ID Validation
// ============================================================

func isStripeRefundID(
	value string,
) bool {
	return strings.HasPrefix(
		value,
		"re_",
	) &&
		len(value) >
			len("re_")
}

func isStripeTransferReversalID(
	value string,
) bool {
	return strings.HasPrefix(
		value,
		"trr_",
	) &&
		len(value) >
			len("trr_")
}

// ============================================================
// Debug
// ============================================================

// String provides a concise identifier for logs without exposing Stripe IDs.
func (r Refund) String() string {
	return fmt.Sprintf(
		"refund{id=%s orderId=%s itemIndex=%d status=%s reversalStatus=%s}",
		r.ID,
		r.OrderID,
		r.OrderItemIndex,
		r.Status,
		r.TransferReversalStatus,
	)
}
