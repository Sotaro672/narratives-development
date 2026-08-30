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

func IsValidStatus(status RefundStatus) bool {
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

func IsValidTransferReversalStatus(status TransferReversalStatus) bool {
	if status == "" {
		return false
	}

	_, ok := AllowedTransferReversalStatuses[status]
	return ok
}

// ============================================================
// Constants
// ============================================================

const (
	CurrencyJPY = "JPY"
)

// ============================================================
// Seller Identity
// ============================================================

// SellerType identifies the seller payout identity associated with a Refund.
//
// account: primary List sale paid out to a company Account.
// avatar: consumer resale paid out to an Avatar owner payout account.
type SellerType string

const (
	SellerTypeAccount SellerType = "account"
	SellerTypeAvatar  SellerType = "avatar"
)

var AllowedSellerTypes = map[SellerType]struct{}{
	SellerTypeAccount: {},
	SellerTypeAvatar:  {},
}

// SellerIdentity is the immutable seller payout identity captured by a Refund.
//
// account seller:
//   - CompanyID, AccountID and StripeAccountID are required.
//   - AvatarID, UserID and PayoutAccountID must be empty.
//
// avatar seller:
//   - AvatarID, UserID, PayoutAccountID and StripeAccountID are required.
//   - CompanyID and AccountID must be empty.
type SellerIdentity struct {
	Type SellerType

	CompanyID string
	AccountID string

	AvatarID        string
	UserID          string
	PayoutAccountID string

	StripeAccountID string
}

func IsValidSellerType(sellerType SellerType) bool {
	if sellerType == "" {
		return false
	}

	_, ok := AllowedSellerTypes[sellerType]
	return ok
}

func (s SellerIdentity) Validate() error {
	if !IsValidSellerType(s.Type) {
		return ErrInvalidSellerType
	}

	if !isStripeAccountID(s.StripeAccountID) {
		return ErrInvalidStripeAccountID
	}

	switch s.Type {
	case SellerTypeAccount:
		if s.CompanyID == "" {
			return ErrInvalidCompanyID
		}

		if s.AccountID == "" {
			return ErrInvalidAccountID
		}

		if s.AvatarID != "" || s.UserID != "" || s.PayoutAccountID != "" {
			return ErrInvalidSellerIdentity
		}

	case SellerTypeAvatar:
		if s.AvatarID == "" {
			return ErrInvalidAvatarID
		}

		if s.UserID == "" {
			return ErrInvalidUserID
		}

		if s.PayoutAccountID == "" {
			return ErrInvalidPayoutAccountID
		}

		if s.CompanyID != "" || s.AccountID != "" {
			return ErrInvalidSellerIdentity
		}

	default:
		return ErrInvalidSellerType
	}

	return nil
}

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
// Policy:
//
//	Policy == ""
//
// represents the unopened-return flow.
//
// For that flow:
//
//	OutboundShippingAmount == 0
//	OutboundShippingTaxAmount == 0
//	ReturnShippingAmount == 0
//	ReturnShippingTaxAmount == 0
//
// and:
//
//	RefundAmount = MerchandiseAmount + MerchandiseTaxAmount
//
// For an opened return, Policy must be one of the values defined by
// OpenedReturnRefundPolicy.
//
// Stripe purchaser Refund amount:
//
//	RefundAmount
//	  = MerchandiseAmount
//	  + MerchandiseTaxAmount
//	  + OutboundShippingAmount
//	  + OutboundShippingTaxAmount
//
// ReturnShippingAmount and ReturnShippingTaxAmount are intentionally excluded
// from RefundAmount because the return shipment is not part of the purchaser's
// original Stripe Charge.
//
// Total company burden can therefore be calculated as:
//
//	RefundAmount
//	+ ReturnShippingAmount
//	+ ReturnShippingTaxAmount
//
// TransferReversalAmount represents the seller-side amount that must be
// reclaimed from the Settlement when the seller Transfer has already completed.
//
// TransferReversalAmount may be smaller than RefundAmount because AMOL platform
// fees may remain outside the seller Transfer.
type Refund struct {
	ID string

	InquiryID string

	OrderID        string
	PaymentID      string
	OrderItemIndex int

	SellerType SellerType

	CompanyID string
	AccountID string

	AvatarID        string
	UserID          string
	PayoutAccountID string

	StripeAccountID string

	SettlementID string

	Policy OpenedReturnRefundPolicy

	MerchandiseAmount    int
	MerchandiseTaxAmount int

	OutboundShippingAmount    int
	OutboundShippingTaxAmount int

	ReturnShippingAmount    int
	ReturnShippingTaxAmount int

	RefundAmount int

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

	ErrInvalidSellerType = errors.New(
		"refund: invalid sellerType",
	)

	ErrInvalidSellerIdentity = errors.New(
		"refund: invalid seller identity",
	)

	ErrInvalidCompanyID = errors.New(
		"refund: invalid companyId",
	)

	ErrInvalidAccountID = errors.New(
		"refund: invalid accountId",
	)

	ErrInvalidAvatarID = errors.New(
		"refund: invalid avatarId",
	)

	ErrInvalidUserID = errors.New(
		"refund: invalid userId",
	)

	ErrInvalidPayoutAccountID = errors.New(
		"refund: invalid payoutAccountId",
	)

	ErrInvalidStripeAccountID = errors.New(
		"refund: invalid stripeAccountId",
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

	ErrInvalidOutboundShippingAmount = errors.New(
		"refund: invalid outboundShippingAmount",
	)

	ErrInvalidOutboundShippingTaxAmount = errors.New(
		"refund: invalid outboundShippingTaxAmount",
	)

	ErrInvalidReturnShippingAmount = errors.New(
		"refund: invalid returnShippingAmount",
	)

	ErrInvalidReturnShippingTaxAmount = errors.New(
		"refund: invalid returnShippingTaxAmount",
	)

	ErrInvalidOpenedReturnAmounts = errors.New(
		"refund: invalid opened return amounts",
	)

	ErrInvalidRefundAmount = errors.New(
		"refund: invalid refundAmount",
	)

	ErrRefundAmountMismatch = errors.New(
		"refund: refundAmount does not equal refundable merchandise and outbound shipping amounts",
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
// This allows retries of return receipt endpoints to resolve to the same Refund
// record.
func NewID(orderID string, orderItemIndex int) (string, error) {
	if orderID == "" || strings.Contains(orderID, "/") {
		return "", ErrInvalidOrderID
	}

	if orderItemIndex < 0 {
		return "", ErrInvalidOrderItemIndex
	}

	return orderID + "_" + strconv.Itoa(orderItemIndex), nil
}

// ============================================================
// Constructor
// ============================================================

// New creates one unopened-return item-level Refund.
//
// Policy remains empty and no shipping component is included in RefundAmount.
//
// RefundAmount:
//
//	merchandiseAmount + merchandiseTaxAmount
func New(
	id string,
	inquiryID string,
	orderID string,
	paymentID string,
	orderItemIndex int,
	seller SellerIdentity,
	settlementID string,
	merchandiseAmount int,
	merchandiseTaxAmount int,
	transferReversalAmount int,
	currency string,
	createdAt time.Time,
) (Refund, error) {
	return newRefund(
		id,
		inquiryID,
		orderID,
		paymentID,
		orderItemIndex,
		seller,
		settlementID,
		"",
		merchandiseAmount,
		merchandiseTaxAmount,
		0,
		0,
		0,
		0,
		transferReversalAmount,
		currency,
		createdAt,
	)
}

// NewOpenedReturn creates one item-level Refund for an opened return.
//
// The application layer must calculate every monetary value from the
// authoritative Order snapshot. The frontend must never provide these amounts.
//
// RefundAmount contains only amounts refundable against the original purchaser
// Stripe Charge:
//
//	merchandiseAmount
//	+ merchandiseTaxAmount
//	+ outboundShippingAmount
//	+ outboundShippingTaxAmount
//
// ReturnShippingAmount and ReturnShippingTaxAmount are recorded as additional
// seller-side burden and are not included in the Stripe Refund amount.
func NewOpenedReturn(
	id string,
	inquiryID string,
	orderID string,
	paymentID string,
	orderItemIndex int,
	seller SellerIdentity,
	settlementID string,
	policy OpenedReturnRefundPolicy,
	merchandiseAmount int,
	merchandiseTaxAmount int,
	outboundShippingAmount int,
	outboundShippingTaxAmount int,
	returnShippingAmount int,
	returnShippingTaxAmount int,
	transferReversalAmount int,
	currency string,
	createdAt time.Time,
) (Refund, error) {
	if err := ValidateOpenedReturnRefundPolicy(policy); err != nil {
		return Refund{}, err
	}

	return newRefund(
		id,
		inquiryID,
		orderID,
		paymentID,
		orderItemIndex,
		seller,
		settlementID,
		policy,
		merchandiseAmount,
		merchandiseTaxAmount,
		outboundShippingAmount,
		outboundShippingTaxAmount,
		returnShippingAmount,
		returnShippingTaxAmount,
		transferReversalAmount,
		currency,
		createdAt,
	)
}

func newRefund(
	id string,
	inquiryID string,
	orderID string,
	paymentID string,
	orderItemIndex int,
	seller SellerIdentity,
	settlementID string,
	policy OpenedReturnRefundPolicy,
	merchandiseAmount int,
	merchandiseTaxAmount int,
	outboundShippingAmount int,
	outboundShippingTaxAmount int,
	returnShippingAmount int,
	returnShippingTaxAmount int,
	transferReversalAmount int,
	currency string,
	createdAt time.Time,
) (Refund, error) {
	if err := seller.Validate(); err != nil {
		return Refund{}, err
	}

	createdAt = createdAt.UTC()

	refundAmount, err := calculateRefundAmount(
		merchandiseAmount,
		merchandiseTaxAmount,
		outboundShippingAmount,
		outboundShippingTaxAmount,
	)
	if err != nil {
		return Refund{}, err
	}

	transferReversalStatus := TransferReversalStatusNotRequired
	if transferReversalAmount > 0 {
		transferReversalStatus = TransferReversalStatusPending
	}

	r := Refund{
		ID: id,

		InquiryID: inquiryID,

		OrderID:        orderID,
		PaymentID:      paymentID,
		OrderItemIndex: orderItemIndex,

		SellerType: seller.Type,

		CompanyID: seller.CompanyID,
		AccountID: seller.AccountID,

		AvatarID:        seller.AvatarID,
		UserID:          seller.UserID,
		PayoutAccountID: seller.PayoutAccountID,

		StripeAccountID: seller.StripeAccountID,

		SettlementID: settlementID,

		Policy: policy,

		MerchandiseAmount:    merchandiseAmount,
		MerchandiseTaxAmount: merchandiseTaxAmount,

		OutboundShippingAmount:    outboundShippingAmount,
		OutboundShippingTaxAmount: outboundShippingTaxAmount,

		ReturnShippingAmount:    returnShippingAmount,
		ReturnShippingTaxAmount: returnShippingTaxAmount,

		RefundAmount: refundAmount,

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

// NewForOrderItem creates an unopened-return Refund using the deterministic
// Refund ID.
func NewForOrderItem(
	inquiryID string,
	orderID string,
	paymentID string,
	orderItemIndex int,
	seller SellerIdentity,
	settlementID string,
	merchandiseAmount int,
	merchandiseTaxAmount int,
	transferReversalAmount int,
	currency string,
	createdAt time.Time,
) (Refund, error) {
	id, err := NewID(orderID, orderItemIndex)
	if err != nil {
		return Refund{}, err
	}

	return New(
		id,
		inquiryID,
		orderID,
		paymentID,
		orderItemIndex,
		seller,
		settlementID,
		merchandiseAmount,
		merchandiseTaxAmount,
		transferReversalAmount,
		currency,
		createdAt,
	)
}

// NewOpenedReturnForOrderItem creates an opened-return Refund using the
// deterministic Refund ID.
func NewOpenedReturnForOrderItem(
	inquiryID string,
	orderID string,
	paymentID string,
	orderItemIndex int,
	seller SellerIdentity,
	settlementID string,
	policy OpenedReturnRefundPolicy,
	merchandiseAmount int,
	merchandiseTaxAmount int,
	outboundShippingAmount int,
	outboundShippingTaxAmount int,
	returnShippingAmount int,
	returnShippingTaxAmount int,
	transferReversalAmount int,
	currency string,
	createdAt time.Time,
) (Refund, error) {
	id, err := NewID(orderID, orderItemIndex)
	if err != nil {
		return Refund{}, err
	}

	return NewOpenedReturn(
		id,
		inquiryID,
		orderID,
		paymentID,
		orderItemIndex,
		seller,
		settlementID,
		policy,
		merchandiseAmount,
		merchandiseTaxAmount,
		outboundShippingAmount,
		outboundShippingTaxAmount,
		returnShippingAmount,
		returnShippingTaxAmount,
		transferReversalAmount,
		currency,
		createdAt,
	)
}

// TotalBrandBurdenAmount returns the total company burden represented by this
// Refund.
//
// Return shipping is deliberately excluded from Stripe RefundAmount because it
// was not part of the original purchaser Charge.
func (r Refund) TotalBrandBurdenAmount() (int, error) {
	total, err := safeAddRefundAmount(r.RefundAmount, r.ReturnShippingAmount)
	if err != nil {
		return 0, err
	}

	return safeAddRefundAmount(total, r.ReturnShippingTaxAmount)
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

	if !isStripeRefundID(stripeRefundID) {
		return ErrInvalidStripeRefundID
	}

	if !IsValidStatus(status) || status == StatusCreated {
		return ErrInvalidStatus
	}

	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}

	if r.StripeRefundID != "" && r.StripeRefundID != stripeRefundID {
		return ErrInvalidStripeRefundID
	}

	if !canTransitionRefundStatus(r.Status, status) {
		return ErrInvalidStatusTransition
	}

	next := *r
	next.StripeRefundID = stripeRefundID
	next.Status = status

	switch status {
	case StatusSucceeded:
		if refundedAt == nil || refundedAt.IsZero() {
			return ErrInvalidRefundedAt
		}

		value := refundedAt.UTC()
		next.RefundedAt = &value

	case StatusPending, StatusRequiresAction, StatusFailed, StatusCanceled:
		if refundedAt != nil {
			return ErrInvalidRefundedAt
		}

		next.RefundedAt = nil

	default:
		return ErrInvalidStatus
	}

	next.UpdatedAt = now.UTC()

	if err := next.Validate(); err != nil {
		return err
	}

	*r = next
	return nil
}

// MarkStripeRefundPending records a pending Stripe Refund.
func (r *Refund) MarkStripeRefundPending(stripeRefundID string, now time.Time) error {
	return r.ApplyStripeRefund(stripeRefundID, StatusPending, nil, now)
}

// MarkStripeRefundRequiresAction records a Refund requiring additional action.
func (r *Refund) MarkStripeRefundRequiresAction(stripeRefundID string, now time.Time) error {
	return r.ApplyStripeRefund(stripeRefundID, StatusRequiresAction, nil, now)
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

	value := refundedAt.UTC()
	return r.ApplyStripeRefund(stripeRefundID, StatusSucceeded, &value, now)
}

// MarkStripeRefundFailed records a failed Stripe Refund object.
func (r *Refund) MarkStripeRefundFailed(stripeRefundID string, now time.Time) error {
	return r.ApplyStripeRefund(stripeRefundID, StatusFailed, nil, now)
}

// MarkStripeRefundCanceled records a canceled Stripe Refund object.
func (r *Refund) MarkStripeRefundCanceled(stripeRefundID string, now time.Time) error {
	return r.ApplyStripeRefund(stripeRefundID, StatusCanceled, nil, now)
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
func (r *Refund) MarkTransferReversalPending(now time.Time) error {
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
	case TransferReversalStatusPending, TransferReversalStatusFailedRetryable:

	default:
		return ErrInvalidTransferReversalStatusTransition
	}

	next := *r
	next.TransferReversalStatus = TransferReversalStatusPending
	next.UpdatedAt = now.UTC()

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

	if !isStripeTransferReversalID(stripeTransferReversalID) {
		return ErrInvalidStripeTransferReversalID
	}

	if reversedAt.IsZero() {
		return ErrInvalidTransferReversedAt
	}

	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}

	switch r.TransferReversalStatus {
	case TransferReversalStatusPending, TransferReversalStatusFailedRetryable:

	case TransferReversalStatusSucceeded:
		if r.StripeTransferReversalID == stripeTransferReversalID {
			return nil
		}

		return ErrInvalidStripeTransferReversalID

	default:
		return ErrInvalidTransferReversalStatusTransition
	}

	next := *r
	reversedAt = reversedAt.UTC()

	next.StripeTransferReversalID = stripeTransferReversalID
	next.TransferReversalStatus = TransferReversalStatusSucceeded
	next.TransferReversedAt = &reversedAt
	next.UpdatedAt = now.UTC()

	if err := next.Validate(); err != nil {
		return err
	}

	*r = next
	return nil
}

// MarkTransferReversalFailedRetryable records a retryable Stripe Transfer
// Reversal failure.
func (r *Refund) MarkTransferReversalFailedRetryable(now time.Time) error {
	return r.markTransferReversalFailed(
		TransferReversalStatusFailedRetryable,
		now,
	)
}

// MarkTransferReversalFailed records a terminal Stripe Transfer Reversal
// failure.
func (r *Refund) MarkTransferReversalFailed(now time.Time) error {
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

	if status != TransferReversalStatusFailedRetryable &&
		status != TransferReversalStatusFailed {
		return ErrInvalidTransferReversalStatus
	}

	switch r.TransferReversalStatus {
	case TransferReversalStatusPending, TransferReversalStatusFailedRetryable:

	default:
		return ErrInvalidTransferReversalStatusTransition
	}

	next := *r
	next.StripeTransferReversalID = ""
	next.TransferReversalStatus = status
	next.TransferReversedAt = nil
	next.UpdatedAt = now.UTC()

	if err := next.Validate(); err != nil {
		return err
	}

	*r = next
	return nil
}

// ============================================================
// Completion
// ============================================================

// IsFinanciallyCompleted reports whether all purchaser-refund and seller-side
// Transfer Reversal operations required for the item return have completed.
//
// Return-shipping cost is additional company burden and is not itself represented
// by the Stripe Refund / Transfer Reversal lifecycle in this aggregate.
//
// Order.IsReturnCompleted and Inquiry resolved should only be updated after this
// returns true.
func (r Refund) IsFinanciallyCompleted() bool {
	if r.Status != StatusSucceeded {
		return false
	}

	switch r.TransferReversalStatus {
	case TransferReversalStatusNotRequired, TransferReversalStatusSucceeded:
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
	if r.ID == "" || strings.Contains(r.ID, "/") {
		return ErrInvalidID
	}

	if r.InquiryID == "" || strings.Contains(r.InquiryID, "/") {
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

	seller := r.SellerIdentity()
	if err := seller.Validate(); err != nil {
		return err
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

	if r.OutboundShippingAmount < 0 {
		return ErrInvalidOutboundShippingAmount
	}

	if r.OutboundShippingTaxAmount < 0 {
		return ErrInvalidOutboundShippingTaxAmount
	}

	if r.ReturnShippingAmount < 0 {
		return ErrInvalidReturnShippingAmount
	}

	if r.ReturnShippingTaxAmount < 0 {
		return ErrInvalidReturnShippingTaxAmount
	}

	if err := r.validatePolicyAmounts(); err != nil {
		return err
	}

	if r.RefundAmount <= 0 {
		return ErrInvalidRefundAmount
	}

	expectedRefundAmount, err := calculateRefundAmount(
		r.MerchandiseAmount,
		r.MerchandiseTaxAmount,
		r.OutboundShippingAmount,
		r.OutboundShippingTaxAmount,
	)
	if err != nil {
		return err
	}

	if r.RefundAmount != expectedRefundAmount {
		return ErrRefundAmountMismatch
	}

	if _, err := r.TotalBrandBurdenAmount(); err != nil {
		return err
	}

	if r.Currency != CurrencyJPY {
		return ErrInvalidCurrency
	}

	if !IsValidStatus(r.Status) {
		return ErrInvalidStatus
	}

	if err := r.validateStripeRefundState(); err != nil {
		return err
	}

	if r.TransferReversalAmount < 0 ||
		r.TransferReversalAmount > r.RefundAmount {
		return ErrInvalidTransferReversalAmount
	}

	if !IsValidTransferReversalStatus(r.TransferReversalStatus) {
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

	if r.UpdatedAt.Before(r.CreatedAt) {
		return ErrInvalidUpdatedAt
	}

	if r.RefundedAt != nil && r.RefundedAt.Before(r.CreatedAt) {
		return ErrInvalidRefundedAt
	}

	if r.TransferReversedAt != nil &&
		r.TransferReversedAt.Before(r.CreatedAt) {
		return ErrInvalidTransferReversedAt
	}

	return nil
}

// SellerIdentity returns the immutable seller payout identity stored by this
// Refund.
//
// SellerType is mandatory. No legacy seller inference is performed.
func (r Refund) SellerIdentity() SellerIdentity {
	return SellerIdentity{
		Type:            r.SellerType,
		CompanyID:       r.CompanyID,
		AccountID:       r.AccountID,
		AvatarID:        r.AvatarID,
		UserID:          r.UserID,
		PayoutAccountID: r.PayoutAccountID,
		StripeAccountID: r.StripeAccountID,
	}
}

func (r Refund) validatePolicyAmounts() error {
	// Empty Policy represents the unopened-return flow.
	if r.Policy == "" {
		if r.OutboundShippingAmount != 0 ||
			r.OutboundShippingTaxAmount != 0 ||
			r.ReturnShippingAmount != 0 ||
			r.ReturnShippingTaxAmount != 0 {
			return ErrInvalidOpenedReturnAmounts
		}

		return nil
	}

	if err := ValidateOpenedReturnRefundPolicy(r.Policy); err != nil {
		return err
	}

	switch r.Policy {
	case OpenedReturnRefundHalfMerchandise,
		OpenedReturnRefundMerchandiseOnly:
		if r.OutboundShippingAmount != 0 ||
			r.OutboundShippingTaxAmount != 0 ||
			r.ReturnShippingAmount != 0 ||
			r.ReturnShippingTaxAmount != 0 {
			return ErrInvalidOpenedReturnAmounts
		}

	case OpenedReturnRefundMerchandiseRoundTripShipping:
		// Zero shipping is valid when the original or return shipment is free.
		// Exact authoritative values are validated against Order snapshots by
		// the application / Order domain.

	default:
		return ErrInvalidOpenedReturnRefundPolicy
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
		if !isStripeRefundID(r.StripeRefundID) {
			return ErrInvalidStripeRefundID
		}

		if r.RefundedAt != nil {
			return ErrInvalidRefundedAt
		}

		return nil

	case StatusSucceeded:
		if !isStripeRefundID(r.StripeRefundID) {
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
		if r.TransferReversalStatus != TransferReversalStatusNotRequired {
			return ErrInvalidTransferReversalStatus
		}

		if r.StripeTransferReversalID != "" ||
			r.TransferReversedAt != nil {
			return ErrInvalidTransferReversalStatus
		}

		return nil
	}

	if r.TransferReversalStatus == TransferReversalStatusNotRequired {
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

		if !isStripeTransferReversalID(r.StripeTransferReversalID) {
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
// Refund Amount
// ============================================================

func calculateRefundAmount(
	merchandiseAmount int,
	merchandiseTaxAmount int,
	outboundShippingAmount int,
	outboundShippingTaxAmount int,
) (int, error) {
	if merchandiseAmount < 0 {
		return 0, ErrInvalidMerchandiseAmount
	}

	if merchandiseTaxAmount < 0 {
		return 0, ErrInvalidMerchandiseTaxAmount
	}

	if outboundShippingAmount < 0 {
		return 0, ErrInvalidOutboundShippingAmount
	}

	if outboundShippingTaxAmount < 0 {
		return 0, ErrInvalidOutboundShippingTaxAmount
	}

	total, err := safeAddRefundAmount(
		merchandiseAmount,
		merchandiseTaxAmount,
	)
	if err != nil {
		return 0, err
	}

	total, err = safeAddRefundAmount(
		total,
		outboundShippingAmount,
	)
	if err != nil {
		return 0, err
	}

	total, err = safeAddRefundAmount(
		total,
		outboundShippingTaxAmount,
	)
	if err != nil {
		return 0, err
	}

	if total <= 0 {
		return 0, ErrInvalidRefundAmount
	}

	return total, nil
}

func safeAddRefundAmount(left int, right int) (int, error) {
	if left < 0 || right < 0 {
		return 0, ErrInvalidRefundAmount
	}

	maxInt := int(^uint(0) >> 1)
	if left > maxInt-right {
		return 0, ErrInvalidRefundAmount
	}

	return left + right, nil
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

func isStripeAccountID(value string) bool {
	return strings.HasPrefix(value, "acct_") &&
		len(value) > len("acct_")
}

func isStripeRefundID(value string) bool {
	return strings.HasPrefix(value, "re_") &&
		len(value) > len("re_")
}

func isStripeTransferReversalID(value string) bool {
	return strings.HasPrefix(value, "trr_") &&
		len(value) > len("trr_")
}

// ============================================================
// Debug
// ============================================================

// String provides a concise identifier for logs without exposing Stripe IDs.
func (r Refund) String() string {
	return fmt.Sprintf(
		"refund{id=%s orderId=%s itemIndex=%d policy=%s status=%s reversalStatus=%s}",
		r.ID,
		r.OrderID,
		r.OrderItemIndex,
		r.Policy,
		r.Status,
		r.TransferReversalStatus,
	)
}
