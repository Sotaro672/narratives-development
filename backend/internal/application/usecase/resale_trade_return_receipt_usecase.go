// backend/internal/application/usecase/resale_trade_return_receipt_usecase.go
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
	ErrResaleTradeReturnReceiptNotConfigured = errors.New(
		"resale trade return receipt: usecase is not configured",
	)
	ErrResaleTradeReturnReceiptInvalidSeller = errors.New(
		"resale trade return receipt: invalid seller",
	)
	ErrResaleTradeReturnReceiptTradeMismatch = errors.New(
		"resale trade return receipt: trade does not match order item",
	)
	ErrResaleTradeReturnReceiptOrderNotPaid = errors.New(
		"resale trade return receipt: order is not paid",
	)
	ErrResaleTradeReturnReceiptNotEligible = errors.New(
		"resale trade return receipt: trade is not eligible for return receipt",
	)
	ErrResaleTradeReturnReceiptAgreementNotReady = errors.New(
		"resale trade return receipt: return agreement is not ready for receipt",
	)
	ErrResaleTradeReturnReceiptPhysicalReturnNotRequired = errors.New(
		"resale trade return receipt: physical return is not required",
	)
	ErrResaleTradeReturnReceiptDisputed = errors.New(
		"resale trade return receipt: return is disputed",
	)
	ErrResaleTradeReturnReceiptShipmentNotReady = errors.New(
		"resale trade return receipt: return shipment is not ready",
	)
	ErrResaleTradeReturnReceiptRefundMismatch = errors.New(
		"resale trade return receipt: refund does not match agreed return",
	)
	ErrResaleTradeReturnReceiptAgreementCompletionMismatch = errors.New(
		"resale trade return receipt: return agreement completion mismatch",
	)
)

// ResaleTradeReturnReceiptOrderService is the minimum Order application service
// required when a resale seller confirms physical receipt of a returned item.
//
// Trade return lifecycle state is owned by ReturnAgreement. Order is read only
// here and remains authoritative for purchaser identity, paid state, item
// identity and immutable resale seller snapshot.
type ResaleTradeReturnReceiptOrderService interface {
	GetByID(
		ctx context.Context,
		id string,
	) (orderdom.Order, error)
}

// ResaleTradeReturnReceiptItemRefundService is the financial boundary used by
// resale Trade return receipt.
//
// ItemRefundUsecase still accepts an InquiryID-shaped correlation field. This
// usecase supplies the deterministic legacy return correlation ID only for the
// Refund aggregate boundary. It does not load or validate Inquiry state.
type ResaleTradeReturnReceiptItemRefundService interface {
	RefundOrderItem(
		ctx context.Context,
		in RefundOrderItemInput,
	) (refunddom.Refund, error)
}

type ResaleTradeReturnReceiptUsecase struct {
	tradeRepo           tradedom.Repository
	returnAgreementRepo tradedom.ReturnAgreementRepository
	returnShipmentRepo  tradedom.ReturnShipmentRepository
	orderService        ResaleTradeReturnReceiptOrderService
	itemRefundService   ResaleTradeReturnReceiptItemRefundService

	refundCompletionNotifier ReturnReceiptRefundCompletionNotifier

	now func() time.Time
}

type NewResaleTradeReturnReceiptUsecaseInput struct {
	TradeRepository           tradedom.Repository
	ReturnAgreementRepository tradedom.ReturnAgreementRepository
	ReturnShipmentRepository  tradedom.ReturnShipmentRepository
	OrderService              ResaleTradeReturnReceiptOrderService
	ItemRefundService         ResaleTradeReturnReceiptItemRefundService
	RefundCompletionNotifier  ReturnReceiptRefundCompletionNotifier
}

func NewResaleTradeReturnReceiptUsecase(
	in NewResaleTradeReturnReceiptUsecaseInput,
) *ResaleTradeReturnReceiptUsecase {
	return &ResaleTradeReturnReceiptUsecase{
		tradeRepo:                in.TradeRepository,
		returnAgreementRepo:      in.ReturnAgreementRepository,
		returnShipmentRepo:       in.ReturnShipmentRepository,
		orderService:             in.OrderService,
		itemRefundService:        in.ItemRefundService,
		refundCompletionNotifier: in.RefundCompletionNotifier,
		now:                      time.Now,
	}
}

// SetNowFunc replaces the server clock for tests.
func (uc *ResaleTradeReturnReceiptUsecase) SetNowFunc(
	now func() time.Time,
) {
	if uc == nil || now == nil {
		return
	}

	uc.now = now
}

// ReceiveResaleTradeReturnInput identifies one seller-side physical receipt.
//
// TradeID comes from the route. SellerAvatarID must come from authenticated
// Mall AvatarContext and must never be trusted from the request body.
//
// Financial conditions are derived from the accepted ReturnProposal instead of
// frontend input.
type ReceiveResaleTradeReturnInput struct {
	TradeID        string
	SellerAvatarID string
}

// ResaleTradeReturnReceiptResult represents one seller receipt/refund attempt.
//
// ReturnAgreement is authoritative for the Trade return lifecycle. Order is
// not mutated into the legacy ReturnRequest/Inquiry completion state.
type ResaleTradeReturnReceiptResult struct {
	Trade     tradedom.Trade
	Agreement tradedom.ReturnAgreement
	Proposal  tradedom.ReturnProposal
	Shipment  tradedom.ReturnShipment
	Order     orderdom.Order
	Refund    refunddom.Refund

	FinanciallyCompleted bool
	ReturnCompleted      bool
	NotificationEnsured  bool
	AlreadyCompleted     bool
}

// ReceiveReturn confirms seller-side physical receipt and coordinates the
// agreed purchaser refund.
//
// Current execution:
//
//	seller Avatar
//	-> Trade
//	-> authoritative Order/item
//	-> accepted ReturnAgreement proposal
//	-> locally persisted ReturnShipment readiness
//	-> seller explicit receipt confirmation
//	-> ReturnAgreement return_received
//	-> ReturnAgreement refund_processing
//	-> ItemRefundUsecase using accepted Proposal.RefundAmount
//	-> refund completion notification
//	-> ReturnAgreement completed
//
// Current shipping-refund policy:
//   - RefundOutboundShipping=false
//   - CoverReturnShipping=false
//
// These flags are not currently represented by ReturnProposal, so they must not
// be accepted again from the seller at receipt time. If they become negotiable,
// they should be added to ReturnProposal and locked at buyer acceptance.
//
// This usecase intentionally does not depend on:
//   - Order.IsReturnRequested
//   - Order.ReturnRequestKind
//   - purchaser Inquiry state
//   - carrier shipment notifications
//   - frontend-selected refund conditions
func (uc *ResaleTradeReturnReceiptUsecase) ReceiveReturn(
	ctx context.Context,
	in ReceiveResaleTradeReturnInput,
) (ResaleTradeReturnReceiptResult, error) {
	if err := uc.validateConfigured(); err != nil {
		return ResaleTradeReturnReceiptResult{}, err
	}

	tradeID := strings.TrimSpace(in.TradeID)
	if tradeID == "" {
		return ResaleTradeReturnReceiptResult{}, tradedom.ErrInvalidID
	}

	sellerAvatarID := strings.TrimSpace(in.SellerAvatarID)
	if sellerAvatarID == "" {
		return ResaleTradeReturnReceiptResult{},
			ErrResaleTradeReturnReceiptInvalidSeller
	}

	trade, err := uc.tradeRepo.GetByID(ctx, tradeID)
	if err != nil {
		return ResaleTradeReturnReceiptResult{}, err
	}

	if trade.ID != tradeID ||
		trade.SellerType != tradedom.SellerTypeAvatar ||
		strings.TrimSpace(trade.SellerAvatarID) == "" ||
		trade.SellerAvatarID != sellerAvatarID {
		return ResaleTradeReturnReceiptResult{}, tradedom.ErrNotFound
	}

	if strings.TrimSpace(trade.OrderID) == "" ||
		trade.OrderItemIndex < 0 {
		return ResaleTradeReturnReceiptResult{
			Trade: trade,
		}, ErrResaleTradeReturnReceiptTradeMismatch
	}

	order, err := uc.orderService.GetByID(ctx, trade.OrderID)
	if err != nil {
		return ResaleTradeReturnReceiptResult{
			Trade: trade,
		}, err
	}

	result := ResaleTradeReturnReceiptResult{
		Trade: trade,
		Order: order,
	}

	if order.ID != trade.OrderID ||
		strings.TrimSpace(order.AvatarID) == "" ||
		order.AvatarID != trade.BuyerAvatarID ||
		trade.OrderItemIndex >= len(order.Items) {
		return result, ErrResaleTradeReturnReceiptTradeMismatch
	}

	itemIndex := trade.OrderItemIndex
	targetItem := order.Items[itemIndex]

	if err := validateResaleTradeReturnReceiptTarget(
		trade,
		targetItem,
		sellerAvatarID,
	); err != nil {
		return result, err
	}

	if !order.Paid {
		return result, ErrResaleTradeReturnReceiptOrderNotPaid
	}

	if targetItem.IsCancelled ||
		!targetItem.IsDispatched ||
		targetItem.Transferred {
		return result, ErrResaleTradeReturnReceiptNotEligible
	}

	agreement, err := uc.returnAgreementRepo.GetByTradeID(ctx, tradeID)
	if err != nil {
		return result, err
	}

	proposal, err := validateResaleTradeReturnReceiptAgreement(
		agreement,
		trade,
	)
	if err != nil {
		return result, err
	}

	result.Agreement = agreement
	result.Proposal = proposal

	selection := buildResaleTradeReturnReceiptSelection(proposal)
	if err := refunddom.ValidateReturnRefundSelection(selection); err != nil {
		return result, err
	}

	// completed is the authoritative terminal state for this Trade return.
	// Once reached, financial completion and notification delivery were already
	// confirmed before ReturnAgreement.Complete. An idempotent retry therefore
	// does not depend on the operational ReturnShipment record remaining readable.
	if agreement.Status == tradedom.ReturnStatusCompleted {
		result.FinanciallyCompleted = true
		result.ReturnCompleted = true
		result.NotificationEnsured = true
		result.AlreadyCompleted = true
		return result, nil
	}

	shipment, err := uc.returnShipmentRepo.GetByTradeID(ctx, tradeID)
	if err != nil {
		return result, err
	}

	if err := validateResaleTradeReturnReceiptShipment(
		shipment,
		trade,
		agreement,
		proposal,
	); err != nil {
		return result, err
	}

	result.Shipment = shipment

	agreement, err = uc.ensureSellerReceipt(
		ctx,
		agreement,
	)
	if err != nil {
		return result, err
	}
	result.Agreement = agreement

	agreement, err = uc.ensureRefundProcessing(
		ctx,
		agreement,
	)
	if err != nil {
		return result, err
	}
	result.Agreement = agreement

	// ItemRefundUsecase still persists an InquiryID-shaped correlation value.
	// Use the existing deterministic return ID for compatibility only. No Inquiry
	// is loaded and this value is not used to authorize or validate the return.
	refundSourceID := returnInquiryID(order.ID, itemIndex)

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

	if err := validateResaleTradeReturnReceiptRefund(
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
	result.FinanciallyCompleted = refund.IsFinanciallyCompleted()

	if !result.FinanciallyCompleted {
		return result, nil
	}

	_, err = uc.refundCompletionNotifier.EnsureDelivery(
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

	agreement, err = uc.ensureCompleted(ctx, agreement)
	if err != nil {
		return result, err
	}

	result.Agreement = agreement
	result.ReturnCompleted = agreement.Status == tradedom.ReturnStatusCompleted

	if !result.ReturnCompleted ||
		agreement.CompletedAt == nil ||
		agreement.CompletedAt.IsZero() {
		return result,
			ErrResaleTradeReturnReceiptAgreementCompletionMismatch
	}

	return result, nil
}

func (uc *ResaleTradeReturnReceiptUsecase) ensureSellerReceipt(
	ctx context.Context,
	agreement tradedom.ReturnAgreement,
) (tradedom.ReturnAgreement, error) {
	switch agreement.Status {
	case tradedom.ReturnStatusAgreed:
		if err := agreement.MarkReturnReceivedBySeller(
			uc.nowUTC(),
		); err != nil {
			return agreement, err
		}

		return uc.returnAgreementRepo.Update(
			ctx,
			agreement.TradeID,
			agreement,
		)

	case tradedom.ReturnStatusReturnShipped:
		if err := agreement.MarkReturnReceived(
			uc.nowUTC(),
		); err != nil {
			return agreement, err
		}

		return uc.returnAgreementRepo.Update(
			ctx,
			agreement.TradeID,
			agreement,
		)

	case tradedom.ReturnStatusReturnReceived,
		tradedom.ReturnStatusRefundProcessing:
		return agreement, nil

	case tradedom.ReturnStatusCompleted:
		return agreement, nil

	case tradedom.ReturnStatusDisputed:
		return agreement,
			ErrResaleTradeReturnReceiptDisputed

	default:
		return agreement,
			ErrResaleTradeReturnReceiptAgreementNotReady
	}
}

func (uc *ResaleTradeReturnReceiptUsecase) ensureRefundProcessing(
	ctx context.Context,
	agreement tradedom.ReturnAgreement,
) (tradedom.ReturnAgreement, error) {
	switch agreement.Status {
	case tradedom.ReturnStatusReturnReceived:
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
			ErrResaleTradeReturnReceiptDisputed

	default:
		return agreement,
			ErrResaleTradeReturnReceiptAgreementNotReady
	}
}

func (uc *ResaleTradeReturnReceiptUsecase) ensureCompleted(
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
			ErrResaleTradeReturnReceiptDisputed

	default:
		return agreement,
			ErrResaleTradeReturnReceiptAgreementNotReady
	}
}

func (uc *ResaleTradeReturnReceiptUsecase) validateConfigured() error {
	if uc == nil ||
		uc.tradeRepo == nil ||
		uc.returnAgreementRepo == nil ||
		uc.returnShipmentRepo == nil ||
		uc.orderService == nil ||
		uc.itemRefundService == nil ||
		uc.refundCompletionNotifier == nil ||
		uc.now == nil {
		return ErrResaleTradeReturnReceiptNotConfigured
	}

	return nil
}

func (uc *ResaleTradeReturnReceiptUsecase) nowUTC() time.Time {
	return uc.now().UTC()
}

func buildResaleTradeReturnReceiptSelection(
	proposal tradedom.ReturnProposal,
) refunddom.ReturnRefundSelection {
	return refunddom.ReturnRefundSelection{
		MerchandiseRefundAmount: proposal.RefundAmount,
		RefundOutboundShipping:  false,
		CoverReturnShipping:     false,
	}
}

func validateResaleTradeReturnReceiptAgreement(
	agreement tradedom.ReturnAgreement,
	trade tradedom.Trade,
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
		proposal.Agreement != tradedom.ReturnProposalAgreementAgree ||
		proposal.RejectedAt != nil ||
		agreement.AgreedAt == nil ||
		agreement.AgreedAt.IsZero() {
		return tradedom.ReturnProposal{},
			ErrResaleTradeReturnReceiptAgreementNotReady
	}

	if proposal.ReturnRequirement ==
		tradedom.ReturnRequirementNotRequired {
		return tradedom.ReturnProposal{},
			ErrResaleTradeReturnReceiptPhysicalReturnNotRequired
	}

	if proposal.ReturnRequirement !=
		tradedom.ReturnRequirementRequired {
		return tradedom.ReturnProposal{},
			tradedom.ErrInvalidReturnRequirement
	}

	if proposal.RefundAmount <= 0 {
		return tradedom.ReturnProposal{},
			tradedom.ErrInvalidReturnRefundAmount
	}

	switch agreement.Status {
	case tradedom.ReturnStatusAgreed,
		tradedom.ReturnStatusReturnShipped,
		tradedom.ReturnStatusReturnReceived,
		tradedom.ReturnStatusRefundProcessing,
		tradedom.ReturnStatusCompleted:
		return proposal, nil

	case tradedom.ReturnStatusDisputed:
		return tradedom.ReturnProposal{},
			ErrResaleTradeReturnReceiptDisputed

	default:
		return tradedom.ReturnProposal{},
			ErrResaleTradeReturnReceiptAgreementNotReady
	}
}

// validateResaleTradeReturnReceiptShipment validates only AMOL's locally
// persisted reverse-logistics state. It does not query PUDO/Yamato and does not
// infer carrier acceptance or delivery from the existence of a QR payload.
func validateResaleTradeReturnReceiptShipment(
	shipment tradedom.ReturnShipment,
	trade tradedom.Trade,
	agreement tradedom.ReturnAgreement,
	proposal tradedom.ReturnProposal,
) error {
	if shipment.ID != trade.ID ||
		shipment.TradeID != trade.ID ||
		shipment.ReturnAgreementID != agreement.ID ||
		shipment.ProposalID != proposal.ID ||
		shipment.Carrier != tradedom.ReturnShipmentCarrierYamato ||
		shipment.DropOffMethod != tradedom.ReturnShipmentDropOffMethodPUDO {
		return tradedom.ErrReturnShipmentConflict
	}

	switch shipment.Status {
	case tradedom.ReturnShipmentStatusReadyForDropOff,
		tradedom.ReturnShipmentStatusShipped,
		tradedom.ReturnShipmentStatusDelivered:
		return nil

	case tradedom.ReturnShipmentStatusPending,
		tradedom.ReturnShipmentStatusCancelled:
		return ErrResaleTradeReturnReceiptShipmentNotReady

	default:
		return ErrResaleTradeReturnReceiptShipmentNotReady
	}
}

// validateResaleTradeReturnReceiptRefund validates that the Refund persisted by
// ItemRefundUsecase represents exactly the same consumer-resale return and the
// immutable conditions accepted in ReturnProposal.
func validateResaleTradeReturnReceiptRefund(
	refundSourceID string,
	order orderdom.Order,
	itemIndex int,
	selection refunddom.ReturnRefundSelection,
	refund refunddom.Refund,
) error {
	if err := refund.Validate(); err != nil {
		return fmt.Errorf(
			"%w: %v",
			ErrResaleTradeReturnReceiptRefundMismatch,
			err,
		)
	}

	if strings.TrimSpace(refundSourceID) == "" ||
		refund.InquiryID != refundSourceID ||
		refund.OrderID != order.ID ||
		refund.PaymentID != order.ID ||
		refund.OrderItemIndex != itemIndex {
		return ErrResaleTradeReturnReceiptRefundMismatch
	}

	if itemIndex < 0 || itemIndex >= len(order.Items) {
		return ErrResaleTradeReturnReceiptRefundMismatch
	}

	targetItem := order.Items[itemIndex]
	if targetItem.Type != orderdom.OrderItemTypeResale {
		return ErrResaleTradeReturnReceiptRefundMismatch
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
		return ErrResaleTradeReturnReceiptRefundMismatch
	}

	expectedSalesReceivableID, err := salesreceivabledom.NewID(
		order.ID,
		itemIndex,
	)
	if err != nil ||
		refund.SalesReceivableID != expectedSalesReceivableID {
		return ErrResaleTradeReturnReceiptRefundMismatch
	}

	if refund.Currency != refunddom.CurrencyJPY {
		return ErrResaleTradeReturnReceiptRefundMismatch
	}

	if err := refunddom.ValidateReturnRefundSelection(selection); err != nil {
		return err
	}

	expectedAmount, err := refunddom.CalculateReturnRefundAmount(
		order,
		itemIndex,
		selection,
	)
	if err != nil {
		return err
	}

	if refund.RequestedMerchandiseRefundAmount != selection.MerchandiseRefundAmount ||
		refund.RefundOutboundShipping != selection.RefundOutboundShipping ||
		refund.CoverReturnShipping != selection.CoverReturnShipping ||
		refund.MerchandiseAmount != expectedAmount.MerchandiseAmount ||
		refund.MerchandiseTaxAmount != expectedAmount.MerchandiseTaxAmount ||
		refund.OutboundShippingAmount != expectedAmount.OutboundShippingAmount ||
		refund.OutboundShippingTaxAmount != expectedAmount.OutboundShippingTaxAmount ||
		refund.ReturnShippingAmount != expectedAmount.ReturnShippingAmount ||
		refund.ReturnShippingTaxAmount != expectedAmount.ReturnShippingTaxAmount ||
		refund.RefundAmount != expectedAmount.StripeRefundAmount {
		return ErrResaleTradeReturnReceiptRefundMismatch
	}

	totalSellerBurdenAmount, err := refund.TotalSellerBurdenAmount()
	if err != nil {
		return ErrResaleTradeReturnReceiptRefundMismatch
	}

	if totalSellerBurdenAmount != expectedAmount.TotalSellerBurdenAmount {
		return ErrResaleTradeReturnReceiptRefundMismatch
	}

	return nil
}

// validateResaleTradeReturnReceiptTarget confirms that the authoritative Order
// item is exactly the consumer-resale item represented by the Trade and that the
// authenticated Avatar is the immutable resale seller.
func validateResaleTradeReturnReceiptTarget(
	trade tradedom.Trade,
	targetItem orderdom.OrderItemSnapshot,
	sellerAvatarID string,
) error {
	if targetItem.Type != orderdom.OrderItemTypeResale ||
		strings.TrimSpace(targetItem.ResaleID) == "" {
		return ErrResaleTradeReturnReceiptTradeMismatch
	}

	seller := targetItem.SellerSnapshot

	if strings.TrimSpace(seller.AvatarID) == "" ||
		seller.AvatarID != sellerAvatarID ||
		seller.AvatarID != trade.SellerAvatarID {
		return tradedom.ErrNotFound
	}

	if strings.TrimSpace(seller.UserID) == "" ||
		strings.TrimSpace(seller.PayoutAccountID) == "" ||
		seller.PayoutAccountID != seller.UserID ||
		seller.BrandID != "" ||
		seller.CompanyID != "" ||
		seller.AccountID != "" ||
		seller.StripeAccountID != "" {
		return ErrResaleTradeReturnReceiptTradeMismatch
	}

	return nil
}
