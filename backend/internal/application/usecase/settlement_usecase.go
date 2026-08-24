// backend/internal/application/usecase/settlement_usecase.go
package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
// The calculator must guarantee that allocations are grouped by AccountID.
// Multiple Brands sharing one AccountID must therefore produce one allocation.
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
	CompanyID    string
	AccountID    string
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
	ErrSettlementDuplicateAccount = errors.New(
		"settlement: duplicate account allocation",
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
		transferLease =
			defaultSettlementTransferLease
	}

	return &SettlementUsecase{
		repo:                  in.Repository,
		calculator:            in.Calculator,
		stripeTransferGateway: in.StripeTransferGateway,
		transferLease:         transferLease,
		now:                   now,
	}
}

// ============================================================
// Queries
// ============================================================

func (u *SettlementUsecase) GetByID(
	ctx context.Context,
	settlementID string,
) (settlementdom.Settlement, error) {
	if u == nil || u.repo == nil {
		return settlementdom.Settlement{}, ErrSettlementRepositoryMissing
	}

	settlementID = strings.TrimSpace(settlementID)
	if settlementID == "" {
		return settlementdom.Settlement{}, settlementdom.ErrInvalidID
	}

	return u.repo.GetByID(
		ctx,
		settlementID,
	)
}

func (u *SettlementUsecase) ListByPaymentID(
	ctx context.Context,
	paymentID string,
) ([]settlementdom.Settlement, error) {
	if u == nil || u.repo == nil {
		return nil, ErrSettlementRepositoryMissing
	}

	paymentID = strings.TrimSpace(paymentID)
	if paymentID == "" {
		return nil, settlementdom.ErrInvalidPaymentID
	}

	return u.repo.ListByPaymentID(
		ctx,
		paymentID,
	)
}

func (u *SettlementUsecase) ListByOrderID(
	ctx context.Context,
	orderID string,
) ([]settlementdom.Settlement, error) {
	if u == nil || u.repo == nil {
		return nil, ErrSettlementRepositoryMissing
	}

	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return nil, settlementdom.ErrInvalidOrderID
	}

	return u.repo.ListByOrderID(
		ctx,
		orderID,
	)
}

func (u *SettlementUsecase) ListByCompanyID(
	ctx context.Context,
	companyID string,
) ([]settlementdom.Settlement, error) {
	if u == nil || u.repo == nil {
		return nil, ErrSettlementRepositoryMissing
	}

	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, settlementdom.ErrInvalidCompanyID
	}

	return u.repo.ListByCompanyID(
		ctx,
		companyID,
	)
}

func (u *SettlementUsecase) ListByAccountID(
	ctx context.Context,
	accountID string,
) ([]settlementdom.Settlement, error) {
	if u == nil || u.repo == nil {
		return nil, ErrSettlementRepositoryMissing
	}

	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return nil, settlementdom.ErrInvalidAccountID
	}

	return u.repo.ListByAccountID(
		ctx,
		accountID,
	)
}

// ============================================================
// Settlement creation
// ============================================================

// EnsureForSucceededPayment creates Account-level Settlements for one
// successfully completed Payment.
//
// This method is idempotent because each Settlement ID is deterministic:
//
//	paymentID + "_" + accountID
//
// Existing Settlement records are loaded and verified instead of overwritten.
//
// This method creates Settlements as ready but does not send money.
// Actual Stripe Transfer execution occurs separately, normally after the
// corresponding seller's Order items have been dispatched.
func (u *SettlementUsecase) EnsureForSucceededPayment(
	ctx context.Context,
	order orderdom.Order,
	payment paymentdom.Payment,
) ([]settlementdom.Settlement, error) {
	if u == nil || u.repo == nil {
		return nil, ErrSettlementRepositoryMissing
	}

	if u.calculator == nil {
		return nil, ErrSettlementCalculatorMissing
	}

	if err := validateSettlementSource(
		order,
		payment,
	); err != nil {
		return nil, err
	}

	if err := validateSettlementOrderItems(order); err != nil {
		return nil, err
	}

	allocations, err := u.calculator.Calculate(
		ctx,
		order,
		payment,
	)
	if err != nil {
		return nil, err
	}

	if err := validateSettlementAllocations(
		payment,
		allocations,
	); err != nil {
		return nil, err
	}

	now := u.now().UTC()

	result := make(
		[]settlementdom.Settlement,
		0,
		len(allocations),
	)

	for _, allocation := range allocations {
		settlementID, err := settlementdom.NewID(
			payment.PaymentID,
			allocation.AccountID,
		)
		if err != nil {
			return nil, err
		}

		entity, err := settlementdom.New(
			settlementID,
			order.ID,
			payment.PaymentID,
			allocation.CompanyID,
			allocation.AccountID,
			allocation.StripeAccountID,
			payment.StripePaymentIntentID,
			payment.StripeChargeID,
			payment.TransferGroup,
			allocation.GrossAmount,
			allocation.PlatformFeeAmount,
			allocation.TransferAmount,
			settlementdom.CurrencyJPY,
			settlementdom.StatusReady,
			now,
		)
		if err != nil {
			return nil, err
		}

		created, err := u.repo.Create(
			ctx,
			settlementdom.CreateSettlementInput{
				SettlementID: entity.ID,
				OrderID:      entity.OrderID,
				PaymentID:    entity.PaymentID,

				CompanyID: entity.CompanyID,
				AccountID: entity.AccountID,

				StripeAccountID: entity.StripeAccountID,

				StripePaymentIntentID: entity.StripePaymentIntentID,
				StripeChargeID:        entity.StripeChargeID,

				TransferGroup: entity.TransferGroup,

				GrossAmount:       entity.GrossAmount,
				PlatformFeeAmount: entity.PlatformFeeAmount,
				TransferAmount:    entity.TransferAmount,

				Currency: entity.Currency,
				Status:   entity.Status,
			},
		)
		if err != nil {
			if !errors.Is(
				err,
				settlementdom.ErrConflict,
			) {
				return nil, err
			}

			existing, getErr := u.repo.GetByID(
				ctx,
				settlementID,
			)
			if getErr != nil {
				return nil, getErr
			}

			if err := validateExistingSettlement(
				existing,
				entity,
			); err != nil {
				return nil, err
			}

			result = append(
				result,
				existing,
			)
			continue
		}

		result = append(
			result,
			created,
		)
	}

	return result, nil
}

// ============================================================
// Stripe Transfer
// ============================================================

// TransferByPaymentAndAccount executes one Account-level Stripe Transfer.
//
// This is useful after dispatch because an Order item already stores
// SellerSnapshot.AccountID.
func (u *SettlementUsecase) TransferByPaymentAndAccount(
	ctx context.Context,
	paymentID string,
	accountID string,
) (settlementdom.Settlement, error) {
	paymentID = strings.TrimSpace(paymentID)
	accountID = strings.TrimSpace(accountID)

	if paymentID == "" {
		return settlementdom.Settlement{},
			settlementdom.ErrInvalidPaymentID
	}

	if accountID == "" {
		return settlementdom.Settlement{},
			settlementdom.ErrInvalidAccountID
	}

	settlementID, err := settlementdom.NewID(
		paymentID,
		accountID,
	)
	if err != nil {
		return settlementdom.Settlement{}, err
	}

	return u.TransferByID(
		ctx,
		settlementID,
	)
}

// TransferByID executes a Stripe Connect Transfer for one Settlement.
//
// Execution order:
//  1. Atomically claim the Settlement.
//  2. Call Stripe with a deterministic Idempotency-Key.
//  3. Persist StripeTransferID and transferred state.
//  4. If Stripe fails, persist failed_retryable or failed.
//
// The atomic claim prevents two workers from intentionally starting the same
// Settlement at the same time.
//
// The Stripe idempotency key protects against an uncertain network result where
// Stripe may have accepted the request before the worker received the response.
func (u *SettlementUsecase) TransferByID(
	ctx context.Context,
	settlementID string,
) (settlementdom.Settlement, error) {
	if u == nil || u.repo == nil {
		return settlementdom.Settlement{},
			ErrSettlementRepositoryMissing
	}

	if u.stripeTransferGateway == nil {
		return settlementdom.Settlement{},
			ErrSettlementStripeTransferGatewayMissing
	}

	settlementID = strings.TrimSpace(settlementID)
	if settlementID == "" {
		return settlementdom.Settlement{},
			settlementdom.ErrInvalidID
	}

	now := u.now().UTC()

	transferLease := u.transferLease
	if transferLease <= 0 {
		transferLease =
			defaultSettlementTransferLease
	}

	staleBefore :=
		now.Add(
			-transferLease,
		)

	claim, err := u.repo.ClaimForTransfer(
		ctx,
		settlementID,
		now,
		staleBefore,
	)
	if err != nil {
		return settlementdom.Settlement{}, err
	}

	if !claim.Claimed {
		if claim.Settlement.Status ==
			settlementdom.StatusTransferred {
			return claim.Settlement, nil
		}

		return claim.Settlement,
			ErrSettlementTransferNotReady
	}

	settlement := claim.Settlement

	if settlement.Status !=
		settlementdom.StatusTransferring {
		return settlementdom.Settlement{},
			ErrSettlementTransferNotReady
	}

	if settlement.StripeChargeID == "" {
		return u.failClaimedSettlement(
			ctx,
			settlement,
			false,
			nil,
			nil,
			ErrSettlementStripeChargeIDMissing,
		)
	}

	if settlement.TransferGroup == "" {
		return u.failClaimedSettlement(
			ctx,
			settlement,
			false,
			nil,
			nil,
			ErrSettlementTransferGroupMissing,
		)
	}

	idempotencyKey := fmt.Sprintf(
		"settlement:%s:%s",
		settlement.PaymentID,
		settlement.AccountID,
	)

	stripeResult, stripeErr :=
		u.stripeTransferGateway.CreateTransfer(
			ctx,
			CreateStripeSettlementTransferInput{
				Amount: settlement.TransferAmount,

				Currency: strings.ToLower(
					settlement.Currency,
				),

				DestinationStripeAccountID: settlement.StripeAccountID,
				SourceTransaction:          settlement.StripeChargeID,
				TransferGroup:              settlement.TransferGroup,
				IdempotencyKey:             idempotencyKey,

				OrderID:      settlement.OrderID,
				PaymentID:    settlement.PaymentID,
				SettlementID: settlement.ID,
				CompanyID:    settlement.CompanyID,
				AccountID:    settlement.AccountID,
			},
		)

	if stripeErr != nil {
		return u.failClaimedSettlement(
			ctx,
			settlement,
			isSettlementTransferErrorRetryable(
				stripeErr,
			),
			settlementErrorType(
				stripeErr,
			),
			settlementErrorCode(
				stripeErr,
			),
			stripeErr,
		)
	}

	if stripeResult == nil {
		return u.failClaimedSettlement(
			ctx,
			settlement,
			true,
			nil,
			nil,
			ErrSettlementStripeTransferResultEmpty,
		)
	}

	stripeTransferID := strings.TrimSpace(
		stripeResult.StripeTransferID,
	)
	if stripeTransferID == "" {
		return u.failClaimedSettlement(
			ctx,
			settlement,
			true,
			nil,
			nil,
			ErrSettlementStripeTransferIDEmpty,
		)
	}

	completedAt := u.now().UTC()

	completed, err := u.repo.CompleteTransfer(
		ctx,
		settlement.ID,
		stripeTransferID,
		completedAt,
	)
	if err != nil {
		// Stripe may already have completed the Transfer here.
		// Do not attempt another transfer with a different idempotency key.
		// The deterministic key allows reconciliation/retry to recover the same
		// Stripe Transfer.
		return settlementdom.Settlement{},
			fmt.Errorf(
				"settlement: persist completed Stripe Transfer %q: %w",
				stripeTransferID,
				err,
			)
	}

	return completed, nil
}

func (u *SettlementUsecase) failClaimedSettlement(
	ctx context.Context,
	settlement settlementdom.Settlement,
	retryable bool,
	errorType *string,
	errorCode *string,
	cause error,
) (settlementdom.Settlement, error) {
	if cause == nil {
		cause = errors.New(
			"settlement: Stripe transfer failed",
		)
	}

	errorMessage := cause.Error()

	nextStatus := settlementdom.StatusFailed
	if retryable {
		nextStatus = settlementdom.StatusFailedRetryable
	}

	failed, persistErr := u.repo.FailTransfer(
		ctx,
		settlement.ID,
		nextStatus,
		normalizeSettlementErrorString(
			errorType,
		),
		normalizeSettlementErrorString(
			errorCode,
		),
		&errorMessage,
		u.now().UTC(),
	)
	if persistErr != nil {
		return settlementdom.Settlement{},
			fmt.Errorf(
				"settlement: Stripe transfer failed and failure state could not be persisted: transferErr=%v persistErr=%w",
				cause,
				persistErr,
			)
	}

	return failed, cause
}

// ============================================================
// Validation
// ============================================================

func validateSettlementSource(
	order orderdom.Order,
	payment paymentdom.Payment,
) error {
	if order.ID == "" {
		return ErrSettlementOrderIDInvalid
	}

	if payment.PaymentID == "" {
		return ErrSettlementPaymentIDInvalid
	}

	if payment.PaymentID != order.ID {
		return ErrSettlementPaymentOrderMismatch
	}

	if payment.Status != paymentdom.StatusSucceeded {
		return ErrSettlementPaymentNotSucceeded
	}

	if payment.StripePaymentIntentID == "" {
		return ErrSettlementStripePaymentIntentIDMissing
	}

	if payment.StripeChargeID == "" {
		return ErrSettlementStripeChargeIDMissing
	}

	if payment.TransferGroup == "" {
		return ErrSettlementTransferGroupMissing
	}

	if payment.Amount <= 0 {
		return ErrSettlementAllocationAmountMismatch
	}

	return nil
}

func validateSettlementOrderItems(
	order orderdom.Order,
) error {
	if len(order.Items) == 0 {
		return ErrSettlementUnsupportedOrderItem
	}

	for _, item := range order.Items {
		if item.IsCancelled {
			continue
		}

		// Current settlement implementation supports primary List sales only.
		//
		// Resale.BrandID identifies the product brand, not necessarily the
		// resale seller. A separate consumer payout identity is required before
		// resale proceeds can safely use Stripe Connect.
		if item.Type != orderdom.OrderItemTypeList {
			return ErrSettlementUnsupportedOrderItem
		}

		if item.SellerSnapshot.BrandID == "" ||
			item.SellerSnapshot.CompanyID == "" ||
			item.SellerSnapshot.AccountID == "" ||
			item.SellerSnapshot.StripeAccountID == "" {
			return orderdom.ErrInvalidSellerSnapshot
		}
	}

	return nil
}

func validateSettlementAllocations(
	payment paymentdom.Payment,
	allocations []settlementdom.Allocation,
) error {
	if len(allocations) == 0 {
		return ErrSettlementAllocationEmpty
	}

	maxInt := int(^uint(0) >> 1)

	seenAccounts := make(
		map[string]struct{},
		len(allocations),
	)

	total := 0

	for _, allocation := range allocations {
		if allocation.CompanyID == "" ||
			allocation.AccountID == "" ||
			allocation.StripeAccountID == "" {
			return ErrSettlementAllocationInvalid
		}

		if _, exists := seenAccounts[allocation.AccountID]; exists {
			return ErrSettlementDuplicateAccount
		}

		seenAccounts[allocation.AccountID] = struct{}{}

		if allocation.GrossAmount <= 0 ||
			allocation.PlatformFeeAmount < 0 ||
			allocation.TransferAmount <= 0 {
			return ErrSettlementAllocationInvalid
		}

		if allocation.PlatformFeeAmount >
			allocation.GrossAmount {
			return ErrSettlementAllocationInvalid
		}

		if allocation.TransferAmount >
			allocation.GrossAmount {
			return ErrSettlementAllocationInvalid
		}

		if allocation.GrossAmount-
			allocation.PlatformFeeAmount !=
			allocation.TransferAmount {
			return ErrSettlementAllocationInvalid
		}

		if total >
			maxInt-allocation.GrossAmount {
			return ErrSettlementAllocationAmountMismatch
		}

		total += allocation.GrossAmount
	}

	if total != payment.Amount {
		return ErrSettlementAllocationAmountMismatch
	}

	return nil
}

func validateExistingSettlement(
	existing settlementdom.Settlement,
	expected settlementdom.Settlement,
) error {
	if existing.ID != expected.ID ||
		existing.OrderID != expected.OrderID ||
		existing.PaymentID != expected.PaymentID ||
		existing.CompanyID != expected.CompanyID ||
		existing.AccountID != expected.AccountID ||
		existing.StripeAccountID != expected.StripeAccountID ||
		existing.StripePaymentIntentID != expected.StripePaymentIntentID ||
		existing.StripeChargeID != expected.StripeChargeID ||
		existing.TransferGroup != expected.TransferGroup ||
		existing.GrossAmount != expected.GrossAmount ||
		existing.PlatformFeeAmount != expected.PlatformFeeAmount ||
		existing.TransferAmount != expected.TransferAmount ||
		existing.Currency != expected.Currency {
		return settlementdom.ErrConflict
	}

	return nil
}

// ============================================================
// Stripe error helpers
// ============================================================

func isSettlementTransferErrorRetryable(
	err error,
) bool {
	if err == nil {
		return false
	}

	var retryableError RetryableStripeSettlementError
	if errors.As(
		err,
		&retryableError,
	) {
		return retryableError.Retryable()
	}

	// Unknown transport/infrastructure failures are retried.
	//
	// The deterministic Stripe Idempotency-Key prevents a retry from creating
	// another Transfer when Stripe accepted the original request but the
	// response was lost.
	return true
}

func settlementErrorType(
	err error,
) *string {
	if err == nil {
		return nil
	}

	var metadata StripeSettlementErrorMetadata
	if !errors.As(
		err,
		&metadata,
	) {
		return nil
	}

	value := metadata.ErrorType()
	if value == "" {
		return nil
	}

	return &value
}

func settlementErrorCode(
	err error,
) *string {
	if err == nil {
		return nil
	}

	var metadata StripeSettlementErrorMetadata
	if !errors.As(
		err,
		&metadata,
	) {
		return nil
	}

	value := metadata.ErrorCode()
	if value == "" {
		return nil
	}

	return &value
}

func normalizeSettlementErrorString(
	value *string,
) *string {
	if value == nil {
		return nil
	}

	if *value == "" {
		return nil
	}

	return value
}
