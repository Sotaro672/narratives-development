// backend/internal/application/usecase/order_usecase.go
package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	cartdom "narratives/internal/domain/cart"
	common "narratives/internal/domain/common"
	orderdom "narratives/internal/domain/order"
)

// OrderRepo is the persistence port required by OrderUsecase.
type OrderRepo interface {
	// Queries
	GetByID(
		ctx context.Context,
		id string,
	) (orderdom.Order, error)

	ListByAvatarID(
		ctx context.Context,
		avatarID string,
		sort common.Sort,
		page common.Page,
	) (common.PageResult[orderdom.Order], error)

	// Commands
	Create(
		ctx context.Context,
		order orderdom.Order,
	) (orderdom.Order, error)

	Update(
		ctx context.Context,
		order orderdom.Order,
		opts *common.SaveOptions,
	) (orderdom.Order, error)
}

// OrderUsecase orchestrates order operations.
//
// - /mall/me/orders は Order テーブル起票のみ
// - Invoice 起票は /mall/me/invoices の責務
// - Payment 起票は /mall/me/payment(s) の責務
type OrderUsecase struct {
	repo     OrderRepo
	cartRepo cartdom.Repository
	now      func() time.Time
}

func NewOrderUsecase(
	repo OrderRepo,
	cartRepo cartdom.Repository,
) *OrderUsecase {
	return &OrderUsecase{
		repo:     repo,
		cartRepo: cartRepo,
		now:      time.Now,
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

type CreateOrderInput struct {
	ID       string
	UserID   string
	AvatarID string
	CartID   string

	ShippingSnapshot      orderdom.ShippingSnapshot
	PaymentMethodSnapshot orderdom.PaymentMethodSnapshot
	Items                 []orderdom.OrderItemSnapshot

	CreatedAt *time.Time // optional; defaults to now
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

	// IDはdomainで必須。未指定ならここで生成してからNewする。
	id := in.ID
	if id == "" {
		id = u.newOrderID(now)
	}

	shipping := orderdom.ShippingSnapshot{
		ZipCode: in.ShippingSnapshot.ZipCode,
		State:   in.ShippingSnapshot.State,
		City:    in.ShippingSnapshot.City,
		Street:  in.ShippingSnapshot.Street,
		Street2: in.ShippingSnapshot.Street2,
		Country: in.ShippingSnapshot.Country,
	}

	paymentMethod := orderdom.PaymentMethodSnapshot{
		CustomerID:     in.PaymentMethodSnapshot.CustomerID,
		Brand:          in.PaymentMethodSnapshot.Brand,
		Last4:          in.PaymentMethodSnapshot.Last4,
		ExpMonth:       in.PaymentMethodSnapshot.ExpMonth,
		ExpYear:        in.PaymentMethodSnapshot.ExpYear,
		CardholderName: in.PaymentMethodSnapshot.CardholderName,
		IsDefault:      in.PaymentMethodSnapshot.IsDefault,
	}

	// --- fetch cart to resolve listId for list items only ---
	cartID := in.CartID

	var cart cartdom.Cart
	cartLoaded := false

	if u.cartRepo != nil && cartID != "" {
		foundCart, err := u.cartRepo.GetByAvatarID(
			ctx,
			cartID,
		)
		if err != nil {
			return orderdom.Order{}, err
		}

		if foundCart != nil {
			cart = *foundCart
			cartLoaded = true
		}
	}

	items, err := normalizeOrderItems(
		in.Items,
		cart,
		cartLoaded,
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
		paymentMethod,
		items,
		createdAt,
	)
	if err != nil {
		return orderdom.Order{}, err
	}

	order.Paid = false

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

	ShippingSnapshot      *orderdom.ShippingSnapshot
	PaymentMethodSnapshot *orderdom.PaymentMethodSnapshot

	ReplaceItems *[]orderdom.OrderItemSnapshot
}

func (u *OrderUsecase) Update(
	ctx context.Context,
	in UpdateOrderInput,
) (orderdom.Order, error) {
	order, err := u.repo.GetByID(ctx, in.ID)
	if err != nil {
		return orderdom.Order{}, err
	}

	if order.CreatedAt.IsZero() {
		order.CreatedAt = u.now().UTC()
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

	if in.ShippingSnapshot != nil {
		shipping := orderdom.ShippingSnapshot{
			ZipCode: in.ShippingSnapshot.ZipCode,
			State:   in.ShippingSnapshot.State,
			City:    in.ShippingSnapshot.City,
			Street:  in.ShippingSnapshot.Street,
			Street2: in.ShippingSnapshot.Street2,
			Country: in.ShippingSnapshot.Country,
		}

		if err := order.UpdateShippingSnapshot(shipping); err != nil {
			return orderdom.Order{}, err
		}
	}

	if in.PaymentMethodSnapshot != nil {
		paymentMethod := orderdom.PaymentMethodSnapshot{
			CustomerID:     in.PaymentMethodSnapshot.CustomerID,
			Brand:          in.PaymentMethodSnapshot.Brand,
			Last4:          in.PaymentMethodSnapshot.Last4,
			ExpMonth:       in.PaymentMethodSnapshot.ExpMonth,
			ExpYear:        in.PaymentMethodSnapshot.ExpYear,
			CardholderName: in.PaymentMethodSnapshot.CardholderName,
			IsDefault:      in.PaymentMethodSnapshot.IsDefault,
		}

		if err := order.UpdatePaymentMethodSnapshot(
			paymentMethod,
		); err != nil {
			return orderdom.Order{}, err
		}
	}

	if in.ReplaceItems != nil {
		cartID := order.CartID

		var cart cartdom.Cart
		cartLoaded := false

		if u.cartRepo != nil && cartID != "" {
			foundCart, err := u.cartRepo.GetByAvatarID(
				ctx,
				cartID,
			)
			if err != nil {
				return orderdom.Order{}, err
			}

			if foundCart != nil {
				cart = *foundCart
				cartLoaded = true
			}
		}

		items, err := normalizeOrderItems(
			*in.ReplaceItems,
			cart,
			cartLoaded,
		)
		if err != nil {
			return orderdom.Order{}, err
		}

		if err := order.ReplaceItems(items); err != nil {
			return orderdom.Order{}, err
		}
	}

	checked, err := orderdom.New(
		order.ID,
		order.UserID,
		order.AvatarID,
		order.CartID,
		order.ShippingSnapshot,
		order.PaymentMethodSnapshot,
		order.Items,
		order.CreatedAt,
	)
	if err != nil {
		return orderdom.Order{}, err
	}

	checked.Paid = order.Paid

	if in.ReplaceItems == nil {
		checked.Items = order.Items
	}

	return u.repo.Update(ctx, checked, nil)
}

// ------------------------------------------------------------
// item normalization
// ------------------------------------------------------------

func normalizeOrderItems(
	input []orderdom.OrderItemSnapshot,
	cart cartdom.Cart,
	cartLoaded bool,
) ([]orderdom.OrderItemSnapshot, error) {
	items := make(
		[]orderdom.OrderItemSnapshot,
		0,
		len(input),
	)

	for _, item := range input {
		switch inferOrderInputItemType(item) {
		case orderdom.OrderItemTypeList:
			normalized, err := normalizeListOrderItem(
				item,
				cart,
				cartLoaded,
			)
			if err != nil {
				return nil, err
			}

			items = append(items, normalized)

		case orderdom.OrderItemTypeResale:
			normalized, err := normalizeResaleOrderItem(item)
			if err != nil {
				return nil, err
			}

			items = append(items, normalized)

		default:
			return nil, orderdom.ErrInvalidItemSnapshot
		}
	}

	return items, nil
}

func normalizeListOrderItem(
	item orderdom.OrderItemSnapshot,
	cart cartdom.Cart,
	cartLoaded bool,
) (orderdom.OrderItemSnapshot, error) {
	modelID := item.ModelID
	inventoryID := item.InventoryID

	listID := item.ListID
	if listID == "" && cartLoaded {
		resolved, err := resolveListIDFromCart(
			cart,
			inventoryID,
			modelID,
		)
		if err != nil {
			return orderdom.OrderItemSnapshot{}, err
		}

		listID = resolved
	}

	return orderdom.OrderItemSnapshot{
		Type:          orderdom.OrderItemTypeList,
		ModelID:       modelID,
		InventoryID:   inventoryID,
		ListID:        listID,
		Qty:           item.Qty,
		Price:         item.Price,
		IsCanceled:    false,
		IsDispatched:  false,
		Transferred:   false,
		TransferredAt: nil,
	}, nil
}

func normalizeResaleOrderItem(
	item orderdom.OrderItemSnapshot,
) (orderdom.OrderItemSnapshot, error) {
	qty := item.Qty
	if qty <= 0 {
		qty = 1
	}

	if qty != 1 {
		return orderdom.OrderItemSnapshot{},
			orderdom.ErrInvalidItemSnapshot
	}

	return orderdom.OrderItemSnapshot{
		Type:               orderdom.OrderItemTypeResale,
		ResaleID:           item.ResaleID,
		ProductID:          item.ProductID,
		ProductBlueprintID: item.ProductBlueprintID,
		TokenBlueprintID:   item.TokenBlueprintID,
		BrandID:            item.BrandID,
		Qty:                1,
		Price:              item.Price,
		IsCanceled:         false,
		IsDispatched:       false,
		Transferred:        false,
		TransferredAt:      nil,
	}, nil
}

func inferOrderInputItemType(
	item orderdom.OrderItemSnapshot,
) orderdom.OrderItemType {
	switch item.Type {
	case orderdom.OrderItemTypeList,
		orderdom.OrderItemTypeResale:
		return item.Type
	}

	if item.ResaleID != "" || item.ProductID != "" {
		return orderdom.OrderItemTypeResale
	}

	if item.ModelID != "" ||
		item.InventoryID != "" ||
		item.ListID != "" {
		return orderdom.OrderItemTypeList
	}

	return ""
}

// ------------------------------------------------------------
// listId resolution
// ------------------------------------------------------------

// resolveListIDFromCart finds listId for (inventoryId, modelId)
// from cart items.
//
// If multiple listIds match, returns an error (ambiguous).
func resolveListIDFromCart(
	cart cartdom.Cart,
	inventoryID string,
	modelID string,
) (string, error) {
	if inventoryID == "" || modelID == "" {
		return "",
			fmt.Errorf(
				"order_uc: invalid inventoryId/modelId for listId resolution",
			)
	}

	if len(cart.Items) == 0 {
		return "",
			fmt.Errorf(
				"order_uc: cart has no items (cannot resolve listId)",
			)
	}

	found := ""

	for _, item := range cart.Items {
		if item.InventoryID != inventoryID ||
			item.ModelID != modelID {
			continue
		}

		listID := item.ListID
		if listID == "" {
			continue
		}

		if found == "" {
			found = listID
			continue
		}

		if found != listID {
			return "",
				fmt.Errorf(
					"order_uc: ambiguous listId for inv=%s model=%s",
					inventoryID,
					modelID,
				)
		}
	}

	if found == "" {
		return "",
			fmt.Errorf(
				"order_uc: listId not found in cart for inv=%s model=%s",
				inventoryID,
				modelID,
			)
	}

	return found, nil
}

// ------------------------------------------------------------
// ID generation
// ------------------------------------------------------------

// newOrderID generates an order id when client didn't specify one.
func (u *OrderUsecase) newOrderID(t time.Time) string {
	return fmt.Sprintf(
		"ord_%d",
		t.UTC().UnixNano(),
	)
}
