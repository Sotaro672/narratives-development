// backend/internal/domain/refund/respository_port.go
package refund

import (
	"context"
	"errors"
	"time"
)

// ============================================================
// Create Input
// ============================================================

// CreateRefundInput represents the persistence input for one item-level Refund.
//
// RefundID should be generated deterministically from:
//
//	orderID + orderItemIndex
//
// by NewID.
//
// A newly created Refund always starts with:
//
//	StatusCreated
//	StripeRefundID = ""
//	RefundedAt = nil
//
// Policy:
//
//	Policy == ""
//
// represents the legacy / unopened-return flow.
//
// For an opened return, Policy must be one of the valid
// OpenedReturnRefundPolicy values.
//
// Monetary values must be calculated by the application / Order domain from
// authoritative persisted snapshots. The frontend must never supply refund
// amounts directly.
//
// RefundAmount itself is not accepted here because Refund derives it from:
//
//	MerchandiseAmount
//	+ MerchandiseTaxAmount
//	+ OutboundShippingAmount
//	+ OutboundShippingTaxAmount
//
// ReturnShippingAmount and ReturnShippingTaxAmount are additional company
// burden and are intentionally excluded from the purchaser Stripe Refund.
//
// TransferReversalStatus is derived by Refund constructors:
//
// - transferReversalAmount == 0 -> not_required
// - transferReversalAmount > 0  -> pending
//
// Stripe-generated IDs are intentionally not accepted during creation.
type CreateRefundInput struct {
	RefundID string

	InquiryID string

	OrderID        string
	PaymentID      string
	OrderItemIndex int

	CompanyID string
	AccountID string

	SettlementID string

	Policy OpenedReturnRefundPolicy

	MerchandiseAmount    int
	MerchandiseTaxAmount int

	OutboundShippingAmount    int
	OutboundShippingTaxAmount int

	ReturnShippingAmount    int
	ReturnShippingTaxAmount int

	TransferReversalAmount int

	Currency string

	CreatedAt time.Time
}

// ============================================================
// Update Operation
// ============================================================

// UpdateRefundOperation identifies one validated Refund domain transition.
//
// Refund contains two independent but coordinated state machines:
//
//  1. purchaser-side Stripe Refund
//  2. seller-side Stripe Transfer Reversal
//
// UpdateByID must not mutate persistence fields directly. The repository must
// load the current Refund and execute the corresponding Refund domain method.
type UpdateRefundOperation string

const (
	UpdateOperationApplyStripeRefund UpdateRefundOperation = "apply_stripe_refund"

	UpdateOperationMarkTransferReversalPending UpdateRefundOperation = "mark_transfer_reversal_pending"

	UpdateOperationMarkTransferReversalSucceeded UpdateRefundOperation = "mark_transfer_reversal_succeeded"

	UpdateOperationMarkTransferReversalFailedRetryable UpdateRefundOperation = "mark_transfer_reversal_failed_retryable"

	UpdateOperationMarkTransferReversalFailed UpdateRefundOperation = "mark_transfer_reversal_failed"
)

var AllowedUpdateOperations = map[UpdateRefundOperation]struct{}{
	UpdateOperationApplyStripeRefund:                   {},
	UpdateOperationMarkTransferReversalPending:         {},
	UpdateOperationMarkTransferReversalSucceeded:       {},
	UpdateOperationMarkTransferReversalFailedRetryable: {},
	UpdateOperationMarkTransferReversalFailed:          {},
}

func IsValidUpdateOperation(
	operation UpdateRefundOperation,
) bool {
	if operation == "" {
		return false
	}

	_, ok := AllowedUpdateOperations[operation]
	return ok
}

// ============================================================
// Update Input
// ============================================================

// UpdateRefundInput represents one validated Refund state transition.
//
// The fields used depend on Operation.
//
// apply_stripe_refund:
//
//	StripeRefundID
//	RefundStatus
//	RefundedAt
//	UpdatedAt
//
// mark_transfer_reversal_pending:
//
//	UpdatedAt
//
// mark_transfer_reversal_succeeded:
//
//	StripeTransferReversalID
//	TransferReversedAt
//	UpdatedAt
//
// mark_transfer_reversal_failed_retryable:
//
//	UpdatedAt
//
// mark_transfer_reversal_failed:
//
//	UpdatedAt
//
// Repository implementations must reject fields that are invalid for the
// requested Operation and execute the corresponding Refund domain method.
//
// Direct field mutation must not be used because doing so could bypass:
//
// - Refund status transition validation
// - Stripe Refund ID validation
// - Transfer Reversal status validation
// - purchaser Refund completion requirement
// - timestamp invariants
type UpdateRefundInput struct {
	Operation UpdateRefundOperation

	StripeRefundID string
	RefundStatus   RefundStatus
	RefundedAt     *time.Time

	StripeTransferReversalID string
	TransferReversedAt       *time.Time

	UpdatedAt time.Time
}

// ============================================================
// Repository Port
// ============================================================

// RepositoryPort is the persistence contract for item-level Refund aggregates.
//
// Refund is intentionally separate from:
//
// - payment.RepositoryPort
// - settlement.Repository
// - Order return-request state
// - Inquiry status
//
// Payment continues to own full-payment refund state.
//
// Refund owns one item-level partial purchaser refund and the seller-side
// partial Transfer Reversal attributable to that same returned Order item.
//
// For opened returns, Refund additionally records:
//
// - selected refund Policy
// - outbound shipping refunded against the original Charge
// - outbound shipping consumption tax
// - return shipping borne by the seller
// - return shipping consumption tax
//
// Return shipping is additional company burden and is not necessarily part of
// the purchaser's original Stripe Charge.
//
// The application layer coordinates:
//
//	Refund
//		-> Order return completion
//		-> Inquiry resolution
//
// Order and Inquiry must not be marked completed until
// Refund.IsFinanciallyCompleted returns true.
type RepositoryPort interface {
	// GetByID returns one Refund by its deterministic Refund document ID.
	GetByID(
		ctx context.Context,
		refundID string,
	) (*Refund, error)

	// GetByInquiryID returns the Refund created from one return Inquiry.
	//
	// One return Inquiry may create at most one Refund.
	//
	// Implementations must treat multiple Refund documents for the same
	// InquiryID as a persistence conflict.
	GetByInquiryID(
		ctx context.Context,
		inquiryID string,
	) (*Refund, error)

	// GetByOrderItem returns the Refund belonging to one Order item.
	//
	// The current return policy allows at most one item-level Refund for one
	// Order item.
	//
	// The deterministic Refund ID is generated from:
	//
	//	orderID + orderItemIndex
	GetByOrderItem(
		ctx context.Context,
		orderID string,
		orderItemIndex int,
	) (*Refund, error)

	// ListByOrderID returns every item-level Refund belonging to one Order.
	//
	// Results should be returned deterministically by OrderItemIndex and then
	// Refund ID.
	ListByOrderID(
		ctx context.Context,
		orderID string,
	) ([]Refund, error)

	// ListByPaymentID returns every item-level Refund belonging to one Payment.
	//
	// PaymentID currently equals OrderID, but this method keeps the financial
	// boundary explicit for application and reconciliation use.
	ListByPaymentID(
		ctx context.Context,
		paymentID string,
	) ([]Refund, error)

	// ListBySettlementID returns item-level Refunds whose seller-side amount is
	// attributable to one Settlement.
	//
	// This is required when calculating cumulative partial Transfer Reversals
	// against a Settlement.
	ListBySettlementID(
		ctx context.Context,
		settlementID string,
	) ([]Refund, error)

	// ListByCompanyID returns Refunds attributable to one seller Company.
	//
	// This may be used by future Console refund management and reconciliation
	// queries.
	ListByCompanyID(
		ctx context.Context,
		companyID string,
	) ([]Refund, error)

	// Create creates one item-level Refund.
	//
	// Implementations must:
	//
	// - reject an existing RefundID instead of overwriting it
	// - reject a duplicate InquiryID
	// - reject a duplicate OrderID + OrderItemIndex
	// - use Refund.New when Policy == ""
	// - use Refund.NewOpenedReturn when Policy is set
	// - persist the complete validated Refund
	//
	// Financial values must never be accepted directly from an untrusted
	// frontend request. The application layer must calculate them from the
	// authoritative Order and Settlement state before calling Create.
	//
	// StripeRefundID and StripeTransferReversalID are not accepted during
	// creation because neither Stripe operation has occurred yet.
	Create(
		ctx context.Context,
		in CreateRefundInput,
	) (*Refund, error)

	// UpdateByID executes one validated Refund domain state transition.
	//
	// Implementations must load the current Refund before applying the update.
	//
	// Operation mapping:
	//
	// apply_stripe_refund:
	//
	//	Refund.ApplyStripeRefund
	//
	// mark_transfer_reversal_pending:
	//
	//	Refund.MarkTransferReversalPending
	//
	// mark_transfer_reversal_succeeded:
	//
	//	Refund.MarkTransferReversalSucceeded
	//
	// mark_transfer_reversal_failed_retryable:
	//
	//	Refund.MarkTransferReversalFailedRetryable
	//
	// mark_transfer_reversal_failed:
	//
	//	Refund.MarkTransferReversalFailed
	//
	// The repository must validate the complete resulting Refund before
	// persisting it.
	//
	// Implementations should perform the read, domain transition, and write
	// atomically where the persistence backend supports transactions.
	UpdateByID(
		ctx context.Context,
		refundID string,
		in UpdateRefundInput,
	) (*Refund, error)
}

// ============================================================
// Repository Errors
// ============================================================

var (
	ErrNotFound = errors.New(
		"refund: not found",
	)

	ErrConflict = errors.New(
		"refund: conflict",
	)

	ErrDuplicateInquiry = errors.New(
		"refund: refund already exists for inquiry",
	)

	ErrDuplicateOrderItem = errors.New(
		"refund: refund already exists for order item",
	)

	ErrInvalidUpdateOperation = errors.New(
		"refund: invalid update operation",
	)
)
