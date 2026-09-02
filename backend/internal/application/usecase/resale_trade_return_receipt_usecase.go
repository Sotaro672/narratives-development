// backend/internal/application/usecase/resale_trade_return_receipt_usecase.go
package usecase

import (
	"context"
	"errors"
	"strings"

	inquirydom "narratives/internal/domain/inquiry"
	orderdom "narratives/internal/domain/order"
	refunddom "narratives/internal/domain/refund"
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
	ErrResaleTradeReturnReceiptReturnNotRequested = errors.New(
		"resale trade return receipt: return is not requested",
	)
	ErrResaleTradeReturnReceiptInquiryMismatch = errors.New(
		"resale trade return receipt: inquiry does not match trade",
	)
	ErrResaleTradeReturnReceiptInquiryClosed = errors.New(
		"resale trade return receipt: inquiry is closed",
	)
	ErrResaleTradeReturnReceiptInquiryResolved = errors.New(
		"resale trade return receipt: inquiry was resolved before return completion",
	)
	ErrResaleTradeReturnReceiptReturnKindMismatch = errors.New(
		"resale trade return receipt: return kind does not match inquiry",
	)
	ErrResaleTradeReturnReceiptUnopenedStateInvalid = errors.New(
		"resale trade return receipt: item is no longer unopened",
	)
	ErrResaleTradeReturnReceiptUnexpectedPolicy = errors.New(
		"resale trade return receipt: refund policy must be empty for unopened return",
	)
	ErrResaleTradeReturnReceiptOrderCompletionMismatch = errors.New(
		"resale trade return receipt: order return completion mismatch",
	)
)

// ResaleTradeReturnReceiptOrderService is the minimum Order application service
// required when a resale seller confirms physical receipt of a returned item.
type ResaleTradeReturnReceiptOrderService interface {
	GetByID(
		ctx context.Context,
		id string,
	) (orderdom.Order, error)

	CompleteReturnItem(
		ctx context.Context,
		in CompleteReturnOrderItemInput,
	) (orderdom.Order, error)
}

// ResaleTradeReturnReceiptItemRefundService is the financial boundary used by
// resale return receipt.
//
// ItemRefundUsecase already supports consumer resale items. Seller identity,
// refund amounts, SalesReceivable, BrandFeeSettlement and Stripe operations are
// resolved from authoritative persisted state rather than request values.
type ResaleTradeReturnReceiptItemRefundService interface {
	RefundOrderItem(
		ctx context.Context,
		in RefundOrderItemInput,
	) (refunddom.Refund, error)

	RefundOpenedReturnOrderItem(
		ctx context.Context,
		in RefundOpenedReturnItemInput,
	) (refunddom.Refund, error)
}

type ResaleTradeReturnReceiptUsecase struct {
	tradeRepo tradedom.Repository

	orderService ResaleTradeReturnReceiptOrderService

	inquiryRepo inquirydom.Repository

	itemRefundService ResaleTradeReturnReceiptItemRefundService

	refundCompletionNotifier ReturnReceiptRefundCompletionNotifier
}

type NewResaleTradeReturnReceiptUsecaseInput struct {
	TradeRepository tradedom.Repository

	OrderService ResaleTradeReturnReceiptOrderService

	InquiryRepository inquirydom.Repository

	ItemRefundService ResaleTradeReturnReceiptItemRefundService

	RefundCompletionNotifier ReturnReceiptRefundCompletionNotifier
}

func NewResaleTradeReturnReceiptUsecase(
	in NewResaleTradeReturnReceiptUsecaseInput,
) *ResaleTradeReturnReceiptUsecase {
	return &ResaleTradeReturnReceiptUsecase{
		tradeRepo:                in.TradeRepository,
		orderService:             in.OrderService,
		inquiryRepo:              in.InquiryRepository,
		itemRefundService:        in.ItemRefundService,
		refundCompletionNotifier: in.RefundCompletionNotifier,
	}
}

// ReceiveResaleTradeReturnInput identifies one seller-side return receipt.
//
// TradeID comes from the route.
//
// SellerAvatarID must come from authenticated Mall AvatarContext and must never
// be trusted from the request body.
//
// Policy is required only when the purchaser declared an opened return.
// Monetary amounts are never accepted from the frontend.
type ReceiveResaleTradeReturnInput struct {
	TradeID        string
	SellerAvatarID string
	Policy         refunddom.OpenedReturnRefundPolicy
}

// ResaleTradeReturnReceiptResult represents one seller return-receipt attempt.
//
// FinanciallyCompleted becomes true only after the purchaser Stripe Refund and
// every required resale seller/Brand financial operation have completed.
//
// OrderCompleted becomes true only after the target Order item has persisted
// IsReturnCompleted=true.
//
// NotificationEnsured means refund-completion notification delivery has been
// scheduled or confirmed idempotently.
//
// Inquiry resolution is intentionally outside this usecase. The current Inquiry
// domain exposes member-side resolve and purchaser Avatar close semantics; this
// usecase must not impersonate a company member with the resale seller Avatar.
type ResaleTradeReturnReceiptResult struct {
	Trade   tradedom.Trade
	Inquiry inquirydom.Inquiry
	Order   orderdom.Order
	Refund  refunddom.Refund

	FinanciallyCompleted bool
	OrderCompleted       bool
	NotificationEnsured  bool
	AlreadyCompleted     bool
}

// ReceiveReturn confirms physical receipt of one returned consumer-resale item.
//
// Execution:
//
//	seller Avatar
//	-> Trade
//	-> authoritative Order item
//	-> deterministic return Inquiry
//	-> unopened/opened ItemRefundUsecase
//	-> Stripe purchaser Refund
//	-> resale SalesReceivable / BrandFeeSettlement handling
//	-> Order IsReturnCompleted
//	-> refund completion notification
//
// The operation is designed to remain safe under retries. ItemRefundUsecase,
// Refund aggregate IDs and Stripe idempotency keys are deterministic, and
// CompleteReturnItem is idempotent for an already-completed return.
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

	trade, err := uc.tradeRepo.GetByID(
		ctx,
		tradeID,
	)
	if err != nil {
		return ResaleTradeReturnReceiptResult{}, err
	}

	if trade.ID != tradeID ||
		trade.SellerType != tradedom.SellerTypeAvatar ||
		trade.SellerAvatarID == "" ||
		trade.SellerAvatarID != sellerAvatarID {
		return ResaleTradeReturnReceiptResult{}, tradedom.ErrNotFound
	}

	if trade.OrderID == "" || trade.OrderItemIndex < 0 {
		return ResaleTradeReturnReceiptResult{
			Trade: trade,
		}, ErrResaleTradeReturnReceiptTradeMismatch
	}

	order, err := uc.orderService.GetByID(
		ctx,
		trade.OrderID,
	)
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
		order.AvatarID == "" ||
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
		!targetItem.IsReturnRequested ||
		targetItem.ReturnRequestedAt == nil ||
		targetItem.ReturnRequestedAt.IsZero() {
		return result, ErrResaleTradeReturnReceiptReturnNotRequested
	}

	inquiryID := returnInquiryID(
		order.ID,
		itemIndex,
	)

	inquiry, err := uc.inquiryRepo.GetByID(
		ctx,
		inquiryID,
	)
	if err != nil {
		return result, err
	}

	result.Inquiry = inquiry

	if err := validateResaleTradeReturnInquiry(
		inquiry,
		trade,
		order,
		targetItem,
	); err != nil {
		return result, err
	}

	alreadyCompleted := targetItem.IsReturnCompleted

	var refund refunddom.Refund

	switch targetItem.ReturnRequestKind {
	case orderdom.ReturnRequestKindUnopened:
		if in.Policy != "" {
			return result, ErrResaleTradeReturnReceiptUnexpectedPolicy
		}

		if targetItem.Transferred ||
			targetItem.TokenTransferVerifiedAt != nil {
			return result, ErrResaleTradeReturnReceiptUnopenedStateInvalid
		}

		refund, err = uc.itemRefundService.RefundOrderItem(
			ctx,
			RefundOrderItemInput{
				InquiryID: inquiry.ID,
				OrderID:   order.ID,
				ItemIndex: itemIndex,

				// Consumer resale does not use CompanyID as seller identity.
				// ItemRefundUsecase resolves the immutable Avatar/User/
				// PayoutAccount seller from OrderItemSnapshot.SellerSnapshot.
				CompanyID: "",
			},
		)
		if err != nil {
			result.Refund = refund
			return result, err
		}

		if err := validateReturnReceiptRefund(
			inquiry,
			order,
			itemIndex,
			refund,
		); err != nil {
			result.Refund = refund
			return result, err
		}

	case orderdom.ReturnRequestKindOpened:
		if err := refunddom.ValidateOpenedReturnRefundPolicy(
			in.Policy,
		); err != nil {
			return result, err
		}

		refund, err = uc.itemRefundService.RefundOpenedReturnOrderItem(
			ctx,
			RefundOpenedReturnItemInput{
				InquiryID: inquiry.ID,
				OrderID:   order.ID,
				ItemIndex: itemIndex,

				// CompanyID is intentionally empty for consumer resale.
				CompanyID: "",

				Policy: in.Policy,
			},
		)
		if err != nil {
			result.Refund = refund
			return result, err
		}

		if err := validateOpenedReturnReceiptRefund(
			inquiry,
			order,
			itemIndex,
			in.Policy,
			refund,
		); err != nil {
			result.Refund = refund
			return result, err
		}

	default:
		return result, ErrResaleTradeReturnReceiptReturnKindMismatch
	}

	result.Refund = refund
	result.FinanciallyCompleted =
		refund.IsFinanciallyCompleted()

	if !result.FinanciallyCompleted {
		return result, nil
	}

	if !alreadyCompleted {
		completedOrder, err :=
			uc.orderService.CompleteReturnItem(
				ctx,
				CompleteReturnOrderItemInput{
					ID:        order.ID,
					ItemIndex: itemIndex,
				},
			)
		if err != nil {
			return result, err
		}

		order = completedOrder
		result.Order = completedOrder
	}

	if itemIndex < 0 ||
		itemIndex >= len(order.Items) ||
		!order.Items[itemIndex].IsReturnCompleted ||
		order.Items[itemIndex].ReturnCompletedAt == nil ||
		order.Items[itemIndex].ReturnCompletedAt.IsZero() {
		return result,
			ErrResaleTradeReturnReceiptOrderCompletionMismatch
	}

	result.OrderCompleted = true

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
	result.AlreadyCompleted = alreadyCompleted

	return result, nil
}

func (uc *ResaleTradeReturnReceiptUsecase) validateConfigured() error {
	if uc == nil ||
		uc.tradeRepo == nil ||
		uc.orderService == nil ||
		uc.inquiryRepo == nil ||
		uc.itemRefundService == nil ||
		uc.refundCompletionNotifier == nil {
		return ErrResaleTradeReturnReceiptNotConfigured
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
		targetItem.ResaleID == "" {
		return ErrResaleTradeReturnReceiptTradeMismatch
	}

	seller := targetItem.SellerSnapshot

	if seller.AvatarID == "" ||
		seller.AvatarID != sellerAvatarID ||
		seller.AvatarID != trade.SellerAvatarID {
		return tradedom.ErrNotFound
	}

	if seller.UserID == "" ||
		seller.PayoutAccountID == "" ||
		seller.PayoutAccountID != seller.UserID ||
		seller.BrandID != "" ||
		seller.CompanyID != "" ||
		seller.AccountID != "" ||
		seller.StripeAccountID != "" {
		return ErrResaleTradeReturnReceiptTradeMismatch
	}

	return nil
}

// validateResaleTradeReturnInquiry confirms that the deterministic purchaser
// return Inquiry belongs to the same buyer, Order and Order item represented by
// the Trade.
//
// A seller cannot receive a return through an arbitrary Inquiry ID because the
// HTTP layer does not supply an Inquiry ID. The usecase derives the deterministic
// return Inquiry ID from authoritative Order identity.
func validateResaleTradeReturnInquiry(
	inquiry inquirydom.Inquiry,
	trade tradedom.Trade,
	order orderdom.Order,
	targetItem orderdom.OrderItemSnapshot,
) error {
	expectedInquiryID := returnInquiryID(
		order.ID,
		trade.OrderItemIndex,
	)

	if inquiry.ID == "" ||
		inquiry.ID != expectedInquiryID ||
		inquiry.DeletedAt != nil {
		return ErrResaleTradeReturnReceiptInquiryMismatch
	}

	if inquiry.OrderID != order.ID ||
		inquiry.OrderItemIndex == nil ||
		*inquiry.OrderItemIndex != trade.OrderItemIndex ||
		inquiry.AvatarID == "" ||
		inquiry.AvatarID != trade.BuyerAvatarID ||
		inquiry.AvatarID != order.AvatarID {
		return ErrResaleTradeReturnReceiptInquiryMismatch
	}

	if inquiry.Status == inquirydom.InquiryStatusClosed {
		return ErrResaleTradeReturnReceiptInquiryClosed
	}

	if inquiry.Status == inquirydom.InquiryStatusResolved &&
		!targetItem.IsReturnCompleted {
		return ErrResaleTradeReturnReceiptInquiryResolved
	}

	switch targetItem.ReturnRequestKind {
	case orderdom.ReturnRequestKindUnopened:
		if inquiry.InquiryType !=
			inquirydom.InquiryTypeReturnUnopened {
			return ErrResaleTradeReturnReceiptReturnKindMismatch
		}

	case orderdom.ReturnRequestKindOpened:
		if inquiry.InquiryType !=
			inquirydom.InquiryTypeReturnOpened {
			return ErrResaleTradeReturnReceiptReturnKindMismatch
		}

	default:
		return ErrResaleTradeReturnReceiptReturnKindMismatch
	}

	return nil
}
