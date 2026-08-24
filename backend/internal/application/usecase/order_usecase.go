// backend/internal/application/usecase/order_usecase.go
package usecase

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	applicationport "narratives/internal/application/port"
	accountdom "narratives/internal/domain/account"
	branddom "narratives/internal/domain/brand"
	cartdom "narratives/internal/domain/cart"
	common "narratives/internal/domain/common"
	inventorydom "narratives/internal/domain/inventory"
	listdom "narratives/internal/domain/list"
	orderdom "narratives/internal/domain/order"
	paymentmethoddom "narratives/internal/domain/paymentMethod"
	productblueprintdom "narratives/internal/domain/productBlueprint"
	productblueprintcategorydom "narratives/internal/domain/productBlueprintCategory"
	resaledom "narratives/internal/domain/resale"
	shippingaddressdom "narratives/internal/domain/shippingAddress"
)

type OrderCancellationMailerPort interface {
	SendOrderCancellationReceipt(
		ctx context.Context,
		toEmail string,
		orderID string,
		itemIndex int,
	) error
}

// OrderUsecase orchestrates order operations.
//
// - /mall/me/orders は Order の取得・作成を担当する
// - Invoice の作成は /mall/me/invoices の責務
// - Payment の作成は /mall/me/payments の責務
// - ReturnItem は返品申請のみを記録し、返金実行は担当しない
// - Stripe Refund / Transfer Reversal は RefundUsecase の責務
type OrderUsecase struct {
	repo                 orderdom.Repository
	cartRepo             cartdom.Repository
	listRepo             listdom.Repository
	inventoryRepo        inventorydom.RepositoryPort
	productBlueprintRepo productblueprintdom.Repository
	brandRepo            branddom.Repository
	accountRepo          accountdom.Repository
	resaleRepo           resaledom.Repository
	paymentMethodRepo    paymentmethoddom.RepositoryPort
	shippingAddressRepo  shippingaddressdom.RepositoryPort
	shippingQuoteUC      *ShippingQuoteUsecase

	authUserReader     applicationport.AuthUserReader
	cancellationMailer OrderCancellationMailerPort

	now func() time.Time
}

func NewOrderUsecase(
	repo orderdom.Repository,
	listRepo listdom.Repository,
	inventoryRepo inventorydom.RepositoryPort,
	productBlueprintRepo productblueprintdom.Repository,
	resaleRepo resaledom.Repository,
	paymentMethodRepo paymentmethoddom.RepositoryPort,
	shippingAddressRepo shippingaddressdom.RepositoryPort,
	shippingQuoteUC *ShippingQuoteUsecase,
) *OrderUsecase {
	return &OrderUsecase{
		repo:                 repo,
		listRepo:             listRepo,
		inventoryRepo:        inventoryRepo,
		productBlueprintRepo: productBlueprintRepo,
		resaleRepo:           resaleRepo,
		paymentMethodRepo:    paymentMethodRepo,
		shippingAddressRepo:  shippingAddressRepo,
		shippingQuoteUC:      shippingQuoteUC,
		now:                  time.Now,
	}
}

func (u *OrderUsecase) WithCartRepository(cartRepo cartdom.Repository) *OrderUsecase {
	if u == nil {
		return u
	}

	u.cartRepo = cartRepo
	return u
}

func (u *OrderUsecase) WithSellerRepositories(
	brandRepo branddom.Repository,
	accountRepo accountdom.Repository,
) *OrderUsecase {
	if u == nil {
		return u
	}

	u.brandRepo = brandRepo
	u.accountRepo = accountRepo
	return u
}

func (u *OrderUsecase) WithCancellationNotification(
	authUserReader applicationport.AuthUserReader,
	mailer OrderCancellationMailerPort,
) *OrderUsecase {
	if u == nil {
		return u
	}

	u.authUserReader = authUserReader
	u.cancellationMailer = mailer
	return u
}

// =======================
// Queries
// =======================

func (u *OrderUsecase) GetByID(
	ctx context.Context,
	id string,
) (orderdom.Order, error) {
	return u.repo.GetByID(ctx, id)
}

func (u *OrderUsecase) ListByAvatarID(
	ctx context.Context,
	avatarID string,
	sort common.Sort,
	page common.Page,
) (common.PageResult[orderdom.Order], error) {
	avatarID = strings.TrimSpace(avatarID)
	if avatarID == "" {
		return common.PageResult[orderdom.Order]{},
			fmt.Errorf("order usecase: avatarId is required")
	}

	return u.repo.ListByAvatarID(
		ctx,
		avatarID,
		sort,
		page,
	)
}

// =======================
// Commands
// =======================

// CreateOrderItemInput contains only values that the client is allowed to
// select when creating an order.
//
// Price, InventoryID, ProductID, ProductBlueprintID,
// TokenBlueprintID, BrandID, and seller settlement destination are resolved
// from server-side repositories.
type CreateOrderItemInput struct {
	Type orderdom.OrderItemType

	// list item identifiers
	ListID  string
	ModelID string

	// resale item identifier
	ResaleID string

	Qty int

	// Reserved for future order creation behavior.
	// The current creation policy always persists false.
	IsCancelled  bool
	IsDispatched bool
}

type CreateOrderInput struct {
	ID       string
	UserID   string
	AvatarID string
	CartID   string

	ShippingAddressID string
	PaymentMethodID   string
	Items             []CreateOrderItemInput

	CreatedAt *time.Time
}

func (u *OrderUsecase) Create(
	ctx context.Context,
	in CreateOrderInput,
) (orderdom.Order, error) {
	now := u.now().UTC()

	createdAt := now
	if in.CreatedAt != nil && !in.CreatedAt.IsZero() {
		createdAt = in.CreatedAt.UTC()
	}

	id := in.ID
	if id == "" {
		id = u.newOrderID(now)
	}

	shippingAddressID := strings.TrimSpace(in.ShippingAddressID)

	shipping, err := u.resolveShippingSnapshot(
		ctx,
		in.UserID,
		shippingAddressID,
	)
	if err != nil {
		return orderdom.Order{}, err
	}

	paymentMethod, err := u.resolvePaymentMethodSnapshot(
		ctx,
		in.UserID,
		in.PaymentMethodID,
	)
	if err != nil {
		return orderdom.Order{}, err
	}

	items, err := u.resolveOrderItems(
		ctx,
		in.Items,
	)
	if err != nil {
		return orderdom.Order{}, err
	}

	shippingQuote, err := u.resolveShippingQuoteSnapshot(
		ctx,
		in.UserID,
		shippingAddressID,
		in.Items,
	)
	if err != nil {
		return orderdom.Order{}, err
	}

	order, err := orderdom.New(
		id,
		in.UserID,
		in.AvatarID,
		in.CartID,
		shipping,
		shippingQuote,
		paymentMethod,
		items,
		createdAt,
	)
	if err != nil {
		return orderdom.Order{}, err
	}

	order.Paid = false

	// Repository.Create must persist the Order and replace its canonical
	// orderTransferItems projection in the same Firestore transaction.
	created, err := u.repo.Create(ctx, order)
	if err != nil {
		return orderdom.Order{}, err
	}

	// 注文作成が確定した時点で、注文元のcartを削除する。
	//
	// Orderは既に永続化済みのため、cart削除失敗を購入APIの失敗として
	// 返すと、クライアント再試行による重複注文を誘発する可能性がある。
	// そのためcart削除はbest-effortとし、失敗はログへ残す。
	if u.cartRepo != nil {
		cartID := strings.TrimSpace(created.CartID)

		if cartID != "" {
			if err := u.cartRepo.DeleteByAvatarID(
				ctx,
				cartID,
			); err != nil {
				log.Printf(
					"order usecase: clear cart after order failed orderId=%q cartId=%q err=%v",
					created.ID,
					cartID,
					err,
				)
			}
		}
	}

	return created, nil
}

type UpdateOrderInput struct {
	ID string

	UserID   *string
	AvatarID *string
	CartID   *string

	ShippingAddressID *string
	PaymentMethodID   *string

	ReplaceItems *[]CreateOrderItemInput
}

func (u *OrderUsecase) Update(
	ctx context.Context,
	in UpdateOrderInput,
) (orderdom.Order, error) {
	order, err := u.repo.GetByID(ctx, in.ID)
	if err != nil {
		return orderdom.Order{}, err
	}

	if in.UserID != nil {
		order.UserID = *in.UserID
	}

	if in.AvatarID != nil {
		order.AvatarID = *in.AvatarID
	}

	if in.CartID != nil {
		order.CartID = *in.CartID
	}

	shippingAddressID, err := resolveOrderDestinationShippingAddressID(
		order.ShippingQuoteSnapshot,
	)
	if err != nil {
		return orderdom.Order{}, err
	}

	if in.ShippingAddressID != nil {
		shippingAddressID = strings.TrimSpace(*in.ShippingAddressID)

		if shippingAddressID == "" {
			return orderdom.Order{},
				orderdom.ErrInvalidShippingSnapshot
		}
	}

	shouldRefreshShipping :=
		in.ShippingAddressID != nil ||
			in.UserID != nil

	if shouldRefreshShipping {
		shipping, err := u.resolveShippingSnapshot(
			ctx,
			order.UserID,
			shippingAddressID,
		)
		if err != nil {
			return orderdom.Order{}, err
		}

		if err := order.UpdateShippingSnapshot(shipping); err != nil {
			return orderdom.Order{}, err
		}
	}

	if in.PaymentMethodID != nil {
		paymentMethod, err := u.resolvePaymentMethodSnapshot(
			ctx,
			order.UserID,
			*in.PaymentMethodID,
		)
		if err != nil {
			return orderdom.Order{}, err
		}

		if err := order.UpdatePaymentMethodSnapshot(
			paymentMethod,
		); err != nil {
			return orderdom.Order{}, err
		}
	}

	var shippingQuoteItems []CreateOrderItemInput

	if in.ReplaceItems != nil {
		items, err := u.resolveOrderItems(
			ctx,
			*in.ReplaceItems,
		)
		if err != nil {
			return orderdom.Order{}, err
		}

		if err := order.ReplaceItems(items); err != nil {
			return orderdom.Order{}, err
		}

		shippingQuoteItems = append(
			[]CreateOrderItemInput(nil),
			(*in.ReplaceItems)...,
		)
	}

	shouldRefreshShippingQuote :=
		in.ReplaceItems != nil ||
			in.ShippingAddressID != nil ||
			in.UserID != nil

	if shouldRefreshShippingQuote {
		if shippingQuoteItems == nil {
			shippingQuoteItems, err =
				createOrderItemInputsFromSnapshots(
					order.Items,
				)
			if err != nil {
				return orderdom.Order{}, err
			}
		}

		shippingQuote, err := u.resolveShippingQuoteSnapshot(
			ctx,
			order.UserID,
			shippingAddressID,
			shippingQuoteItems,
		)
		if err != nil {
			return orderdom.Order{}, err
		}

		if err := order.UpdateShippingQuoteSnapshot(
			shippingQuote,
		); err != nil {
			return orderdom.Order{}, err
		}
	}

	checked, err := orderdom.New(
		order.ID,
		order.UserID,
		order.AvatarID,
		order.CartID,
		order.ShippingSnapshot,
		order.ShippingQuoteSnapshot,
		order.PaymentMethodSnapshot,
		order.Items,
		order.CreatedAt,
	)
	if err != nil {
		return orderdom.Order{}, err
	}

	checked.Paid = order.Paid

	// Repository.Update must persist the Order and replace its canonical
	// orderTransferItems projection in the same Firestore transaction.
	return u.repo.Update(ctx, checked, nil)
}

type CancelOrderItemInput struct {
	ID        string
	AvatarID  string
	ItemIndex int
}

func (u *OrderUsecase) CancelItem(
	ctx context.Context,
	in CancelOrderItemInput,
) (orderdom.Order, error) {
	orderID := strings.TrimSpace(in.ID)
	if orderID == "" {
		return orderdom.Order{}, orderdom.ErrInvalidID
	}

	avatarID := strings.TrimSpace(in.AvatarID)
	if avatarID == "" {
		return orderdom.Order{}, orderdom.ErrInvalidAvatarID
	}

	if in.ItemIndex < 0 {
		return orderdom.Order{}, orderdom.ErrInvalidItems
	}

	order, err := u.repo.GetByID(ctx, orderID)
	if err != nil {
		return orderdom.Order{}, err
	}

	if order.AvatarID != avatarID {
		return orderdom.Order{}, orderdom.ErrNotFound
	}

	if in.ItemIndex >= len(order.Items) {
		return orderdom.Order{}, orderdom.ErrNotFound
	}

	targetItem := order.Items[in.ItemIndex]
	cancelledNow := !targetItem.IsCancelled

	if cancelledNow {
		if order.Paid ||
			targetItem.IsDispatched ||
			targetItem.Transferred {
			return orderdom.Order{}, orderdom.ErrConflict
		}

		shippingAddressID, err :=
			resolveOrderDestinationShippingAddressID(
				order.ShippingQuoteSnapshot,
			)
		if err != nil {
			return orderdom.Order{}, err
		}

		if err := order.CancelItem(in.ItemIndex); err != nil {
			return orderdom.Order{}, err
		}

		shippingQuoteItems, err :=
			createOrderItemInputsFromSnapshots(
				order.Items,
			)
		if err != nil {
			return orderdom.Order{}, err
		}

		if len(shippingQuoteItems) > 0 {
			shippingQuote, err :=
				u.resolveShippingQuoteSnapshot(
					ctx,
					order.UserID,
					shippingAddressID,
					shippingQuoteItems,
				)
			if err != nil {
				return orderdom.Order{}, err
			}

			if err := order.UpdateShippingQuoteSnapshot(
				shippingQuote,
			); err != nil {
				return orderdom.Order{}, err
			}
		}

		updated, err := u.repo.Update(ctx, order, nil)
		if err != nil {
			return orderdom.Order{}, err
		}

		order = updated
		targetItem = order.Items[in.ItemIndex]

		u.sendCancellationReceiptBestEffort(
			ctx,
			order,
			in.ItemIndex,
		)
	}

	if targetItem.Type == orderdom.OrderItemTypeList {
		remainingQty := 0

		for _, item := range order.Items {
			if item.Type != orderdom.OrderItemTypeList {
				continue
			}

			if item.InventoryID != targetItem.InventoryID ||
				item.ModelID != targetItem.ModelID {
				continue
			}

			if item.IsCancelled || item.Transferred {
				continue
			}

			remainingQty += item.Qty
		}

		if remainingQty > 0 {
			if err := u.inventoryRepo.ReserveByOrder(
				ctx,
				targetItem.InventoryID,
				targetItem.ModelID,
				order.ID,
				remainingQty,
			); err != nil {
				return orderdom.Order{}, err
			}
		} else {
			now := u.now().UTC()

			if err := u.inventoryRepo.ReleaseReservationByOrder(
				ctx,
				targetItem.InventoryID,
				targetItem.ModelID,
				order.ID,
				now,
			); err != nil {
				return orderdom.Order{}, err
			}
		}
	}

	return order, nil
}

// ReturnOrderItemInput identifies one Order item for which the purchaser
// requests a return.
//
// This input represents a return request only. It is not a refund instruction.
type ReturnOrderItemInput struct {
	ID        string
	AvatarID  string
	ItemIndex int
}

// ReturnItem records a purchaser return request.
//
// This method intentionally does not execute:
//
// - Stripe Refund
// - Stripe Transfer Reversal
// - Payment refund-state mutation
// - Settlement cancellation or reversal
//
// Those financial operations belong to RefundUsecase and must be started from
// a separate return-approval flow.
//
// The current RefundUsecase supports full Payment refunds only, while this
// method records an item-level request. A caller must therefore not translate
// one ReturnItem call directly into RefundByPaymentID without first confirming
// that the approved refund policy covers the complete Payment.
func (u *OrderUsecase) ReturnItem(
	ctx context.Context,
	in ReturnOrderItemInput,
) (orderdom.Order, error) {
	orderID := strings.TrimSpace(in.ID)
	if orderID == "" {
		return orderdom.Order{}, orderdom.ErrInvalidID
	}

	avatarID := strings.TrimSpace(in.AvatarID)
	if avatarID == "" {
		return orderdom.Order{}, orderdom.ErrInvalidAvatarID
	}

	if in.ItemIndex < 0 {
		return orderdom.Order{}, orderdom.ErrInvalidItems
	}

	order, err := u.repo.GetByID(ctx, orderID)
	if err != nil {
		return orderdom.Order{}, err
	}

	if order.AvatarID != avatarID {
		return orderdom.Order{}, orderdom.ErrNotFound
	}

	if in.ItemIndex >= len(order.Items) {
		return orderdom.Order{}, orderdom.ErrNotFound
	}

	targetItem := order.Items[in.ItemIndex]

	if targetItem.IsCancelled ||
		!targetItem.IsDispatched ||
		targetItem.Transferred {
		return orderdom.Order{}, orderdom.ErrConflict
	}

	if targetItem.IsReturnRequested {
		return order, nil
	}

	if err := order.RequestItemReturn(
		in.ItemIndex,
		u.now().UTC(),
	); err != nil {
		return orderdom.Order{}, err
	}

	updated, err := u.repo.Update(ctx, order, nil)
	if err != nil {
		return orderdom.Order{}, err
	}

	return updated, nil
}

func (u *OrderUsecase) sendCancellationReceiptBestEffort(
	ctx context.Context,
	order orderdom.Order,
	itemIndex int,
) {
	if u == nil ||
		u.authUserReader == nil ||
		u.cancellationMailer == nil {
		return
	}

	toEmail, err := u.authUserReader.GetEmailByUID(
		ctx,
		order.UserID,
	)
	if err != nil {
		log.Printf(
			"order cancellation mail: resolve email failed orderId=%q userId=%q err=%v",
			order.ID,
			order.UserID,
			err,
		)
		return
	}

	toEmail = strings.TrimSpace(toEmail)
	if toEmail == "" {
		log.Printf(
			"order cancellation mail: resolved email is empty orderId=%q userId=%q",
			order.ID,
			order.UserID,
		)
		return
	}

	if err := u.cancellationMailer.SendOrderCancellationReceipt(
		ctx,
		toEmail,
		order.ID,
		itemIndex,
	); err != nil {
		log.Printf(
			"order cancellation mail: send failed orderId=%q itemIndex=%d err=%v",
			order.ID,
			itemIndex,
			err,
		)
	}
}

type DispatchOrderItemsInput struct {
	ID string

	AllowedInventoryIDs map[string]struct{}
}

type DispatchOrderItemsResult struct {
	Order orderdom.Order

	TargetItems []orderdom.OrderItemSnapshot
	Changed     bool
}

func (u *OrderUsecase) PrepareDispatchItems(
	ctx context.Context,
	in DispatchOrderItemsInput,
) (DispatchOrderItemsResult, error) {
	if in.ID == "" {
		return DispatchOrderItemsResult{},
			orderdom.ErrInvalidID
	}

	order, err := u.repo.GetByID(
		ctx,
		in.ID,
	)
	if err != nil {
		return DispatchOrderItemsResult{}, err
	}

	targetItems := make(
		[]orderdom.OrderItemSnapshot,
		0,
		len(order.Items),
	)

	for _, item := range order.Items {
		if _, ok :=
			in.AllowedInventoryIDs[item.InventoryID]; !ok {
			continue
		}

		if item.IsCancelled || item.IsReturnRequested {
			continue
		}

		targetItems = append(
			targetItems,
			item,
		)
	}

	if len(targetItems) == 0 {
		return DispatchOrderItemsResult{},
			orderdom.ErrNotFound
	}

	return DispatchOrderItemsResult{
		Order:       order,
		TargetItems: targetItems,
		Changed:     false,
	}, nil
}

func (u *OrderUsecase) DispatchItems(
	ctx context.Context,
	in DispatchOrderItemsInput,
) (DispatchOrderItemsResult, error) {
	if in.ID == "" {
		return DispatchOrderItemsResult{},
			orderdom.ErrInvalidID
	}

	order, err := u.repo.GetByID(
		ctx,
		in.ID,
	)
	if err != nil {
		return DispatchOrderItemsResult{}, err
	}

	if !order.Paid {
		return DispatchOrderItemsResult{},
			orderdom.ErrConflict
	}

	targetItems := make(
		[]orderdom.OrderItemSnapshot,
		0,
		len(order.Items),
	)
	changed := false

	for index := range order.Items {
		item := order.Items[index]

		if _, ok :=
			in.AllowedInventoryIDs[item.InventoryID]; !ok {
			continue
		}

		if item.IsCancelled || item.IsReturnRequested {
			continue
		}

		if !item.IsDispatched {
			if err := order.UpdateItemDispatched(
				index,
				true,
			); err != nil {
				return DispatchOrderItemsResult{}, err
			}

			changed = true
		}

		targetItems = append(
			targetItems,
			order.Items[index],
		)
	}

	if len(targetItems) == 0 {
		return DispatchOrderItemsResult{},
			orderdom.ErrNotFound
	}

	if changed {
		updated, err := u.repo.Update(
			ctx,
			order,
			nil,
		)
		if err != nil {
			return DispatchOrderItemsResult{}, err
		}

		order = updated
	}

	return DispatchOrderItemsResult{
		Order:       order,
		TargetItems: targetItems,
		Changed:     changed,
	}, nil
}

// =======================
// Shipping snapshot
// =======================

func (u *OrderUsecase) resolveShippingSnapshot(
	ctx context.Context,
	userID string,
	shippingAddressID string,
) (orderdom.ShippingSnapshot, error) {
	if u == nil ||
		u.shippingAddressRepo == nil {
		return orderdom.ShippingSnapshot{},
			orderdom.ErrInvalidShippingSnapshot
	}

	userID = strings.TrimSpace(userID)
	shippingAddressID = strings.TrimSpace(shippingAddressID)

	if userID == "" ||
		shippingAddressID == "" {
		return orderdom.ShippingSnapshot{},
			orderdom.ErrInvalidShippingSnapshot
	}

	address, err := u.shippingAddressRepo.GetByUser(
		ctx,
		shippingAddressID,
		userID,
	)
	if err != nil {
		return orderdom.ShippingSnapshot{}, err
	}

	if address == nil {
		return orderdom.ShippingSnapshot{},
			shippingaddressdom.ErrNotFound
	}

	if address.ID != shippingAddressID {
		return orderdom.ShippingSnapshot{},
			shippingaddressdom.ErrNotFound
	}

	if address.UserID != userID {
		return orderdom.ShippingSnapshot{},
			shippingaddressdom.ErrNotFound
	}

	return orderdom.ShippingSnapshot{
		ZipCode: address.ZipCode,
		State:   address.State,
		City:    address.City,
		Street:  address.Street,
		Street2: address.Street2,
		Country: address.Country,
	}, nil
}

// =======================
// Shipping quote snapshot
// =======================

func (u *OrderUsecase) resolveShippingQuoteSnapshot(
	ctx context.Context,
	userID string,
	shippingAddressID string,
	input []CreateOrderItemInput,
) (orderdom.ShippingQuoteSnapshot, error) {
	if u == nil ||
		u.shippingQuoteUC == nil {
		return orderdom.ShippingQuoteSnapshot{},
			orderdom.ErrInvalidShippingQuote
	}

	userID = strings.TrimSpace(userID)
	shippingAddressID = strings.TrimSpace(shippingAddressID)

	if userID == "" ||
		shippingAddressID == "" ||
		len(input) == 0 {
		return orderdom.ShippingQuoteSnapshot{},
			orderdom.ErrInvalidShippingQuote
	}

	quoteItems := make(
		[]orderdom.ShippingQuoteItemSnapshot,
		0,
		len(input),
	)

	maxInt := int(^uint(0) >> 1)
	total := 0

	for _, item := range input {
		if item.Type != orderdom.OrderItemTypeList {
			return orderdom.ShippingQuoteSnapshot{},
				orderdom.ErrInvalidShippingQuote
		}

		if item.ListID == "" ||
			item.ModelID == "" ||
			item.Qty <= 0 {
			return orderdom.ShippingQuoteSnapshot{},
				orderdom.ErrInvalidShippingQuoteItem
		}

		quote, err := u.shippingQuoteUC.Quote(
			ctx,
			ShippingQuoteInput{
				UserID:                       userID,
				ListID:                       item.ListID,
				ModelID:                      item.ModelID,
				DestinationShippingAddressID: shippingAddressID,
			},
		)
		if err != nil {
			return orderdom.ShippingQuoteSnapshot{}, err
		}

		if quote.Amount < 0 ||
			quote.Amount > int64(maxInt) {
			return orderdom.ShippingQuoteSnapshot{},
				orderdom.ErrInvalidShippingQuoteItem
		}

		unitAmount := int(quote.Amount)

		if unitAmount > 0 &&
			item.Qty > maxInt/unitAmount {
			return orderdom.ShippingQuoteSnapshot{},
				orderdom.ErrInvalidShippingQuoteItem
		}

		lineAmount := unitAmount * item.Qty

		if total > maxInt-lineAmount {
			return orderdom.ShippingQuoteSnapshot{},
				orderdom.ErrInvalidShippingQuote
		}

		total += lineAmount

		quoteItems = append(
			quoteItems,
			orderdom.ShippingQuoteItemSnapshot{
				ListID:                       quote.ListID,
				InventoryID:                  quote.InventoryID,
				ModelID:                      quote.ModelID,
				OriginShippingAddressID:      quote.OriginShippingAddressID,
				DestinationShippingAddressID: quote.DestinationShippingAddressID,
				Carrier:                      string(quote.TransportationOption),
				TransportationID:             quote.TransportationID,
				Size:                         quote.Size,
				Qty:                          item.Qty,
				UnitAmount:                   unitAmount,
				Amount:                       lineAmount,
				Currency:                     quote.Currency,
			},
		)
	}

	return orderdom.ShippingQuoteSnapshot{
		Items:    quoteItems,
		Amount:   total,
		Currency: orderdom.ShippingQuoteCurrencyJPY,
	}, nil
}

func resolveOrderDestinationShippingAddressID(
	snapshot orderdom.ShippingQuoteSnapshot,
) (string, error) {
	if len(snapshot.Items) == 0 {
		return "",
			orderdom.ErrInvalidShippingQuote
	}

	destinationShippingAddressID := strings.TrimSpace(
		snapshot.Items[0].DestinationShippingAddressID,
	)

	if destinationShippingAddressID == "" {
		return "",
			orderdom.ErrInvalidShippingQuote
	}

	for _, item := range snapshot.Items {
		if strings.TrimSpace(
			item.DestinationShippingAddressID,
		) != destinationShippingAddressID {
			return "",
				orderdom.ErrInvalidShippingQuote
		}
	}

	return destinationShippingAddressID, nil
}

func createOrderItemInputsFromSnapshots(
	items []orderdom.OrderItemSnapshot,
) ([]CreateOrderItemInput, error) {
	if len(items) == 0 {
		return nil,
			orderdom.ErrInvalidItems
	}

	result := make(
		[]CreateOrderItemInput,
		0,
		len(items),
	)

	for _, item := range items {
		if item.IsCancelled {
			continue
		}

		switch item.Type {
		case orderdom.OrderItemTypeList:
			result = append(
				result,
				CreateOrderItemInput{
					Type:         orderdom.OrderItemTypeList,
					ListID:       item.ListID,
					ModelID:      item.ModelID,
					Qty:          item.Qty,
					IsCancelled:  item.IsCancelled,
					IsDispatched: item.IsDispatched,
				},
			)

		case orderdom.OrderItemTypeResale:
			return nil,
				orderdom.ErrInvalidShippingQuote

		default:
			return nil,
				orderdom.ErrInvalidItemSnapshot
		}
	}

	return result, nil
}

// =======================
// Payment method snapshot
// =======================

func (u *OrderUsecase) resolvePaymentMethodSnapshot(
	ctx context.Context,
	userID string,
	paymentMethodID string,
) (orderdom.PaymentMethodSnapshot, error) {
	paymentMethod, err := u.paymentMethodRepo.GetByID(
		ctx,
		paymentMethodID,
	)
	if err != nil {
		return orderdom.PaymentMethodSnapshot{}, err
	}

	if paymentMethod == nil ||
		paymentMethod.UserID != userID {
		return orderdom.PaymentMethodSnapshot{},
			orderdom.ErrInvalidPaymentMethod
	}

	return orderdom.PaymentMethodSnapshot{
		PaymentMethodID:       paymentMethod.ID,
		CustomerID:            paymentMethod.StripeCustomerID,
		StripePaymentMethodID: paymentMethod.StripePaymentMethodID,
		Brand:                 paymentMethod.Brand,
		Last4:                 paymentMethod.Last4,
		ExpMonth:              paymentMethod.ExpMonth,
		ExpYear:               paymentMethod.ExpYear,
		CardholderName:        paymentMethod.CardholderName,
		IsDefault:             paymentMethod.IsDefault,
	}, nil
}

// =======================
// Order item snapshots
// =======================

func (u *OrderUsecase) resolveOrderItems(
	ctx context.Context,
	input []CreateOrderItemInput,
) ([]orderdom.OrderItemSnapshot, error) {
	items := make(
		[]orderdom.OrderItemSnapshot,
		0,
		len(input),
	)

	for _, item := range input {
		switch item.Type {
		case orderdom.OrderItemTypeList:
			resolved, err := u.resolveListOrderItem(
				ctx,
				item,
			)
			if err != nil {
				return nil, err
			}

			items = append(items, resolved)

		case orderdom.OrderItemTypeResale:
			resolved, err := u.resolveResaleOrderItem(
				ctx,
				item,
			)
			if err != nil {
				return nil, err
			}

			items = append(items, resolved)

		default:
			return nil, orderdom.ErrInvalidItemSnapshot
		}
	}

	return items, nil
}

func (u *OrderUsecase) resolveProductBlueprintTaxSnapshot(
	ctx context.Context,
	productBlueprintID string,
) (
	[]string,
	int,
	error,
) {
	if u == nil ||
		u.productBlueprintRepo == nil {
		return nil,
			0,
			orderdom.ErrInvalidItemSnapshot
	}

	productBlueprintID = strings.TrimSpace(
		productBlueprintID,
	)

	if productBlueprintID == "" {
		return nil,
			0,
			orderdom.ErrInvalidItemSnapshot
	}

	productBlueprint, err := u.productBlueprintRepo.GetByID(
		ctx,
		productBlueprintID,
	)
	if err != nil {
		return nil,
			0,
			err
	}

	if productBlueprint.ID != productBlueprintID {
		return nil,
			0,
			orderdom.ErrInvalidItemSnapshot
	}

	categoryPath := append(
		[]string(nil),
		productBlueprint.ProductBlueprintCategoryPath...,
	)

	taxRate, err :=
		productblueprintcategorydom.GetConsumptionTaxRate(
			categoryPath,
		)
	if err != nil {
		return nil,
			0,
			err
	}

	return categoryPath,
		int(taxRate),
		nil
}

func (u *OrderUsecase) resolveSellerSnapshotByProductBlueprintID(
	ctx context.Context,
	productBlueprintID string,
) (orderdom.SellerSnapshot, error) {
	if u == nil ||
		u.productBlueprintRepo == nil ||
		u.brandRepo == nil ||
		u.accountRepo == nil {
		return orderdom.SellerSnapshot{},
			orderdom.ErrInvalidSellerSnapshot
	}

	productBlueprint, err := u.productBlueprintRepo.GetByID(
		ctx,
		productBlueprintID,
	)
	if err != nil {
		return orderdom.SellerSnapshot{}, err
	}

	if productBlueprint.ID != productBlueprintID ||
		productBlueprint.BrandID == "" ||
		productBlueprint.CompanyID == "" {
		return orderdom.SellerSnapshot{},
			orderdom.ErrInvalidSellerSnapshot
	}

	brand, err := u.brandRepo.GetByID(
		ctx,
		productBlueprint.BrandID,
	)
	if err != nil {
		return orderdom.SellerSnapshot{}, err
	}

	if brand.ID != productBlueprint.BrandID ||
		brand.CompanyID != productBlueprint.CompanyID ||
		brand.AccountID == "" ||
		!brand.IsActive {
		return orderdom.SellerSnapshot{},
			orderdom.ErrInvalidSellerSnapshot
	}

	account, err := u.accountRepo.GetByID(
		ctx,
		brand.AccountID,
	)
	if err != nil {
		return orderdom.SellerSnapshot{}, err
	}

	if account.ID != brand.AccountID ||
		account.CompanyID != brand.CompanyID ||
		account.Status != accountdom.StatusActive ||
		account.StripeAccountID == "" {
		return orderdom.SellerSnapshot{},
			orderdom.ErrInvalidSellerSnapshot
	}

	return orderdom.SellerSnapshot{
		BrandID:         brand.ID,
		CompanyID:       brand.CompanyID,
		AccountID:       account.ID,
		StripeAccountID: account.StripeAccountID,
	}, nil
}

func (u *OrderUsecase) resolveListOrderItem(
	ctx context.Context,
	item CreateOrderItemInput,
) (orderdom.OrderItemSnapshot, error) {
	if item.ListID == "" ||
		item.ModelID == "" ||
		item.Qty <= 0 {
		return orderdom.OrderItemSnapshot{},
			orderdom.ErrInvalidItemSnapshot
	}

	list, err := u.listRepo.GetByID(
		ctx,
		item.ListID,
	)
	if err != nil {
		return orderdom.OrderItemSnapshot{}, err
	}

	if list.Status != listdom.StatusListing {
		return orderdom.OrderItemSnapshot{},
			orderdom.ErrInvalidItemSnapshot
	}

	inventory, err := u.inventoryRepo.GetByID(
		ctx,
		list.InventoryID,
	)
	if err != nil {
		return orderdom.OrderItemSnapshot{}, err
	}

	if inventory.ProductBlueprintID == "" ||
		inventory.TokenBlueprintID == "" {
		return orderdom.OrderItemSnapshot{},
			orderdom.ErrInvalidItemSnapshot
	}

	productBlueprintCategoryPath,
		consumptionTaxRate,
		err :=
		u.resolveProductBlueprintTaxSnapshot(
			ctx,
			inventory.ProductBlueprintID,
		)
	if err != nil {
		return orderdom.OrderItemSnapshot{}, err
	}

	sellerSnapshot, err :=
		u.resolveSellerSnapshotByProductBlueprintID(
			ctx,
			inventory.ProductBlueprintID,
		)
	if err != nil {
		return orderdom.OrderItemSnapshot{}, err
	}

	stock, ok := inventory.Stock[item.ModelID]
	if !ok {
		return orderdom.OrderItemSnapshot{},
			orderdom.ErrInvalidItemSnapshot
	}

	available := stock.Accumulation - stock.ReservedCount
	if available < item.Qty {
		return orderdom.OrderItemSnapshot{},
			orderdom.ErrInvalidItemSnapshot
	}

	price, err := resolveListModelPrice(
		list,
		item.ModelID,
	)
	if err != nil {
		return orderdom.OrderItemSnapshot{}, err
	}

	return orderdom.OrderItemSnapshot{
		Type:               orderdom.OrderItemTypeList,
		ModelID:            item.ModelID,
		InventoryID:        list.InventoryID,
		ListID:             list.ID,
		ProductBlueprintID: inventory.ProductBlueprintID,
		TokenBlueprintID:   inventory.TokenBlueprintID,

		SellerSnapshot: sellerSnapshot,

		ProductBlueprintCategoryPath: productBlueprintCategoryPath,
		ConsumptionTaxRate:           consumptionTaxRate,

		Qty:           item.Qty,
		Price:         price,
		IsCancelled:   false,
		IsDispatched:  false,
		Transferred:   false,
		TransferredAt: nil,
	}, nil
}

func resolveListModelPrice(
	list listdom.List,
	modelID string,
) (int, error) {
	for _, price := range list.Prices {
		if price.ModelID == modelID {
			return price.Price, nil
		}
	}

	return 0, orderdom.ErrInvalidItemSnapshot
}

func (u *OrderUsecase) resolveResaleOrderItem(
	ctx context.Context,
	item CreateOrderItemInput,
) (orderdom.OrderItemSnapshot, error) {
	if item.ResaleID == "" {
		return orderdom.OrderItemSnapshot{},
			orderdom.ErrInvalidItemSnapshot
	}

	resale, err := u.resaleRepo.GetByID(
		ctx,
		item.ResaleID,
	)
	if err != nil {
		return orderdom.OrderItemSnapshot{}, err
	}

	if resale.Status != resaledom.StatusListing {
		return orderdom.OrderItemSnapshot{},
			orderdom.ErrInvalidItemSnapshot
	}

	// A resale BrandID identifies the product brand, not the resale seller.
	// Company Brand.Account must therefore never be used as the payout
	// destination for a consumer resale. A separate resale seller payout
	// destination must be implemented before resale checkout is enabled.
	return orderdom.OrderItemSnapshot{},
		orderdom.ErrInvalidSellerSnapshot
}

// =======================
// ID generation
// =======================

func (u *OrderUsecase) newOrderID(t time.Time) string {
	return fmt.Sprintf(
		"ord_%d",
		t.UTC().UnixNano(),
	)
}
