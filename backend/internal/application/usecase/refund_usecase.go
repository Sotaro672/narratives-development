// backend/internal/application/usecase/refund_usecase.go
package usecase

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	paymentdom "narratives/internal/domain/payment"
	settlementdom "narratives/internal/domain/settlement"
)

// ============================================================
// Payment Port
// ============================================================

// RefundPaymentReader provides the authoritative Payment required for a full
// Stripe refund.
//
// Payment.Status mirrors PaymentIntent lifecycle and remains succeeded after
// a refund. Refund lifecycle is persisted separately on Payment.
type RefundPaymentReader interface {
	GetByPaymentID(
		ctx context.Context,
		paymentID string,
	) (*paymentdom.Payment, error)
}

// ============================================================
// Settlement Port
// ============================================================

// RefundSettlementRepository provides the Settlement operations required to
// coordinate a card refund with seller payout state.
//
// UpdateByID must execute validated Settlement domain transitions.
type RefundSettlementRepository interface {
	ListByPaymentID(
		ctx context.Context,
		paymentID string,
	) ([]settlementdom.Settlement, error)

	UpdateByID(
		ctx context.Context,
		settlementID string,
		patch settlementdom.UpdateSettlementInput,
	) (settlementdom.Settlement, error)
}

// ============================================================
// Stripe Refund Port
// ============================================================

// StripeRefundGateway executes a Stripe Charge refund.
//
// AMOL uses Separate Charges and Transfers. Refunding the platform Charge does
// not automatically reverse seller Transfers, so transferred Settlements must
// also be processed through StripeTransferReversalGateway.
type StripeRefundGateway interface {
	CreateRefund(
		ctx context.Context,
		in CreateStripeRefundInput,
	) (*CreateStripeRefundResult, error)
}

type CreateStripeRefundInput struct {
	StripeChargeID string
	Amount         int
	IdempotencyKey string
	PaymentID      string
	RefundID       string
}

type CreateStripeRefundResult struct {
	StripeRefundID string
	Status         paymentdom.RefundStatus
	CreatedAt      time.Time
}

// ============================================================
// Stripe Transfer Reversal Port
// ============================================================

// StripeTransferReversalGateway reverses a completed Stripe Connect Transfer.
type StripeTransferReversalGateway interface {
	CreateTransferReversal(
		ctx context.Context,
		in CreateStripeTransferReversalInput,
	) (*CreateStripeTransferReversalResult, error)
}

type CreateStripeTransferReversalInput struct {
	StripeTransferID string

	Amount int

	IdempotencyKey string

	OrderID      string
	PaymentID    string
	SettlementID string
	CompanyID    string
	AccountID    string
}

type CreateStripeTransferReversalResult struct {
	StripeTransferReversalID string
}

// ============================================================
// Errors
// ============================================================

var (
	ErrRefundPaymentReaderMissing = errors.New(
		"refund: payment reader is not configured",
	)
	ErrRefundSettlementRepositoryMissing = errors.New(
		"refund: settlement repository is not configured",
	)
	ErrRefundStripeRefundGatewayMissing = errors.New(
		"refund: Stripe refund gateway is not configured",
	)
	ErrRefundStripeTransferReversalGatewayMissing = errors.New(
		"refund: Stripe transfer reversal gateway is not configured",
	)
	ErrRefundPaymentNotSucceeded = errors.New(
		"refund: payment is not succeeded",
	)
	ErrRefundStripeChargeIDMissing = errors.New(
		"refund: Stripe charge id is missing",
	)
	ErrRefundSettlementEmpty = errors.New(
		"refund: settlement is empty",
	)
	ErrRefundSettlementPaymentMismatch = errors.New(
		"refund: settlement does not belong to payment",
	)
	ErrRefundSettlementAmountMismatch = errors.New(
		"refund: settlement total does not match payment amount",
	)
	ErrRefundSettlementDuplicate = errors.New(
		"refund: duplicate settlement",
	)
	ErrRefundSettlementTransferring = errors.New(
		"refund: settlement transfer is in progress",
	)
	ErrRefundSettlementFailed = errors.New(
		"refund: settlement is in terminal transfer failure state",
	)
	ErrRefundSettlementStatusUnsupported = errors.New(
		"refund: unsupported settlement status",
	)
	ErrRefundStripeRefundResultEmpty = errors.New(
		"refund: Stripe refund result is empty",
	)
	ErrRefundStripeRefundIDEmpty = errors.New(
		"refund: Stripe refund id is empty",
	)
	ErrRefundStripeRefundStatusInvalid = errors.New(
		"refund: Stripe refund status is invalid",
	)
	ErrRefundStripeRefundCreatedAtInvalid = errors.New(
		"refund: Stripe refund createdAt is invalid",
	)
	ErrRefundStripeRefundMismatch = errors.New(
		"refund: Stripe refund does not match payment",
	)
	ErrRefundStripeTransferReversalResultEmpty = errors.New(
		"refund: Stripe transfer reversal result is empty",
	)
	ErrRefundStripeTransferReversalIDEmpty = errors.New(
		"refund: Stripe transfer reversal id is empty",
	)
)

// ============================================================
// Result
// ============================================================

type RefundSettlementAction string

const (
	RefundSettlementActionCanceled        RefundSettlementAction = "canceled"
	RefundSettlementActionAlreadyCanceled RefundSettlementAction = "already_canceled"
	RefundSettlementActionReversed        RefundSettlementAction = "reversed"
	RefundSettlementActionAlreadyReversed RefundSettlementAction = "already_reversed"
)

type RefundSettlementResult struct {
	SettlementID string
	AccountID    string

	PreviousStatus settlementdom.SettlementStatus
	Status         settlementdom.SettlementStatus

	StripeTransferID         string
	StripeTransferReversalID string

	Action RefundSettlementAction
}

type RefundByPaymentIDInput struct {
	PaymentID string
}

type RefundByPaymentIDResult struct {
	PaymentID string

	Amount int

	StripeRefundID string
	RefundStatus   paymentdom.RefundStatus
	RefundedAmount int
	RefundedAt     *time.Time

	Settlements []RefundSettlementResult
}

type CompleteSucceededRefundInput struct {
	PaymentID string

	StripeRefundID string
}

// ============================================================
// Usecase
// ============================================================

// RefundUsecase coordinates a full purchaser refund with seller-side Stripe
// Connect Settlement state.
//
// This first implementation intentionally supports full refunds only.
// Partial refunds require an explicit allocation policy for:
//
// - merchandise
// - consumption tax
// - shipping
// - shipping tax
// - platform fee
// - seller transfer reversal amount
//
// and must not be inferred here.
//
// Execution order:
//
//  1. Load and validate the succeeded Payment.
//  2. Load and validate all Account-level Settlements.
//  3. Reject any Settlement whose Transfer result is currently uncertain.
//  4. Cancel untransferred Settlements so a payout cannot begin during refund.
//  5. Create the Stripe Charge refund with a deterministic idempotency key.
//  6. Respect the actual Stripe Refund status returned by Stripe.
//  7. Reverse already-transferred seller Transfers only when Refund succeeded.
//  8. Persist each successful Transfer Reversal as Settlement reversed.
//
// If the process stops after Stripe accepted a request, retries reuse
// deterministic Stripe idempotency keys.
type RefundUsecase struct {
	paymentReader                 RefundPaymentReader
	settlementRepo                RefundSettlementRepository
	stripeRefundGateway           StripeRefundGateway
	stripeTransferReversalGateway StripeTransferReversalGateway
}

type NewRefundUsecaseInput struct {
	PaymentReader                 RefundPaymentReader
	SettlementRepository          RefundSettlementRepository
	StripeRefundGateway           StripeRefundGateway
	StripeTransferReversalGateway StripeTransferReversalGateway
}

func NewRefundUsecase(
	in NewRefundUsecaseInput,
) *RefundUsecase {
	return &RefundUsecase{
		paymentReader:                 in.PaymentReader,
		settlementRepo:                in.SettlementRepository,
		stripeRefundGateway:           in.StripeRefundGateway,
		stripeTransferReversalGateway: in.StripeTransferReversalGateway,
	}
}

// RefundByPaymentID performs a full refund for one succeeded Payment.
//
// PaymentID is currently the same value as OrderID.
//
// Payment.Status remains succeeded because that status mirrors the original
// PaymentIntent lifecycle. Refund lifecycle is returned separately through
// RefundStatus / RefundedAmount / RefundedAt.
func (u *RefundUsecase) RefundByPaymentID(
	ctx context.Context,
	in RefundByPaymentIDInput,
) (*RefundByPaymentIDResult, error) {
	if u == nil || u.paymentReader == nil {
		return nil,
			ErrRefundPaymentReaderMissing
	}

	if u.settlementRepo == nil {
		return nil,
			ErrRefundSettlementRepositoryMissing
	}

	if u.stripeRefundGateway == nil {
		return nil,
			ErrRefundStripeRefundGatewayMissing
	}

	paymentID := in.PaymentID
	if paymentID == "" {
		return nil,
			paymentdom.ErrInvalidPaymentID
	}

	payment, err := u.paymentReader.GetByPaymentID(
		ctx,
		paymentID,
	)
	if err != nil {
		return nil, err
	}

	if payment == nil ||
		payment.PaymentID != paymentID {
		return nil,
			paymentdom.ErrNotFound
	}

	if payment.Status != paymentdom.StatusSucceeded {
		return nil,
			ErrRefundPaymentNotSucceeded
	}

	if payment.StripeChargeID == "" {
		return nil,
			ErrRefundStripeChargeIDMissing
	}

	if payment.Amount <= 0 {
		return nil,
			paymentdom.ErrInvalidAmount
	}

	settlements, err :=
		u.settlementRepo.ListByPaymentID(
			ctx,
			paymentID,
		)
	if err != nil {
		return nil, err
	}

	if len(settlements) == 0 {
		return nil,
			ErrRefundSettlementEmpty
	}

	sort.Slice(
		settlements,
		func(i, j int) bool {
			return settlements[i].ID <
				settlements[j].ID
		},
	)

	hasTransferredSettlement, err :=
		validateRefundSettlements(
			paymentID,
			payment.Amount,
			settlements,
		)
	if err != nil {
		return nil, err
	}

	if hasTransferredSettlement &&
		u.stripeTransferReversalGateway == nil {
		return nil,
			ErrRefundStripeTransferReversalGatewayMissing
	}

	result := newRefundByPaymentIDResult(
		*payment,
		settlements,
	)

	settlementResults :=
		newRefundSettlementResultMap(
			settlements,
		)

	// Freeze every Settlement that has not sent money to a seller yet.
	//
	// This is deliberately done before the purchaser refund so a ready worker
	// cannot intentionally start a new payout after the refund begins.
	if err := u.cancelUntransferredSettlementsForRefund(
		ctx,
		settlements,
		settlementResults,
	); err != nil {
		result.Settlements =
			buildRefundSettlementResults(
				settlements,
				settlementResults,
			)

		return result, err
	}

	refundResult, refundErr :=
		u.stripeRefundGateway.CreateRefund(
			ctx,
			CreateStripeRefundInput{
				StripeChargeID: payment.StripeChargeID,
				Amount:         payment.Amount,
				IdempotencyKey: refundIdempotencyKey(
					paymentID,
				),
				PaymentID: paymentID,
			},
		)
	if refundErr != nil {
		result.Settlements =
			buildRefundSettlementResults(
				settlements,
				settlementResults,
			)

		return result,
			fmt.Errorf(
				"refund: create Stripe refund: %w",
				refundErr,
			)
	}

	if refundResult == nil {
		result.Settlements =
			buildRefundSettlementResults(
				settlements,
				settlementResults,
			)

		return result,
			ErrRefundStripeRefundResultEmpty
	}

	if err := applyCreateStripeRefundResult(
		result,
		payment.Amount,
		refundResult,
	); err != nil {
		result.Settlements =
			buildRefundSettlementResults(
				settlements,
				settlementResults,
			)

		return result, err
	}

	switch result.RefundStatus {
	case paymentdom.RefundStatusPending,
		paymentdom.RefundStatusRequiresAction,
		paymentdom.RefundStatusFailed,
		paymentdom.RefundStatusCanceled:
		// Stripe accepted or completed the Refund request, but purchaser funds
		// are not confirmed as refunded. Keep seller payouts frozen and do not
		// reverse already-transferred seller funds yet.
		result.Settlements =
			buildRefundSettlementResults(
				settlements,
				settlementResults,
			)

		return result, nil

	case paymentdom.RefundStatusSucceeded:
		// A Charge refund does not reverse Separate Charges and Transfers
		// payouts. Reverse each already-transferred Settlement independently.
		if err := u.reverseTransferredSettlementsForRefund(
			ctx,
			settlements,
			settlementResults,
		); err != nil {
			result.Settlements =
				buildRefundSettlementResults(
					settlements,
					settlementResults,
				)

			return result, err
		}

	default:
		result.Settlements =
			buildRefundSettlementResults(
				settlements,
				settlementResults,
			)

		return result,
			ErrRefundStripeRefundStatusInvalid
	}

	result.Settlements =
		buildRefundSettlementResults(
			settlements,
			settlementResults,
		)

	return result, nil
}

// CompleteSucceededRefund completes seller-side settlement handling after a
// Stripe Refund has become succeeded asynchronously.
//
// This method is intended to be called from the verified Stripe refund webhook
// after Payment refund state has been persisted as succeeded.
//
// It is idempotent at the application level:
//
// - CANCELED Settlement remains CANCELED.
// - REVERSED Settlement remains REVERSED.
// - TRANSFERRED Settlement uses the deterministic Stripe reversal key.
func (u *RefundUsecase) CompleteSucceededRefund(
	ctx context.Context,
	in CompleteSucceededRefundInput,
) (*RefundByPaymentIDResult, error) {
	if u == nil || u.paymentReader == nil {
		return nil,
			ErrRefundPaymentReaderMissing
	}

	if u.settlementRepo == nil {
		return nil,
			ErrRefundSettlementRepositoryMissing
	}

	paymentID := in.PaymentID
	if paymentID == "" {
		return nil,
			paymentdom.ErrInvalidPaymentID
	}

	if !isStripeRefundID(in.StripeRefundID) {
		return nil,
			ErrRefundStripeRefundIDEmpty
	}

	payment, err := u.paymentReader.GetByPaymentID(
		ctx,
		paymentID,
	)
	if err != nil {
		return nil, err
	}

	if payment == nil ||
		payment.PaymentID != paymentID {
		return nil,
			paymentdom.ErrNotFound
	}

	if payment.Status != paymentdom.StatusSucceeded {
		return nil,
			ErrRefundPaymentNotSucceeded
	}

	if payment.RefundStatus !=
		paymentdom.RefundStatusSucceeded {
		return nil,
			paymentdom.ErrInvalidRefundState
	}

	if payment.StripeRefundID !=
		in.StripeRefundID {
		return nil,
			ErrRefundStripeRefundMismatch
	}

	if payment.RefundedAmount != payment.Amount ||
		payment.RefundedAt == nil {
		return nil,
			paymentdom.ErrInvalidRefundState
	}

	settlements, err :=
		u.settlementRepo.ListByPaymentID(
			ctx,
			paymentID,
		)
	if err != nil {
		return nil, err
	}

	if len(settlements) == 0 {
		return nil,
			ErrRefundSettlementEmpty
	}

	sort.Slice(
		settlements,
		func(i, j int) bool {
			return settlements[i].ID <
				settlements[j].ID
		},
	)

	hasTransferredSettlement, err :=
		validateRefundSettlements(
			paymentID,
			payment.Amount,
			settlements,
		)
	if err != nil {
		return nil, err
	}

	if hasTransferredSettlement &&
		u.stripeTransferReversalGateway == nil {
		return nil,
			ErrRefundStripeTransferReversalGatewayMissing
	}

	result := newRefundByPaymentIDResult(
		*payment,
		settlements,
	)

	settlementResults :=
		newRefundSettlementResultMap(
			settlements,
		)

	// A succeeded refund must keep every not-yet-transferred seller payout
	// frozen. This also repairs a partially completed earlier attempt.
	if err := u.cancelUntransferredSettlementsForRefund(
		ctx,
		settlements,
		settlementResults,
	); err != nil {
		result.Settlements =
			buildRefundSettlementResults(
				settlements,
				settlementResults,
			)

		return result, err
	}

	if err := u.reverseTransferredSettlementsForRefund(
		ctx,
		settlements,
		settlementResults,
	); err != nil {
		result.Settlements =
			buildRefundSettlementResults(
				settlements,
				settlementResults,
			)

		return result, err
	}

	result.Settlements =
		buildRefundSettlementResults(
			settlements,
			settlementResults,
		)

	return result, nil
}

func newRefundByPaymentIDResult(
	payment paymentdom.Payment,
	settlements []settlementdom.Settlement,
) *RefundByPaymentIDResult {
	result := &RefundByPaymentIDResult{
		PaymentID: payment.PaymentID,
		Amount:    payment.Amount,
		Settlements: make(
			[]RefundSettlementResult,
			0,
			len(settlements),
		),
	}

	if payment.RefundStatus != "" &&
		payment.RefundStatus !=
			paymentdom.RefundStatusNone {
		result.StripeRefundID =
			payment.StripeRefundID
		result.RefundStatus =
			payment.RefundStatus
		result.RefundedAmount =
			payment.RefundedAmount

		if payment.RefundedAt != nil {
			value := payment.RefundedAt.UTC()
			result.RefundedAt = &value
		}
	}

	return result
}

func newRefundSettlementResultMap(
	settlements []settlementdom.Settlement,
) map[string]RefundSettlementResult {
	result := make(
		map[string]RefundSettlementResult,
		len(settlements),
	)

	for _, settlement := range settlements {
		result[settlement.ID] =
			RefundSettlementResult{
				SettlementID: settlement.ID,
				AccountID:    settlement.AccountID,

				PreviousStatus: settlement.Status,
				Status:         settlement.Status,

				StripeTransferID: settlement.StripeTransferID,

				StripeTransferReversalID: settlement.StripeTransferReversalID,
			}
	}

	return result
}

func applyCreateStripeRefundResult(
	result *RefundByPaymentIDResult,
	paymentAmount int,
	refundResult *CreateStripeRefundResult,
) error {
	if result == nil || refundResult == nil {
		return ErrRefundStripeRefundResultEmpty
	}

	stripeRefundID :=
		refundResult.StripeRefundID
	if !isStripeRefundID(
		stripeRefundID,
	) {
		return ErrRefundStripeRefundIDEmpty
	}

	if !paymentdom.IsValidRefundStatus(
		refundResult.Status,
	) ||
		refundResult.Status ==
			paymentdom.RefundStatusNone {
		return ErrRefundStripeRefundStatusInvalid
	}

	if refundResult.CreatedAt.IsZero() {
		return ErrRefundStripeRefundCreatedAtInvalid
	}

	result.StripeRefundID =
		stripeRefundID
	result.RefundStatus =
		refundResult.Status
	result.RefundedAmount = 0
	result.RefundedAt = nil

	if refundResult.Status ==
		paymentdom.RefundStatusSucceeded {
		if paymentAmount <= 0 {
			return paymentdom.ErrInvalidAmount
		}

		result.RefundedAmount =
			paymentAmount

		value := refundResult.CreatedAt.UTC()
		result.RefundedAt = &value
	}

	return nil
}

func (u *RefundUsecase) cancelUntransferredSettlementsForRefund(
	ctx context.Context,
	settlements []settlementdom.Settlement,
	settlementResults map[string]RefundSettlementResult,
) error {
	for _, settlement := range settlements {
		switch settlement.Status {
		case settlementdom.StatusPending,
			settlementdom.StatusReady,
			settlementdom.StatusFailedRetryable:

			updated, err :=
				u.cancelSettlementForRefund(
					ctx,
					settlement,
				)
			if err != nil {
				return fmt.Errorf(
					"refund: cancel settlement %q: %w",
					settlement.ID,
					err,
				)
			}

			item := settlementResults[settlement.ID]
			item.Status = updated.Status
			item.Action = RefundSettlementActionCanceled
			settlementResults[settlement.ID] = item

		case settlementdom.StatusCanceled:
			item := settlementResults[settlement.ID]
			item.Action = RefundSettlementActionAlreadyCanceled
			settlementResults[settlement.ID] = item

		case settlementdom.StatusTransferred,
			settlementdom.StatusReversed:
			// Transferred funds are handled only after purchaser Refund succeeded.
		}
	}

	return nil
}

func (u *RefundUsecase) reverseTransferredSettlementsForRefund(
	ctx context.Context,
	settlements []settlementdom.Settlement,
	settlementResults map[string]RefundSettlementResult,
) error {
	for _, settlement := range settlements {
		switch settlement.Status {
		case settlementdom.StatusTransferred:
			updated, err :=
				u.reverseTransferredSettlement(
					ctx,
					settlement,
				)
			if err != nil {
				return fmt.Errorf(
					"refund: reverse settlement %q: %w",
					settlement.ID,
					err,
				)
			}

			item := settlementResults[settlement.ID]
			item.Status = updated.Status
			item.StripeTransferReversalID =
				updated.StripeTransferReversalID
			item.Action = RefundSettlementActionReversed
			settlementResults[settlement.ID] = item

		case settlementdom.StatusReversed:
			item := settlementResults[settlement.ID]
			item.Action = RefundSettlementActionAlreadyReversed
			settlementResults[settlement.ID] = item
		}
	}

	return nil
}

func (u *RefundUsecase) cancelSettlementForRefund(
	ctx context.Context,
	settlement settlementdom.Settlement,
) (settlementdom.Settlement, error) {
	status := settlementdom.StatusCanceled

	updated, err := u.settlementRepo.UpdateByID(
		ctx,
		settlement.ID,
		settlementdom.UpdateSettlementInput{
			Status: &status,
		},
	)
	if err != nil {
		return settlementdom.Settlement{}, err
	}

	if updated.ID != settlement.ID ||
		updated.PaymentID != settlement.PaymentID ||
		updated.AccountID != settlement.AccountID {
		return settlementdom.Settlement{},
			ErrRefundSettlementPaymentMismatch
	}

	if updated.Status !=
		settlementdom.StatusCanceled {
		return settlementdom.Settlement{},
			settlementdom.ErrInvalidStatusTransition
	}

	return updated, nil
}

func (u *RefundUsecase) reverseTransferredSettlement(
	ctx context.Context,
	settlement settlementdom.Settlement,
) (settlementdom.Settlement, error) {
	if u.stripeTransferReversalGateway == nil {
		return settlementdom.Settlement{},
			ErrRefundStripeTransferReversalGatewayMissing
	}

	if settlement.Status !=
		settlementdom.StatusTransferred {
		return settlementdom.Settlement{},
			settlementdom.ErrInvalidStatusTransition
	}

	if settlement.StripeTransferID == "" {
		return settlementdom.Settlement{},
			settlementdom.ErrInvalidStripeTransferID
	}

	if settlement.TransferAmount <= 0 {
		return settlementdom.Settlement{},
			settlementdom.ErrInvalidTransferAmount
	}

	reversalResult, reversalErr :=
		u.stripeTransferReversalGateway.CreateTransferReversal(
			ctx,
			CreateStripeTransferReversalInput{
				StripeTransferID: settlement.StripeTransferID,
				Amount:           settlement.TransferAmount,
				IdempotencyKey: transferReversalIdempotencyKey(
					settlement.ID,
				),
				OrderID:      settlement.OrderID,
				PaymentID:    settlement.PaymentID,
				SettlementID: settlement.ID,
				CompanyID:    settlement.CompanyID,
				AccountID:    settlement.AccountID,
			},
		)
	if reversalErr != nil {
		return settlementdom.Settlement{},
			fmt.Errorf(
				"Stripe transfer reversal: %w",
				reversalErr,
			)
	}

	if reversalResult == nil {
		return settlementdom.Settlement{},
			ErrRefundStripeTransferReversalResultEmpty
	}

	reversalID :=
		reversalResult.StripeTransferReversalID
	if !isStripeTransferReversalIDForRefund(
		reversalID,
	) {
		return settlementdom.Settlement{},
			ErrRefundStripeTransferReversalIDEmpty
	}

	status := settlementdom.StatusReversed

	updated, err := u.settlementRepo.UpdateByID(
		ctx,
		settlement.ID,
		settlementdom.UpdateSettlementInput{
			StripeTransferReversalID: &reversalID,
			Status:                   &status,
		},
	)
	if err != nil {
		// Stripe may already have completed the Reversal here.
		// A retry uses the same deterministic idempotency key and can recover the
		// same Stripe Reversal before persisting it again.
		return settlementdom.Settlement{},
			fmt.Errorf(
				"persist Stripe transfer reversal %q: %w",
				reversalID,
				err,
			)
	}

	if updated.Status != settlementdom.StatusReversed ||
		updated.StripeTransferReversalID != reversalID {
		return settlementdom.Settlement{},
			settlementdom.ErrInvalidStatusTransition
	}

	return updated, nil
}

// ============================================================
// Validation
// ============================================================

func validateRefundSettlements(
	paymentID string,
	paymentAmount int,
	settlements []settlementdom.Settlement,
) (bool, error) {
	if paymentID == "" {
		return false,
			paymentdom.ErrInvalidPaymentID
	}

	if paymentAmount <= 0 {
		return false,
			paymentdom.ErrInvalidAmount
	}

	if len(settlements) == 0 {
		return false,
			ErrRefundSettlementEmpty
	}

	seenSettlementIDs := make(
		map[string]struct{},
		len(settlements),
	)

	seenAccountIDs := make(
		map[string]struct{},
		len(settlements),
	)

	maxInt := int(^uint(0) >> 1)
	total := 0
	hasTransferredSettlement := false

	for _, settlement := range settlements {
		if settlement.ID == "" ||
			settlement.PaymentID != paymentID ||
			settlement.OrderID != paymentID {
			return false,
				ErrRefundSettlementPaymentMismatch
		}

		if _, exists :=
			seenSettlementIDs[settlement.ID]; exists {
			return false,
				ErrRefundSettlementDuplicate
		}

		seenSettlementIDs[settlement.ID] =
			struct{}{}

		if settlement.AccountID == "" {
			return false,
				settlementdom.ErrInvalidAccountID
		}

		if _, exists :=
			seenAccountIDs[settlement.AccountID]; exists {
			return false,
				ErrRefundSettlementDuplicate
		}

		seenAccountIDs[settlement.AccountID] =
			struct{}{}

		if settlement.GrossAmount <= 0 ||
			settlement.TransferAmount <= 0 {
			return false,
				ErrRefundSettlementAmountMismatch
		}

		if total >
			maxInt-settlement.GrossAmount {
			return false,
				ErrRefundSettlementAmountMismatch
		}

		total += settlement.GrossAmount

		switch settlement.Status {
		case settlementdom.StatusPending,
			settlementdom.StatusReady,
			settlementdom.StatusFailedRetryable,
			settlementdom.StatusCanceled:

		case settlementdom.StatusTransferred:
			hasTransferredSettlement = true

		case settlementdom.StatusReversed:

		case settlementdom.StatusTransferring:
			return false,
				ErrRefundSettlementTransferring

		case settlementdom.StatusFailed:
			// Current Settlement domain does not allow failed -> canceled.
			// Do not start the Stripe refund while financial state cannot be
			// represented consistently.
			return false,
				ErrRefundSettlementFailed

		default:
			return false,
				ErrRefundSettlementStatusUnsupported
		}
	}

	if total != paymentAmount {
		return false,
			ErrRefundSettlementAmountMismatch
	}

	return hasTransferredSettlement, nil
}

func buildRefundSettlementResults(
	settlements []settlementdom.Settlement,
	values map[string]RefundSettlementResult,
) []RefundSettlementResult {
	result := make(
		[]RefundSettlementResult,
		0,
		len(settlements),
	)

	for _, settlement := range settlements {
		value, exists := values[settlement.ID]
		if !exists {
			continue
		}

		result = append(
			result,
			value,
		)
	}

	return result
}

// ============================================================
// Stripe idempotency / identifiers
// ============================================================

func refundIdempotencyKey(
	paymentID string,
) string {
	return fmt.Sprintf(
		"refund:%s",
		paymentID,
	)
}

func transferReversalIdempotencyKey(
	settlementID string,
) string {
	return fmt.Sprintf(
		"settlement-reversal:%s",
		settlementID,
	)
}

func isStripeRefundID(
	value string,
) bool {
	return strings.HasPrefix(
		value,
		"re_",
	) &&
		len(value) > len("re_")
}

func isStripeTransferReversalIDForRefund(
	value string,
) bool {
	return strings.HasPrefix(
		value,
		"trr_",
	) &&
		len(value) > len("trr_")
}
