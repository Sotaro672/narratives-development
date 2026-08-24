// backend/internal/domain/settlement/repository_port.go
package settlement

import (
	"context"
	"errors"
	"time"
)

// CreateSettlementInput represents the persistence input used when a new
// seller-side settlement is created.
//
// SettlementID should be generated deterministically from:
//
//	paymentID + accountID
//
// so that one Payment cannot create duplicate Settlement records for the same
// seller Account.
//
// A newly created Settlement must be pending.
//
// Payment success only creates the seller-side financial allocation.
// ready represents the dispatch boundary and must only be reached later through
// a validated pending -> ready state transition.
//
// Status may be empty so a repository implementation can normalize it to
// pending. Any non-empty status other than pending must be rejected.
type CreateSettlementInput struct {
	SettlementID string

	OrderID   string
	PaymentID string

	CompanyID string
	AccountID string

	StripeAccountID string

	StripePaymentIntentID string
	StripeChargeID        string

	TransferGroup string

	GrossAmount       int
	PlatformFeeAmount int
	TransferAmount    int

	Currency string

	Status SettlementStatus
}

// UpdateSettlementInput represents a partial Settlement update.
//
// nil means that the corresponding field is not updated.
//
// StripeTransferID is normally set when a Stripe Connect Transfer succeeds.
//
// StripeTransferReversalID is normally set when a completed Stripe Transfer
// is reversed.
//
// Status changes should be performed together with the fields required by the
// target Settlement state.
//
// In particular, pending -> ready is the explicit seller dispatch boundary.
// Payment success alone must not transition a Settlement to ready.
//
// failed_retryable does not need to transition back to ready. A retryable
// Settlement has already crossed the dispatch boundary and may be claimed
// directly for another transfer attempt.
type UpdateSettlementInput struct {
	StripeTransferID *string

	StripeTransferReversalID *string

	Status *SettlementStatus

	ErrorType *string
	ErrorCode *string
	ErrorMsg  *string
}

// ListTransferCandidatesInput represents the recovery boundary used by
// Settlement reconciliation.
//
// The repository must return only Settlements that may need another transfer
// task:
//
// - ready
// - failed_retryable
// - transferring whose UpdatedAt is equal to or before StaleBefore
//
// pending must never be returned because pending means the seller Account has
// not yet crossed the dispatch boundary.
//
// A non-stale transferring Settlement must not be returned because another
// worker may still own its transfer lease.
//
// Limit is the maximum total number of Settlements returned by one scan.
// The application layer must provide a positive value.
type ListTransferCandidatesInput struct {
	StaleBefore time.Time

	Limit int
}

// Repository is the persistence contract for seller-side Stripe Connect
// settlements.
//
// Settlement is intentionally separate from:
//
// - payment.RepositoryPort
// - Order item token transfer state
//
// One Payment may have multiple Settlement records when one Order contains
// products belonging to multiple seller Accounts.
type Repository interface {
	// GetByID returns a Settlement by its Settlement document ID.
	GetByID(
		ctx context.Context,
		settlementID string,
	) (Settlement, error)

	// ListByPaymentID returns every seller Settlement belonging to one Payment.
	//
	// The result is expected to contain at most one Settlement per AccountID.
	ListByPaymentID(
		ctx context.Context,
		paymentID string,
	) ([]Settlement, error)

	// ListByOrderID returns every seller Settlement belonging to one Order.
	//
	// PaymentID currently equals OrderID, but this method keeps the Order
	// boundary explicit for application/query use.
	ListByOrderID(
		ctx context.Context,
		orderID string,
	) ([]Settlement, error)

	// ListByCompanyID returns Settlements attributable to one Company.
	//
	// This is used by Console transaction and settlement management queries.
	ListByCompanyID(
		ctx context.Context,
		companyID string,
	) ([]Settlement, error)

	// ListByAccountID returns Settlements attributable to one payout Account.
	ListByAccountID(
		ctx context.Context,
		accountID string,
	) ([]Settlement, error)

	// ListTransferCandidates returns Settlements that may require a transfer
	// Cloud Task to be present or retried.
	//
	// Implementations must include:
	//
	// - ready
	// - failed_retryable
	// - transferring with UpdatedAt <= StaleBefore
	//
	// Implementations must exclude:
	//
	// - pending
	// - non-stale transferring
	// - transferred
	// - failed
	// - canceled
	// - reversed
	//
	// pending is excluded because it represents a Settlement whose seller
	// Account has not yet completed the dispatch boundary.
	//
	// The result must be deterministic and contain no duplicate Settlement ID.
	//
	// Implementations should order older UpdatedAt values first and use ID as
	// the tie breaker before applying Limit.
	ListTransferCandidates(
		ctx context.Context,
		in ListTransferCandidatesInput,
	) ([]Settlement, error)

	// Create creates a new pending Settlement.
	//
	// Implementations must reject an existing SettlementID instead of
	// overwriting it so deterministic Settlement IDs provide idempotency.
	//
	// Implementations must persist the new Settlement as pending.
	//
	// An empty input Status may be normalized to pending. Any non-empty Status
	// other than pending must be rejected.
	//
	// ready must never be accepted during creation because ready means that the
	// corresponding seller Account has completed dispatch and is eligible for
	// Stripe Transfer.
	//
	// Implementations must reconstruct or validate the complete Settlement
	// entity before persisting it.
	Create(
		ctx context.Context,
		in CreateSettlementInput,
	) (Settlement, error)

	// UpdateByID partially updates an existing Settlement.
	//
	// Implementations must read the current Settlement and execute validated
	// domain state transitions before persisting the resulting entity.
	//
	// In particular:
	//
	// - pending -> ready is the seller dispatch boundary
	// - ready -> transferring begins Stripe Transfer processing
	// - failed_retryable -> transferring retries a previously eligible transfer
	//
	// Implementations must validate the complete resulting Settlement before
	// persisting the update.
	UpdateByID(
		ctx context.Context,
		settlementID string,
		patch UpdateSettlementInput,
	) (Settlement, error)
}

// Repository errors.
var (
	ErrNotFound = errors.New(
		"settlement: not found",
	)
	ErrConflict = errors.New(
		"settlement: conflict",
	)
)
