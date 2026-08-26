// backend/internal/application/usecase/item_refund_usecase.go
package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	orderdom "narratives/internal/domain/order"
	paymentdom "narratives/internal/domain/payment"
	refunddom "narratives/internal/domain/refund"
	settlementdom "narratives/internal/domain/settlement"
)

// ============================================================
// Ports
// ============================================================

// ItemRefundOrderReader provides the authoritative Order snapshot required for
// one item-level refund.
//
// Refund amount and seller identity must always be derived from the Order.
// They must never be accepted from the frontend.
type ItemRefundOrderReader interface {
	GetByID(
		ctx context.Context,
		id string,
	) (orderdom.Order, error)
}

// ItemRefundPaymentReader provides the original succeeded Stripe Payment.
//
// Payment is read-only in this flow.
//
// Item-level partial refund state is persisted in refund.Refund instead of
// Payment because Payment's current refund fields represent the legacy
// full-payment refund lifecycle.
type ItemRefundPaymentReader interface {
	GetByPaymentID(
		ctx context.Context,
		paymentID string,
	) (*paymentdom.Payment, error)
}

// ItemRefundSettlementRepository provides seller Settlement state.
//
// The current item-level refund flow does not mark Settlement reversed because
// one Settlement may contain multiple Order items and one item return performs
// only a partial Transfer Reversal.
//
// Each partial reversal is persisted independently in refund.Refund.
type ItemRefundSettlementRepository interface {
	ListByPaymentID(
		ctx context.Context,
		paymentID string,
	) ([]settlementdom.Settlement, error)
}

// ============================================================
// Errors
// ============================================================

var (
	ErrItemRefundNotConfigured = errors.New(
		"item refund: usecase is not configured",
	)

	ErrItemRefundInvalidInquiryID = errors.New(
		"item refund: invalid inquiryId",
	)

	ErrItemRefundInvalidCompanyID = errors.New(
		"item refund: invalid companyId",
	)

	ErrItemRefundOrderMismatch = errors.New(
		"item refund: order does not match refund target",
	)

	ErrItemRefundCompanyMismatch = errors.New(
		"item refund: order item does not belong to company",
	)

	ErrItemRefundAccountMissing = errors.New(
		"item refund: seller account is missing",
	)

	ErrItemRefundPaymentNotSucceeded = errors.New(
		"item refund: payment is not succeeded",
	)

	ErrItemRefundPaymentAlreadyRefunding = errors.New(
		"item refund: payment full refund has already started",
	)

	ErrItemRefundStripeChargeMissing = errors.New(
		"item refund: Stripe charge id is missing",
	)

	ErrItemRefundSettlementNotFound = errors.New(
		"item refund: seller settlement is not found",
	)

	ErrItemRefundSettlementDuplicate = errors.New(
		"item refund: duplicate seller settlement",
	)

	ErrItemRefundSettlementMismatch = errors.New(
		"item refund: settlement does not match order item",
	)

	ErrItemRefundSettlementNotTransferred = errors.New(
		"item refund: settlement has not completed seller transfer",
	)

	ErrItemRefundSettlementUnavailable = errors.New(
		"item refund: settlement cannot be used for item refund",
	)

	ErrItemRefundInvalidPlatformFee = errors.New(
		"item refund: invalid item platform fee",
	)

	ErrItemRefundPaymentAmountExceeded = errors.New(
		"item refund: cumulative refund amount exceeds payment amount",
	)

	ErrItemRefundTransferReversalAmountExceeded = errors.New(
		"item refund: cumulative transfer reversal amount exceeds settlement transfer amount",
	)

	ErrItemRefundExistingRefundMismatch = errors.New(
		"item refund: existing refund does not match request",
	)

	ErrItemRefundStripeRefundResultEmpty = errors.New(
		"item refund: Stripe refund result is empty",
	)

	ErrItemRefundStripeRefundStatusInvalid = errors.New(
		"item refund: Stripe refund status is invalid",
	)

	ErrItemRefundStripeRefundTerminal = errors.New(
		"item refund: Stripe refund is in terminal failure state",
	)

	ErrItemRefundStripeTransferReversalResultEmpty = errors.New(
		"item refund: Stripe transfer reversal result is empty",
	)

	ErrItemRefundStripeTransferReversalIDInvalid = errors.New(
		"item refund: Stripe transfer reversal id is invalid",
	)

	ErrItemRefundTransferReversalTerminal = errors.New(
		"item refund: transfer reversal is in terminal failure state",
	)
)

// ============================================================
// Usecase
// ============================================================

// ItemRefundUsecase coordinates one purchaser-side partial Stripe Refund and,
// when required, one seller-side partial Stripe Transfer Reversal.
//
// The authoritative refund amount is:
//
//	merchandise amount
//	+ merchandise consumption tax
//
// Shipping and shipping consumption tax are intentionally excluded.
//
// Seller-side reversal amount is:
//
//	refund amount
//	- platform fee attributable to the returned merchandise/tax
//
// Platform fee calculation uses the same PlatformFeeCalculator policy used by
// Settlement creation, with:
//
//	ShippingAmount    = 0
//	ShippingTaxAmount = 0
//
// Execution order:
//
//  1. Load authoritative Order.
//  2. Calculate exact item refund amount.
//  3. Load succeeded Payment.
//  4. Resolve seller Settlement.
//  5. Load or create deterministic Refund aggregate.
//  6. Create Stripe purchaser Refund.
//  7. Persist Stripe Refund result.
//  8. If Refund succeeded, create seller partial Transfer Reversal.
//  9. Persist Transfer Reversal result.
//
// Order return completion and Inquiry resolution are intentionally NOT handled
// here. ReturnReceiptUsecase performs them only after:
//
//	Refund.IsFinanciallyCompleted() == true
type ItemRefundUsecase struct {
	orderReader                   ItemRefundOrderReader
	paymentReader                 ItemRefundPaymentReader
	settlementRepo                ItemRefundSettlementRepository
	refundRepo                    refunddom.RepositoryPort
	platformFeeCalculator         settlementdom.PlatformFeeCalculator
	stripeRefundGateway           StripeRefundGateway
	stripeTransferReversalGateway StripeTransferReversalGateway
	now                           func() time.Time
}

type NewItemRefundUsecaseInput struct {
	OrderReader                   ItemRefundOrderReader
	PaymentReader                 ItemRefundPaymentReader
	SettlementRepository          ItemRefundSettlementRepository
	RefundRepository              refunddom.RepositoryPort
	PlatformFeeCalculator         settlementdom.PlatformFeeCalculator
	StripeRefundGateway           StripeRefundGateway
	StripeTransferReversalGateway StripeTransferReversalGateway
}

func NewItemRefundUsecase(
	in NewItemRefundUsecaseInput,
) *ItemRefundUsecase {
	return &ItemRefundUsecase{
		orderReader:                   in.OrderReader,
		paymentReader:                 in.PaymentReader,
		settlementRepo:                in.SettlementRepository,
		refundRepo:                    in.RefundRepository,
		platformFeeCalculator:         in.PlatformFeeCalculator,
		stripeRefundGateway:           in.StripeRefundGateway,
		stripeTransferReversalGateway: in.StripeTransferReversalGateway,
		now:                           time.Now,
	}
}

// SetNowFunc replaces the current-time source for tests.
func (u *ItemRefundUsecase) SetNowFunc(
	now func() time.Time,
) {
	if u == nil ||
		now == nil {
		return
	}

	u.now = now
}

// ============================================================
// Refund Order Item
// ============================================================

// RefundOrderItem executes or resumes one item-level refund.
//
// The operation is idempotent:
//
// - Refund ID is deterministic from orderId + itemIndex.
// - Stripe Refund uses a deterministic idempotency key.
// - Stripe Transfer Reversal uses a deterministic idempotency key.
// - RefundRepository persists every completed financial transition.
//
// A persisted pending/requires_action Stripe Refund is returned without
// completing the seller reversal. A verified Stripe webhook should later
// update the same Refund aggregate to succeeded, after which retrying this
// method resumes from the seller reversal step.
func (u *ItemRefundUsecase) RefundOrderItem(
	ctx context.Context,
	in RefundOrderItemInput,
) (refunddom.Refund, error) {
	if err := u.validateConfigured(); err != nil {
		return refunddom.Refund{}, err
	}

	if in.InquiryID == "" {
		return refunddom.Refund{},
			ErrItemRefundInvalidInquiryID
	}

	if in.OrderID == "" ||
		in.ItemIndex < 0 {
		return refunddom.Refund{},
			ErrItemRefundOrderMismatch
	}

	if in.CompanyID == "" {
		return refunddom.Refund{},
			ErrItemRefundInvalidCompanyID
	}

	order, err :=
		u.orderReader.GetByID(
			ctx,
			in.OrderID,
		)
	if err != nil {
		return refunddom.Refund{}, err
	}

	if order.ID != in.OrderID ||
		in.ItemIndex >= len(order.Items) {
		return refunddom.Refund{},
			ErrItemRefundOrderMismatch
	}

	targetItem :=
		order.Items[in.ItemIndex]

	if targetItem.SellerSnapshot.CompanyID !=
		in.CompanyID {
		return refunddom.Refund{},
			ErrItemRefundCompanyMismatch
	}

	if targetItem.SellerSnapshot.AccountID == "" {
		return refunddom.Refund{},
			ErrItemRefundAccountMissing
	}

	amountSummary, err :=
		orderdom.CalculateOrderItemRefundAmount(
			order,
			in.ItemIndex,
		)
	if err != nil {
		return refunddom.Refund{}, err
	}

	if amountSummary.RefundAmount <= 0 {
		return refunddom.Refund{},
			refunddom.ErrInvalidRefundAmount
	}

	payment, err :=
		u.paymentReader.GetByPaymentID(
			ctx,
			order.ID,
		)
	if err != nil {
		return refunddom.Refund{}, err
	}

	if payment == nil ||
		payment.PaymentID != order.ID {
		return refunddom.Refund{},
			paymentdom.ErrNotFound
	}

	if payment.Status !=
		paymentdom.StatusSucceeded {
		return refunddom.Refund{},
			ErrItemRefundPaymentNotSucceeded
	}

	if payment.StripeChargeID == "" ||
		!strings.HasPrefix(
			payment.StripeChargeID,
			"ch_",
		) {
		return refunddom.Refund{},
			ErrItemRefundStripeChargeMissing
	}

	settlement, err :=
		u.resolveSettlement(
			ctx,
			*payment,
			targetItem,
		)
	if err != nil {
		return refunddom.Refund{}, err
	}

	refundID, err :=
		refunddom.NewID(
			order.ID,
			in.ItemIndex,
		)
	if err != nil {
		return refunddom.Refund{}, err
	}

	existingRefund, err :=
		u.refundRepo.GetByID(
			ctx,
			refundID,
		)
	if err == nil {
		if err := validateExistingItemRefund(
			*existingRefund,
			in,
			order,
			targetItem,
			settlement,
			amountSummary,
		); err != nil {
			return refunddom.Refund{}, err
		}

		return u.resumeRefund(
			ctx,
			*payment,
			settlement,
			*existingRefund,
		)
	}

	if !errors.Is(
		err,
		refunddom.ErrNotFound,
	) {
		return refunddom.Refund{}, err
	}

	if payment.RefundStatus !=
		paymentdom.RefundStatusNone {
		return refunddom.Refund{},
			ErrItemRefundPaymentAlreadyRefunding
	}

	if err := u.validatePaymentRefundCapacity(
		ctx,
		payment.PaymentID,
		refundID,
		amountSummary.RefundAmount,
		payment.Amount,
	); err != nil {
		return refunddom.Refund{}, err
	}

	transferReversalAmount, err :=
		u.resolveTransferReversalAmount(
			ctx,
			settlement,
			targetItem,
			amountSummary,
		)
	if err != nil {
		return refunddom.Refund{}, err
	}

	if err := u.validateTransferReversalCapacity(
		ctx,
		settlement,
		refundID,
		transferReversalAmount,
	); err != nil {
		return refunddom.Refund{}, err
	}

	now :=
		u.nowUTC()

	createdRefund, err :=
		u.refundRepo.Create(
			ctx,
			refunddom.CreateRefundInput{
				RefundID:               refundID,
				InquiryID:              in.InquiryID,
				OrderID:                order.ID,
				PaymentID:              payment.PaymentID,
				OrderItemIndex:         in.ItemIndex,
				CompanyID:              targetItem.SellerSnapshot.CompanyID,
				AccountID:              targetItem.SellerSnapshot.AccountID,
				SettlementID:           settlement.ID,
				MerchandiseAmount:      amountSummary.MerchandiseAmount,
				MerchandiseTaxAmount:   amountSummary.MerchandiseTaxAmount,
				TransferReversalAmount: transferReversalAmount,
				Currency:               refunddom.CurrencyJPY,
				CreatedAt:              now,
			},
		)
	if err != nil {
		if !isRefundCreateConflict(err) {
			return refunddom.Refund{}, err
		}

		existing, getErr :=
			u.refundRepo.GetByID(
				ctx,
				refundID,
			)
		if getErr != nil {
			return refunddom.Refund{}, err
		}

		if err := validateExistingItemRefund(
			*existing,
			in,
			order,
			targetItem,
			settlement,
			amountSummary,
		); err != nil {
			return refunddom.Refund{}, err
		}

		return u.resumeRefund(
			ctx,
			*payment,
			settlement,
			*existing,
		)
	}

	if createdRefund == nil {
		return refunddom.Refund{},
			refunddom.ErrConflict
	}

	return u.resumeRefund(
		ctx,
		*payment,
		settlement,
		*createdRefund,
	)
}

// ============================================================
// Resume
// ============================================================

func (u *ItemRefundUsecase) resumeRefund(
	ctx context.Context,
	payment paymentdom.Payment,
	settlement settlementdom.Settlement,
	refund refunddom.Refund,
) (refunddom.Refund, error) {
	if err := refund.Validate(); err != nil {
		return refunddom.Refund{}, err
	}

	switch refund.Status {
	case refunddom.StatusCreated:
		updated, err :=
			u.createPurchaserRefund(
				ctx,
				payment,
				refund,
			)
		if err != nil {
			return updated, err
		}

		refund = updated

	case refunddom.StatusPending,
		refunddom.StatusRequiresAction:
		// Stripe accepted the Refund, but purchaser-side completion has not
		// been confirmed yet.
		//
		// Do not execute seller Transfer Reversal.
		return refund, nil

	case refunddom.StatusSucceeded:
		// Continue below.

	case refunddom.StatusFailed,
		refunddom.StatusCanceled:
		return refund,
			ErrItemRefundStripeRefundTerminal

	default:
		return refund,
			refunddom.ErrInvalidStatus
	}

	if refund.Status !=
		refunddom.StatusSucceeded {
		return refund, nil
	}

	return u.completeTransferReversal(
		ctx,
		payment,
		settlement,
		refund,
	)
}

// ============================================================
// Purchaser Refund
// ============================================================

func (u *ItemRefundUsecase) createPurchaserRefund(
	ctx context.Context,
	payment paymentdom.Payment,
	refund refunddom.Refund,
) (refunddom.Refund, error) {
	result, err :=
		u.stripeRefundGateway.CreateRefund(
			ctx,
			CreateStripeRefundInput{
				StripeChargeID: payment.StripeChargeID,
				Amount:         refund.RefundAmount,
				IdempotencyKey: itemRefundIdempotencyKey(
					"refund",
					refund.ID,
				),

				PaymentID: payment.PaymentID,
			},
		)
	if err != nil {
		return refund,
			fmt.Errorf(
				"item refund: create Stripe refund: %w",
				err,
			)
	}

	if result == nil {
		return refund,
			ErrItemRefundStripeRefundResultEmpty
	}

	if result.StripeRefundID == "" ||
		!strings.HasPrefix(
			result.StripeRefundID,
			"re_",
		) {
		return refund,
			refunddom.ErrInvalidStripeRefundID
	}

	if result.CreatedAt.IsZero() {
		return refund,
			refunddom.ErrInvalidRefundedAt
	}

	status, refundedAt, err :=
		mapStripeRefundResult(
			result,
		)
	if err != nil {
		return refund, err
	}

	if status ==
		refunddom.StatusSucceeded &&
		refundedAt != nil &&
		refundedAt.Before(
			refund.CreatedAt,
		) {
		delta :=
			refund.CreatedAt.Sub(
				*refundedAt,
			)

		if delta >= time.Second {
			return refund,
				refunddom.ErrInvalidRefundedAt
		}

		normalized :=
			refund.CreatedAt.UTC()

		refundedAt =
			&normalized
	}

	updated, err :=
		u.refundRepo.UpdateByID(
			ctx,
			refund.ID,
			refunddom.UpdateRefundInput{
				Operation:      refunddom.UpdateOperationApplyStripeRefund,
				StripeRefundID: result.StripeRefundID,
				RefundStatus:   status,
				RefundedAt:     refundedAt,
				UpdatedAt:      u.nowUTC(),
			},
		)
	if err != nil {
		return refund,
			fmt.Errorf(
				"item refund: persist Stripe refund result: %w",
				err,
			)
	}

	if updated == nil {
		return refund,
			refunddom.ErrConflict
	}

	switch updated.Status {
	case refunddom.StatusPending,
		refunddom.StatusRequiresAction,
		refunddom.StatusSucceeded:
		return *updated, nil

	case refunddom.StatusFailed,
		refunddom.StatusCanceled:
		return *updated,
			ErrItemRefundStripeRefundTerminal

	default:
		return *updated,
			ErrItemRefundStripeRefundStatusInvalid
	}
}

func mapStripeRefundResult(
	result *CreateStripeRefundResult,
) (
	refunddom.RefundStatus,
	*time.Time,
	error,
) {
	if result == nil {
		return "",
			nil,
			ErrItemRefundStripeRefundResultEmpty
	}

	switch result.Status {
	case paymentdom.RefundStatusPending:
		return refunddom.StatusPending,
			nil,
			nil

	case paymentdom.RefundStatusRequiresAction:
		return refunddom.StatusRequiresAction,
			nil,
			nil

	case paymentdom.RefundStatusSucceeded:
		if result.CreatedAt.IsZero() {
			return "",
				nil,
				refunddom.ErrInvalidRefundedAt
		}

		refundedAt :=
			result.CreatedAt.UTC()

		return refunddom.StatusSucceeded,
			&refundedAt,
			nil

	case paymentdom.RefundStatusFailed:
		return refunddom.StatusFailed,
			nil,
			nil

	case paymentdom.RefundStatusCanceled:
		return refunddom.StatusCanceled,
			nil,
			nil

	default:
		return "",
			nil,
			ErrItemRefundStripeRefundStatusInvalid
	}
}

// ============================================================
// Seller Transfer Reversal
// ============================================================

func (u *ItemRefundUsecase) completeTransferReversal(
	ctx context.Context,
	payment paymentdom.Payment,
	settlement settlementdom.Settlement,
	refund refunddom.Refund,
) (refunddom.Refund, error) {
	if refund.TransferReversalAmount == 0 {
		if refund.TransferReversalStatus !=
			refunddom.TransferReversalStatusNotRequired {
			return refund,
				refunddom.ErrInvalidTransferReversalStatus
		}

		return refund, nil
	}

	if settlement.Status !=
		settlementdom.StatusTransferred {
		return refund,
			ErrItemRefundSettlementNotTransferred
	}

	if settlement.StripeTransferID == "" ||
		!strings.HasPrefix(
			settlement.StripeTransferID,
			"tr_",
		) {
		return refund,
			ErrItemRefundSettlementMismatch
	}

	switch refund.TransferReversalStatus {
	case refunddom.TransferReversalStatusSucceeded:
		return refund, nil

	case refunddom.TransferReversalStatusNotRequired:
		return refund,
			refunddom.ErrInvalidTransferReversalStatus

	case refunddom.TransferReversalStatusFailed:
		return refund,
			ErrItemRefundTransferReversalTerminal

	case refunddom.TransferReversalStatusFailedRetryable:
		updated, err :=
			u.refundRepo.UpdateByID(
				ctx,
				refund.ID,
				refunddom.UpdateRefundInput{
					Operation: refunddom.UpdateOperationMarkTransferReversalPending,

					UpdatedAt: u.nowUTC(),
				},
			)
		if err != nil {
			return refund, err
		}

		if updated == nil {
			return refund,
				refunddom.ErrConflict
		}

		refund = *updated

	case refunddom.TransferReversalStatusPending:
		// Continue.

	default:
		return refund,
			refunddom.ErrInvalidTransferReversalStatus
	}

	result, reversalErr :=
		u.stripeTransferReversalGateway.CreateTransferReversal(
			ctx,
			CreateStripeTransferReversalInput{
				StripeTransferID: settlement.StripeTransferID,
				Amount:           refund.TransferReversalAmount,
				IdempotencyKey: itemRefundIdempotencyKey(
					"transfer-reversal",
					refund.ID,
				),
				OrderID:      refund.OrderID,
				PaymentID:    payment.PaymentID,
				SettlementID: settlement.ID,
				CompanyID:    refund.CompanyID,
				AccountID:    refund.AccountID,
			},
		)
	if reversalErr != nil {
		return u.persistTransferReversalFailure(
			ctx,
			refund,
			reversalErr,
		)
	}

	if result == nil {
		return u.persistTransferReversalFailure(
			ctx,
			refund,
			ErrItemRefundStripeTransferReversalResultEmpty,
		)
	}

	if result.StripeTransferReversalID == "" ||
		!strings.HasPrefix(
			result.StripeTransferReversalID,
			"trr_",
		) {
		return u.persistTransferReversalFailure(
			ctx,
			refund,
			ErrItemRefundStripeTransferReversalIDInvalid,
		)
	}

	now :=
		u.nowUTC()

	updated, err :=
		u.refundRepo.UpdateByID(
			ctx,
			refund.ID,
			refunddom.UpdateRefundInput{
				Operation:                refunddom.UpdateOperationMarkTransferReversalSucceeded,
				StripeTransferReversalID: result.StripeTransferReversalID,
				TransferReversedAt:       &now,
				UpdatedAt:                now,
			},
		)
	if err != nil {
		return refund,
			fmt.Errorf(
				"item refund: persist Stripe transfer reversal result: %w",
				err,
			)
	}

	if updated == nil {
		return refund,
			refunddom.ErrConflict
	}

	return *updated, nil
}

func (u *ItemRefundUsecase) persistTransferReversalFailure(
	ctx context.Context,
	refund refunddom.Refund,
	reversalErr error,
) (refunddom.Refund, error) {
	operation :=
		refunddom.UpdateOperationMarkTransferReversalFailed

	if isRetryableItemRefundError(
		reversalErr,
	) {
		operation =
			refunddom.UpdateOperationMarkTransferReversalFailedRetryable
	}

	updated, persistErr :=
		u.refundRepo.UpdateByID(
			ctx,
			refund.ID,
			refunddom.UpdateRefundInput{
				Operation: operation,

				UpdatedAt: u.nowUTC(),
			},
		)
	if persistErr != nil {
		return refund,
			fmt.Errorf(
				"item refund: Stripe transfer reversal failed: %w; persist failure state: %v",
				reversalErr,
				persistErr,
			)
	}

	if updated != nil {
		refund = *updated
	}

	return refund,
		fmt.Errorf(
			"item refund: create Stripe transfer reversal: %w",
			reversalErr,
		)
}

// ============================================================
// Settlement Resolution
// ============================================================

func (u *ItemRefundUsecase) resolveSettlement(
	ctx context.Context,
	payment paymentdom.Payment,
	targetItem orderdom.OrderItemSnapshot,
) (settlementdom.Settlement, error) {
	settlements, err :=
		u.settlementRepo.ListByPaymentID(
			ctx,
			payment.PaymentID,
		)
	if err != nil {
		return settlementdom.Settlement{}, err
	}

	var matched *settlementdom.Settlement

	for index := range settlements {
		settlement :=
			settlements[index]

		if settlement.AccountID !=
			targetItem.SellerSnapshot.AccountID {
			continue
		}

		if matched != nil {
			return settlementdom.Settlement{},
				ErrItemRefundSettlementDuplicate
		}

		copy :=
			settlement

		matched =
			&copy
	}

	if matched == nil {
		return settlementdom.Settlement{},
			ErrItemRefundSettlementNotFound
	}

	if matched.PaymentID !=
		payment.PaymentID ||
		matched.OrderID !=
			payment.PaymentID ||
		matched.CompanyID !=
			targetItem.SellerSnapshot.CompanyID ||
		matched.AccountID !=
			targetItem.SellerSnapshot.AccountID {
		return settlementdom.Settlement{},
			ErrItemRefundSettlementMismatch
	}

	if matched.Currency !=
		settlementdom.CurrencyJPY {
		return settlementdom.Settlement{},
			ErrItemRefundSettlementMismatch
	}

	if matched.StripeChargeID !=
		payment.StripeChargeID {
		return settlementdom.Settlement{},
			ErrItemRefundSettlementMismatch
	}

	if targetItem.SellerSnapshot.StripeAccountID != "" &&
		matched.StripeAccountID !=
			targetItem.SellerSnapshot.StripeAccountID {
		return settlementdom.Settlement{},
			ErrItemRefundSettlementMismatch
	}

	switch matched.Status {
	case settlementdom.StatusTransferred:
		return *matched, nil

	case settlementdom.StatusPending,
		settlementdom.StatusReady,
		settlementdom.StatusTransferring,
		settlementdom.StatusFailedRetryable:
		return *matched, nil

	case settlementdom.StatusFailed,
		settlementdom.StatusCanceled:
		if matched.StripeTransferID != "" ||
			matched.TransferredAt != nil {
			return settlementdom.Settlement{},
				ErrItemRefundSettlementMismatch
		}

		return *matched, nil

	case settlementdom.StatusReversed:
		return settlementdom.Settlement{},
			ErrItemRefundSettlementUnavailable

	default:
		return settlementdom.Settlement{},
			ErrItemRefundSettlementUnavailable
	}
}

// ============================================================
// Amount Calculation
// ============================================================

func (u *ItemRefundUsecase) resolveTransferReversalAmount(
	ctx context.Context,
	settlement settlementdom.Settlement,
	targetItem orderdom.OrderItemSnapshot,
	amountSummary orderdom.OrderItemRefundAmountSummary,
) (int, error) {
	switch settlement.Status {
	case settlementdom.StatusTransferred:
		if settlement.StripeTransferID == "" ||
			!strings.HasPrefix(
				settlement.StripeTransferID,
				"tr_",
			) {
			return 0,
				ErrItemRefundSettlementMismatch
		}

		return u.calculateTransferReversalAmount(
			ctx,
			targetItem,
			amountSummary,
		)

	case settlementdom.StatusFailed,
		settlementdom.StatusCanceled:
		if settlement.StripeTransferID != "" ||
			settlement.TransferredAt != nil {
			return 0,
				ErrItemRefundSettlementMismatch
		}

		// Seller Transfer never completed, so purchaser refund does not need
		// a seller-side Transfer Reversal.
		return 0, nil

	case settlementdom.StatusPending,
		settlementdom.StatusReady,
		settlementdom.StatusTransferring,
		settlementdom.StatusFailedRetryable:
		// These states may still result in a future seller Transfer.
		// Refunding the purchaser now would risk paying the seller afterward.
		return 0,
			ErrItemRefundSettlementNotTransferred

	case settlementdom.StatusReversed:
		return 0,
			ErrItemRefundSettlementUnavailable

	default:
		return 0,
			ErrItemRefundSettlementUnavailable
	}
}

func (u *ItemRefundUsecase) calculateTransferReversalAmount(
	ctx context.Context,
	targetItem orderdom.OrderItemSnapshot,
	amountSummary orderdom.OrderItemRefundAmountSummary,
) (int, error) {
	platformFeeAmount, err :=
		u.platformFeeCalculator.CalculatePlatformFee(
			ctx,
			settlementdom.PlatformFeeInput{
				CompanyID:            targetItem.SellerSnapshot.CompanyID,
				AccountID:            targetItem.SellerSnapshot.AccountID,
				MerchandiseAmount:    amountSummary.MerchandiseAmount,
				MerchandiseTaxAmount: amountSummary.MerchandiseTaxAmount,
				ShippingAmount:       0,
				ShippingTaxAmount:    0,
				GrossAmount:          amountSummary.RefundAmount,
			},
		)
	if err != nil {
		return 0, err
	}

	if platformFeeAmount < 0 ||
		platformFeeAmount >
			amountSummary.RefundAmount {
		return 0,
			ErrItemRefundInvalidPlatformFee
	}

	return amountSummary.RefundAmount -
			platformFeeAmount,
		nil
}

func (u *ItemRefundUsecase) validatePaymentRefundCapacity(
	ctx context.Context,
	paymentID string,
	currentRefundID string,
	newRefundAmount int,
	paymentAmount int,
) error {
	if newRefundAmount <= 0 ||
		paymentAmount <= 0 {
		return refunddom.ErrInvalidRefundAmount
	}

	refunds, err :=
		u.refundRepo.ListByPaymentID(
			ctx,
			paymentID,
		)
	if err != nil {
		return err
	}

	total :=
		0

	for _, refund := range refunds {
		if refund.ID ==
			currentRefundID {
			continue
		}

		if !reservesPurchaserRefundAmount(
			refund,
		) {
			continue
		}

		total +=
			refund.RefundAmount

		if total >
			paymentAmount {
			return ErrItemRefundPaymentAmountExceeded
		}
	}

	if total >
		paymentAmount-newRefundAmount {
		return ErrItemRefundPaymentAmountExceeded
	}

	return nil
}

func (u *ItemRefundUsecase) validateTransferReversalCapacity(
	ctx context.Context,
	settlement settlementdom.Settlement,
	currentRefundID string,
	newReversalAmount int,
) error {
	if newReversalAmount < 0 {
		return refunddom.ErrInvalidTransferReversalAmount
	}

	if newReversalAmount >
		settlement.TransferAmount {
		return ErrItemRefundTransferReversalAmountExceeded
	}

	refunds, err :=
		u.refundRepo.ListBySettlementID(
			ctx,
			settlement.ID,
		)
	if err != nil {
		return err
	}

	total :=
		0

	for _, refund := range refunds {
		if refund.ID ==
			currentRefundID {
			continue
		}

		if !reservesTransferReversalAmount(
			refund,
		) {
			continue
		}

		total +=
			refund.TransferReversalAmount

		if total >
			settlement.TransferAmount {
			return ErrItemRefundTransferReversalAmountExceeded
		}
	}

	if total >
		settlement.TransferAmount-
			newReversalAmount {
		return ErrItemRefundTransferReversalAmountExceeded
	}

	return nil
}

func reservesPurchaserRefundAmount(
	refund refunddom.Refund,
) bool {
	switch refund.Status {
	case refunddom.StatusCreated,
		refunddom.StatusPending,
		refunddom.StatusRequiresAction,
		refunddom.StatusSucceeded:
		return true

	case refunddom.StatusFailed,
		refunddom.StatusCanceled:
		return false

	default:
		return true
	}
}

func reservesTransferReversalAmount(
	refund refunddom.Refund,
) bool {
	if refund.TransferReversalAmount <= 0 {
		return false
	}

	switch refund.Status {
	case refunddom.StatusFailed,
		refunddom.StatusCanceled:
		return false

	default:
		return true
	}
}

// ============================================================
// Existing Refund Validation
// ============================================================

func validateExistingItemRefund(
	refund refunddom.Refund,
	in RefundOrderItemInput,
	order orderdom.Order,
	targetItem orderdom.OrderItemSnapshot,
	settlement settlementdom.Settlement,
	amountSummary orderdom.OrderItemRefundAmountSummary,
) error {
	if err := refund.Validate(); err != nil {
		return err
	}

	if refund.InquiryID !=
		in.InquiryID {
		return ErrItemRefundExistingRefundMismatch
	}

	if refund.OrderID !=
		order.ID ||
		refund.PaymentID !=
			order.ID ||
		refund.OrderItemIndex !=
			in.ItemIndex {
		return ErrItemRefundExistingRefundMismatch
	}

	if refund.CompanyID !=
		targetItem.SellerSnapshot.CompanyID ||
		refund.AccountID !=
			targetItem.SellerSnapshot.AccountID {
		return ErrItemRefundExistingRefundMismatch
	}

	if refund.SettlementID !=
		settlement.ID {
		return ErrItemRefundExistingRefundMismatch
	}

	if refund.MerchandiseAmount !=
		amountSummary.MerchandiseAmount ||
		refund.MerchandiseTaxAmount !=
			amountSummary.MerchandiseTaxAmount ||
		refund.RefundAmount !=
			amountSummary.RefundAmount {
		return ErrItemRefundExistingRefundMismatch
	}

	if refund.Currency !=
		refunddom.CurrencyJPY {
		return ErrItemRefundExistingRefundMismatch
	}

	if refund.TransferReversalAmount < 0 ||
		refund.TransferReversalAmount >
			settlement.TransferAmount {
		return ErrItemRefundExistingRefundMismatch
	}

	if (settlement.Status ==
		settlementdom.StatusFailed ||
		settlement.Status ==
			settlementdom.StatusCanceled) &&
		refund.TransferReversalAmount != 0 {
		return ErrItemRefundExistingRefundMismatch
	}

	return nil
}

// ============================================================
// Validation
// ============================================================

func (u *ItemRefundUsecase) validateConfigured() error {
	if u == nil ||
		u.orderReader == nil ||
		u.paymentReader == nil ||
		u.settlementRepo == nil ||
		u.refundRepo == nil ||
		u.platformFeeCalculator == nil ||
		u.stripeRefundGateway == nil ||
		u.stripeTransferReversalGateway == nil ||
		u.now == nil {
		return ErrItemRefundNotConfigured
	}

	return nil
}

// ============================================================
// Helpers
// ============================================================

type itemRefundRetryableError interface {
	Retryable() bool
}

func isRetryableItemRefundError(
	err error,
) bool {
	if err == nil {
		return false
	}

	var retryableError itemRefundRetryableError

	if errors.As(
		err,
		&retryableError,
	) {
		return retryableError.Retryable()
	}

	return false
}

func isRefundCreateConflict(
	err error,
) bool {
	return errors.Is(
		err,
		refunddom.ErrConflict,
	) ||
		errors.Is(
			err,
			refunddom.ErrDuplicateInquiry,
		) ||
		errors.Is(
			err,
			refunddom.ErrDuplicateOrderItem,
		)
}

func itemRefundIdempotencyKey(
	operation string,
	refundID string,
) string {
	value :=
		operation +
			"|" +
			refundID

	hash :=
		sha256.Sum256(
			[]byte(value),
		)

	return "amol_item_refund_" +
		operation +
		"_" +
		hex.EncodeToString(
			hash[:],
		)
}

func (u *ItemRefundUsecase) nowUTC() time.Time {
	return u.now().
		UTC()
}
