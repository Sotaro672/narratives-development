// backend/internal/application/usecase/order_usecase.go
package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	common "narratives/internal/domain/common"
	inventorydom "narratives/internal/domain/inventory"
	listdom "narratives/internal/domain/list"
	orderdom "narratives/internal/domain/order"
	paymentmethoddom "narratives/internal/domain/paymentMethod"
	resaledom "narratives/internal/domain/resale"
	shippingaddressdom "narratives/internal/domain/shippingAddress"
)

// OrderUsecase orchestrates order operations.
//
// - /mall/me/orders は Order の取得・作成を担当する
// - Invoice の作成は /mall/me/invoices の責務
// - Payment の作成は /mall/me/payments の責務
type OrderUsecase struct {
	repo                orderdom.Repository
	listRepo            listdom.Repository
	inventoryRepo       inventorydom.RepositoryPort
	resaleRepo          resaledom.Repository
	paymentMethodRepo   paymentmethoddom.RepositoryPort
	shippingAddressRepo shippingaddressdom.RepositoryPort
	shippingQuoteUC     *ShippingQuoteUsecase
	now                 func() time.Time
}

func NewOrderUsecase(
	repo orderdom.Repository,
	listRepo listdom.Repository,
	inventoryRepo inventorydom.RepositoryPort,
	resaleRepo resaledom.Repository,
	paymentMethodRepo paymentmethoddom.RepositoryPort,
	shippingAddressRepo shippingaddressdom.RepositoryPort,
	shippingQuoteUC *ShippingQuoteUsecase,
) *OrderUsecase {
	return &OrderUsecase{
		repo:                repo,
		listRepo:            listRepo,
		inventoryRepo:       inventoryRepo,
		resaleRepo:          resaleRepo,
		paymentMethodRepo:   paymentMethodRepo,
		shippingAddressRepo: shippingAddressRepo,
		shippingQuoteUC:     shippingQuoteUC,
		now:                 time.Now,
	}
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
// TokenBlueprintID and BrandID are resolved from server-side repositories.
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
	IsCanceled   bool
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

	shippingAddressID :=
		strings.TrimSpace(
			in.ShippingAddressID,
		)

	shipping, err :=
		u.resolveShippingSnapshot(
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

	shippingQuote, err :=
		u.resolveShippingQuoteSnapshot(
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

	shippingAddressID, err :=
		resolveOrderDestinationShippingAddressID(
			order.ShippingQuoteSnapshot,
		)
	if err != nil {
		return orderdom.Order{}, err
	}

	if in.ShippingAddressID != nil {
		shippingAddressID =
			strings.TrimSpace(
				*in.ShippingAddressID,
			)

		if shippingAddressID == "" {
			return orderdom.Order{},
				orderdom.ErrInvalidShippingSnapshot
		}
	}

	shouldRefreshShipping :=
		in.ShippingAddressID != nil ||
			in.UserID != nil

	if shouldRefreshShipping {
		shipping, err :=
			u.resolveShippingSnapshot(
				ctx,
				order.UserID,
				shippingAddressID,
			)
		if err != nil {
			return orderdom.Order{}, err
		}

		if err :=
			order.UpdateShippingSnapshot(
				shipping,
			); err != nil {
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

		shippingQuoteItems =
			append(
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

		if err :=
			order.UpdateShippingQuoteSnapshot(
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

	userID =
		strings.TrimSpace(
			userID,
		)

	shippingAddressID =
		strings.TrimSpace(
			shippingAddressID,
		)

	if userID == "" ||
		shippingAddressID == "" {
		return orderdom.ShippingSnapshot{},
			orderdom.ErrInvalidShippingSnapshot
	}

	address, err :=
		u.shippingAddressRepo.GetByUser(
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

	if address.ID !=
		shippingAddressID {
		return orderdom.ShippingSnapshot{},
			shippingaddressdom.ErrNotFound
	}

	if address.UserID !=
		userID {
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

	userID =
		strings.TrimSpace(
			userID,
		)

	shippingAddressID =
		strings.TrimSpace(
			shippingAddressID,
		)

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

	maxInt :=
		int(^uint(0) >> 1)

	total := 0

	for _, item := range input {
		if item.Type !=
			orderdom.OrderItemTypeList {
			return orderdom.ShippingQuoteSnapshot{},
				orderdom.ErrInvalidShippingQuote
		}

		if item.ListID == "" ||
			item.ModelID == "" ||
			item.Qty <= 0 {
			return orderdom.ShippingQuoteSnapshot{},
				orderdom.ErrInvalidShippingQuoteItem
		}

		quote, err :=
			u.shippingQuoteUC.Quote(
				ctx,
				ShippingQuoteInput{
					UserID: userID,

					ListID: item.ListID,

					ModelID: item.ModelID,

					DestinationShippingAddressID: shippingAddressID,
				},
			)
		if err != nil {
			return orderdom.ShippingQuoteSnapshot{},
				err
		}

		if quote.Amount < 0 ||
			quote.Amount >
				int64(maxInt) {
			return orderdom.ShippingQuoteSnapshot{},
				orderdom.ErrInvalidShippingQuoteItem
		}

		unitAmount :=
			int(
				quote.Amount,
			)

		if unitAmount > 0 &&
			item.Qty >
				maxInt/unitAmount {
			return orderdom.ShippingQuoteSnapshot{},
				orderdom.ErrInvalidShippingQuoteItem
		}

		lineAmount :=
			unitAmount *
				item.Qty

		if total >
			maxInt-lineAmount {
			return orderdom.ShippingQuoteSnapshot{},
				orderdom.ErrInvalidShippingQuote
		}

		total +=
			lineAmount

		quoteItems =
			append(
				quoteItems,
				orderdom.ShippingQuoteItemSnapshot{
					ListID: quote.ListID,

					InventoryID: quote.InventoryID,

					ModelID: quote.ModelID,

					OriginShippingAddressID: quote.OriginShippingAddressID,

					DestinationShippingAddressID: quote.DestinationShippingAddressID,

					Carrier: string(
						quote.TransportationOption,
					),

					TransportationID: quote.TransportationID,

					Size: quote.Size,

					Qty: item.Qty,

					UnitAmount: unitAmount,

					Amount: lineAmount,

					Currency: quote.Currency,
				},
			)
	}

	return orderdom.ShippingQuoteSnapshot{
		Items: quoteItems,

		Amount: total,

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

	destinationShippingAddressID :=
		strings.TrimSpace(
			snapshot.Items[0].
				DestinationShippingAddressID,
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

	return destinationShippingAddressID,
		nil
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
		switch item.Type {
		case orderdom.OrderItemTypeList:
			result =
				append(
					result,
					CreateOrderItemInput{
						Type: orderdom.OrderItemTypeList,

						ListID: item.ListID,

						ModelID: item.ModelID,

						Qty: item.Qty,

						IsCanceled: item.IsCanceled,

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

	if paymentMethod == nil || paymentMethod.UserID != userID {
		return orderdom.PaymentMethodSnapshot{},
			orderdom.ErrInvalidPaymentMethod
	}

	return orderdom.PaymentMethodSnapshot{
		CustomerID:     paymentMethod.StripeCustomerID,
		Brand:          paymentMethod.Brand,
		Last4:          paymentMethod.Last4,
		ExpMonth:       paymentMethod.ExpMonth,
		ExpYear:        paymentMethod.ExpYear,
		CardholderName: paymentMethod.CardholderName,
		IsDefault:      paymentMethod.IsDefault,
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
		Qty:                item.Qty,
		Price:              price,
		IsCanceled:         false,
		IsDispatched:       false,
		Transferred:        false,
		TransferredAt:      nil,
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

	return orderdom.OrderItemSnapshot{
		Type:               orderdom.OrderItemTypeResale,
		ResaleID:           resale.ID,
		ProductID:          resale.ProductID,
		ProductBlueprintID: resale.ProductBlueprintID,
		TokenBlueprintID:   resale.TokenBlueprintID,
		BrandID:            resale.BrandID,
		Qty:                1,
		Price:              resale.Price,
		IsCanceled:         false,
		IsDispatched:       false,
		Transferred:        false,
		TransferredAt:      nil,
	}, nil
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
