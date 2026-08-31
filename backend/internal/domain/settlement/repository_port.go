// backend/internal/domain/settlement/repository_port.go
package settlement

import (
	"context"
	"errors"
	"time"
)

// CreateSettlementInput represents the persistence input used when a new
// primary-sale seller Settlement is created.
//
// SettlementID must be generated deterministically with NewID from:
//
//	paymentID + "_account_" + accountID
//
// so that one Payment cannot create duplicate Settlement records for the same
// Account payout identity.
//
// Seller contains the immutable primary-sale Account payout identity captured
// at Order/payment time.
//
// A newly created Settlement must be pending.
//
// Payment success only creates the seller-side financial allocation.
// ready represents the dispatch boundary and must only be reached later through
// a validated pending -> ready state transition.
//
// Status may be empty so a repository implementation can normalize it to
// pending. Any non-empty status other than pending must be rejected.
//
// Consumer resale proceeds must not be represented by Settlement. They belong
// to the salesReceivable domain.
type CreateSettlementInput struct {
	SettlementID string

	OrderID   string
	PaymentID string

	Seller SellerIdentity

	StripePaymentIntentID string
	StripeChargeID        string

	TransferGroup string

	GrossAmount       int
	PlatformFeeAmount int
	TransferAmount    int

	Currency string
	Status   SettlementStatus
}

// UpdateSettlementInput represents a partial Settlement state transition.
//
// nil means that the corresponding field is not updated.
//
// StripeTransferID is set when a Stripe Connect Transfer succeeds.
//
// StripeTransferReversalID is set when a completed Stripe Transfer is reversed.
//
// Seller identity is immutable and therefore cannot be changed through this
// input.
//
// pending -> ready is the explicit seller dispatch boundary. Payment success
// alone must not transition a Settlement to ready.
//
// failed_retryable does not transition back to ready. A retryable Settlement
// has already crossed the dispatch boundary and may be claimed directly for
// another transfer attempt.
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
// pending must never be returned because the seller has not yet crossed the
// dispatch boundary.
//
// A non-stale transferring Settlement must not be returned because another
// worker may still own its transfer lease.
//
// Limit is the maximum total number of Settlements returned by one scan.
type ListTransferCandidatesInput struct {
	StaleBefore time.Time
	Limit       int
}

// Repository is the persistence contract for primary-sale seller-side Stripe
// Connect Settlements.
//
// Settlement is intentionally separate from:
//
// - payment.RepositoryPort
// - Order item token transfer state
// - salesReceivable.Repository
//
// Settlement represents only:
//
//	AMOL Stripe Platform -> primary-sale Stripe Connected Account
//
// One Payment may contain multiple primary-sale Account sellers and therefore
// multiple Settlement records.
//
// Multiple Brands sharing the same AccountID are aggregated into the same
// Settlement.
//
// A Payment must contain at most one Settlement for each AccountID.
//
// Consumer resale proceeds must never be stored or queried through this
// repository.
type Repository interface {
	// GetByID returns one Settlement by deterministic Settlement document ID.
	GetByID(
		ctx context.Context,
		settlementID string,
	) (Settlement, error)

	// ListByPaymentID returns every primary-sale Settlement belonging to one
	// Payment.
	//
	// The result must contain at most one Settlement per AccountID.
	ListByPaymentID(
		ctx context.Context,
		paymentID string,
	) ([]Settlement, error)

	// ListByOrderID returns every primary-sale Settlement belonging to one Order.
	//
	// PaymentID currently equals OrderID, but this method keeps the Order
	// boundary explicit for application/query use.
	ListByOrderID(
		ctx context.Context,
		orderID string,
	) ([]Settlement, error)

	// ListByCompanyID returns primary-sale Settlements attributable to one
	// Company.
	//
	// This is used by Console transaction and settlement management queries.
	ListByCompanyID(
		ctx context.Context,
		companyID string,
	) ([]Settlement, error)

	// ListByAccountID returns primary-sale Settlements attributable to one Stripe
	// Connected Account payout identity.
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
	// The result must be deterministic and contain no duplicate Settlement ID.
	//
	// Implementations should order older UpdatedAt values first and use ID as
	// the tie breaker before applying Limit.
	ListTransferCandidates(
		ctx context.Context,
		in ListTransferCandidatesInput,
	) ([]Settlement, error)

	// Create creates one new pending primary-sale Settlement.
	//
	// Implementations must reject an existing SettlementID instead of
	// overwriting it so deterministic Settlement IDs provide idempotency.
	//
	// Seller must be SellerTypeAccount and requires:
	//
	// - CompanyID
	// - AccountID
	// - StripeAccountID
	//
	// Implementations must persist the complete immutable Seller identity.
	//
	// An empty input Status may be normalized to pending. Any non-empty Status
	// other than pending must be rejected.
	//
	// ready must never be accepted during creation because ready means that the
	// corresponding seller has completed dispatch and is eligible for Stripe
	// Transfer.
	//
	// Consumer resale financial state must not be persisted through this method.
	Create(
		ctx context.Context,
		in CreateSettlementInput,
	) (Settlement, error)

	// UpdateByID executes one validated Settlement state transition.
	//
	// Implementations must read the current Settlement and execute domain state
	// transitions before persisting the resulting entity.
	//
	// Seller identity must not be modified by this operation.
	//
	// In particular:
	//
	// - pending -> ready is the seller dispatch boundary
	// - ready -> transferring begins Stripe Transfer processing
	// - failed_retryable -> transferring retries a previously eligible transfer
	// - transferring -> transferred records Stripe Transfer success
	// - transferring -> failed_retryable/failed records Stripe Transfer failure
	// - pending/ready/failed_retryable -> canceled prevents seller payout
	// - transferred -> reversed records a completed Stripe Transfer reversal
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
