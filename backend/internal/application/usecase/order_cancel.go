// backend/internal/application/usecase/order_cancel.go
package usecase

import (
	"context"
	"log"
	"strings"

	orderdom "narratives/internal/domain/order"
)

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

		if err := order.CancelItem(in.ItemIndex); err != nil {
			return orderdom.Order{}, err
		}

		// キャンセル後も有効な商品が残っている場合のみ送料を再計算する。
		// 全商品がキャンセル済みの場合は既存のShippingQuoteSnapshotを
		// 注文時点の履歴として保持する。
		hasActiveItems := false
		for _, item := range order.Items {
			if !item.IsCancelled {
				hasActiveItems = true
				break
			}
		}

		if hasActiveItems {
			shippingAddressID, err :=
				resolveOrderDestinationShippingAddressID(
					order.ShippingQuoteSnapshot,
				)
			if err != nil {
				return orderdom.Order{}, err
			}

			shippingQuoteItems, err :=
				createOrderItemInputsFromSnapshots(
					order.Items,
				)
			if err != nil {
				return orderdom.Order{}, err
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

		u.sendResaleOrderCancellationNotificationBestEffort(
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

func (u *OrderUsecase) sendResaleOrderCancellationNotificationBestEffort(
	ctx context.Context,
	order orderdom.Order,
	itemIndex int,
) {
	if u == nil ||
		u.authUserReader == nil ||
		u.resaleOrderNotificationMailer == nil {
		return
	}

	if itemIndex < 0 || itemIndex >= len(order.Items) {
		return
	}

	item := order.Items[itemIndex]
	if item.Type != orderdom.OrderItemTypeResale {
		return
	}

	sellerUserID := item.SellerSnapshot.UserID
	resaleID := item.ResaleID
	if sellerUserID == "" || resaleID == "" {
		return
	}

	toEmail, err := u.authUserReader.GetEmailByUID(
		ctx,
		sellerUserID,
	)
	if err != nil || toEmail == "" {
		return
	}

	_ = u.resaleOrderNotificationMailer.SendResaleOrderCancellationNotification(
		ctx,
		toEmail,
		order.ID,
		itemIndex,
		resaleID,
	)
}
