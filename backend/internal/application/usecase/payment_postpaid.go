// backend/internal/application/usecase/payment_postpaid.go
package usecase

/*
責任と機能:
- Payment が succeeded になった後の「後続処理のオーケストレーション」を担う。
  前提: payment / order の docId は同一（= rootID）である。
  具体的には:
  0) order.Paid=true 更新
  1) 注文確定メール送信
  2) inventory reserve 更新（best-effort）
  3) cart clear（best-effort）
- 外部依存（cartRepo/inventoryRepo/orderRepo/userRepo/mailSender）は PaymentUsecase が保持し、
  このファイルは処理順序と優先順位を管理する。
- payment.PaymentID は order.ID と同じ値として扱う。
*/

import (
	"context"
	"log"

	orderdom "narratives/internal/domain/order"
	paymentdom "narratives/internal/domain/payment"
)

func isPaidStatus(st paymentdom.PaymentStatus) bool {
	return st == paymentdom.PaymentStatusSucceeded
}

// handlePostPaidBestEffort runs post-paid side effects in best-effort manner.
//
// 前提: payment / order の docId は同じ。
// したがって rootID = payment.PaymentID をそのまま使う。
// payment.PaymentID は order.ID と同じ値である。
func (u *PaymentUsecase) handlePostPaidBestEffort(ctx context.Context, p *paymentdom.Payment) {
	if u == nil || p == nil {
		return
	}

	rootID := p.PaymentID
	if rootID == "" {
		return
	}

	var ord *orderdom.Order
	if u.orderRepo != nil {
		o, getErr := u.orderRepo.GetByID(ctx, rootID)
		if getErr != nil {
			log.Printf("[payment_uc] WARN: order fetch failed orderId=%s err=%v", rootID, getErr)
		} else {
			ord = &o
		}
	}

	// 0) order.Paid=true
	if u.orderRepo != nil {
		updatedOrder, mkErr := u.markOrderPaidTrue(ctx, rootID, ord)
		if mkErr != nil {
			log.Printf("[payment_uc] WARN: order mark paid failed orderId=%s err=%v", rootID, mkErr)
		} else if updatedOrder != nil {
			ord = updatedOrder
		}
	}

	// 1) 注文確定メール送信
	if ord != nil && u.userRepo != nil && u.mailSender != nil && u.mailFrom != "" {
		if mailErr := u.sendOrderConfirmationMail(ctx, *ord); mailErr != nil {
			log.Printf("[payment_uc] WARN: order confirmation mail failed orderId=%s userId=%s err=%v", ord.ID, ord.UserID, mailErr)
		} else {
			log.Printf("[payment_uc] order confirmation mail sent orderId=%s userId=%s", ord.ID, ord.UserID)
		}
	}

	// 2) inventory reserve
	if u.inventoryRepo != nil && ord != nil {
		rawItems := extractOrderItems(*ord)
		agg := aggregateReserveItems(rawItems)

		for _, it := range agg {
			invID := normalizeInventoryDocIDBestEffort(it.InventoryID)
			if invID == "" || it.ModelID == "" || it.Qty <= 0 {
				continue
			}

			if rErr := u.inventoryRepo.ReserveByOrder(ctx, invID, it.ModelID, rootID, it.Qty); rErr != nil {
				log.Printf(
					"[payment_uc] WARN: inventory reserve failed inventoryId=%s modelId=%s orderId=%s qty=%d err=%v",
					invID,
					it.ModelID,
					rootID,
					it.Qty,
					rErr,
				)
			} else {
				log.Printf(
					"[payment_uc] inventory reserved inventoryId=%s modelId=%s orderId=%s qty=%d",
					invID,
					it.ModelID,
					rootID,
					it.Qty,
				)
			}
		}
	}

	// 3) cart clear
	if u.cartRepo != nil && ord != nil {
		cartID := ord.CartID
		if cartID == "" {
			log.Printf("[payment_uc] WARN: cartId empty (skip clear) rootId=%s", rootID)
		} else {
			if clrErr := u.cartRepo.Clear(ctx, cartID); clrErr != nil {
				log.Printf("[payment_uc] WARN: cart clear failed cartId=%s rootId=%s err=%v", cartID, rootID, clrErr)
			} else {
				log.Printf("[payment_uc] cart cleared cartId=%s rootId=%s", cartID, rootID)
			}
		}
	}
}

// ------------------------------------------------------------
// order.Paid = true
// ------------------------------------------------------------

func (u *PaymentUsecase) markOrderPaidTrue(
	ctx context.Context,
	orderID string,
	ord *orderdom.Order,
) (*orderdom.Order, error) {
	if u == nil || u.orderRepo == nil {
		return ord, nil
	}
	if orderID == "" {
		return ord, nil
	}

	var current orderdom.Order
	if ord != nil {
		current = *ord
	} else {
		fetched, err := u.orderRepo.GetByID(ctx, orderID)
		if err != nil {
			return nil, err
		}
		current = fetched
	}

	if current.Paid {
		return &current, nil
	}

	current.Paid = true

	saved, err := u.orderRepo.Save(ctx, current, nil)
	if err != nil {
		return nil, err
	}

	return &saved, nil
}

// ------------------------------------------------------------
// mail
// ------------------------------------------------------------

func (u *PaymentUsecase) sendOrderConfirmationMail(ctx context.Context, ord orderdom.Order) error {
	if u == nil || u.userRepo == nil || u.mailSender == nil || u.mailFrom == "" {
		return nil
	}

	if ord.ID == "" || ord.UserID == "" {
		return nil
	}

	to, err := u.userRepo.GetEmailByID(ctx, ord.UserID)
	if err != nil {
		return err
	}
	if to == "" {
		return nil
	}

	return u.mailSender.SendOrderConfirmation(ctx, u.mailFrom, to, ord)
}
