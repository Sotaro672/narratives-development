// backend/internal/application/usecase/refund_usecase.go
package usecase

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	paymentdom "narratives/internal/domain/payment"
	settlementdom "narratives/internal/domain/settlement"
)

// ============================================================
// Payment Port
// ============================================================

// RefundPaymentReader provides the authoritative Payment required for a full
// Stripe refund.
//
// The current Payment domain mirrors PaymentIntent lifecycle and does not yet
// persist refund-specific fields. RefundUsecase therefore reads Payment but
// does not change Payment.Status.
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

	Amount int

	IdempotencyKey string

	PaymentID string
}

type CreateStripeRefundResult struct {
	StripeRefundID string
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
	RefundSettlementActionCanceled RefundSettlementAction = "canceled"

	RefundSettlementActionAlreadyCanceled RefundSettlementAction = "already_canceled"

	RefundSettlementActionReversed RefundSettlementAction = "reversed"

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

	Settlements []RefundSettlementResult
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
//  6. Reverse each already-transferred seller Transfer.
//  7. Persist each successful Transfer Reversal as Settlement reversed.
//
// If the process stops after Stripe accepted a request, retries reuse
// deterministic Stripe idempotency keys.
type RefundUsecase struct {
	paymentReader RefundPaymentReader

	settlementRepo RefundSettlementRepository

	stripeRefundGateway StripeRefundGateway

	stripeTransferReversalGateway StripeTransferReversalGateway
}

type NewRefundUsecaseInput struct {
	PaymentReader RefundPaymentReader

	SettlementRepository RefundSettlementRepository

	StripeRefundGateway StripeRefundGateway

	StripeTransferReversalGateway StripeTransferReversalGateway
}

func NewRefundUsecase(
	in NewRefundUsecaseInput,
) *RefundUsecase {
	return &RefundUsecase{
		paymentReader: in.PaymentReader,

		settlementRepo: in.SettlementRepository,

		stripeRefundGateway: in.StripeRefundGateway,

		stripeTransferReversalGateway: in.StripeTransferReversalGateway,
	}
}

// RefundByPaymentID performs a full refund for one succeeded Payment.
//
// PaymentID is currently the same value as OrderID.
//
// Payment.Status remains succeeded because that status mirrors the original
// PaymentIntent lifecycle. Refund-specific persistence should be added to the
// Payment domain separately rather than overloading PaymentStatus.
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

	result := &RefundByPaymentIDResult{
		PaymentID: paymentID,
		Amount:    payment.Amount,
		Settlements: make(
			[]RefundSettlementResult,
			0,
			len(settlements),
		),
	}

	settlementResults := make(
		map[string]RefundSettlementResult,
		len(settlements),
	)

	for _, settlement := range settlements {
		settlementResults[settlement.ID] =
			RefundSettlementResult{
				SettlementID: settlement.ID,
				AccountID:    settlement.AccountID,

				PreviousStatus: settlement.Status,
				Status:         settlement.Status,

				StripeTransferID: settlement.StripeTransferID,

				StripeTransferReversalID: settlement.StripeTransferReversalID,
			}
	}

	// Freeze every Settlement that has not sent money to a seller yet.
	//
	// This is deliberately done before the purchaser refund so a ready worker
	// cannot intentionally start a new payout after the refund begins.
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
				result.Settlements =
					buildRefundSettlementResults(
						settlements,
						settlementResults,
					)

				return result,
					fmt.Errorf(
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
			// These states are handled after the purchaser Charge refund.
		}
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

	stripeRefundID :=
		refundResult.StripeRefundID
	if !isStripeRefundID(
		stripeRefundID,
	) {
		result.Settlements =
			buildRefundSettlementResults(
				settlements,
				settlementResults,
			)

		return result,
			ErrRefundStripeRefundIDEmpty
	}

	result.StripeRefundID =
		stripeRefundID

	// A Charge refund does not reverse Separate Charges and Transfers payouts.
	// Reverse each already-transferred Settlement independently.
	for _, settlement := range settlements {
		switch settlement.Status {
		case settlementdom.StatusTransferred:
			updated, err :=
				u.reverseTransferredSettlement(
					ctx,
					settlement,
				)
			if err != nil {
				result.Settlements =
					buildRefundSettlementResults(
						settlements,
						settlementResults,
					)

				return result,
					fmt.Errorf(
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

	result.Settlements =
		buildRefundSettlementResults(
			settlements,
			settlementResults,
		)

	return result, nil
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

		case settlementdom.StatusTransferred,
			settlementdom.StatusReversed:
			hasTransferredSettlement = true

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
