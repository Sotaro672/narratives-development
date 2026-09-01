// backend/internal/application/usecase/order_create.go
package usecase

import (
	"context"
	"fmt"
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

	// resale商品の注文受付が確定した時点で、出品者へ発注通知メールを送る。
	// Orderは既に永続化済みのため、メール送信はbest-effortとする。
	u.sendResaleOrderNotificationsBestEffort(
		ctx,
		created,
	)

	// 注文作成が確定した時点で、注文元のcartを削除する。
	// Orderは既に永続化済みのため、cart削除はbest-effortとする。
	if u.cartRepo != nil {
		cartID := created.CartID

		if cartID != "" {
			_ = u.cartRepo.DeleteByAvatarID(
				ctx,
				cartID,
			)
		}
	}

	return created, nil
}

func (u *OrderUsecase) sendResaleOrderNotificationsBestEffort(
	ctx context.Context,
	order orderdom.Order,
) {
	if u == nil ||
		u.authUserReader == nil ||
		u.resaleOrderNotificationMailer == nil {
		return
	}

	for itemIndex, item := range order.Items {
		if item.Type != orderdom.OrderItemTypeResale {
			continue
		}

		if item.IsCancelled {
			continue
		}

		resaleID := item.ResaleID
		sellerUserID := item.SellerSnapshot.UserID

		if resaleID == "" || sellerUserID == "" {
			continue
		}

		toEmail, err := u.authUserReader.GetEmailByUID(
			ctx,
			sellerUserID,
		)
		if err != nil || toEmail == "" {
			continue
		}

		_ = u.resaleOrderNotificationMailer.SendResaleOrderNotification(
			ctx,
			toEmail,
			order.ID,
			itemIndex,
			resaleID,
			item.Price,
		)
	}

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
