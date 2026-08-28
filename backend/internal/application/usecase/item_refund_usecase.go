// backend/internal/application/usecase/item_refund_usecase.go
package usecase

import (
	"context"
	"errors"
	"time"

	applicationport "narratives/internal/application/port"
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
// Unopened returns use the existing merchandise-only calculation:
//
//	merchandise amount
//	+ merchandise consumption tax
//
// Opened returns require an explicit OpenedReturnRefundPolicy and may include:
//
//	merchandise amount
//	+ merchandise consumption tax
//	+ outbound shipping amount
//	+ outbound shipping consumption tax
//
// Return shipping is recorded as additional company burden in refund.Refund but
// is not included in the purchaser Stripe Refund because it was not part of the
// original Charge.
//
// Seller-side reversal amount is:
//
//	purchaser Stripe Refund amount
//	- platform fee attributable to the refunded components
//
// Execution order:
//
//  1. Load authoritative Order.
//  2. Calculate the exact refund amount from the Order snapshot.
//  3. Load succeeded Payment.
//  4. Resolve seller Settlement.
//  5. Load or create deterministic Refund aggregate.
//  6. Create Stripe purchaser Refund.
//  7. Persist Stripe Refund result.
//  8. If Refund succeeded, create seller partial Transfer Reversal.
//  9. Persist Transfer Reversal result.
//
// Order return completion and Inquiry resolution are intentionally NOT handled
// here. Return receipt orchestrators perform them only after:
//
//	Refund.IsFinanciallyCompleted() == true
type ItemRefundUsecase struct {
	orderReader                   ItemRefundOrderReader
	paymentReader                 ItemRefundPaymentReader
	settlementRepo                ItemRefundSettlementRepository
	refundRepo                    refunddom.RepositoryPort
	platformFeeCalculator         settlementdom.PlatformFeeCalculator
	stripeRefundGateway           applicationport.StripeRefundGateway
	stripeTransferReversalGateway applicationport.StripeTransferReversalGateway
	now                           func() time.Time
}

type NewItemRefundUsecaseInput struct {
	OrderReader                   ItemRefundOrderReader
	PaymentReader                 ItemRefundPaymentReader
	SettlementRepository          ItemRefundSettlementRepository
	RefundRepository              refunddom.RepositoryPort
	PlatformFeeCalculator         settlementdom.PlatformFeeCalculator
	StripeRefundGateway           applicationport.StripeRefundGateway
	StripeTransferReversalGateway applicationport.StripeTransferReversalGateway
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
	if u == nil || now == nil {
		return
	}

	u.now = now
}

// ============================================================
// Refund Order Item
// ============================================================

// RefundOpenedReturnItemInput identifies one opened-return financial refund.
//
// Only Policy is accepted in addition to identity fields. All monetary amounts
// are calculated from the authoritative Order snapshot.
type RefundOpenedReturnItemInput struct {
	InquiryID string
	OrderID   string
	ItemIndex int
	CompanyID string
	Policy    refunddom.OpenedReturnRefundPolicy
}

type itemRefundRequest struct {
	InquiryID string
	OrderID   string
	ItemIndex int
	CompanyID string
	Policy    refunddom.OpenedReturnRefundPolicy
}

type itemRefundAmountSummary struct {
	Policy refunddom.OpenedReturnRefundPolicy

	MerchandiseAmount    int
	MerchandiseTaxAmount int

	OutboundShippingAmount    int
	OutboundShippingTaxAmount int

	ReturnShippingAmount    int
	ReturnShippingTaxAmount int

	RefundAmount int
}

// RefundOrderItem executes or resumes the existing unopened-return refund.
func (u *ItemRefundUsecase) RefundOrderItem(
	ctx context.Context,
	in RefundOrderItemInput,
) (refunddom.Refund, error) {
	return u.refundOrderItem(
		ctx,
		itemRefundRequest{
			InquiryID: in.InquiryID,
			OrderID:   in.OrderID,
			ItemIndex: in.ItemIndex,
			CompanyID: in.CompanyID,
		},
	)
}

// RefundOpenedReturnOrderItem executes or resumes one opened-return refund.
//
// The operation is idempotent. Policy is persisted in the deterministic Refund
// aggregate and a retry must use the same Policy.
func (u *ItemRefundUsecase) RefundOpenedReturnOrderItem(
	ctx context.Context,
	in RefundOpenedReturnItemInput,
) (refunddom.Refund, error) {
	if err := refunddom.ValidateOpenedReturnRefundPolicy(in.Policy); err != nil {
		return refunddom.Refund{}, err
	}

	return u.refundOrderItem(
		ctx,
		itemRefundRequest{
			InquiryID: in.InquiryID,
			OrderID:   in.OrderID,
			ItemIndex: in.ItemIndex,
			CompanyID: in.CompanyID,
			Policy:    in.Policy,
		},
	)
}
