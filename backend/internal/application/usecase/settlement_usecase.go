// backend/internal/application/usecase/settlement_usecase.go
package usecase

import (
	"context"
	"errors"
	"time"

	orderdom "narratives/internal/domain/order"
	paymentdom "narratives/internal/domain/payment"
	settlementdom "narratives/internal/domain/settlement"
)

// ============================================================
// Calculator Port
// ============================================================

// SettlementCalculator calculates seller-side settlement allocations from an
// authoritative Order and Payment.
//
// Calculation rules such as:
// - item tax allocation
// - shipping allocation
// - platform fee
// - integer rounding
//
// must remain outside SettlementUsecase.
//
// The calculator must guarantee that allocations are grouped by seller payout
// identity.
//
// Primary List sales are grouped by AccountID.
// Resale transactions are grouped by PayoutAccountID.
type SettlementCalculator interface {
	Calculate(
		ctx context.Context,
		order orderdom.Order,
		payment paymentdom.Payment,
	) ([]settlementdom.Allocation, error)
}

// ============================================================
// Stripe Transfer Port
// ============================================================

// StripeSettlementTransferGateway executes Stripe Connect Transfers.
//
// This Transfer is a fiat settlement transfer:
//
//	AMOL Stripe Platform -> Stripe Connected Account
//
// It is unrelated to Solana token transfer.
type StripeSettlementTransferGateway interface {
	CreateTransfer(
		ctx context.Context,
		in CreateStripeSettlementTransferInput,
	) (*CreateStripeSettlementTransferResult, error)
}

// CreateStripeSettlementTransferInput represents one Stripe Connect Transfer.
//
// Seller is the immutable seller payout identity stored in the Settlement.
//
// DestinationStripeAccountID remains explicit because it is the actual Stripe
// destination used by the transfer request. It must match
// Seller.StripeAccountID before the gateway is called.
type CreateStripeSettlementTransferInput struct {
	Amount int

	Currency string

	DestinationStripeAccountID string

	// SourceTransaction is the Stripe Charge ID.
	SourceTransaction string

	TransferGroup string

	IdempotencyKey string

	OrderID      string
	PaymentID    string
	SettlementID string

	Seller settlementdom.SellerIdentity
}

type CreateStripeSettlementTransferResult struct {
	StripeTransferID string
}

// RetryableStripeSettlementError may be implemented by a Stripe adapter error
// when the adapter can determine whether the failure should be retried.
//
// Unknown infrastructure errors are treated as retryable because Stripe
// Transfer requests use deterministic idempotency keys.
type RetryableStripeSettlementError interface {
	error
	Retryable() bool
}

// StripeSettlementErrorMetadata may be implemented by a Stripe adapter error
// to expose Stripe error type/code without coupling the application layer to
// Stripe SDK types.
type StripeSettlementErrorMetadata interface {
	error
	ErrorType() string
	ErrorCode() string
}

// ============================================================
// Transfer Queue Port
// ============================================================

// SettlementTransferQueue is the minimal outbound contract used by settlement
// reconciliation.
//
// The queue payload must contain only SettlementID. Financial values and seller
// identity must be loaded again from the authoritative Settlement document by
// the worker.
type SettlementTransferQueue interface {
	EnqueueSettlementTransfer(
		ctx context.Context,
		settlementID string,
	) error
}

// ============================================================
// Repository Port
// ============================================================

// SettlementTransferRepository extends the domain Settlement repository with
// atomic state transitions required for safe financial transfer execution.
//
// CreateStripeTransfer must never be called after a plain GetByID followed by
// a non-transactional status update. Two workers could otherwise send the same
// Settlement concurrently.
//
// ClaimForTransfer must atomically:
//  1. Read the Settlement.
//  2. Accept ready or failed_retryable.
//  3. Accept transferring only when UpdatedAt is not after staleBefore.
//  4. Change/keep status as transferring.
//  5. Persist UpdatedAt as now.
//  6. Return Claimed=true.
//
// If another worker still owns a non-stale transferring claim or the
// Settlement is already completed/terminal, Claimed must be false.
type SettlementTransferRepository interface {
	settlementdom.Repository

	ClaimForTransfer(
		ctx context.Context,
		settlementID string,
		now time.Time,
		staleBefore time.Time,
	) (ClaimSettlementTransferResult, error)

	CompleteTransfer(
		ctx context.Context,
		settlementID string,
		stripeTransferID string,
		now time.Time,
	) (settlementdom.Settlement, error)

	FailTransfer(
		ctx context.Context,
		settlementID string,
		status settlementdom.SettlementStatus,
		errorType *string,
		errorCode *string,
		errorMsg *string,
		now time.Time,
	) (settlementdom.Settlement, error)
}

type ClaimSettlementTransferResult struct {
	Settlement settlementdom.Settlement
	Claimed    bool
}

// ============================================================
// Errors
// ============================================================

var (
	ErrSettlementRepositoryMissing = errors.New(
		"settlement: repository is not configured",
	)
	ErrSettlementCalculatorMissing = errors.New(
		"settlement: calculator is not configured",
	)
	ErrSettlementStripeTransferGatewayMissing = errors.New(
		"settlement: Stripe transfer gateway is not configured",
	)
	ErrSettlementTransferQueueMissing = errors.New(
		"settlement: transfer queue is not configured",
	)
	ErrSettlementOrderIDInvalid = errors.New(
		"settlement: invalid order id",
	)
	ErrSettlementPaymentIDInvalid = errors.New(
		"settlement: invalid payment id",
	)
	ErrSettlementPaymentOrderMismatch = errors.New(
		"settlement: payment does not belong to order",
	)
	ErrSettlementPaymentNotSucceeded = errors.New(
		"settlement: payment is not succeeded",
	)
	ErrSettlementStripePaymentIntentIDMissing = errors.New(
		"settlement: stripe payment intent id is missing",
	)
	ErrSettlementStripeChargeIDMissing = errors.New(
		"settlement: stripe charge id is missing",
	)
	ErrSettlementTransferGroupMissing = errors.New(
		"settlement: transfer group is missing",
	)
	ErrSettlementAllocationEmpty = errors.New(
		"settlement: allocation is empty",
	)
	ErrSettlementAllocationInvalid = errors.New(
		"settlement: invalid allocation",
	)
	ErrSettlementDuplicateSeller = errors.New(
		"settlement: duplicate seller allocation",
	)
	ErrSettlementAllocationAmountMismatch = errors.New(
		"settlement: allocation total does not match payment amount",
	)
	ErrSettlementTransferNotReady = errors.New(
		"settlement: transfer is not ready",
	)
	ErrSettlementStripeTransferResultEmpty = errors.New(
		"settlement: Stripe transfer result is empty",
	)
	ErrSettlementStripeTransferIDEmpty = errors.New(
		"settlement: Stripe transfer id is empty",
	)
	ErrSettlementUnsupportedOrderItem = errors.New(
		"settlement: unsupported order item",
	)
)

// ============================================================
// Usecase
// ============================================================

const (
	defaultSettlementTransferLease = 15 * time.Minute

	defaultSettlementTransferDispatchLimit = 50
	maxSettlementTransferDispatchLimit     = 200
)

type SettlementUsecase struct {
	repo SettlementTransferRepository

	calculator SettlementCalculator

	stripeTransferGateway StripeSettlementTransferGateway

	transferLease time.Duration

	now func() time.Time
}

type NewSettlementUsecaseInput struct {
	Repository SettlementTransferRepository

	Calculator SettlementCalculator

	StripeTransferGateway StripeSettlementTransferGateway

	TransferLease time.Duration

	Now func() time.Time
}

func NewSettlementUsecase(
	in NewSettlementUsecaseInput,
) *SettlementUsecase {
	now := in.Now
	if now == nil {
		now = time.Now
	}

	transferLease := in.TransferLease
	if transferLease <= 0 {
		transferLease = defaultSettlementTransferLease
	}

	return &SettlementUsecase{
		repo:                  in.Repository,
		calculator:            in.Calculator,
		stripeTransferGateway: in.StripeTransferGateway,
		transferLease:         transferLease,
		now:                   now,
	}
}
