// backend/internal/application/usecase/resale_trade_dispatch_usecase.go
package usecase

import (
	"context"
	"errors"
	"strings"

	applicationport "narratives/internal/application/port"
	brandfeesettlementdom "narratives/internal/domain/brandFeeSettlement"
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
	ErrResaleTradeDispatchBrandFeeSettlementUsecaseMissing = errors.New(
		"resale trade dispatch: brand fee settlement usecase is not configured",
	)
	ErrResaleTradeDispatchBrandFeeSettlementQueueMissing = errors.New(
		"resale trade dispatch: brand fee settlement queue is not configured",
	)
	ErrResaleTradeDispatchAuthUserReaderMissing = errors.New(
		"resale trade dispatch: auth user reader is not configured",
	)
	ErrResaleTradeDispatchProductBlueprintReaderMissing = errors.New(
		"resale trade dispatch: product blueprint reader is not configured",
	)
	ErrResaleTradeDispatchNotificationMailerMissing = errors.New(
		"resale trade dispatch: dispatch notification mailer is not configured",
	)
)

type ResaleTradeDispatchUsecase struct {
	tradeRepo     tradedom.Repository
	orderRepo     orderdom.Repository
	paymentFlowUC *PaymentFlowUsecase
	paymentUC     *PaymentUsecase
	settlementUC  *SettlementUsecase

	brandFeeSettlementUC    *BrandFeeSettlementUsecase
	brandFeeSettlementQueue BrandFeeSettlementTransferQueue

	authUserReader             applicationport.AuthUserReader
	productBlueprintReader     applicationport.ProductBlueprintGetter
	dispatchNotificationMailer applicationport.OrderDispatchNotificationMailerPort
}

type NewResaleTradeDispatchUsecaseInput struct {
	TradeRepository tradedom.Repository
	OrderRepository orderdom.Repository

	PaymentFlowUsecase *PaymentFlowUsecase
	PaymentUsecase     *PaymentUsecase
	SettlementUsecase  *SettlementUsecase

	BrandFeeSettlementUsecase *BrandFeeSettlementUsecase
	BrandFeeSettlementQueue   BrandFeeSettlementTransferQueue

	AuthUserReader             applicationport.AuthUserReader
	ProductBlueprintReader     applicationport.ProductBlueprintGetter
	DispatchNotificationMailer applicationport.OrderDispatchNotificationMailerPort
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
		tradeRepo:                  in.TradeRepository,
		orderRepo:                  in.OrderRepository,
		paymentFlowUC:              in.PaymentFlowUsecase,
		paymentUC:                  in.PaymentUsecase,
		settlementUC:               in.SettlementUsecase,
		brandFeeSettlementUC:       in.BrandFeeSettlementUsecase,
		brandFeeSettlementQueue:    in.BrandFeeSettlementQueue,
		authUserReader:             in.AuthUserReader,
		productBlueprintReader:     in.ProductBlueprintReader,
		dispatchNotificationMailer: in.DispatchNotificationMailer,
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
	if u.brandFeeSettlementUC == nil {
		return DispatchResaleTradeResult{},
			ErrResaleTradeDispatchBrandFeeSettlementUsecaseMissing
	}
	if u.brandFeeSettlementQueue == nil {
		return DispatchResaleTradeResult{},
			ErrResaleTradeDispatchBrandFeeSettlementQueueMissing
	}
	if u.authUserReader == nil {
		return DispatchResaleTradeResult{},
			ErrResaleTradeDispatchAuthUserReaderMissing
	}
	if u.productBlueprintReader == nil {
		return DispatchResaleTradeResult{},
			ErrResaleTradeDispatchProductBlueprintReaderMissing
	}
	if u.dispatchNotificationMailer == nil {
		return DispatchResaleTradeResult{},
			ErrResaleTradeDispatchNotificationMailerMissing
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

	// 発送保存後にBrandFeeSettlementのready化やqueue投入だけ失敗した場合、
	// 再実行時にはここへ入る。
	//
	// financial recordを再保証し、Brand feeをready化して再enqueueすることで
	// 「発送済みだがBrand feeがpendingのまま」を復旧する。
	if item.IsDispatched {
		if err := u.reconcileDispatchedBrandFee(
			ctx,
			order,
			trade.OrderItemIndex,
		); err != nil {
			return DispatchResaleTradeResult{}, err
		}

		if err := u.notifyBuyerDispatched(
			ctx,
			trade,
			order,
			item,
		); err != nil {
			return DispatchResaleTradeResult{}, err
		}

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
	// Resale itemではSalesReceivable pendingとBrandFeeSettlement pendingが
	// 作成される。
	//
	// SalesReceivableのavailable化はtoken transfer完了時の責務であり、
	// 発送時には行わない。
	//
	// BrandFeeSettlementもここではまだpendingのままとし、発送状態が
	// 永続化された後にだけreadyへ遷移させる。
	if _, err := u.settlementUC.EnsureForSucceededPayment(
		ctx,
		paidOrder,
		*payment,
	); err != nil {
		return DispatchResaleTradeResult{}, err
	}

	// 並行リクエスト等ですでに発送済みになっていれば、発送境界は既に
	// 永続化されているためBrand feeをready化してenqueueする。
	if paidItem.IsDispatched {
		if err := u.ensureBrandFeeSettlementReadyAndEnqueued(
			ctx,
			payment.PaymentID,
			trade.OrderItemIndex,
		); err != nil {
			return DispatchResaleTradeResult{}, err
		}

		if err := u.notifyBuyerDispatched(
			ctx,
			trade,
			paidOrder,
			paidItem,
		); err != nil {
			return DispatchResaleTradeResult{}, err
		}

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

	updatedItem := updatedOrder.Items[trade.OrderItemIndex]
	if !updatedItem.IsDispatched {
		return DispatchResaleTradeResult{},
			orderdom.ErrConflict
	}

	// Brand feeは「発送済み」が永続化された後にだけ送金可能とする。
	//
	// MarkReadyとCloud Tasks enqueueの間で失敗しても、次回Dispatch時の
	// IsDispatched分岐またはDispatchDue reconciliationから復旧できる。
	if err := u.ensureBrandFeeSettlementReadyAndEnqueued(
		ctx,
		payment.PaymentID,
		trade.OrderItemIndex,
	); err != nil {
		return DispatchResaleTradeResult{}, err
	}

	if err := u.notifyBuyerDispatched(
		ctx,
		trade,
		updatedOrder,
		updatedItem,
	); err != nil {
		return DispatchResaleTradeResult{}, err
	}

	return DispatchResaleTradeResult{
		Trade:   trade,
		Order:   updatedOrder,
		Item:    updatedItem,
		Changed: true,
	}, nil
}

// notifyBuyerDispatched sends the buyer a dispatch notification after the
// authoritative Order item has been persisted as dispatched.
//
// The Trade ID is used as the mail provider idempotency key so retries of the
// same Trade dispatch do not produce duplicate notification emails.
func (u *ResaleTradeDispatchUsecase) notifyBuyerDispatched(
	ctx context.Context,
	trade tradedom.Trade,
	order orderdom.Order,
	item orderdom.OrderItemSnapshot,
) error {
	if !item.IsDispatched {
		return orderdom.ErrConflict
	}
	if strings.TrimSpace(order.UserID) == "" {
		return orderdom.ErrInvalidUserID
	}
	if strings.TrimSpace(item.ProductBlueprintID) == "" {
		return orderdom.ErrConflict
	}

	toEmail, err := u.authUserReader.GetEmailByUID(
		ctx,
		order.UserID,
	)
	if err != nil {
		return err
	}

	toEmail = strings.TrimSpace(toEmail)
	if toEmail == "" {
		return errors.New(
			"resale trade dispatch: buyer email is empty",
		)
	}

	productBlueprint, err := u.productBlueprintReader.GetByID(
		ctx,
		item.ProductBlueprintID,
	)
	if err != nil {
		return err
	}
	if productBlueprint.ID != item.ProductBlueprintID {
		return orderdom.ErrConflict
	}

	productName := strings.TrimSpace(
		productBlueprint.ProductName,
	)
	if productName == "" {
		return errors.New(
			"resale trade dispatch: product name is empty",
		)
	}

	qty := item.Qty
	if qty <= 0 {
		return orderdom.ErrConflict
	}

	_, err = u.dispatchNotificationMailer.SendOrderDispatchNotification(
		ctx,
		applicationport.OrderDispatchNotificationMailMessage{
			IdempotencyKey: "resale-trade-dispatch:" + trade.ID,
			ToEmail:        toEmail,
			OrderID:        order.ID,
			Items: []applicationport.OrderDispatchNotificationMailItem{
				{
					ProductName: productName,
					Qty:         qty,
				},
			},
		},
	)

	return err
}

// reconcileDispatchedBrandFee repairs the financial side of an Order item whose
// dispatch state has already been persisted.
//
// This path is required because the previous request may have succeeded in
// persisting IsDispatched=true and then failed before BrandFeeSettlement was
// marked ready or its Cloud Task was enqueued.
func (u *ResaleTradeDispatchUsecase) reconcileDispatchedBrandFee(
	ctx context.Context,
	order orderdom.Order,
	orderItemIndex int,
) error {
	if !order.Paid {
		return orderdom.ErrConflict
	}

	payment, err := u.paymentUC.GetByPaymentID(
		ctx,
		order.ID,
	)
	if err != nil {
		return err
	}
	if payment == nil ||
		payment.PaymentID != order.ID ||
		payment.Status != paymentdom.StatusSucceeded {
		return ErrPaymentFlowDispatchNotSucceeded
	}

	// SalesReceivableとBrandFeeSettlementの存在を冪等に保証する。
	if _, err := u.settlementUC.EnsureForSucceededPayment(
		ctx,
		order,
		*payment,
	); err != nil {
		return err
	}

	return u.ensureBrandFeeSettlementReadyAndEnqueued(
		ctx,
		payment.PaymentID,
		orderItemIndex,
	)
}

// ensureBrandFeeSettlementReadyAndEnqueued moves the deterministic Brand fee
// record across the fulfillment boundary and schedules Stripe Transfer.
//
// It must only be called after IsDispatched=true has been persisted.
func (u *ResaleTradeDispatchUsecase) ensureBrandFeeSettlementReadyAndEnqueued(
	ctx context.Context,
	paymentID string,
	orderItemIndex int,
) error {
	brandFeeSettlementID, err := brandfeesettlementdom.NewID(
		paymentID,
		orderItemIndex,
	)
	if err != nil {
		return err
	}

	brandFeeSettlement, err := u.brandFeeSettlementUC.MarkReady(
		ctx,
		brandFeeSettlementID,
	)
	if err != nil {
		return err
	}

	switch brandFeeSettlement.Status {
	case brandfeesettlementdom.StatusReady,
		brandfeesettlementdom.StatusFailedRetryable,
		brandfeesettlementdom.StatusTransferring:

		return u.brandFeeSettlementQueue.EnqueueBrandFeeSettlementTransfer(
			ctx,
			brandFeeSettlement.ID,
		)

	case brandfeesettlementdom.StatusTransferred,
		brandfeesettlementdom.StatusFailed,
		brandfeesettlementdom.StatusCanceled,
		brandfeesettlementdom.StatusReversed:

		return nil

	default:
		return ErrBrandFeeSettlementCannotReady
	}
}

func (u *ResaleTradeDispatchUsecase) ensureResaleShippingQuote(
	ctx context.Context,
	order orderdom.Order,
	item orderdom.OrderItemSnapshot,
	quote transportationdom.Quote,
) (orderdom.Order, error) {
	if quote.Carrier == "" ||
		!transportationdom.IsValidResaleShippingCarrier(
			quote.Carrier,
		) {
		return orderdom.Order{},
			transportationdom.ErrInvalidCarrier
	}

	if !transportationdom.IsValidResaleBoxSize(
		quote.Size,
	) {
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
			current.Currency ==
				orderdom.ShippingQuoteCurrencyJPY

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
	current.Currency =
		orderdom.ShippingQuoteCurrencyJPY

	quoteItems[targetIndex] = current

	totalAmount := 0
	maxOrderAmount := int(^uint(0) >> 1)

	for _, quoteItem := range quoteItems {
		if quoteItem.Amount < 0 {
			return orderdom.Order{},
				orderdom.ErrInvalidShippingQuote
		}

		if totalAmount >
			maxOrderAmount-quoteItem.Amount {
			return orderdom.Order{},
				orderdom.ErrInvalidShippingQuote
		}

		totalAmount += quoteItem.Amount
	}

	nextShippingQuote :=
		orderdom.ShippingQuoteSnapshot{
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
		item.SellerSnapshot.AvatarID !=
			trade.SellerAvatarID ||
		item.SellerSnapshot.AvatarID !=
			sellerAvatarID {
		return orderdom.OrderItemSnapshot{},
			tradedom.ErrNotFound
	}

	return item, nil
}
