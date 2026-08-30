// backend/internal/domain/settlement/repository_port.go
package settlement

import (
	"context"
	"errors"
	"time"
)

// CreateSettlementInput represents the persistence input used when a new
// seller-side Settlement is created.
//
// SettlementID should be generated deterministically with NewID from:
//
//	account seller:
//	  paymentID + "_account_" + accountID
//
//	avatar seller:
//	  paymentID + "_avatar_" + payoutAccountID
//
// so that one Payment cannot create duplicate Settlement records for the same
// seller payout identity.
//
// Seller contains the immutable payout identity captured at Order/payment time.
// The repository must persist its fields into the resulting Settlement.
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

// UpdateSettlementInput represents a partial Settlement update.
//
// nil means that the corresponding field is not updated.
//
// StripeTransferID is normally set when a Stripe Connect Transfer succeeds.
//
// StripeTransferReversalID is normally set when a completed Stripe Transfer
// is reversed.
//
// Seller identity is intentionally immutable and therefore cannot be changed
// through this input.
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
// pending must never be returned because pending means the seller has not yet
// crossed the dispatch boundary.
//
// A non-stale transferring Settlement must not be returned because another
// worker may still own its transfer lease.
//
// Limit is the maximum total number of Settlements returned by one scan.
// The application layer must provide a positive value.
type ListTransferCandidatesInput struct {
	StaleBefore time.Time
	Limit       int
}

// Repository is the persistence contract for seller-side Stripe Connect
// Settlements.
//
// Settlement is intentionally separate from:
//
// - payment.RepositoryPort
// - Order item token transfer state
//
// One Payment may have multiple Settlement records when one Order contains
// multiple seller payout identities.
//
// Primary List sales use SellerTypeAccount.
// Resale transactions use SellerTypeAvatar.
//
// A Payment must contain at most one Settlement for each normalized seller key:
//
// - account seller: AccountID
// - avatar seller: PayoutAccountID
type Repository interface {
	// GetByID returns a Settlement by its Settlement document ID.
	GetByID(
		ctx context.Context,
		settlementID string,
	) (Settlement, error)

	// ListByPaymentID returns every seller Settlement belonging to one Payment.
	//
	// The result is expected to contain at most one Settlement per normalized
	// seller payout identity.
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

	// ListByCompanyID returns primary-sale Settlements attributable to one
	// Company.
	//
	// Avatar/resale Settlements do not have CompanyID and must not be returned.
	//
	// This is used by Console transaction and settlement management queries.
	ListByCompanyID(
		ctx context.Context,
		companyID string,
	) ([]Settlement, error)

	// ListByAccountID returns primary-sale Settlements attributable to one
	// company payout Account.
	//
	// Avatar/resale Settlements do not have AccountID and must not be returned.
	ListByAccountID(
		ctx context.Context,
		accountID string,
	) ([]Settlement, error)

	// ListByAvatarID returns resale Settlements attributable to one Avatar.
	//
	// Account/primary-sale Settlements do not have AvatarID and must not be
	// returned.
	ListByAvatarID(
		ctx context.Context,
		avatarID string,
	) ([]Settlement, error)

	// ListByPayoutAccountID returns resale Settlements attributable to one
	// consumer payout account.
	//
	// PayoutAccountID is the stable payout identity used for resale Settlement
	// aggregation and deterministic Settlement IDs.
	ListByPayoutAccountID(
		ctx context.Context,
		payoutAccountID string,
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
	// pending is excluded because it represents a Settlement whose seller has
	// not yet completed the dispatch boundary.
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
	// Implementations must persist the complete immutable Seller identity.
	//
	// SellerTypeAccount requires:
	//
	// - CompanyID
	// - AccountID
	// - StripeAccountID
	//
	// SellerTypeAvatar requires:
	//
	// - AvatarID
	// - UserID
	// - PayoutAccountID
	// - StripeAccountID
	//
	// Implementations must persist the new Settlement as pending.
	//
	// An empty input Status may be normalized to pending. Any non-empty Status
	// other than pending must be rejected.
	//
	// ready must never be accepted during creation because ready means that the
	// corresponding seller has completed dispatch and is eligible for Stripe
	// Transfer.
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
	// Seller identity must not be modified by this operation.
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
