// backend/internal/application/usecase/resale_trade_dispatch_usecase.go
package usecase

import (
	"context"
	"errors"

	orderdom "narratives/internal/domain/order"
	paymentdom "narratives/internal/domain/payment"
	tradedom "narratives/internal/domain/trade"
	transportationdom "narratives/internal/domain/transportation"
)

var (
	ErrResaleTradeDispatchTradeRepositoryMissing = errors.New(
		"resale trade dispatch: trade repository is not configured",
	)
	ErrResaleTradeDispatchOrderRepositoryMissing = errors.New(
		"resale trade dispatch: order repository is not configured",
	)
	ErrResaleTradeDispatchPaymentFlowUsecaseMissing = errors.New(
		"resale trade dispatch: payment flow usecase is not configured",
	)
	ErrResaleTradeDispatchPaymentUsecaseMissing = errors.New(
		"resale trade dispatch: payment usecase is not configured",
	)
	ErrResaleTradeDispatchSettlementUsecaseMissing = errors.New(
		"resale trade dispatch: settlement usecase is not configured",
	)
)

type ResaleTradeDispatchUsecase struct {
	tradeRepo     tradedom.Repository
	orderRepo     orderdom.Repository
	paymentFlowUC *PaymentFlowUsecase
	paymentUC     *PaymentUsecase
	settlementUC  *SettlementUsecase
}

type NewResaleTradeDispatchUsecaseInput struct {
	TradeRepository tradedom.Repository
	OrderRepository orderdom.Repository

	PaymentFlowUsecase *PaymentFlowUsecase
	PaymentUsecase     *PaymentUsecase
	SettlementUsecase  *SettlementUsecase
}

type DispatchResaleTradeInput struct {
	TradeID        string
	SellerAvatarID string
	Carrier        transportationdom.Carrier
	BoxSize        int
}

type DispatchResaleTradeResult struct {
	Trade tradedom.Trade
	Order orderdom.Order
	Item  orderdom.OrderItemSnapshot

	Changed bool
}

func NewResaleTradeDispatchUsecase(
	in NewResaleTradeDispatchUsecaseInput,
) *ResaleTradeDispatchUsecase {
	return &ResaleTradeDispatchUsecase{
		tradeRepo:     in.TradeRepository,
		orderRepo:     in.OrderRepository,
		paymentFlowUC: in.PaymentFlowUsecase,
		paymentUC:     in.PaymentUsecase,
		settlementUC:  in.SettlementUsecase,
	}
}

func (u *ResaleTradeDispatchUsecase) Dispatch(
	ctx context.Context,
	in DispatchResaleTradeInput,
) (DispatchResaleTradeResult, error) {
	if u == nil || u.tradeRepo == nil {
		return DispatchResaleTradeResult{},
			ErrResaleTradeDispatchTradeRepositoryMissing
	}
	if u.orderRepo == nil {
		return DispatchResaleTradeResult{},
			ErrResaleTradeDispatchOrderRepositoryMissing
	}
	if u.paymentFlowUC == nil {
		return DispatchResaleTradeResult{},
			ErrResaleTradeDispatchPaymentFlowUsecaseMissing
	}
	if u.paymentUC == nil {
		return DispatchResaleTradeResult{},
			ErrResaleTradeDispatchPaymentUsecaseMissing
	}
	if u.settlementUC == nil {
		return DispatchResaleTradeResult{},
			ErrResaleTradeDispatchSettlementUsecaseMissing
	}

	if in.TradeID == "" {
		return DispatchResaleTradeResult{},
			tradedom.ErrInvalidID
	}
	if in.SellerAvatarID == "" {
		return DispatchResaleTradeResult{},
			tradedom.ErrInvalidSellerAvatarID
	}

	trade, err := u.tradeRepo.GetByID(
		ctx,
		in.TradeID,
	)
	if err != nil {
		return DispatchResaleTradeResult{}, err
	}

	if trade.ID != in.TradeID {
		return DispatchResaleTradeResult{},
			tradedom.ErrNotFound
	}
	if trade.SellerType != tradedom.SellerTypeAvatar {
		return DispatchResaleTradeResult{},
			tradedom.ErrNotFound
	}
	if trade.SellerAvatarID != in.SellerAvatarID {
		return DispatchResaleTradeResult{},
			tradedom.ErrNotFound
	}
	if trade.Status != tradedom.StatusActive {
		return DispatchResaleTradeResult{},
			tradedom.ErrConflict
	}

	order, err := u.orderRepo.GetByID(
		ctx,
		trade.OrderID,
	)
	if err != nil {
		return DispatchResaleTradeResult{}, err
	}

	item, err := validateResaleTradeDispatchTarget(
		order,
		trade,
		in.SellerAvatarID,
	)
	if err != nil {
		return DispatchResaleTradeResult{}, err
	}

	if item.IsDispatched {
		return DispatchResaleTradeResult{
			Trade:   trade,
			Order:   order,
			Item:    item,
			Changed: false,
		}, nil
	}

	if item.IsCancelled ||
		item.IsReturnRequested ||
		item.IsReturnCompleted ||
		item.Transferred ||
		item.TokenTransferVerifiedAt != nil {
		return DispatchResaleTradeResult{},
			orderdom.ErrConflict
	}

	if !transportationdom.IsValidResaleShippingCarrier(in.Carrier) {
		return DispatchResaleTradeResult{},
			transportationdom.ErrInvalidCarrier
	}
	if !transportationdom.IsValidResaleBoxSize(in.BoxSize) {
		return DispatchResaleTradeResult{},
			transportationdom.ErrInvalidResaleBoxSize
	}

	shippingQuote, err := transportationdom.CalculateResaleFlatRate(
		in.Carrier,
		in.BoxSize,
	)
	if err != nil {
		return DispatchResaleTradeResult{}, err
	}

	order, err = u.ensureResaleShippingQuote(
		ctx,
		order,
		item,
		shippingQuote,
	)
	if err != nil {
		return DispatchResaleTradeResult{}, err
	}

	// Orderに保存されたPaymentMethodSnapshotと、
	// backendで確定・保存したShippingQuoteSnapshotを使って
	// off-session決済する。
	//
	// succeeded Paymentが既に存在する場合は既存Paymentを再利用するため、
	// 同一Tradeに対する再実行でも二重課金しない。
	if err := u.paymentFlowUC.EnsureOrderPaidForDispatch(
		ctx,
		order.ID,
	); err != nil {
		return DispatchResaleTradeResult{}, err
	}

	payment, err := u.paymentUC.GetByPaymentID(
		ctx,
		order.ID,
	)
	if err != nil {
		return DispatchResaleTradeResult{}, err
	}
	if payment == nil ||
		payment.PaymentID != order.ID ||
		payment.Status != paymentdom.StatusSucceeded {
		return DispatchResaleTradeResult{},
			ErrPaymentFlowDispatchNotSucceeded
	}

	// 決済処理によってOrder.Paidが更新されるため、最新Orderを再取得する。
	paidOrder, err := u.orderRepo.GetByID(
		ctx,
		order.ID,
	)
	if err != nil {
		return DispatchResaleTradeResult{}, err
	}
	if paidOrder.ID != order.ID || !paidOrder.Paid {
		return DispatchResaleTradeResult{},
			orderdom.ErrConflict
	}

	paidItem, err := validateResaleTradeDispatchTarget(
		paidOrder,
		trade,
		in.SellerAvatarID,
	)
	if err != nil {
		return DispatchResaleTradeResult{}, err
	}

	if paidItem.IsCancelled ||
		paidItem.IsReturnRequested ||
		paidItem.IsReturnCompleted ||
		paidItem.Transferred ||
		paidItem.TokenTransferVerifiedAt != nil {
		return DispatchResaleTradeResult{},
			orderdom.ErrConflict
	}

	// Webhook到着順に依存せず、succeeded Paymentからseller側の
	// financial recordを冪等に保証する。
	//
	// Resale itemではStripe SettlementではなくSalesReceivable pendingが
	// 作成される。availableへの遷移はtoken transfer完了時の責務であり、
	// 発送時には行わない。
	if _, err := u.settlementUC.EnsureForSucceededPayment(
		ctx,
		paidOrder,
		*payment,
	); err != nil {
		return DispatchResaleTradeResult{}, err
	}

	// 並行リクエスト等ですでに発送済みになっていれば、
	// financial recordの保証だけ行い冪等成功として返す。
	if paidItem.IsDispatched {
		return DispatchResaleTradeResult{
			Trade:   trade,
			Order:   paidOrder,
			Item:    paidItem,
			Changed: false,
		}, nil
	}

	if err := paidOrder.UpdateItemDispatched(
		trade.OrderItemIndex,
		true,
	); err != nil {
		return DispatchResaleTradeResult{}, err
	}

	updatedOrder, err := u.orderRepo.Update(
		ctx,
		paidOrder,
		nil,
	)
	if err != nil {
		return DispatchResaleTradeResult{}, err
	}

	if trade.OrderItemIndex < 0 ||
		trade.OrderItemIndex >= len(updatedOrder.Items) {
		return DispatchResaleTradeResult{},
			orderdom.ErrNotFound
	}

	return DispatchResaleTradeResult{
		Trade:   trade,
		Order:   updatedOrder,
		Item:    updatedOrder.Items[trade.OrderItemIndex],
		Changed: true,
	}, nil
}

func (u *ResaleTradeDispatchUsecase) ensureResaleShippingQuote(
	ctx context.Context,
	order orderdom.Order,
	item orderdom.OrderItemSnapshot,
	quote transportationdom.Quote,
) (orderdom.Order, error) {
	if quote.Carrier == "" ||
		!transportationdom.IsValidResaleShippingCarrier(quote.Carrier) {
		return orderdom.Order{},
			transportationdom.ErrInvalidCarrier
	}
	if !transportationdom.IsValidResaleBoxSize(quote.Size) {
		return orderdom.Order{},
			transportationdom.ErrInvalidResaleBoxSize
	}
	if quote.Amount <= 0 {
		return orderdom.Order{},
			transportationdom.ErrInvalidRateAmount
	}

	maxInt := int64(^uint(0) >> 1)
	if quote.Amount > maxInt {
		return orderdom.Order{},
			transportationdom.ErrInvalidRateAmount
	}

	amount := int(quote.Amount)
	quoteItems := append(
		[]orderdom.ShippingQuoteItemSnapshot(nil),
		order.ShippingQuoteSnapshot.Items...,
	)

	targetIndex := -1

	for i := range quoteItems {
		quoteItem := quoteItems[i]

		if quoteItem.Type != orderdom.OrderItemTypeResale ||
			quoteItem.ResaleID != item.ResaleID {
			continue
		}

		if targetIndex >= 0 {
			return orderdom.Order{},
				orderdom.ErrInvalidShippingQuote
		}

		targetIndex = i
	}

	if targetIndex < 0 {
		return orderdom.Order{},
			orderdom.ErrInvalidShippingQuote
	}

	current := quoteItems[targetIndex]

	alreadyConfigured :=
		current.Carrier == string(quote.Carrier) &&
			current.Size == quote.Size &&
			current.UnitAmount == amount &&
			current.Amount == amount &&
			current.Currency == orderdom.ShippingQuoteCurrencyJPY

	if alreadyConfigured {
		return order, nil
	}

	// 決済済みOrderの金額を発送側から変更してはいけない。
	if order.Paid {
		return orderdom.Order{},
			orderdom.ErrConflict
	}

	current.Carrier = string(quote.Carrier)
	current.TransportationID = ""
	current.Size = quote.Size
	current.Qty = 1
	current.UnitAmount = amount
	current.Amount = amount
	current.Currency = orderdom.ShippingQuoteCurrencyJPY

	quoteItems[targetIndex] = current

	totalAmount := 0
	maxOrderAmount := int(^uint(0) >> 1)

	for _, quoteItem := range quoteItems {
		if quoteItem.Amount < 0 {
			return orderdom.Order{},
				orderdom.ErrInvalidShippingQuote
		}
		if totalAmount > maxOrderAmount-quoteItem.Amount {
			return orderdom.Order{},
				orderdom.ErrInvalidShippingQuote
		}

		totalAmount += quoteItem.Amount
	}

	nextShippingQuote := orderdom.ShippingQuoteSnapshot{
		Items:    quoteItems,
		Amount:   totalAmount,
		Currency: orderdom.ShippingQuoteCurrencyJPY,
	}

	if err := order.UpdateShippingQuoteSnapshot(
		nextShippingQuote,
	); err != nil {
		return orderdom.Order{}, err
	}

	updatedOrder, err := u.orderRepo.Update(
		ctx,
		order,
		nil,
	)
	if err != nil {
		return orderdom.Order{}, err
	}

	return updatedOrder, nil
}

func validateResaleTradeDispatchTarget(
	order orderdom.Order,
	trade tradedom.Trade,
	sellerAvatarID string,
) (orderdom.OrderItemSnapshot, error) {
	if order.ID == "" ||
		order.ID != trade.OrderID ||
		order.AvatarID != trade.BuyerAvatarID {
		return orderdom.OrderItemSnapshot{},
			tradedom.ErrNotFound
	}

	if trade.OrderItemIndex < 0 ||
		trade.OrderItemIndex >= len(order.Items) {
		return orderdom.OrderItemSnapshot{},
			tradedom.ErrNotFound
	}

	item := order.Items[trade.OrderItemIndex]

	if item.Type != orderdom.OrderItemTypeResale {
		return orderdom.OrderItemSnapshot{},
			tradedom.ErrNotFound
	}
	if item.ResaleID == "" {
		return orderdom.OrderItemSnapshot{},
			tradedom.ErrNotFound
	}
	if item.SellerSnapshot.AvatarID == "" ||
		item.SellerSnapshot.AvatarID != trade.SellerAvatarID ||
		item.SellerSnapshot.AvatarID != sellerAvatarID {
		return orderdom.OrderItemSnapshot{},
			tradedom.ErrNotFound
	}

	return item, nil
}
