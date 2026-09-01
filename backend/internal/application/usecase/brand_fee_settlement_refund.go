// backend/internal/application/usecase/brand_fee_settlement_refund.go
package usecase

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	applicationport "narratives/internal/application/port"
	brandfeesettlementdom "narratives/internal/domain/brandFeeSettlement"
	settlementdom "narratives/internal/domain/settlement"
)

// ============================================================
// Repository Port
// ============================================================

// BrandFeeSettlementRefundRepository is the minimal persistence contract
// required to coordinate purchaser Refunds with productBlueprint Brand fees.
//
// Brand fee transfer execution itself remains owned by
// BrandFeeSettlementTransferUsecase. This port is used only to:
//
//   - load BrandFeeSettlements
//   - cancel Brand fees that have not transferred
//   - persist completed Stripe Transfer Reversals
type BrandFeeSettlementRefundRepository interface {
	GetByID(
		ctx context.Context,
		brandFeeSettlementID string,
	) (brandfeesettlementdom.BrandFeeSettlement, error)

	ListByPaymentID(
		ctx context.Context,
		paymentID string,
	) ([]brandfeesettlementdom.BrandFeeSettlement, error)

	UpdateByID(
		ctx context.Context,
		brandFeeSettlementID string,
		patch brandfeesettlementdom.UpdateBrandFeeSettlementInput,
	) (brandfeesettlementdom.BrandFeeSettlement, error)

	ReverseTransfer(
		ctx context.Context,
		brandFeeSettlementID string,
		stripeTransferReversalID string,
		now time.Time,
	) (brandfeesettlementdom.BrandFeeSettlement, error)
}

// ============================================================
// Consumer Port
// ============================================================

// BrandFeeSettlementRefundService is shared by:
//
//   - full Payment Refund
//   - item-level resale Refund
//
// Refund execution is intentionally split into two phases.
//
// Prepare:
//
//   - pending          -> canceled
//   - ready            -> canceled
//   - failed_retryable -> canceled
//   - transferring     -> rejected
//   - transferred      -> retained for post-Refund reversal
//   - failed           -> accepted because no Transfer completed
//   - canceled         -> idempotent success
//   - reversed         -> idempotent success
//
// Complete:
//
//   - transferred      -> Stripe Transfer Reversal -> reversed
//   - pending/ready/failed_retryable -> canceled as recovery
//   - transferring     -> rejected
//   - failed/canceled/reversed -> idempotent success
//
// The purchaser Stripe Refund must be created only after Prepare succeeds.
// Complete must run only after purchaser Refund success is authoritative.
type BrandFeeSettlementRefundService interface {
	PrepareByPaymentID(
		ctx context.Context,
		paymentID string,
	) ([]BrandFeeSettlementRefundResult, error)

	CompleteByPaymentID(
		ctx context.Context,
		paymentID string,
	) ([]BrandFeeSettlementRefundResult, error)

	PrepareByPaymentAndOrderItem(
		ctx context.Context,
		paymentID string,
		orderItemIndex int,
	) (BrandFeeSettlementRefundResult, error)

	CompleteByPaymentAndOrderItem(
		ctx context.Context,
		paymentID string,
		orderItemIndex int,
	) (BrandFeeSettlementRefundResult, error)
}

// ============================================================
// Errors
// ============================================================

var (
	ErrBrandFeeSettlementRefundRepositoryMissing = errors.New(
		"brandFeeSettlement refund: repository is not configured",
	)

	ErrBrandFeeSettlementRefundStripeTransferReversalGatewayMissing = errors.New(
		"brandFeeSettlement refund: Stripe transfer reversal gateway is not configured",
	)

	ErrBrandFeeSettlementRefundClockMissing = errors.New(
		"brandFeeSettlement refund: clock is not configured",
	)

	ErrBrandFeeSettlementRefundTransferring = errors.New(
		"brandFeeSettlement refund: Brand fee transfer is in progress",
	)

	ErrBrandFeeSettlementRefundStatusUnsupported = errors.New(
		"brandFeeSettlement refund: unsupported Brand fee status",
	)

	ErrBrandFeeSettlementRefundMismatch = errors.New(
		"brandFeeSettlement refund: Brand fee does not match refund target",
	)

	ErrBrandFeeSettlementRefundDuplicate = errors.New(
		"brandFeeSettlement refund: duplicate Brand fee settlement",
	)

	ErrBrandFeeSettlementRefundStripeTransferReversalResultEmpty = errors.New(
		"brandFeeSettlement refund: Stripe transfer reversal result is empty",
	)

	ErrBrandFeeSettlementRefundStripeTransferReversalIDEmpty = errors.New(
		"brandFeeSettlement refund: Stripe transfer reversal id is empty",
	)
)

// ============================================================
// Result
// ============================================================

type BrandFeeSettlementRefundAction string

const (
	BrandFeeSettlementRefundActionCanceled BrandFeeSettlementRefundAction = "canceled"

	BrandFeeSettlementRefundActionAlreadyCanceled BrandFeeSettlementRefundAction = "already_canceled"

	BrandFeeSettlementRefundActionReversalRequired BrandFeeSettlementRefundAction = "reversal_required"

	BrandFeeSettlementRefundActionReversed BrandFeeSettlementRefundAction = "reversed"

	BrandFeeSettlementRefundActionAlreadyReversed BrandFeeSettlementRefundAction = "already_reversed"

	BrandFeeSettlementRefundActionNoTransfer BrandFeeSettlementRefundAction = "no_transfer"
)

type BrandFeeSettlementRefundResult struct {
	BrandFeeSettlementID string

	OrderID        string
	PaymentID      string
	OrderItemIndex int
	ResaleID       string

	BrandID         string
	CompanyID       string
	AccountID       string
	StripeAccountID string

	BrandFeeAmount int
	Currency       string

	PreviousStatus brandfeesettlementdom.Status
	Status         brandfeesettlementdom.Status

	StripeTransferID         string
	StripeTransferReversalID string

	Action BrandFeeSettlementRefundAction
}

// ============================================================
// Usecase
// ============================================================

// BrandFeeSettlementRefundUsecase coordinates Brand fee state with purchaser
// Stripe Refunds.
//
// Brand fee Transfer and purchaser Refund are separate Stripe operations.
//
// A successfully transferred Brand fee therefore requires:
//
//	purchaser Stripe Refund
//		+
//	Brand Stripe Transfer Reversal
//
// The service never changes BrandFeeAmount or payout destination. All financial
// identity is loaded from the persisted BrandFeeSettlement.
//
// Transfer reversal uses the complete BrandFeeAmount because each
// BrandFeeSettlement represents exactly one resale Order item's Brand share.
type BrandFeeSettlementRefundUsecase struct {
	repo BrandFeeSettlementRefundRepository

	stripeTransferReversalGateway applicationport.StripeTransferReversalGateway

	now func() time.Time
}

type NewBrandFeeSettlementRefundUsecaseInput struct {
	Repository BrandFeeSettlementRefundRepository

	StripeTransferReversalGateway applicationport.StripeTransferReversalGateway

	Now func() time.Time
}

func NewBrandFeeSettlementRefundUsecase(
	in NewBrandFeeSettlementRefundUsecaseInput,
) *BrandFeeSettlementRefundUsecase {
	now := in.Now
	if now == nil {
		now = time.Now
	}

	return &BrandFeeSettlementRefundUsecase{
		repo:                          in.Repository,
		stripeTransferReversalGateway: in.StripeTransferReversalGateway,
		now:                           now,
	}
}

// ============================================================
// Prepare Full Payment Refund
// ============================================================

// PrepareByPaymentID prepares every BrandFeeSettlement belonging to one
// purchaser Payment.
//
// This method must run before purchaser Stripe Refund creation.
//
// Untransferred Brand fees are canceled so they cannot be paid after the buyer
// has been refunded.
//
// A currently transferring record stops Refund execution because the Stripe
// Transfer result is uncertain.
//
// Already transferred records remain transferred until the purchaser Refund has
// succeeded. They are reversed by CompleteByPaymentID.
func (u *BrandFeeSettlementRefundUsecase) PrepareByPaymentID(
	ctx context.Context,
	paymentID string,
) ([]BrandFeeSettlementRefundResult, error) {
	if err := u.validateRepositoryReady(); err != nil {
		return nil, err
	}
	if paymentID == "" {
		return nil, brandfeesettlementdom.ErrInvalidPaymentID
	}

	brandFeeSettlements, err := u.loadByPaymentID(
		ctx,
		paymentID,
	)
	if err != nil {
		return nil, err
	}

	results := make(
		[]BrandFeeSettlementRefundResult,
		0,
		len(brandFeeSettlements),
	)

	for _, brandFeeSettlement := range brandFeeSettlements {
		result, err := u.prepareOne(
			ctx,
			brandFeeSettlement,
		)
		if err != nil {
			return results, fmt.Errorf(
				"brandFeeSettlement refund: prepare %q: %w",
				brandFeeSettlement.ID,
				err,
			)
		}

		results = append(
			results,
			result,
		)
	}

	return results, nil
}

// ============================================================
// Complete Full Payment Refund
// ============================================================

// CompleteByPaymentID completes Brand fee handling after the purchaser Stripe
// Refund has succeeded.
//
// Transferred Brand fees are fully reversed.
//
// If an untransferred record somehow remains pending/ready/failed_retryable,
// Complete cancels it as recovery. This makes asynchronous Refund completion
// resilient to a crash between earlier preparation steps.
//
// A transferring record remains unsafe and returns an error until transfer
// reconciliation determines whether Stripe completed it.
func (u *BrandFeeSettlementRefundUsecase) CompleteByPaymentID(
	ctx context.Context,
	paymentID string,
) ([]BrandFeeSettlementRefundResult, error) {
	if err := u.validateRepositoryReady(); err != nil {
		return nil, err
	}
	if paymentID == "" {
		return nil, brandfeesettlementdom.ErrInvalidPaymentID
	}

	brandFeeSettlements, err := u.loadByPaymentID(
		ctx,
		paymentID,
	)
	if err != nil {
		return nil, err
	}

	results := make(
		[]BrandFeeSettlementRefundResult,
		0,
		len(brandFeeSettlements),
	)

	for _, brandFeeSettlement := range brandFeeSettlements {
		result, err := u.completeOne(
			ctx,
			brandFeeSettlement,
		)
		if err != nil {
			return results, fmt.Errorf(
				"brandFeeSettlement refund: complete %q: %w",
				brandFeeSettlement.ID,
				err,
			)
		}

		results = append(
			results,
			result,
		)
	}

	return results, nil
}

// ============================================================
// Prepare Item Refund
// ============================================================

// PrepareByPaymentAndOrderItem prepares exactly one resale Order item's Brand
// fee before an item-level purchaser Refund.
func (u *BrandFeeSettlementRefundUsecase) PrepareByPaymentAndOrderItem(
	ctx context.Context,
	paymentID string,
	orderItemIndex int,
) (BrandFeeSettlementRefundResult, error) {
	if err := u.validateRepositoryReady(); err != nil {
		return BrandFeeSettlementRefundResult{}, err
	}

	brandFeeSettlement, err := u.loadByPaymentAndOrderItem(
		ctx,
		paymentID,
		orderItemIndex,
	)
	if err != nil {
		return BrandFeeSettlementRefundResult{}, err
	}

	result, err := u.prepareOne(
		ctx,
		brandFeeSettlement,
	)
	if err != nil {
		return result, fmt.Errorf(
			"brandFeeSettlement refund: prepare %q: %w",
			brandFeeSettlement.ID,
			err,
		)
	}

	return result, nil
}

// ============================================================
// Complete Item Refund
// ============================================================

// CompleteByPaymentAndOrderItem completes exactly one resale Order item's Brand
// fee handling after its purchaser Stripe Refund has succeeded.
func (u *BrandFeeSettlementRefundUsecase) CompleteByPaymentAndOrderItem(
	ctx context.Context,
	paymentID string,
	orderItemIndex int,
) (BrandFeeSettlementRefundResult, error) {
	if err := u.validateRepositoryReady(); err != nil {
		return BrandFeeSettlementRefundResult{}, err
	}

	brandFeeSettlement, err := u.loadByPaymentAndOrderItem(
		ctx,
		paymentID,
		orderItemIndex,
	)
	if err != nil {
		return BrandFeeSettlementRefundResult{}, err
	}

	result, err := u.completeOne(
		ctx,
		brandFeeSettlement,
	)
	if err != nil {
		return result, fmt.Errorf(
			"brandFeeSettlement refund: complete %q: %w",
			brandFeeSettlement.ID,
			err,
		)
	}

	return result, nil
}

// ============================================================
// Prepare One
// ============================================================

func (u *BrandFeeSettlementRefundUsecase) prepareOne(
	ctx context.Context,
	current brandfeesettlementdom.BrandFeeSettlement,
) (BrandFeeSettlementRefundResult, error) {
	result := newBrandFeeSettlementRefundResult(
		current,
	)

	if err := current.Validate(); err != nil {
		return result, fmt.Errorf(
			"%w: %v",
			ErrBrandFeeSettlementRefundMismatch,
			err,
		)
	}

	switch current.Status {
	case brandfeesettlementdom.StatusPending,
		brandfeesettlementdom.StatusReady,
		brandfeesettlementdom.StatusFailedRetryable:

		updated, err := u.cancelOne(
			ctx,
			current,
		)
		if err != nil {
			return result, err
		}

		result = completedBrandFeeSettlementRefundResult(
			result,
			updated,
			BrandFeeSettlementRefundActionCanceled,
		)

		return result, nil

	case brandfeesettlementdom.StatusTransferring:
		return result,
			ErrBrandFeeSettlementRefundTransferring

	case brandfeesettlementdom.StatusTransferred:
		if u.stripeTransferReversalGateway == nil {
			return result,
				ErrBrandFeeSettlementRefundStripeTransferReversalGatewayMissing
		}

		result.Action =
			BrandFeeSettlementRefundActionReversalRequired

		return result, nil

	case brandfeesettlementdom.StatusFailed:
		result.Action =
			BrandFeeSettlementRefundActionNoTransfer

		return result, nil

	case brandfeesettlementdom.StatusCanceled:
		result.Action =
			BrandFeeSettlementRefundActionAlreadyCanceled

		return result, nil

	case brandfeesettlementdom.StatusReversed:
		result.Action =
			BrandFeeSettlementRefundActionAlreadyReversed

		return result, nil

	default:
		return result,
			ErrBrandFeeSettlementRefundStatusUnsupported
	}
}

// ============================================================
// Complete One
// ============================================================

func (u *BrandFeeSettlementRefundUsecase) completeOne(
	ctx context.Context,
	current brandfeesettlementdom.BrandFeeSettlement,
) (BrandFeeSettlementRefundResult, error) {
	result := newBrandFeeSettlementRefundResult(
		current,
	)

	if err := current.Validate(); err != nil {
		return result, fmt.Errorf(
			"%w: %v",
			ErrBrandFeeSettlementRefundMismatch,
			err,
		)
	}

	switch current.Status {
	case brandfeesettlementdom.StatusPending,
		brandfeesettlementdom.StatusReady,
		brandfeesettlementdom.StatusFailedRetryable:

		updated, err := u.cancelOne(
			ctx,
			current,
		)
		if err != nil {
			return result, err
		}

		result = completedBrandFeeSettlementRefundResult(
			result,
			updated,
			BrandFeeSettlementRefundActionCanceled,
		)

		return result, nil

	case brandfeesettlementdom.StatusTransferring:
		return result,
			ErrBrandFeeSettlementRefundTransferring

	case brandfeesettlementdom.StatusTransferred:
		updated, err := u.reverseOne(
			ctx,
			current,
		)
		if err != nil {
			return result, err
		}

		result = completedBrandFeeSettlementRefundResult(
			result,
			updated,
			BrandFeeSettlementRefundActionReversed,
		)

		return result, nil

	case brandfeesettlementdom.StatusFailed:
		result.Action =
			BrandFeeSettlementRefundActionNoTransfer

		return result, nil

	case brandfeesettlementdom.StatusCanceled:
		result.Action =
			BrandFeeSettlementRefundActionAlreadyCanceled

		return result, nil

	case brandfeesettlementdom.StatusReversed:
		result.Action =
			BrandFeeSettlementRefundActionAlreadyReversed

		return result, nil

	default:
		return result,
			ErrBrandFeeSettlementRefundStatusUnsupported
	}
}

// ============================================================
// Cancellation
// ============================================================

func (u *BrandFeeSettlementRefundUsecase) cancelOne(
	ctx context.Context,
	current brandfeesettlementdom.BrandFeeSettlement,
) (brandfeesettlementdom.BrandFeeSettlement, error) {
	switch current.Status {
	case brandfeesettlementdom.StatusPending,
		brandfeesettlementdom.StatusReady,
		brandfeesettlementdom.StatusFailedRetryable:
	default:
		return brandfeesettlementdom.BrandFeeSettlement{},
			brandfeesettlementdom.ErrInvalidStatusTransition
	}

	nextStatus := brandfeesettlementdom.StatusCanceled

	updated, err := u.repo.UpdateByID(
		ctx,
		current.ID,
		brandfeesettlementdom.UpdateBrandFeeSettlementInput{
			Status: &nextStatus,
		},
	)
	if err != nil {
		return brandfeesettlementdom.BrandFeeSettlement{},
			err
	}

	if err := validateBrandFeeSettlementRefundMutation(
		current,
		updated,
	); err != nil {
		return brandfeesettlementdom.BrandFeeSettlement{},
			err
	}

	if updated.Status !=
		brandfeesettlementdom.StatusCanceled {
		return brandfeesettlementdom.BrandFeeSettlement{},
			brandfeesettlementdom.ErrInvalidStatusTransition
	}

	if updated.StripeTransferID != "" ||
		updated.StripeTransferReversalID != "" ||
		updated.TransferredAt != nil ||
		updated.ReversedAt != nil {
		return brandfeesettlementdom.BrandFeeSettlement{},
			ErrBrandFeeSettlementRefundMismatch
	}

	return updated, nil
}

// ============================================================
// Transfer Reversal
// ============================================================

func (u *BrandFeeSettlementRefundUsecase) reverseOne(
	ctx context.Context,
	current brandfeesettlementdom.BrandFeeSettlement,
) (brandfeesettlementdom.BrandFeeSettlement, error) {
	if u.stripeTransferReversalGateway == nil {
		return brandfeesettlementdom.BrandFeeSettlement{},
			ErrBrandFeeSettlementRefundStripeTransferReversalGatewayMissing
	}
	if u.now == nil {
		return brandfeesettlementdom.BrandFeeSettlement{},
			ErrBrandFeeSettlementRefundClockMissing
	}
	if current.Status !=
		brandfeesettlementdom.StatusTransferred {
		return brandfeesettlementdom.BrandFeeSettlement{},
			brandfeesettlementdom.ErrInvalidStatusTransition
	}
	if err := current.Validate(); err != nil {
		return brandfeesettlementdom.BrandFeeSettlement{},
			fmt.Errorf(
				"%w: %v",
				ErrBrandFeeSettlementRefundMismatch,
				err,
			)
	}
	if current.StripeTransferID == "" ||
		!strings.HasPrefix(
			current.StripeTransferID,
			"tr_",
		) {
		return brandfeesettlementdom.BrandFeeSettlement{},
			brandfeesettlementdom.ErrInvalidStripeTransferID
	}
	if current.BrandFeeAmount <= 0 {
		return brandfeesettlementdom.BrandFeeSettlement{},
			brandfeesettlementdom.ErrInvalidBrandFeeAmount
	}

	brand := current.BrandIdentity()
	if err := brand.Validate(); err != nil {
		return brandfeesettlementdom.BrandFeeSettlement{},
			ErrBrandFeeSettlementRefundMismatch
	}

	// The existing Stripe Transfer Reversal gateway accepts the same
	// Account-shaped immutable payout identity used by the Brand fee Transfer.
	//
	// This SellerIdentity is only the low-level Stripe gateway identity.
	// It does not create or imply a primary-sale Settlement.
	gatewaySeller := settlementdom.SellerIdentity{
		Type: settlementdom.SellerTypeAccount,

		CompanyID: brand.CompanyID,
		AccountID: brand.AccountID,

		StripeAccountID: brand.StripeAccountID,
	}

	if err := gatewaySeller.Validate(); err != nil {
		return brandfeesettlementdom.BrandFeeSettlement{},
			ErrBrandFeeSettlementRefundMismatch
	}

	reversalResult, reversalErr :=
		u.stripeTransferReversalGateway.CreateTransferReversal(
			ctx,
			applicationport.CreateStripeTransferReversalInput{
				StripeTransferID: current.StripeTransferID,

				Amount: current.BrandFeeAmount,

				IdempotencyKey: brandFeeSettlementRefundReversalIdempotencyKey(
					current.ID,
				),

				OrderID:      current.OrderID,
				PaymentID:    current.PaymentID,
				SettlementID: current.ID,

				Seller: gatewaySeller,
			},
		)
	if reversalErr != nil {
		return brandfeesettlementdom.BrandFeeSettlement{},
			fmt.Errorf(
				"brandFeeSettlement refund: Stripe transfer reversal: %w",
				reversalErr,
			)
	}

	if reversalResult == nil {
		return brandfeesettlementdom.BrandFeeSettlement{},
			ErrBrandFeeSettlementRefundStripeTransferReversalResultEmpty
	}

	stripeTransferReversalID :=
		reversalResult.StripeTransferReversalID

	if !isBrandFeeSettlementStripeTransferReversalID(
		stripeTransferReversalID,
	) {
		return brandfeesettlementdom.BrandFeeSettlement{},
			ErrBrandFeeSettlementRefundStripeTransferReversalIDEmpty
	}

	updated, err := u.repo.ReverseTransfer(
		ctx,
		current.ID,
		stripeTransferReversalID,
		u.now().UTC(),
	)
	if err != nil {
		// Stripe may already have completed the reversal.
		//
		// Every retry uses the same deterministic Stripe Idempotency-Key so the
		// same reversal is returned instead of creating a second reversal.
		return brandfeesettlementdom.BrandFeeSettlement{},
			fmt.Errorf(
				"brandFeeSettlement refund: persist Stripe transfer reversal %q: %w",
				stripeTransferReversalID,
				err,
			)
	}

	if err := validateBrandFeeSettlementRefundMutation(
		current,
		updated,
	); err != nil {
		return brandfeesettlementdom.BrandFeeSettlement{},
			err
	}

	if updated.Status !=
		brandfeesettlementdom.StatusReversed {
		return brandfeesettlementdom.BrandFeeSettlement{},
			brandfeesettlementdom.ErrInvalidStatusTransition
	}

	if updated.StripeTransferID != current.StripeTransferID ||
		updated.StripeTransferReversalID != stripeTransferReversalID ||
		updated.ReversedAt == nil ||
		updated.ReversedAt.IsZero() {
		return brandfeesettlementdom.BrandFeeSettlement{},
			ErrBrandFeeSettlementRefundMismatch
	}

	return updated, nil
}

// ============================================================
// Loading
// ============================================================

func (u *BrandFeeSettlementRefundUsecase) loadByPaymentID(
	ctx context.Context,
	paymentID string,
) ([]brandfeesettlementdom.BrandFeeSettlement, error) {
	if paymentID == "" {
		return nil, brandfeesettlementdom.ErrInvalidPaymentID
	}

	values, err := u.repo.ListByPaymentID(
		ctx,
		paymentID,
	)
	if err != nil {
		return nil, err
	}

	seenIDs := make(
		map[string]struct{},
		len(values),
	)

	seenItemIndexes := make(
		map[int]struct{},
		len(values),
	)

	for _, value := range values {
		if err := validateBrandFeeSettlementRefundTarget(
			value,
			paymentID,
			value.OrderItemIndex,
		); err != nil {
			return nil, err
		}

		if _, exists := seenIDs[value.ID]; exists {
			return nil,
				ErrBrandFeeSettlementRefundDuplicate
		}

		seenIDs[value.ID] = struct{}{}

		if _, exists :=
			seenItemIndexes[value.OrderItemIndex]; exists {
			return nil,
				ErrBrandFeeSettlementRefundDuplicate
		}

		seenItemIndexes[value.OrderItemIndex] = struct{}{}
	}

	sort.Slice(
		values,
		func(i, j int) bool {
			if values[i].OrderItemIndex !=
				values[j].OrderItemIndex {
				return values[i].OrderItemIndex <
					values[j].OrderItemIndex
			}

			return values[i].ID < values[j].ID
		},
	)

	return values, nil
}

func (u *BrandFeeSettlementRefundUsecase) loadByPaymentAndOrderItem(
	ctx context.Context,
	paymentID string,
	orderItemIndex int,
) (brandfeesettlementdom.BrandFeeSettlement, error) {
	if paymentID == "" {
		return brandfeesettlementdom.BrandFeeSettlement{},
			brandfeesettlementdom.ErrInvalidPaymentID
	}
	if orderItemIndex < 0 {
		return brandfeesettlementdom.BrandFeeSettlement{},
			brandfeesettlementdom.ErrInvalidOrderItemIndex
	}

	brandFeeSettlementID, err :=
		brandfeesettlementdom.NewID(
			paymentID,
			orderItemIndex,
		)
	if err != nil {
		return brandfeesettlementdom.BrandFeeSettlement{},
			err
	}

	value, err := u.repo.GetByID(
		ctx,
		brandFeeSettlementID,
	)
	if err != nil {
		return brandfeesettlementdom.BrandFeeSettlement{},
			err
	}

	if err := validateBrandFeeSettlementRefundTarget(
		value,
		paymentID,
		orderItemIndex,
	); err != nil {
		return brandfeesettlementdom.BrandFeeSettlement{},
			err
	}

	return value, nil
}

// ============================================================
// Validation
// ============================================================

func (u *BrandFeeSettlementRefundUsecase) validateRepositoryReady() error {
	if u == nil || u.repo == nil {
		return ErrBrandFeeSettlementRefundRepositoryMissing
	}

	if u.now == nil {
		return ErrBrandFeeSettlementRefundClockMissing
	}

	return nil
}

func validateBrandFeeSettlementRefundTarget(
	value brandfeesettlementdom.BrandFeeSettlement,
	paymentID string,
	orderItemIndex int,
) error {
	if paymentID == "" {
		return brandfeesettlementdom.ErrInvalidPaymentID
	}
	if orderItemIndex < 0 {
		return brandfeesettlementdom.ErrInvalidOrderItemIndex
	}

	if err := value.Validate(); err != nil {
		return fmt.Errorf(
			"%w: %v",
			ErrBrandFeeSettlementRefundMismatch,
			err,
		)
	}

	expectedID, err :=
		brandfeesettlementdom.NewID(
			paymentID,
			orderItemIndex,
		)
	if err != nil {
		return err
	}

	if value.ID != expectedID ||
		value.PaymentID != paymentID ||
		value.OrderID != paymentID ||
		value.OrderItemIndex != orderItemIndex {
		return ErrBrandFeeSettlementRefundMismatch
	}

	return nil
}

func validateBrandFeeSettlementRefundMutation(
	before brandfeesettlementdom.BrandFeeSettlement,
	after brandfeesettlementdom.BrandFeeSettlement,
) error {
	if err := after.Validate(); err != nil {
		return fmt.Errorf(
			"%w: %v",
			ErrBrandFeeSettlementRefundMismatch,
			err,
		)
	}

	if !sameBrandFeeSettlementRefundAllocation(
		before,
		after,
	) {
		return ErrBrandFeeSettlementRefundMismatch
	}

	return nil
}

func sameBrandFeeSettlementRefundAllocation(
	left brandfeesettlementdom.BrandFeeSettlement,
	right brandfeesettlementdom.BrandFeeSettlement,
) bool {
	return left.ID == right.ID &&
		left.OrderID == right.OrderID &&
		left.PaymentID == right.PaymentID &&
		left.OrderItemIndex == right.OrderItemIndex &&
		left.ResaleID == right.ResaleID &&
		left.BrandID == right.BrandID &&
		left.CompanyID == right.CompanyID &&
		left.AccountID == right.AccountID &&
		left.StripeAccountID == right.StripeAccountID &&
		left.StripePaymentIntentID == right.StripePaymentIntentID &&
		left.StripeChargeID == right.StripeChargeID &&
		left.StripeTransferID == right.StripeTransferID &&
		left.TransferGroup == right.TransferGroup &&
		left.BrandFeeAmount == right.BrandFeeAmount &&
		left.Currency == right.Currency &&
		left.CreatedAt.Equal(right.CreatedAt) &&
		sameBrandFeeSettlementRefundTime(
			left.TransferredAt,
			right.TransferredAt,
		)
}

func sameBrandFeeSettlementRefundTime(
	left *time.Time,
	right *time.Time,
) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}

	return left.Equal(*right)
}

// ============================================================
// Result Helpers
// ============================================================

func newBrandFeeSettlementRefundResult(
	value brandfeesettlementdom.BrandFeeSettlement,
) BrandFeeSettlementRefundResult {
	return BrandFeeSettlementRefundResult{
		BrandFeeSettlementID: value.ID,

		OrderID:        value.OrderID,
		PaymentID:      value.PaymentID,
		OrderItemIndex: value.OrderItemIndex,
		ResaleID:       value.ResaleID,

		BrandID:         value.BrandID,
		CompanyID:       value.CompanyID,
		AccountID:       value.AccountID,
		StripeAccountID: value.StripeAccountID,

		BrandFeeAmount: value.BrandFeeAmount,
		Currency:       value.Currency,

		PreviousStatus: value.Status,
		Status:         value.Status,

		StripeTransferID: value.StripeTransferID,

		StripeTransferReversalID: value.StripeTransferReversalID,
	}
}

func completedBrandFeeSettlementRefundResult(
	result BrandFeeSettlementRefundResult,
	value brandfeesettlementdom.BrandFeeSettlement,
	action BrandFeeSettlementRefundAction,
) BrandFeeSettlementRefundResult {
	result.Status = value.Status

	result.StripeTransferID =
		value.StripeTransferID

	result.StripeTransferReversalID =
		value.StripeTransferReversalID

	result.Action = action

	return result
}

// ============================================================
// Stripe Idempotency / Identifiers
// ============================================================

func brandFeeSettlementRefundReversalIdempotencyKey(
	brandFeeSettlementID string,
) string {
	return fmt.Sprintf(
		"brand_fee_settlement_reversal:%s",
		brandFeeSettlementID,
	)
}

func isBrandFeeSettlementStripeTransferReversalID(
	value string,
) bool {
	return strings.HasPrefix(
		value,
		"trr_",
	) &&
		len(value) > len("trr_") &&
		!strings.Contains(
			value,
			"/",
		)
}
