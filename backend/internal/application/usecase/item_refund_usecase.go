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
	salesreceivabledom "narratives/internal/domain/salesReceivable"
	settlementdom "narratives/internal/domain/settlement"
)

// ============================================================
// Ports
// ============================================================

// ItemRefundOrderReader provides the authoritative Order snapshot required for
// one item-level refund.
//
// Refund amount, seller identity, item type, and seller-side financial reference
// must always be derived from authoritative persisted state. They must never be
// accepted from the frontend.
type ItemRefundOrderReader interface {
	GetByID(
		ctx context.Context,
		id string,
	) (orderdom.Order, error)
}

// ItemRefundPaymentReader provides the original succeeded Stripe Payment.
//
// Payment is read-only in this item-level flow.
//
// Item-level partial refund state is persisted in refund.Refund instead of
// Payment because Payment refund fields represent the full-payment refund
// lifecycle.
type ItemRefundPaymentReader interface {
	GetByPaymentID(
		ctx context.Context,
		paymentID string,
	) (*paymentdom.Payment, error)
}

// ItemRefundSettlementRepository provides primary List-sale Settlement state.
//
// One Settlement may contain multiple List Order items belonging to the same
// Account seller. Therefore one item return may perform only a partial Stripe
// Transfer Reversal.
//
// Each partial reversal is persisted independently in refund.Refund.
//
// Consumer resale items never use this repository.
type ItemRefundSettlementRepository interface {
	ListByPaymentID(
		ctx context.Context,
		paymentID string,
	) ([]settlementdom.Settlement, error)
}

// ItemRefundSalesReceivableService provides the item-level resale financial
// state required by a consumer resale Refund.
//
// One resale Order item maps to exactly one SalesReceivable:
//
//	NewID(PaymentID, OrderItemIndex)
//
// Before the purchaser Stripe Refund is created:
//
//   - pending   -> canceled
//   - available -> canceled
//   - canceled  -> accepted as an idempotent retry
//   - reserved  -> rejected until BankPayout coordination is implemented
//   - paid      -> rejected until recovery/adjustment handling is implemented
//
// The Refund flow must never rewrite a paid SalesReceivable.
type ItemRefundSalesReceivableService interface {
	GetByID(
		ctx context.Context,
		receivableID string,
	) (*salesreceivabledom.SalesReceivable, error)

	Cancel(
		ctx context.Context,
		receivableID string,
	) (*salesreceivabledom.SalesReceivable, error)
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

	ErrItemRefundSalesReceivableNotConfigured = errors.New(
		"item refund: sales receivable service is not configured",
	)

	ErrItemRefundSalesReceivableNotFound = errors.New(
		"item refund: resale sales receivable is not found",
	)

	ErrItemRefundSalesReceivableMismatch = errors.New(
		"item refund: sales receivable does not match order item",
	)

	ErrItemRefundSalesReceivableUnavailable = errors.New(
		"item refund: sales receivable cannot be used for item refund",
	)

	ErrItemRefundSalesReceivableReserved = errors.New(
		"item refund: sales receivable is reserved for bank payout",
	)

	ErrItemRefundSalesReceivablePaid = errors.New(
		"item refund: sales receivable has already been paid",
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

// ItemRefundUsecase coordinates one item-level purchaser Stripe Refund and the
// corresponding seller-side financial state.
//
// Primary List sale:
//
//	Order item
//		-> Settlement
//		-> purchaser Stripe Refund
//		-> optional partial Stripe Transfer Reversal
//
// Consumer resale:
//
//	Order item
//		-> SalesReceivable
//		-> cancel unpaid receivable
//		-> purchaser Stripe Refund
//
// Consumer resale never uses Stripe Connect Transfer Reversal.
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
// Return shipping is recorded as additional seller-side burden in refund.Refund
// but is not included in the purchaser Stripe Refund because it was not part of
// the original Charge.
//
// For a transferred primary List Settlement, seller-side reversal amount is:
//
//	purchaser Stripe Refund amount
//	- platform fee attributable to the refunded components
//
// Resale TransferReversalAmount is always zero.
//
// Common execution order:
//
//  1. Load authoritative Order.
//  2. Validate the target Order item.
//  3. Calculate the exact refund amount from the Order snapshot.
//  4. Load succeeded Payment.
//  5. Resolve seller-side financial state.
//
// List path:
//
//  6. Resolve primary-sale Settlement.
//  7. Calculate any required partial Stripe Transfer Reversal.
//  8. Load or create deterministic Refund aggregate.
//  9. Create or resume purchaser Stripe Refund.
//  10. If purchaser Refund succeeded, execute any required Transfer Reversal.
//
// Resale path:
//
//  6. Resolve the exact item-level SalesReceivable.
//  7. Require pending, available, or already canceled.
//  8. Cancel pending/available SalesReceivable before purchaser Refund creation.
//  9. Load or create deterministic Refund aggregate referencing SalesReceivable.
//  10. Create or resume purchaser Stripe Refund.
//  11. No Stripe Transfer Reversal is performed.
//
// reserved and paid SalesReceivables are intentionally rejected until
// BankPayout coordination and paid-receivable recovery are implemented.
//
// Order return completion and Inquiry resolution are intentionally NOT handled
// here. Return receipt orchestrators perform them only after:
//
//	Refund.IsFinanciallyCompleted() == true
type ItemRefundUsecase struct {
	orderReader ItemRefundOrderReader

	paymentReader ItemRefundPaymentReader

	settlementRepo ItemRefundSettlementRepository

	salesReceivableService ItemRefundSalesReceivableService

	refundRepo refunddom.RepositoryPort

	platformFeeCalculator settlementdom.PlatformFeeCalculator

	stripeRefundGateway applicationport.StripeRefundGateway

	stripeTransferReversalGateway applicationport.StripeTransferReversalGateway

	now func() time.Time
}

type NewItemRefundUsecaseInput struct {
	OrderReader ItemRefundOrderReader

	PaymentReader ItemRefundPaymentReader

	SettlementRepository ItemRefundSettlementRepository

	SalesReceivableService ItemRefundSalesReceivableService

	RefundRepository refunddom.RepositoryPort

	PlatformFeeCalculator settlementdom.PlatformFeeCalculator

	StripeRefundGateway applicationport.StripeRefundGateway

	StripeTransferReversalGateway applicationport.StripeTransferReversalGateway
}

func NewItemRefundUsecase(
	in NewItemRefundUsecaseInput,
) *ItemRefundUsecase {
	return &ItemRefundUsecase{
		orderReader:                   in.OrderReader,
		paymentReader:                 in.PaymentReader,
		settlementRepo:                in.SettlementRepository,
		salesReceivableService:        in.SalesReceivableService,
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
//
// CompanyID remains present because the current Console return-receipt endpoint
// supplies it. It is authoritative only for primary List items. Consumer resale
// seller identity is resolved from the Order SellerSnapshot instead.
type RefundOpenedReturnItemInput struct {
	InquiryID string

	OrderID   string
	ItemIndex int

	CompanyID string

	Policy refunddom.OpenedReturnRefundPolicy
}

// itemRefundRequest is the internal normalized request boundary shared by
// unopened and opened return flows.
//
// CompanyID is required for a primary List item but is not used as the resale
// seller identity.
type itemRefundRequest struct {
	InquiryID string

	OrderID   string
	ItemIndex int

	CompanyID string

	Policy refunddom.OpenedReturnRefundPolicy
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
//
// List items validate CompanyID against the seller Company.
//
// Resale items ignore CompanyID for seller identity and instead validate the
// immutable resale SellerSnapshot and SalesReceivable.
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
//
// List items validate CompanyID against the seller Company.
//
// Resale items resolve seller identity from the authoritative Order snapshot and
// SalesReceivable.
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
