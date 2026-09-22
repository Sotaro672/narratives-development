// backend/internal/application/usecase/resale_trade_return_refund_usecase.go
package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	orderdom "narratives/internal/domain/order"
	refunddom "narratives/internal/domain/refund"
	salesreceivabledom "narratives/internal/domain/salesReceivable"
	tradedom "narratives/internal/domain/trade"
)

var (
	ErrResaleTradeReturnRefundNotConfigured = errors.New(
		"resale trade return refund: usecase is not configured",
	)
	ErrResaleTradeReturnRefundInvalidBuyer = errors.New(
		"resale trade return refund: invalid buyer",
	)
	ErrResaleTradeReturnRefundTradeMismatch = errors.New(
		"resale trade return refund: trade does not match order item",
	)
	ErrResaleTradeReturnRefundOrderNotPaid = errors.New(
		"resale trade return refund: order is not paid",
	)
	ErrResaleTradeReturnRefundNotEligible = errors.New(
		"resale trade return refund: trade is not eligible for return refund",
	)
	ErrResaleTradeReturnRefundAgreementNotReady = errors.New(
		"resale trade return refund: return agreement is not ready for refund",
	)
	ErrResaleTradeReturnRefundPhysicalReturnRequired = errors.New(
		"resale trade return refund: physical return is required",
	)
	ErrResaleTradeReturnRefundDisputed = errors.New(
		"resale trade return refund: return is disputed",
	)
	ErrResaleTradeReturnRefundAmountInvalid = errors.New(
		"resale trade return refund: proposal refund amount is invalid",
	)
	ErrResaleTradeReturnRefundMismatch = errors.New(
		"resale trade return refund: refund does not match agreed return",
	)
	ErrResaleTradeReturnRefundAgreementCompletionMismatch = errors.New(
		"resale trade return refund: return agreement completion mismatch",
	)
)

// ResaleTradeReturnRefundOrderService is the minimum Order application service
// required for an accepted resale Trade return that does not require physical
// return.
//
// Order remains authoritative for purchaser identity, paid state, item identity
// and the immutable resale seller snapshot.
type ResaleTradeReturnRefundOrderService interface {
	GetByID(
		ctx context.Context,
		id string,
	) (orderdom.Order, error)
}

// ResaleTradeReturnRefundItemRefundService is the financial boundary used for
// an accepted resale Trade return that does not require physical return.
//
// RefundOrderItem must remain idempotent. Financial conditions are derived from
// the accepted ReturnProposal and are never accepted again from the frontend.
type ResaleTradeReturnRefundItemRefundService interface {
	RefundOrderItem(
		ctx context.Context,
		in RefundOrderItemInput,
	) (refunddom.Refund, error)
}

// ResaleTradeReturnRefundCompletionNotifier is the minimum notification
// contract required after the item-level refund has completed financially.
type ResaleTradeReturnRefundCompletionNotifier interface {
	EnsureDelivery(
		ctx context.Context,
		in EnsureRefundCompletionNotificationInput,
	) (refunddom.CompletionNotificationDelivery, error)
}

// ResaleTradeReturnRefundUsecase coordinates the refund flow for an accepted
// Avatar-to-Avatar Resale Trade return where physical return is not required.
//
// Current execution:
//
//	buyer accepted proposal
//	-> ReturnAgreement agreed
//	-> verify returnRequirement=not_required
//	-> ReturnAgreement refund_processing
//	-> ItemRefundUsecase using accepted Proposal.RefundAmount
//	-> refund completion notification
//	-> ReturnAgreement completed
//
// This usecase intentionally does not depend on:
//   - ReturnShipment
//   - carrier/PUDO/Yamato state
//   - Order.IsReturnRequested
//   - Order.ReturnRequestKind
//   - purchaser Inquiry state
//   - frontend-selected refund conditions
//
// Current shipping-refund policy:
//   - RefundOutboundShipping=false
//   - CoverReturnShipping=false
//
// If shipping conditions become negotiable, they must be stored in
// ReturnProposal before buyer acceptance rather than accepted by this usecase.
type ResaleTradeReturnRefundUsecase struct {
	tradeRepo           tradedom.Repository
	returnAgreementRepo tradedom.ReturnAgreementRepository
	orderService        ResaleTradeReturnRefundOrderService
	itemRefundService   ResaleTradeReturnRefundItemRefundService

	refundCompletionNotifier ResaleTradeReturnRefundCompletionNotifier

	now func() time.Time
}

type NewResaleTradeReturnRefundUsecaseInput struct {
	TradeRepository           tradedom.Repository
	ReturnAgreementRepository tradedom.ReturnAgreementRepository
	OrderService              ResaleTradeReturnRefundOrderService
	ItemRefundService         ResaleTradeReturnRefundItemRefundService
	RefundCompletionNotifier  ResaleTradeReturnRefundCompletionNotifier
}

func NewResaleTradeReturnRefundUsecase(
	in NewResaleTradeReturnRefundUsecaseInput,
) *ResaleTradeReturnRefundUsecase {
	return &ResaleTradeReturnRefundUsecase{
		tradeRepo:                in.TradeRepository,
		returnAgreementRepo:      in.ReturnAgreementRepository,
		orderService:             in.OrderService,
		itemRefundService:        in.ItemRefundService,
		refundCompletionNotifier: in.RefundCompletionNotifier,
		now:                      time.Now,
	}
}

// SetNowFunc replaces the server clock for tests.
func (uc *ResaleTradeReturnRefundUsecase) SetNowFunc(
	now func() time.Time,
) {
	if uc == nil || now == nil {
		return
	}

	uc.now = now
}

// RefundResaleTradeReturnInput identifies one buyer-authorized refund attempt.
//
// TradeID identifies the Trade whose accepted proposal is authoritative.
// BuyerAvatarID must come from authenticated Mall AvatarContext and must never
// be trusted from arbitrary request-body input.
type RefundResaleTradeReturnInput struct {
	TradeID       string
	BuyerAvatarID string
}

// ResaleTradeReturnRefundResult represents one no-physical-return refund
// attempt.
//
// ReturnAgreement is authoritative for the return lifecycle. The legacy Order
// return-request/completion fields are not mutated by this usecase.
type ResaleTradeReturnRefundResult struct {
	Trade     tradedom.Trade
	Agreement tradedom.ReturnAgreement
	Proposal  tradedom.ReturnProposal
	Order     orderdom.Order
	Refund    refunddom.Refund

	FinanciallyCompleted bool
	ReturnCompleted      bool
	NotificationEnsured  bool
	AlreadyCompleted     bool
}

// Refund executes or resumes the refund for an accepted proposal where the
// seller explicitly agreed that the item does not need to be physically
// returned.
//
// The method is retry-safe:
//
//	agreed
//	  -> refund_processing
//
// A later retry resumes from refund_processing using the same immutable
// ReturnProposal conditions.
//
// ItemRefundUsecase owns purchaser Stripe Refund and seller-side financial
// idempotency. This usecase marks ReturnAgreement completed only after those
// financial operations and the purchaser refund-completion notification have
// succeeded.
func (uc *ResaleTradeReturnRefundUsecase) Refund(
	ctx context.Context,
	in RefundResaleTradeReturnInput,
) (ResaleTradeReturnRefundResult, error) {
	if err := uc.validateConfigured(); err != nil {
		return ResaleTradeReturnRefundResult{}, err
	}

	tradeID := strings.TrimSpace(in.TradeID)
	if tradeID == "" {
		return ResaleTradeReturnRefundResult{},
			tradedom.ErrInvalidID
	}

	buyerAvatarID := strings.TrimSpace(in.BuyerAvatarID)
	if buyerAvatarID == "" {
		return ResaleTradeReturnRefundResult{},
			ErrResaleTradeReturnRefundInvalidBuyer
	}

	trade, err := uc.tradeRepo.GetByID(
		ctx,
		tradeID,
	)
	if err != nil {
		return ResaleTradeReturnRefundResult{}, err
	}

	if trade.ID != tradeID ||
		trade.SellerType != tradedom.SellerTypeAvatar ||
		strings.TrimSpace(trade.SellerAvatarID) == "" ||
		strings.TrimSpace(trade.BuyerAvatarID) == "" ||
		trade.BuyerAvatarID != buyerAvatarID {
		return ResaleTradeReturnRefundResult{},
			tradedom.ErrNotFound
	}

	if strings.TrimSpace(trade.OrderID) == "" ||
		trade.OrderItemIndex < 0 {
		return ResaleTradeReturnRefundResult{
			Trade: trade,
		}, ErrResaleTradeReturnRefundTradeMismatch
	}

	order, err := uc.orderService.GetByID(
		ctx,
		trade.OrderID,
	)
	if err != nil {
		return ResaleTradeReturnRefundResult{
			Trade: trade,
		}, err
	}

	result := ResaleTradeReturnRefundResult{
		Trade: trade,
		Order: order,
	}

	if order.ID != trade.OrderID ||
		strings.TrimSpace(order.AvatarID) == "" ||
		order.AvatarID != trade.BuyerAvatarID ||
		order.AvatarID != buyerAvatarID ||
		trade.OrderItemIndex >= len(order.Items) {
		return result,
			ErrResaleTradeReturnRefundTradeMismatch
	}

	itemIndex := trade.OrderItemIndex
	targetItem := order.Items[itemIndex]

	if err := validateResaleTradeReturnRefundTarget(
		trade,
		targetItem,
	); err != nil {
		return result, err
	}

	if !order.Paid {
		return result,
			ErrResaleTradeReturnRefundOrderNotPaid
	}

	if targetItem.IsCancelled ||
		!targetItem.IsDispatched ||
		targetItem.Transferred {
		return result,
			ErrResaleTradeReturnRefundNotEligible
	}

	agreement, err :=
		uc.returnAgreementRepo.GetByTradeID(
			ctx,
			tradeID,
		)
	if err != nil {
		return result, err
	}

	proposal, err :=
		validateResaleTradeReturnRefundAgreement(
			agreement,
			trade,
			order,
			itemIndex,
		)
	if err != nil {
		return result, err
	}

	result.Agreement = agreement
	result.Proposal = proposal

	selection := buildResaleTradeReturnRefundSelection(
		proposal,
	)
	if err := refunddom.ValidateReturnRefundSelection(
		selection,
	); err != nil {
		return result, err
	}

	if agreement.Status == tradedom.ReturnStatusCompleted {
		result.FinanciallyCompleted = true
		result.ReturnCompleted = true
		result.NotificationEnsured = true
		result.AlreadyCompleted = true
		return result, nil
	}

	agreement, err = uc.ensureRefundProcessing(
		ctx,
		agreement,
	)
	if err != nil {
		return result, err
	}

	result.Agreement = agreement

	// ItemRefundUsecase still persists an InquiryID-shaped correlation field.
	// The deterministic legacy return ID is used only as a stable financial
	// correlation key. No Inquiry is loaded or used for authorization here.
	refundSourceID := returnInquiryID(
		order.ID,
		itemIndex,
	)

	refund, err := uc.itemRefundService.RefundOrderItem(
		ctx,
		RefundOrderItemInput{
			InquiryID: refundSourceID,
			OrderID:   order.ID,
			ItemIndex: itemIndex,
			CompanyID: "",
			Selection: selection,
		},
	)
	if err != nil {
		result.Refund = refund
		return result, err
	}

	if err := validateResaleTradeReturnRefund(
		refundSourceID,
		order,
		itemIndex,
		selection,
		refund,
	); err != nil {
		result.Refund = refund
		return result, err
	}

	result.Refund = refund
	result.FinanciallyCompleted =
		refund.IsFinanciallyCompleted()

	if !result.FinanciallyCompleted {
		return result, nil
	}

	_, err =
		uc.refundCompletionNotifier.EnsureDelivery(
			ctx,
			EnsureRefundCompletionNotificationInput{
				PaymentID:      refund.PaymentID,
				OrderID:        refund.OrderID,
				UserID:         order.UserID,
				StripeRefundID: refund.StripeRefundID,
				RefundedAmount: refund.RefundAmount,
			},
		)
	if err != nil {
		return result, err
	}

	result.NotificationEnsured = true

	agreement, err = uc.ensureCompleted(
		ctx,
		agreement,
	)
	if err != nil {
		return result, err
	}

	result.Agreement = agreement
	result.ReturnCompleted =
		agreement.Status ==
			tradedom.ReturnStatusCompleted

	if !result.ReturnCompleted ||
		agreement.CompletedAt == nil ||
		agreement.CompletedAt.IsZero() {
		return result,
			ErrResaleTradeReturnRefundAgreementCompletionMismatch
	}

	return result, nil
}

func (uc *ResaleTradeReturnRefundUsecase) ensureRefundProcessing(
	ctx context.Context,
	agreement tradedom.ReturnAgreement,
) (tradedom.ReturnAgreement, error) {
	switch agreement.Status {
	case tradedom.ReturnStatusAgreed:
		if err := agreement.MarkRefundProcessing(
			uc.nowUTC(),
		); err != nil {
			return agreement, err
		}

		return uc.returnAgreementRepo.Update(
			ctx,
			agreement.TradeID,
			agreement,
		)

	case tradedom.ReturnStatusRefundProcessing,
		tradedom.ReturnStatusCompleted:
		return agreement, nil

	case tradedom.ReturnStatusDisputed:
		return agreement,
			ErrResaleTradeReturnRefundDisputed

	default:
		return agreement,
			ErrResaleTradeReturnRefundAgreementNotReady
	}
}

func (uc *ResaleTradeReturnRefundUsecase) ensureCompleted(
	ctx context.Context,
	agreement tradedom.ReturnAgreement,
) (tradedom.ReturnAgreement, error) {
	switch agreement.Status {
	case tradedom.ReturnStatusRefundProcessing:
		if err := agreement.Complete(
			uc.nowUTC(),
		); err != nil {
			return agreement, err
		}

		return uc.returnAgreementRepo.Update(
			ctx,
			agreement.TradeID,
			agreement,
		)

	case tradedom.ReturnStatusCompleted:
		return agreement, nil

	case tradedom.ReturnStatusDisputed:
		return agreement,
			ErrResaleTradeReturnRefundDisputed

	default:
		return agreement,
			ErrResaleTradeReturnRefundAgreementNotReady
	}
}

func (uc *ResaleTradeReturnRefundUsecase) validateConfigured() error {
	if uc == nil ||
		uc.tradeRepo == nil ||
		uc.returnAgreementRepo == nil ||
		uc.orderService == nil ||
		uc.itemRefundService == nil ||
		uc.refundCompletionNotifier == nil ||
		uc.now == nil {
		return ErrResaleTradeReturnRefundNotConfigured
	}

	return nil
}

func (uc *ResaleTradeReturnRefundUsecase) nowUTC() time.Time {
	return uc.now().UTC()
}

func buildResaleTradeReturnRefundSelection(
	proposal tradedom.ReturnProposal,
) refunddom.ReturnRefundSelection {
	return refunddom.ReturnRefundSelection{
		MerchandiseRefundAmount: proposal.RefundAmount,
		RefundOutboundShipping:  false,
		CoverReturnShipping:     false,
	}
}

func validateResaleTradeReturnRefundAgreement(
	agreement tradedom.ReturnAgreement,
	trade tradedom.Trade,
	order orderdom.Order,
	itemIndex int,
) (tradedom.ReturnProposal, error) {
	if agreement.ID != trade.ID ||
		agreement.TradeID != trade.ID {
		return tradedom.ReturnProposal{},
			tradedom.ErrReturnAgreementConflict
	}

	if agreement.Proposal == nil {
		return tradedom.ReturnProposal{},
			tradedom.ErrReturnProposalNotFound
	}

	proposal := *agreement.Proposal

	if strings.TrimSpace(proposal.ID) == "" ||
		proposal.Agreement !=
			tradedom.ReturnProposalAgreementAgree ||
		proposal.RejectedAt != nil ||
		agreement.AgreedAt == nil ||
		agreement.AgreedAt.IsZero() {
		return tradedom.ReturnProposal{},
			ErrResaleTradeReturnRefundAgreementNotReady
	}

	if proposal.ReturnRequirement ==
		tradedom.ReturnRequirementRequired {
		return tradedom.ReturnProposal{},
			ErrResaleTradeReturnRefundPhysicalReturnRequired
	}

	if proposal.ReturnRequirement !=
		tradedom.ReturnRequirementNotRequired {
		return tradedom.ReturnProposal{},
			tradedom.ErrInvalidReturnRequirement
	}

	if proposal.RefundAmount <= 0 {
		return tradedom.ReturnProposal{},
			tradedom.ErrInvalidReturnRefundAmount
	}

	if itemIndex < 0 ||
		itemIndex >= len(order.Items) {
		return tradedom.ReturnProposal{},
			ErrResaleTradeReturnRefundTradeMismatch
	}

	refundAmountSummary, err :=
		refunddom.CalculateOrderItemRefundAmount(
			order,
			itemIndex,
		)
	if err != nil {
		return tradedom.ReturnProposal{}, err
	}

	if proposal.RefundAmount >
		refundAmountSummary.RefundAmount {
		return tradedom.ReturnProposal{},
			ErrResaleTradeReturnRefundAmountInvalid
	}

	switch agreement.Status {
	case tradedom.ReturnStatusAgreed,
		tradedom.ReturnStatusRefundProcessing,
		tradedom.ReturnStatusCompleted:
		return proposal, nil

	case tradedom.ReturnStatusDisputed:
		return tradedom.ReturnProposal{},
			ErrResaleTradeReturnRefundDisputed

	default:
		return tradedom.ReturnProposal{},
			ErrResaleTradeReturnRefundAgreementNotReady
	}
}

// validateResaleTradeReturnRefundTarget confirms that the authoritative Order
// item is exactly the consumer-resale item represented by the Trade.
func validateResaleTradeReturnRefundTarget(
	trade tradedom.Trade,
	targetItem orderdom.OrderItemSnapshot,
) error {
	if targetItem.Type != orderdom.OrderItemTypeResale ||
		strings.TrimSpace(targetItem.ResaleID) == "" {
		return ErrResaleTradeReturnRefundTradeMismatch
	}

	seller := targetItem.SellerSnapshot

	if strings.TrimSpace(seller.AvatarID) == "" ||
		seller.AvatarID != trade.SellerAvatarID {
		return ErrResaleTradeReturnRefundTradeMismatch
	}

	if strings.TrimSpace(seller.UserID) == "" ||
		strings.TrimSpace(seller.PayoutAccountID) == "" ||
		seller.PayoutAccountID != seller.UserID ||
		seller.BrandID != "" ||
		seller.CompanyID != "" ||
		seller.AccountID != "" ||
		seller.StripeAccountID != "" {
		return ErrResaleTradeReturnRefundTradeMismatch
	}

	return nil
}

// validateResaleTradeReturnRefund validates that the Refund persisted by
// ItemRefundUsecase represents exactly the same consumer-resale return and the
// immutable financial conditions accepted in ReturnProposal.
func validateResaleTradeReturnRefund(
	refundSourceID string,
	order orderdom.Order,
	itemIndex int,
	selection refunddom.ReturnRefundSelection,
	refund refunddom.Refund,
) error {
	if err := refund.Validate(); err != nil {
		return fmt.Errorf(
			"%w: %v",
			ErrResaleTradeReturnRefundMismatch,
			err,
		)
	}

	if strings.TrimSpace(refundSourceID) == "" ||
		refund.InquiryID != refundSourceID ||
		refund.OrderID != order.ID ||
		refund.PaymentID != order.ID ||
		refund.OrderItemIndex != itemIndex {
		return ErrResaleTradeReturnRefundMismatch
	}

	if itemIndex < 0 ||
		itemIndex >= len(order.Items) {
		return ErrResaleTradeReturnRefundMismatch
	}

	targetItem := order.Items[itemIndex]
	if targetItem.Type != orderdom.OrderItemTypeResale {
		return ErrResaleTradeReturnRefundMismatch
	}

	seller := targetItem.SellerSnapshot

	if refund.SellerType != refunddom.SellerTypeResale ||
		refund.CompanyID != "" ||
		refund.AccountID != "" ||
		refund.StripeAccountID != "" ||
		refund.AvatarID != seller.AvatarID ||
		refund.UserID != seller.UserID ||
		refund.PayoutAccountID != seller.PayoutAccountID ||
		refund.SettlementID != "" {
		return ErrResaleTradeReturnRefundMismatch
	}

	expectedSalesReceivableID, err :=
		salesreceivabledom.NewID(
			order.ID,
			itemIndex,
		)
	if err != nil ||
		refund.SalesReceivableID !=
			expectedSalesReceivableID {
		return ErrResaleTradeReturnRefundMismatch
	}

	if refund.Currency != refunddom.CurrencyJPY {
		return ErrResaleTradeReturnRefundMismatch
	}

	if err := refunddom.ValidateReturnRefundSelection(
		selection,
	); err != nil {
		return err
	}

	expectedAmount, err :=
		refunddom.CalculateReturnRefundAmount(
			order,
			itemIndex,
			selection,
		)
	if err != nil {
		return err
	}

	if refund.RequestedMerchandiseRefundAmount !=
		selection.MerchandiseRefundAmount ||
		refund.RefundOutboundShipping !=
			selection.RefundOutboundShipping ||
		refund.CoverReturnShipping !=
			selection.CoverReturnShipping ||
		refund.MerchandiseAmount !=
			expectedAmount.MerchandiseAmount ||
		refund.MerchandiseTaxAmount !=
			expectedAmount.MerchandiseTaxAmount ||
		refund.OutboundShippingAmount !=
			expectedAmount.OutboundShippingAmount ||
		refund.OutboundShippingTaxAmount !=
			expectedAmount.OutboundShippingTaxAmount ||
		refund.ReturnShippingAmount !=
			expectedAmount.ReturnShippingAmount ||
		refund.ReturnShippingTaxAmount !=
			expectedAmount.ReturnShippingTaxAmount ||
		refund.RefundAmount !=
			expectedAmount.StripeRefundAmount {
		return ErrResaleTradeReturnRefundMismatch
	}

	totalSellerBurdenAmount, err :=
		refund.TotalSellerBurdenAmount()
	if err != nil {
		return ErrResaleTradeReturnRefundMismatch
	}

	if totalSellerBurdenAmount !=
		expectedAmount.TotalSellerBurdenAmount {
		return ErrResaleTradeReturnRefundMismatch
	}

	return nil
}
