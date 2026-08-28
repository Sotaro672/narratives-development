// backend/internal/application/usecase/order_create.go
package usecase

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	orderdom "narratives/internal/domain/order"
)

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

// =======================
// ID generation
// =======================

func (u *OrderUsecase) newOrderID(t time.Time) string {
	return fmt.Sprintf(
		"ord_%d",
		t.UTC().UnixNano(),
	)
}
