// backend/internal/application/usecase/refund_usecase.go
package usecase

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	applicationport "narratives/internal/application/port"
	paymentdom "narratives/internal/domain/payment"
	salesreceivabledom "narratives/internal/domain/salesReceivable"
	settlementdom "narratives/internal/domain/settlement"
)

// ============================================================
// Ports
// ============================================================

type RefundPaymentReader interface {
	GetByPaymentID(ctx context.Context, paymentID string) (*paymentdom.Payment, error)
}

type RefundSettlementRepository interface {
	ListByPaymentID(ctx context.Context, paymentID string) ([]settlementdom.Settlement, error)
	UpdateByID(ctx context.Context, settlementID string, patch settlementdom.UpdateSettlementInput) (settlementdom.Settlement, error)
}

type RefundSalesReceivableService interface {
	ListByPaymentID(ctx context.Context, paymentID string) ([]salesreceivabledom.SalesReceivable, error)
	Cancel(ctx context.Context, receivableID string) (*salesreceivabledom.SalesReceivable, error)
}

// ============================================================
// Errors
// ============================================================

var (
	ErrRefundPaymentReaderMissing                 = errors.New("refund: payment reader is not configured")
	ErrRefundSettlementRepositoryMissing          = errors.New("refund: settlement repository is not configured")
	ErrRefundSalesReceivableServiceMissing        = errors.New("refund: sales receivable service is not configured")
	ErrRefundStripeRefundGatewayMissing           = errors.New("refund: Stripe refund gateway is not configured")
	ErrRefundStripeTransferReversalGatewayMissing = errors.New("refund: Stripe transfer reversal gateway is not configured")
	ErrRefundPaymentNotSucceeded                  = errors.New("refund: payment is not succeeded")
	ErrRefundStripeChargeIDMissing                = errors.New("refund: Stripe charge id is missing")
	ErrRefundFinancialSourceEmpty                 = errors.New("refund: seller financial source is empty")
	ErrRefundFinancialSourceAmountMismatch        = errors.New("refund: seller financial source total does not match payment amount")
	ErrRefundSettlementPaymentMismatch            = errors.New("refund: settlement does not belong to payment")
	ErrRefundSettlementAmountMismatch             = errors.New("refund: settlement amount is invalid")
	ErrRefundSettlementDuplicate                  = errors.New("refund: duplicate settlement")
	ErrRefundSettlementTransferring               = errors.New("refund: settlement transfer is in progress")
	ErrRefundSettlementFailed                     = errors.New("refund: settlement is in terminal transfer failure state")
	ErrRefundSettlementStatusUnsupported          = errors.New("refund: unsupported settlement status")
	ErrRefundSalesReceivableMismatch              = errors.New("refund: sales receivable does not belong to payment")
	ErrRefundSalesReceivableDuplicate             = errors.New("refund: duplicate sales receivable")
	ErrRefundSalesReceivableReserved              = errors.New("refund: sales receivable is reserved for bank payout")
	ErrRefundSalesReceivablePaid                  = errors.New("refund: sales receivable has already been paid")
	ErrRefundSalesReceivableStatusUnsupported     = errors.New("refund: unsupported sales receivable status")
	ErrRefundStripeRefundResultEmpty              = errors.New("refund: Stripe refund result is empty")
	ErrRefundStripeRefundIDEmpty                  = errors.New("refund: Stripe refund id is empty")
	ErrRefundStripeRefundStatusInvalid            = errors.New("refund: Stripe refund status is invalid")
	ErrRefundStripeRefundCreatedAtInvalid         = errors.New("refund: Stripe refund createdAt is invalid")
	ErrRefundStripeRefundMismatch                 = errors.New("refund: Stripe refund does not match payment")
	ErrRefundStripeTransferReversalResultEmpty    = errors.New("refund: Stripe transfer reversal result is empty")
	ErrRefundStripeTransferReversalIDEmpty        = errors.New("refund: Stripe transfer reversal id is empty")
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
	SettlementID             string
	SellerType               settlementdom.SellerType
	CompanyID                string
	AccountID                string
	StripeAccountID          string
	PreviousStatus           settlementdom.SettlementStatus
	Status                   settlementdom.SettlementStatus
	StripeTransferID         string
	StripeTransferReversalID string
	Action                   RefundSettlementAction
}

type RefundSalesReceivableAction string

const (
	RefundSalesReceivableActionCanceled        RefundSalesReceivableAction = "canceled"
	RefundSalesReceivableActionAlreadyCanceled RefundSalesReceivableAction = "already_canceled"
)

type RefundSalesReceivableResult struct {
	SalesReceivableID string
	OrderItemIndex    int
	ResaleID          string
	AvatarID          string
	UserID            string
	PayoutAccountID   string
	PreviousStatus    salesreceivabledom.Status
	Status            salesreceivabledom.Status
	Action            RefundSalesReceivableAction
}

type RefundByPaymentIDInput struct {
	PaymentID string
}

type RefundByPaymentIDResult struct {
	PaymentID        string
	Amount           int
	StripeRefundID   string
	RefundStatus     paymentdom.RefundStatus
	RefundedAmount   int
	RefundedAt       *time.Time
	Settlements      []RefundSettlementResult
	SalesReceivables []RefundSalesReceivableResult
}

type CompleteSucceededRefundInput struct {
	PaymentID      string
	StripeRefundID string
}

// ============================================================
// Usecase
// ============================================================

// RefundUsecase coordinates one full purchaser Stripe refund with both seller-side
// financial models used by AMOL.
//
// Primary List items use Settlement and may require a Stripe Transfer Reversal.
// Consumer resale items use item-level SalesReceivable and never use Stripe
// Connect. Unpaid resale receivables are canceled before the purchaser refund.
// Reserved receivables are rejected until BankPayout coordination is implemented;
// paid receivables are rejected until recovery/adjustment handling is implemented.
type RefundUsecase struct {
	paymentReader                 RefundPaymentReader
	settlementRepo                RefundSettlementRepository
	salesReceivableService        RefundSalesReceivableService
	stripeRefundGateway           applicationport.StripeRefundGateway
	stripeTransferReversalGateway applicationport.StripeTransferReversalGateway
}

type NewRefundUsecaseInput struct {
	PaymentReader                 RefundPaymentReader
	SettlementRepository          RefundSettlementRepository
	SalesReceivableService        RefundSalesReceivableService
	StripeRefundGateway           applicationport.StripeRefundGateway
	StripeTransferReversalGateway applicationport.StripeTransferReversalGateway
}

func NewRefundUsecase(in NewRefundUsecaseInput) *RefundUsecase {
	return &RefundUsecase{
		paymentReader:                 in.PaymentReader,
		settlementRepo:                in.SettlementRepository,
		salesReceivableService:        in.SalesReceivableService,
		stripeRefundGateway:           in.StripeRefundGateway,
		stripeTransferReversalGateway: in.StripeTransferReversalGateway,
	}
}

// RefundByPaymentID performs a full refund for one succeeded Payment.
// PaymentID is currently the same value as OrderID.
func (u *RefundUsecase) RefundByPaymentID(ctx context.Context, in RefundByPaymentIDInput) (*RefundByPaymentIDResult, error) {
	if err := u.validateConfigured(true); err != nil {
		return nil, err
	}

	paymentID := in.PaymentID
	if paymentID == "" {
		return nil, paymentdom.ErrInvalidPaymentID
	}

	payment, err := u.paymentReader.GetByPaymentID(ctx, paymentID)
	if err != nil {
		return nil, err
	}
	if payment == nil || payment.PaymentID != paymentID {
		return nil, paymentdom.ErrNotFound
	}
	if payment.Status != paymentdom.StatusSucceeded {
		return nil, ErrRefundPaymentNotSucceeded
	}
	if payment.StripeChargeID == "" {
		return nil, ErrRefundStripeChargeIDMissing
	}
	if payment.Amount <= 0 {
		return nil, paymentdom.ErrInvalidAmount
	}

	settlements, receivables, err := u.loadFinancialSources(ctx, paymentID)
	if err != nil {
		return nil, err
	}

	hasTransferredSettlement, err := validateRefundFinancialSources(paymentID, payment.Amount, settlements, receivables)
	if err != nil {
		return nil, err
	}
	if hasTransferredSettlement && u.stripeTransferReversalGateway == nil {
		return nil, ErrRefundStripeTransferReversalGatewayMissing
	}

	result := newRefundByPaymentIDResult(*payment, settlements, receivables)
	settlementResults := newRefundSettlementResultMap(settlements)
	receivableResults := newRefundSalesReceivableResultMap(receivables)

	if err := u.cancelUntransferredSettlementsForRefund(ctx, settlements, settlementResults); err != nil {
		populateRefundFinancialResults(result, settlements, settlementResults, receivables, receivableResults)
		return result, err
	}
	if err := u.cancelUnpaidSalesReceivablesForRefund(ctx, receivables, receivableResults); err != nil {
		populateRefundFinancialResults(result, settlements, settlementResults, receivables, receivableResults)
		return result, err
	}

	refundResult, refundErr := u.stripeRefundGateway.CreateRefund(ctx, applicationport.CreateStripeRefundInput{
		StripeChargeID: payment.StripeChargeID,
		Amount:         payment.Amount,
		IdempotencyKey: refundIdempotencyKey(paymentID),
		PaymentID:      paymentID,
	})
	if refundErr != nil {
		populateRefundFinancialResults(result, settlements, settlementResults, receivables, receivableResults)
		return result, fmt.Errorf("refund: create Stripe refund: %w", refundErr)
	}
	if refundResult == nil {
		populateRefundFinancialResults(result, settlements, settlementResults, receivables, receivableResults)
		return result, ErrRefundStripeRefundResultEmpty
	}
	if err := applyCreateStripeRefundResult(result, payment.Amount, refundResult); err != nil {
		populateRefundFinancialResults(result, settlements, settlementResults, receivables, receivableResults)
		return result, err
	}

	switch result.RefundStatus {
	case paymentdom.RefundStatusPending, paymentdom.RefundStatusRequiresAction, paymentdom.RefundStatusFailed, paymentdom.RefundStatusCanceled:
		populateRefundFinancialResults(result, settlements, settlementResults, receivables, receivableResults)
		return result, nil
	case paymentdom.RefundStatusSucceeded:
		if err := u.reverseTransferredSettlementsForRefund(ctx, settlements, settlementResults); err != nil {
			populateRefundFinancialResults(result, settlements, settlementResults, receivables, receivableResults)
			return result, err
		}
	default:
		populateRefundFinancialResults(result, settlements, settlementResults, receivables, receivableResults)
		return result, ErrRefundStripeRefundStatusInvalid
	}

	populateRefundFinancialResults(result, settlements, settlementResults, receivables, receivableResults)
	return result, nil
}

// CompleteSucceededRefund completes seller-side handling after an asynchronous
// Stripe Refund has become succeeded. It is idempotent for already canceled
// SalesReceivables and canceled/reversed Settlements.
func (u *RefundUsecase) CompleteSucceededRefund(ctx context.Context, in CompleteSucceededRefundInput) (*RefundByPaymentIDResult, error) {
	if err := u.validateConfigured(false); err != nil {
		return nil, err
	}

	paymentID := in.PaymentID
	if paymentID == "" {
		return nil, paymentdom.ErrInvalidPaymentID
	}
	if !isStripeRefundID(in.StripeRefundID) {
		return nil, ErrRefundStripeRefundIDEmpty
	}

	payment, err := u.paymentReader.GetByPaymentID(ctx, paymentID)
	if err != nil {
		return nil, err
	}
	if payment == nil || payment.PaymentID != paymentID {
		return nil, paymentdom.ErrNotFound
	}
	if payment.Status != paymentdom.StatusSucceeded {
		return nil, ErrRefundPaymentNotSucceeded
	}
	if payment.RefundStatus != paymentdom.RefundStatusSucceeded {
		return nil, paymentdom.ErrInvalidRefundState
	}
	if payment.StripeRefundID != in.StripeRefundID {
		return nil, ErrRefundStripeRefundMismatch
	}
	if payment.RefundedAmount != payment.Amount || payment.RefundedAt == nil {
		return nil, paymentdom.ErrInvalidRefundState
	}

	settlements, receivables, err := u.loadFinancialSources(ctx, paymentID)
	if err != nil {
		return nil, err
	}

	hasTransferredSettlement, err := validateRefundFinancialSources(paymentID, payment.Amount, settlements, receivables)
	if err != nil {
		return nil, err
	}
	if hasTransferredSettlement && u.stripeTransferReversalGateway == nil {
		return nil, ErrRefundStripeTransferReversalGatewayMissing
	}

	result := newRefundByPaymentIDResult(*payment, settlements, receivables)
	settlementResults := newRefundSettlementResultMap(settlements)
	receivableResults := newRefundSalesReceivableResultMap(receivables)

	if err := u.cancelUntransferredSettlementsForRefund(ctx, settlements, settlementResults); err != nil {
		populateRefundFinancialResults(result, settlements, settlementResults, receivables, receivableResults)
		return result, err
	}
	if err := u.cancelUnpaidSalesReceivablesForRefund(ctx, receivables, receivableResults); err != nil {
		populateRefundFinancialResults(result, settlements, settlementResults, receivables, receivableResults)
		return result, err
	}
	if err := u.reverseTransferredSettlementsForRefund(ctx, settlements, settlementResults); err != nil {
		populateRefundFinancialResults(result, settlements, settlementResults, receivables, receivableResults)
		return result, err
	}

	populateRefundFinancialResults(result, settlements, settlementResults, receivables, receivableResults)
	return result, nil
}

func (u *RefundUsecase) validateConfigured(requireStripeRefundGateway bool) error {
	if u == nil || u.paymentReader == nil {
		return ErrRefundPaymentReaderMissing
	}
	if u.settlementRepo == nil {
		return ErrRefundSettlementRepositoryMissing
	}
	if u.salesReceivableService == nil {
		return ErrRefundSalesReceivableServiceMissing
	}
	if requireStripeRefundGateway && u.stripeRefundGateway == nil {
		return ErrRefundStripeRefundGatewayMissing
	}
	return nil
}

func (u *RefundUsecase) loadFinancialSources(ctx context.Context, paymentID string) ([]settlementdom.Settlement, []salesreceivabledom.SalesReceivable, error) {
	settlements, err := u.settlementRepo.ListByPaymentID(ctx, paymentID)
	if err != nil {
		return nil, nil, err
	}
	receivables, err := u.salesReceivableService.ListByPaymentID(ctx, paymentID)
	if err != nil {
		return nil, nil, err
	}
	sort.Slice(settlements, func(i, j int) bool { return settlements[i].ID < settlements[j].ID })
	sort.Slice(receivables, func(i, j int) bool { return receivables[i].ID < receivables[j].ID })
	return settlements, receivables, nil
}

func newRefundByPaymentIDResult(payment paymentdom.Payment, settlements []settlementdom.Settlement, receivables []salesreceivabledom.SalesReceivable) *RefundByPaymentIDResult {
	result := &RefundByPaymentIDResult{
		PaymentID:        payment.PaymentID,
		Amount:           payment.Amount,
		Settlements:      make([]RefundSettlementResult, 0, len(settlements)),
		SalesReceivables: make([]RefundSalesReceivableResult, 0, len(receivables)),
	}
	if payment.RefundStatus != "" && payment.RefundStatus != paymentdom.RefundStatusNone {
		result.StripeRefundID = payment.StripeRefundID
		result.RefundStatus = payment.RefundStatus
		result.RefundedAmount = payment.RefundedAmount
		if payment.RefundedAt != nil {
			value := payment.RefundedAt.UTC()
			result.RefundedAt = &value
		}
	}
	return result
}

func newRefundSettlementResultMap(settlements []settlementdom.Settlement) map[string]RefundSettlementResult {
	result := make(map[string]RefundSettlementResult, len(settlements))
	for _, settlement := range settlements {
		seller := settlement.SellerIdentity()
		result[settlement.ID] = RefundSettlementResult{
			SettlementID:             settlement.ID,
			SellerType:               seller.Type,
			CompanyID:                seller.CompanyID,
			AccountID:                seller.AccountID,
			StripeAccountID:          seller.StripeAccountID,
			PreviousStatus:           settlement.Status,
			Status:                   settlement.Status,
			StripeTransferID:         settlement.StripeTransferID,
			StripeTransferReversalID: settlement.StripeTransferReversalID,
		}
	}
	return result
}

func newRefundSalesReceivableResultMap(receivables []salesreceivabledom.SalesReceivable) map[string]RefundSalesReceivableResult {
	result := make(map[string]RefundSalesReceivableResult, len(receivables))
	for _, receivable := range receivables {
		result[receivable.ID] = RefundSalesReceivableResult{
			SalesReceivableID: receivable.ID,
			OrderItemIndex:    receivable.OrderItemIndex,
			ResaleID:          receivable.ResaleID,
			AvatarID:          receivable.AvatarID,
			UserID:            receivable.UserID,
			PayoutAccountID:   receivable.PayoutAccountID,
			PreviousStatus:    receivable.Status,
			Status:            receivable.Status,
		}
	}
	return result
}

func populateRefundFinancialResults(result *RefundByPaymentIDResult, settlements []settlementdom.Settlement, settlementValues map[string]RefundSettlementResult, receivables []salesreceivabledom.SalesReceivable, receivableValues map[string]RefundSalesReceivableResult) {
	if result == nil {
		return
	}
	result.Settlements = buildRefundSettlementResults(settlements, settlementValues)
	result.SalesReceivables = buildRefundSalesReceivableResults(receivables, receivableValues)
}

func applyCreateStripeRefundResult(result *RefundByPaymentIDResult, paymentAmount int, refundResult *applicationport.CreateStripeRefundResult) error {
	if result == nil || refundResult == nil {
		return ErrRefundStripeRefundResultEmpty
	}
	stripeRefundID := refundResult.StripeRefundID
	if !isStripeRefundID(stripeRefundID) {
		return ErrRefundStripeRefundIDEmpty
	}
	if !paymentdom.IsValidRefundStatus(refundResult.Status) || refundResult.Status == paymentdom.RefundStatusNone {
		return ErrRefundStripeRefundStatusInvalid
	}
	if refundResult.CreatedAt.IsZero() {
		return ErrRefundStripeRefundCreatedAtInvalid
	}

	result.StripeRefundID = stripeRefundID
	result.RefundStatus = refundResult.Status
	result.RefundedAmount = 0
	result.RefundedAt = nil
	if refundResult.Status == paymentdom.RefundStatusSucceeded {
		if paymentAmount <= 0 {
			return paymentdom.ErrInvalidAmount
		}
		result.RefundedAmount = paymentAmount
		value := refundResult.CreatedAt.UTC()
		result.RefundedAt = &value
	}
	return nil
}

// ============================================================
// Seller-side state transitions
// ============================================================

func (u *RefundUsecase) cancelUntransferredSettlementsForRefund(ctx context.Context, settlements []settlementdom.Settlement, settlementResults map[string]RefundSettlementResult) error {
	for _, settlement := range settlements {
		switch settlement.Status {
		case settlementdom.StatusPending, settlementdom.StatusReady, settlementdom.StatusFailedRetryable:
			updated, err := u.cancelSettlementForRefund(ctx, settlement)
			if err != nil {
				return fmt.Errorf("refund: cancel settlement %q: %w", settlement.ID, err)
			}
			item := settlementResults[settlement.ID]
			item.Status = updated.Status
			item.Action = RefundSettlementActionCanceled
			settlementResults[settlement.ID] = item
		case settlementdom.StatusCanceled:
			item := settlementResults[settlement.ID]
			item.Action = RefundSettlementActionAlreadyCanceled
			settlementResults[settlement.ID] = item
		case settlementdom.StatusTransferred, settlementdom.StatusReversed:
		}
	}
	return nil
}

func (u *RefundUsecase) cancelUnpaidSalesReceivablesForRefund(ctx context.Context, receivables []salesreceivabledom.SalesReceivable, receivableResults map[string]RefundSalesReceivableResult) error {
	for _, receivable := range receivables {
		switch receivable.Status {
		case salesreceivabledom.StatusPending, salesreceivabledom.StatusAvailable:
			updated, err := u.salesReceivableService.Cancel(ctx, receivable.ID)
			if err != nil {
				return fmt.Errorf("refund: cancel sales receivable %q: %w", receivable.ID, err)
			}
			if updated == nil || !sameSalesReceivableAllocation(receivable, *updated) || updated.Status != salesreceivabledom.StatusCanceled {
				return ErrRefundSalesReceivableMismatch
			}
			if err := updated.Validate(); err != nil {
				return fmt.Errorf("%w: %v", ErrRefundSalesReceivableMismatch, err)
			}
			item := receivableResults[receivable.ID]
			item.Status = updated.Status
			item.Action = RefundSalesReceivableActionCanceled
			receivableResults[receivable.ID] = item
		case salesreceivabledom.StatusCanceled:
			item := receivableResults[receivable.ID]
			item.Action = RefundSalesReceivableActionAlreadyCanceled
			receivableResults[receivable.ID] = item
		case salesreceivabledom.StatusReserved:
			return ErrRefundSalesReceivableReserved
		case salesreceivabledom.StatusPaid:
			return ErrRefundSalesReceivablePaid
		default:
			return ErrRefundSalesReceivableStatusUnsupported
		}
	}
	return nil
}

func (u *RefundUsecase) reverseTransferredSettlementsForRefund(ctx context.Context, settlements []settlementdom.Settlement, settlementResults map[string]RefundSettlementResult) error {
	for _, settlement := range settlements {
		switch settlement.Status {
		case settlementdom.StatusTransferred:
			updated, err := u.reverseTransferredSettlement(ctx, settlement)
			if err != nil {
				return fmt.Errorf("refund: reverse settlement %q: %w", settlement.ID, err)
			}
			item := settlementResults[settlement.ID]
			item.Status = updated.Status
			item.StripeTransferReversalID = updated.StripeTransferReversalID
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

func (u *RefundUsecase) cancelSettlementForRefund(ctx context.Context, settlement settlementdom.Settlement) (settlementdom.Settlement, error) {
	seller := settlement.SellerIdentity()
	if err := seller.Validate(); err != nil {
		return settlementdom.Settlement{}, ErrRefundSettlementPaymentMismatch
	}
	status := settlementdom.StatusCanceled
	updated, err := u.settlementRepo.UpdateByID(ctx, settlement.ID, settlementdom.UpdateSettlementInput{Status: &status})
	if err != nil {
		return settlementdom.Settlement{}, err
	}
	updatedSeller := updated.SellerIdentity()
	if err := updatedSeller.Validate(); err != nil {
		return settlementdom.Settlement{}, ErrRefundSettlementPaymentMismatch
	}
	if updated.ID != settlement.ID || updated.PaymentID != settlement.PaymentID || updated.OrderID != settlement.OrderID || updatedSeller != seller {
		return settlementdom.Settlement{}, ErrRefundSettlementPaymentMismatch
	}
	if updated.Status != settlementdom.StatusCanceled {
		return settlementdom.Settlement{}, settlementdom.ErrInvalidStatusTransition
	}
	return updated, nil
}

func (u *RefundUsecase) reverseTransferredSettlement(ctx context.Context, settlement settlementdom.Settlement) (settlementdom.Settlement, error) {
	if u.stripeTransferReversalGateway == nil {
		return settlementdom.Settlement{}, ErrRefundStripeTransferReversalGatewayMissing
	}
	if settlement.Status != settlementdom.StatusTransferred {
		return settlementdom.Settlement{}, settlementdom.ErrInvalidStatusTransition
	}
	seller := settlement.SellerIdentity()
	if err := seller.Validate(); err != nil {
		return settlementdom.Settlement{}, ErrRefundSettlementPaymentMismatch
	}
	if settlement.StripeTransferID == "" {
		return settlementdom.Settlement{}, settlementdom.ErrInvalidStripeTransferID
	}
	if settlement.TransferAmount <= 0 {
		return settlementdom.Settlement{}, settlementdom.ErrInvalidTransferAmount
	}

	reversalResult, reversalErr := u.stripeTransferReversalGateway.CreateTransferReversal(ctx, applicationport.CreateStripeTransferReversalInput{
		StripeTransferID: settlement.StripeTransferID,
		Amount:           settlement.TransferAmount,
		IdempotencyKey:   transferReversalIdempotencyKey(settlement.ID),
		OrderID:          settlement.OrderID,
		PaymentID:        settlement.PaymentID,
		SettlementID:     settlement.ID,
		Seller:           seller,
	})
	if reversalErr != nil {
		return settlementdom.Settlement{}, fmt.Errorf("Stripe transfer reversal: %w", reversalErr)
	}
	if reversalResult == nil {
		return settlementdom.Settlement{}, ErrRefundStripeTransferReversalResultEmpty
	}
	reversalID := reversalResult.StripeTransferReversalID
	if !isStripeTransferReversalIDForRefund(reversalID) {
		return settlementdom.Settlement{}, ErrRefundStripeTransferReversalIDEmpty
	}

	status := settlementdom.StatusReversed
	updated, err := u.settlementRepo.UpdateByID(ctx, settlement.ID, settlementdom.UpdateSettlementInput{
		StripeTransferReversalID: &reversalID,
		Status:                   &status,
	})
	if err != nil {
		return settlementdom.Settlement{}, fmt.Errorf("persist Stripe transfer reversal %q: %w", reversalID, err)
	}
	updatedSeller := updated.SellerIdentity()
	if err := updatedSeller.Validate(); err != nil {
		return settlementdom.Settlement{}, ErrRefundSettlementPaymentMismatch
	}
	if updated.ID != settlement.ID || updated.PaymentID != settlement.PaymentID || updated.OrderID != settlement.OrderID || updatedSeller != seller {
		return settlementdom.Settlement{}, ErrRefundSettlementPaymentMismatch
	}
	if updated.Status != settlementdom.StatusReversed || updated.StripeTransferReversalID != reversalID {
		return settlementdom.Settlement{}, settlementdom.ErrInvalidStatusTransition
	}
	return updated, nil
}

// ============================================================
// Validation
// ============================================================

func validateRefundFinancialSources(paymentID string, paymentAmount int, settlements []settlementdom.Settlement, receivables []salesreceivabledom.SalesReceivable) (bool, error) {
	if paymentID == "" {
		return false, paymentdom.ErrInvalidPaymentID
	}
	if paymentAmount <= 0 {
		return false, paymentdom.ErrInvalidAmount
	}
	if len(settlements) == 0 && len(receivables) == 0 {
		return false, ErrRefundFinancialSourceEmpty
	}

	settlementTotal, hasTransferredSettlement, err := validateRefundSettlements(paymentID, settlements)
	if err != nil {
		return false, err
	}
	receivableTotal, err := validateRefundSalesReceivables(paymentID, receivables)
	if err != nil {
		return false, err
	}
	maxInt := int(^uint(0) >> 1)
	if settlementTotal > maxInt-receivableTotal || settlementTotal+receivableTotal != paymentAmount {
		return false, ErrRefundFinancialSourceAmountMismatch
	}
	return hasTransferredSettlement, nil
}

func validateRefundSettlements(paymentID string, settlements []settlementdom.Settlement) (int, bool, error) {
	seenSettlementIDs := make(map[string]struct{}, len(settlements))
	seenSellerKeys := make(map[string]struct{}, len(settlements))
	maxInt := int(^uint(0) >> 1)
	total := 0
	hasTransferredSettlement := false

	for _, settlement := range settlements {
		if settlement.ID == "" || settlement.PaymentID != paymentID || settlement.OrderID != paymentID {
			return 0, false, ErrRefundSettlementPaymentMismatch
		}
		if _, exists := seenSettlementIDs[settlement.ID]; exists {
			return 0, false, ErrRefundSettlementDuplicate
		}
		seenSettlementIDs[settlement.ID] = struct{}{}
		if err := settlement.Validate(); err != nil {
			return 0, false, fmt.Errorf("%w: %v", ErrRefundSettlementPaymentMismatch, err)
		}

		seller := settlement.SellerIdentity()
		if err := seller.Validate(); err != nil {
			return 0, false, ErrRefundSettlementPaymentMismatch
		}
		expectedSettlementID, err := settlementdom.NewID(paymentID, seller)
		if err != nil || expectedSettlementID != settlement.ID {
			return 0, false, ErrRefundSettlementPaymentMismatch
		}
		sellerID, err := seller.Key()
		if err != nil {
			return 0, false, ErrRefundSettlementPaymentMismatch
		}
		sellerKey := string(seller.Type) + ":" + sellerID
		if _, exists := seenSellerKeys[sellerKey]; exists {
			return 0, false, ErrRefundSettlementDuplicate
		}
		seenSellerKeys[sellerKey] = struct{}{}

		if settlement.GrossAmount <= 0 || settlement.TransferAmount <= 0 {
			return 0, false, ErrRefundSettlementAmountMismatch
		}
		if total > maxInt-settlement.GrossAmount {
			return 0, false, ErrRefundFinancialSourceAmountMismatch
		}
		total += settlement.GrossAmount

		switch settlement.Status {
		case settlementdom.StatusPending, settlementdom.StatusReady, settlementdom.StatusFailedRetryable, settlementdom.StatusCanceled:
		case settlementdom.StatusTransferred:
			hasTransferredSettlement = true
		case settlementdom.StatusReversed:
		case settlementdom.StatusTransferring:
			return 0, false, ErrRefundSettlementTransferring
		case settlementdom.StatusFailed:
			return 0, false, ErrRefundSettlementFailed
		default:
			return 0, false, ErrRefundSettlementStatusUnsupported
		}
	}
	return total, hasTransferredSettlement, nil
}

func validateRefundSalesReceivables(paymentID string, receivables []salesreceivabledom.SalesReceivable) (int, error) {
	seenIDs := make(map[string]struct{}, len(receivables))
	seenItemIndexes := make(map[int]struct{}, len(receivables))
	maxInt := int(^uint(0) >> 1)
	total := 0

	for _, receivable := range receivables {
		if receivable.ID == "" || receivable.PaymentID != paymentID || receivable.OrderID != paymentID {
			return 0, ErrRefundSalesReceivableMismatch
		}
		if _, exists := seenIDs[receivable.ID]; exists {
			return 0, ErrRefundSalesReceivableDuplicate
		}
		seenIDs[receivable.ID] = struct{}{}
		if _, exists := seenItemIndexes[receivable.OrderItemIndex]; exists {
			return 0, ErrRefundSalesReceivableDuplicate
		}
		seenItemIndexes[receivable.OrderItemIndex] = struct{}{}
		if err := receivable.Validate(); err != nil {
			return 0, fmt.Errorf("%w: %v", ErrRefundSalesReceivableMismatch, err)
		}
		expectedID, err := salesreceivabledom.NewID(paymentID, receivable.OrderItemIndex)
		if err != nil || expectedID != receivable.ID {
			return 0, ErrRefundSalesReceivableMismatch
		}
		if total > maxInt-receivable.GrossAmount {
			return 0, ErrRefundFinancialSourceAmountMismatch
		}
		total += receivable.GrossAmount

		switch receivable.Status {
		case salesreceivabledom.StatusPending, salesreceivabledom.StatusAvailable, salesreceivabledom.StatusCanceled:
		case salesreceivabledom.StatusReserved:
			return 0, ErrRefundSalesReceivableReserved
		case salesreceivabledom.StatusPaid:
			return 0, ErrRefundSalesReceivablePaid
		default:
			return 0, ErrRefundSalesReceivableStatusUnsupported
		}
	}
	return total, nil
}

func sameSalesReceivableAllocation(left, right salesreceivabledom.SalesReceivable) bool {
	return left.ID == right.ID &&
		left.OrderID == right.OrderID &&
		left.PaymentID == right.PaymentID &&
		left.OrderItemIndex == right.OrderItemIndex &&
		left.ResaleID == right.ResaleID &&
		left.AvatarID == right.AvatarID &&
		left.UserID == right.UserID &&
		left.PayoutAccountID == right.PayoutAccountID &&
		left.GrossAmount == right.GrossAmount &&
		left.PlatformFeeAmount == right.PlatformFeeAmount &&
		left.ReceivableAmount == right.ReceivableAmount &&
		left.Currency == right.Currency &&
		left.CreatedAt.Equal(right.CreatedAt)
}

func buildRefundSettlementResults(settlements []settlementdom.Settlement, values map[string]RefundSettlementResult) []RefundSettlementResult {
	result := make([]RefundSettlementResult, 0, len(settlements))
	for _, settlement := range settlements {
		if value, exists := values[settlement.ID]; exists {
			result = append(result, value)
		}
	}
	return result
}

func buildRefundSalesReceivableResults(receivables []salesreceivabledom.SalesReceivable, values map[string]RefundSalesReceivableResult) []RefundSalesReceivableResult {
	result := make([]RefundSalesReceivableResult, 0, len(receivables))
	for _, receivable := range receivables {
		if value, exists := values[receivable.ID]; exists {
			result = append(result, value)
		}
	}
	return result
}

// ============================================================
// Stripe idempotency / identifiers
// ============================================================

func refundIdempotencyKey(paymentID string) string {
	return fmt.Sprintf("refund:%s", paymentID)
}

func transferReversalIdempotencyKey(settlementID string) string {
	return fmt.Sprintf("settlement-reversal:%s", settlementID)
}

func isStripeRefundID(value string) bool {
	return strings.HasPrefix(value, "re_") && len(value) > len("re_")
}

func isStripeTransferReversalIDForRefund(value string) bool {
	return strings.HasPrefix(value, "trr_") && len(value) > len("trr_")
}
