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
	GetByID(ctx context.Context, id string) (orderdom.Order, error)
	ListByAvatarID(ctx context.Context, avatarID string, sort common.Sort, page common.Page) (common.PageResult[orderdom.Order], error)

	// Commands
	Create(ctx context.Context, o orderdom.Order) (orderdom.Order, error)
	Update(ctx context.Context, o orderdom.Order, opts *common.SaveOptions) (orderdom.Order, error)
}

// CartRepo is the persistence port required to read cart items (for listId lookup).
type CartRepo interface {
	GetByID(ctx context.Context, id string) (cartdom.Cart, error)
}

// OrderUsecase orchestrates order operations.
//
// - /mall/me/orders は Order テーブル起票のみ
// - Invoice 起票は /mall/me/invoices の責務
// - Payment 起票は /mall/me/payment(s) の責務
type OrderUsecase struct {
	repo     OrderRepo
	cartRepo CartRepo
	now      func() time.Time
}

func NewOrderUsecase(repo OrderRepo, cartRepo CartRepo) *OrderUsecase {
	return &OrderUsecase{
		repo:     repo,
		cartRepo: cartRepo,
		now:      time.Now,
	}
}

// =======================
// Queries
// =======================

func (u *OrderUsecase) GetByID(ctx context.Context, id string) (orderdom.Order, error) {
	return u.repo.GetByID(ctx, id)
}

func (u *OrderUsecase) ListByAvatarID(
	ctx context.Context,
	avatarID string,
	s common.Sort,
	p common.Page,
) (common.PageResult[orderdom.Order], error) {
	avatarID = strings.TrimSpace(avatarID)
	if avatarID == "" {
		return common.PageResult[orderdom.Order]{}, fmt.Errorf("order usecase: avatarId is required")
	}

	return u.repo.ListByAvatarID(ctx, avatarID, s, p)
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

func (u *OrderUsecase) Create(ctx context.Context, in CreateOrderInput) (orderdom.Order, error) {
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

	ship := orderdom.ShippingSnapshot{
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
		c, err := u.cartRepo.GetByID(ctx, cartID)
		if err != nil {
			return orderdom.Order{}, err
		}
		cart = c
		cartLoaded = true
	}

	items, err := normalizeOrderItems(in.Items, cart, cartLoaded)
	if err != nil {
		return orderdom.Order{}, err
	}

	o, err := orderdom.New(
		id,
		in.UserID,
		in.AvatarID,
		in.CartID,
		ship,
		paymentMethod,
		items,
		createdAt,
	)
	if err != nil {
		return orderdom.Order{}, err
	}

	o.Paid = false

	created, err := u.repo.Create(ctx, o)
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

func (u *OrderUsecase) Update(ctx context.Context, in UpdateOrderInput) (orderdom.Order, error) {
	o, err := u.repo.GetByID(ctx, in.ID)
	if err != nil {
		return orderdom.Order{}, err
	}

	if o.CreatedAt.IsZero() {
		o.CreatedAt = u.now().UTC()
	}

	if in.UserID != nil {
		o.UserID = *in.UserID
	}
	if in.AvatarID != nil {
		o.AvatarID = *in.AvatarID
	}
	if in.CartID != nil {
		o.CartID = *in.CartID
	}

	if in.ShippingSnapshot != nil {
		s := orderdom.ShippingSnapshot{
			ZipCode: in.ShippingSnapshot.ZipCode,
			State:   in.ShippingSnapshot.State,
			City:    in.ShippingSnapshot.City,
			Street:  in.ShippingSnapshot.Street,
			Street2: in.ShippingSnapshot.Street2,
			Country: in.ShippingSnapshot.Country,
		}
		if err := o.UpdateShippingSnapshot(s); err != nil {
			return orderdom.Order{}, err
		}
	}

	if in.PaymentMethodSnapshot != nil {
		p := orderdom.PaymentMethodSnapshot{
			CustomerID:     in.PaymentMethodSnapshot.CustomerID,
			Brand:          in.PaymentMethodSnapshot.Brand,
			Last4:          in.PaymentMethodSnapshot.Last4,
			ExpMonth:       in.PaymentMethodSnapshot.ExpMonth,
			ExpYear:        in.PaymentMethodSnapshot.ExpYear,
			CardholderName: in.PaymentMethodSnapshot.CardholderName,
			IsDefault:      in.PaymentMethodSnapshot.IsDefault,
		}
		if err := o.UpdatePaymentMethodSnapshot(p); err != nil {
			return orderdom.Order{}, err
		}
	}

	if in.ReplaceItems != nil {
		cartID := o.CartID
		var cart cartdom.Cart
		cartLoaded := false
		if u.cartRepo != nil && cartID != "" {
			c, err := u.cartRepo.GetByID(ctx, cartID)
			if err != nil {
				return orderdom.Order{}, err
			}
			cart = c
			cartLoaded = true
		}

		items, err := normalizeOrderItems(*in.ReplaceItems, cart, cartLoaded)
		if err != nil {
			return orderdom.Order{}, err
		}

		if err := o.ReplaceItems(items); err != nil {
			return orderdom.Order{}, err
		}
	}

	checked, err := orderdom.New(
		o.ID,
		o.UserID,
		o.AvatarID,
		o.CartID,
		o.ShippingSnapshot,
		o.PaymentMethodSnapshot,
		o.Items,
		o.CreatedAt,
	)
	if err != nil {
		return orderdom.Order{}, err
	}

	checked.Paid = o.Paid

	if in.ReplaceItems == nil {
		checked.Items = o.Items
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
	items := make([]orderdom.OrderItemSnapshot, 0, len(input))

	for _, it := range input {
		switch inferOrderInputItemType(it) {
		case orderdom.OrderItemTypeList:
			n, err := normalizeListOrderItem(it, cart, cartLoaded)
			if err != nil {
				return nil, err
			}
			items = append(items, n)

		case orderdom.OrderItemTypeResale:
			n, err := normalizeResaleOrderItem(it)
			if err != nil {
				return nil, err
			}
			items = append(items, n)

		default:
			return nil, orderdom.ErrInvalidItemSnapshot
		}
	}

	return items, nil
}

func normalizeListOrderItem(
	it orderdom.OrderItemSnapshot,
	cart cartdom.Cart,
	cartLoaded bool,
) (orderdom.OrderItemSnapshot, error) {
	modelID := it.ModelID
	inventoryID := it.InventoryID

	listID := it.ListID
	if listID == "" && cartLoaded {
		resolved, err := resolveListIDFromCart(cart, inventoryID, modelID)
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
		Qty:           it.Qty,
		Price:         it.Price,
		IsCanceled:    false,
		IsDispatched:  false,
		Transferred:   false,
		TransferredAt: nil,
	}, nil
}

func normalizeResaleOrderItem(it orderdom.OrderItemSnapshot) (orderdom.OrderItemSnapshot, error) {
	qty := it.Qty
	if qty <= 0 {
		qty = 1
	}

	if qty != 1 {
		return orderdom.OrderItemSnapshot{}, orderdom.ErrInvalidItemSnapshot
	}

	return orderdom.OrderItemSnapshot{
		Type:               orderdom.OrderItemTypeResale,
		ResaleID:           it.ResaleID,
		ProductID:          it.ProductID,
		ProductBlueprintID: it.ProductBlueprintID,
		TokenBlueprintID:   it.TokenBlueprintID,
		BrandID:            it.BrandID,
		Qty:                1,
		Price:              it.Price,
		IsCanceled:         false,
		IsDispatched:       false,
		Transferred:        false,
		TransferredAt:      nil,
	}, nil
}

func inferOrderInputItemType(it orderdom.OrderItemSnapshot) orderdom.OrderItemType {
	switch it.Type {
	case orderdom.OrderItemTypeList, orderdom.OrderItemTypeResale:
		return it.Type
	}

	if it.ResaleID != "" || it.ProductID != "" {
		return orderdom.OrderItemTypeResale
	}

	if it.ModelID != "" || it.InventoryID != "" || it.ListID != "" {
		return orderdom.OrderItemTypeList
	}

	return ""
}

// ------------------------------------------------------------
// listId resolution
// ------------------------------------------------------------

// resolveListIDFromCart finds listId for (inventoryId, modelId) from cart items.
// If multiple listIds match, returns an error (ambiguous).
func resolveListIDFromCart(c cartdom.Cart, inventoryID, modelID string) (string, error) {
	inv := inventoryID
	mid := modelID
	if inv == "" || mid == "" {
		return "", fmt.Errorf("order_uc: invalid inventoryId/modelId for listId resolution")
	}

	if len(c.Items) == 0 {
		return "", fmt.Errorf("order_uc: cart has no items (cannot resolve listId)")
	}

	found := ""
	for _, it := range c.Items {
		if it.InventoryID == inv && it.ModelID == mid {
			lid := it.ListID
			if lid == "" {
				continue
			}
			if found == "" {
				found = lid
				continue
			}
			if found != lid {
				return "", fmt.Errorf("order_uc: ambiguous listId for inv=%s model=%s", inv, mid)
			}
		}
	}

	if found == "" {
		return "", fmt.Errorf("order_uc: listId not found in cart for inv=%s model=%s", inv, mid)
	}
	return found, nil
}

// ------------------------------------------------------------
// ID generation
// ------------------------------------------------------------

// newOrderID generates an order id when client didn't specify one.
func (u *OrderUsecase) newOrderID(t time.Time) string {
	return fmt.Sprintf("ord_%d", t.UTC().UnixNano())
}
