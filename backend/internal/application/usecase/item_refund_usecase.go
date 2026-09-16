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
// Refund amount, seller identity, item type and seller-side financial reference
// must always be derived from authoritative persisted state. They must never be
// accepted from the frontend.
type ItemRefundOrderReader interface {
	GetByID(ctx context.Context, id string) (orderdom.Order, error)
}

// ItemRefundPaymentReader provides the original succeeded Stripe Payment.
//
// Payment is read-only in this item-level flow.
//
// Item-level partial refund state is persisted in refund.Refund instead of
// Payment because Payment refund fields represent the full-payment refund
// lifecycle.
type ItemRefundPaymentReader interface {
	GetByPaymentID(ctx context.Context, paymentID string) (*paymentdom.Payment, error)
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
	ListByPaymentID(ctx context.Context, paymentID string) ([]settlementdom.Settlement, error)
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
	GetByID(ctx context.Context, receivableID string) (*salesreceivabledom.SalesReceivable, error)
	Cancel(ctx context.Context, receivableID string) (*salesreceivabledom.SalesReceivable, error)
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
//		-> cancel unpaid seller proceeds
//		-> BrandFeeSettlement
//		-> cancel untransferred Brand fee before purchaser Refund
//		-> purchaser Stripe Refund
//		-> reverse transferred Brand fee after Refund success
//
// Consumer resale seller proceeds never use Stripe Connect Transfer Reversal.
// The productBlueprint Brand fee is separate and may require a full Stripe
// Transfer Reversal after the purchaser Refund succeeds.
//
// Both unopened and opened returns use the same ReturnRefundSelection:
//
//	MerchandiseRefundAmount
//	RefundOutboundShipping
//	CoverReturnShipping
//
// MerchandiseRefundAmount is tax-inclusive. The backend automatically separates
// the selected amount into merchandise and consumption-tax components using the
// authoritative Order snapshot.
//
// The frontend never provides merchandise tax, outbound shipping amount,
// outbound shipping tax, return shipping amount or return shipping tax.
//
// Purchaser Stripe Refund amount:
//
//	merchandise refund including tax
//	+ outbound shipping including tax when RefundOutboundShipping=true
//
// Return shipping is recorded as additional seller-side burden when
// CoverReturnShipping=true. It is not included in the purchaser Stripe Refund
// because it was not part of the original Charge.
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
//  3. Validate ReturnRefundSelection.
//  4. Calculate exact merchandise/tax/shipping refund amounts from Order.
//  5. Load succeeded Payment.
//  6. Resolve seller-side financial state.
//
// List path:
//
//  7. Resolve primary-sale Settlement.
//  8. Calculate any required partial Stripe Transfer Reversal.
//  9. Load or create deterministic Refund aggregate.
//  10. Create or resume purchaser Stripe Refund.
//  11. If purchaser Refund succeeded, execute any required Transfer Reversal.
//
// Resale path:
//
//  7. Resolve the exact item-level SalesReceivable.
//  8. Require pending, available or already canceled.
//  9. Cancel pending/available SalesReceivable before purchaser Refund creation.
//  10. Prepare the exact item-level BrandFeeSettlement before purchaser Refund.
//  11. Load or create deterministic Refund aggregate referencing SalesReceivable.
//  12. Create or resume purchaser Stripe Refund.
//  13. After Refund success, reverse the Brand fee when it was transferred.
//
// reserved and paid SalesReceivables are intentionally rejected until
// BankPayout coordination and paid-receivable recovery are implemented.
//
// Order return completion and Inquiry resolution are intentionally NOT handled
// here. Return receipt orchestrators perform them only after:
//
//	Refund.IsFinanciallyCompleted() == true
type ItemRefundUsecase struct {
	orderReader                     ItemRefundOrderReader
	paymentReader                   ItemRefundPaymentReader
	settlementRepo                  ItemRefundSettlementRepository
	salesReceivableService          ItemRefundSalesReceivableService
	brandFeeSettlementRefundService BrandFeeSettlementRefundService
	refundRepo                      refunddom.RepositoryPort
	platformFeeCalculator           settlementdom.PlatformFeeCalculator
	stripeRefundGateway             applicationport.StripeRefundGateway
	stripeTransferReversalGateway   applicationport.StripeTransferReversalGateway
	now                             func() time.Time
}

type NewItemRefundUsecaseInput struct {
	OrderReader                     ItemRefundOrderReader
	PaymentReader                   ItemRefundPaymentReader
	SettlementRepository            ItemRefundSettlementRepository
	SalesReceivableService          ItemRefundSalesReceivableService
	BrandFeeSettlementRefundService BrandFeeSettlementRefundService
	RefundRepository                refunddom.RepositoryPort
	PlatformFeeCalculator           settlementdom.PlatformFeeCalculator
	StripeRefundGateway             applicationport.StripeRefundGateway
	StripeTransferReversalGateway   applicationport.StripeTransferReversalGateway
}

func NewItemRefundUsecase(in NewItemRefundUsecaseInput) *ItemRefundUsecase {
	return &ItemRefundUsecase{
		orderReader:                     in.OrderReader,
		paymentReader:                   in.PaymentReader,
		settlementRepo:                  in.SettlementRepository,
		salesReceivableService:          in.SalesReceivableService,
		brandFeeSettlementRefundService: in.BrandFeeSettlementRefundService,
		refundRepo:                      in.RefundRepository,
		platformFeeCalculator:           in.PlatformFeeCalculator,
		stripeRefundGateway:             in.StripeRefundGateway,
		stripeTransferReversalGateway:   in.StripeTransferReversalGateway,
		now:                             time.Now,
	}
}

// SetNowFunc replaces the current-time source for tests.
func (u *ItemRefundUsecase) SetNowFunc(now func() time.Time) {
	if u == nil || now == nil {
		return
	}

	u.now = now
}

// ============================================================
// Refund Order Item
// ============================================================

// itemRefundRequest is the internal normalized request boundary shared by
// unopened and opened return flows.
//
// Selection is the only caller-controlled financial input.
//
// MerchandiseRefundAmount is tax-inclusive. The backend derives the
// corresponding merchandise/tax allocation and every shipping amount from the
// authoritative Order snapshot.
//
// CompanyID is required for a primary List item but is not used as the resale
// seller identity.
type itemRefundRequest struct {
	InquiryID string
	OrderID   string
	ItemIndex int
	CompanyID string
	Selection refunddom.ReturnRefundSelection
}

// itemRefundAmountSummary contains the authoritative monetary result calculated
// from Order plus ReturnRefundSelection.
//
// Selection is retained so creation and idempotent retry validation can confirm
// that an existing Refund represents exactly the same seller decision.
type itemRefundAmountSummary struct {
	Selection refunddom.ReturnRefundSelection

	MerchandiseAmount    int
	MerchandiseTaxAmount int

	OutboundShippingAmount    int
	OutboundShippingTaxAmount int

	ReturnShippingAmount    int
	ReturnShippingTaxAmount int

	RefundAmount            int
	TotalSellerBurdenAmount int
}

// RefundOrderItem executes or resumes one item-level return refund.
//
// This method is shared by return_unopened and return_opened. Physical return
// state is validated by each return-receipt orchestrator before calling this
// financial usecase.
//
// List items validate CompanyID against the seller Company.
//
// Resale items ignore CompanyID for seller identity and instead validate the
// immutable resale SellerSnapshot and SalesReceivable.
//
// The operation is idempotent. ReturnRefundSelection is persisted in the
// deterministic Refund aggregate and a retry must use exactly the same
// selection.
func (u *ItemRefundUsecase) RefundOrderItem(
	ctx context.Context,
	in RefundOrderItemInput,
) (refunddom.Refund, error) {
	if err := refunddom.ValidateReturnRefundSelection(in.Selection); err != nil {
		return refunddom.Refund{}, err
	}

	return u.refundOrderItem(
		ctx,
		itemRefundRequest{
			InquiryID: in.InquiryID,
			OrderID:   in.OrderID,
			ItemIndex: in.ItemIndex,
			CompanyID: in.CompanyID,
			Selection: in.Selection,
		},
	)
}
