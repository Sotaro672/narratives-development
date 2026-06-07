// backend/internal/application/usecase/payment_usecase.go
package usecase

/*
責任と機能:
- PaymentUsecase の公開API（Queries/Commands）と依存注入（DI）を提供する。
- Payment が succeeded になった後の「後続処理のオーケストレーション」を担う。

前提:
- payment document ID = payment.PaymentID
- payment.PaymentID = order.ID
- paymentId は payment document field としては保存しない
- payment records は削除しない
- payment updates は UpdateByPaymentID による partial update のみ

支払い成功後の後続処理:
0) order.Paid=true 更新
1) 注文確定メール送信
2) inventory reserve 更新（best-effort）
3) cart clear（best-effort）
*/

import (
	"context"
	"sort"
	"time"

	common "narratives/internal/domain/common"
	orderdom "narratives/internal/domain/order"
	paymentdom "narratives/internal/domain/payment"
	userdom "narratives/internal/domain/user"
)

// ============================================================
// Ports
// ============================================================

// PaymentRepo defines the minimal persistence port needed by PaymentUsecase.
//
// current design:
// - payment document ID = payment.PaymentID
// - payment.PaymentID must be the same value as order.ID
// - paymentId is NOT stored as a document field
// - payment records are not deleted
// - payment updates are performed by partial update only
type PaymentRepo interface {
	// Reads
	GetByPaymentID(ctx context.Context, paymentID string) (*paymentdom.Payment, error)

	// Writes
	Create(ctx context.Context, in paymentdom.CreatePaymentInput) (*paymentdom.Payment, error)
	UpdateByPaymentID(ctx context.Context, paymentID string, patch paymentdom.UpdatePaymentInput) (*paymentdom.Payment, error)
}

// CartRepoForPayment is the minimal port for clearing carts after payment.
//
// carts/{cartId} を空にする。
type CartRepoForPayment interface {
	Clear(ctx context.Context, cartID string) error
}

// InventoryRepoForPayment is the minimal port for inventory reserve after payment.
type InventoryRepoForPayment interface {
	// ReserveByOrder sets:
	// - stock[modelId].reservedByOrder[orderId] = qty
	// - reservedCount = sum(reservedByOrder) (repo側で正規化)
	ReserveByOrder(ctx context.Context, inventoryID string, modelID string, orderID string, qty int) error
}

// OrderRepoForPayment is the minimal port for reading/updating orders after payment.
type OrderRepoForPayment interface {
	GetByID(ctx context.Context, id string) (orderdom.Order, error)
	Update(ctx context.Context, o orderdom.Order, opts *common.SaveOptions) (orderdom.Order, error)
}

// MailSenderForPayment is the minimal port for sending order confirmation mail.
//
// メール件名・本文の組み立ては application/usecase では行わず、
// adapter 側の OrderMailer に分離する。
type MailSenderForPayment interface {
	SendOrderConfirmation(ctx context.Context, from, to string, ord orderdom.Order) error
}

// ============================================================
// Usecase
// ============================================================

// PaymentUsecase orchestrates payment operations.
type PaymentUsecase struct {
	repo          PaymentRepo
	cartRepo      CartRepoForPayment
	inventoryRepo InventoryRepoForPayment
	orderRepo     OrderRepoForPayment

	// userRepo は domain/user.RepositoryPort を正として接続する。
	// PaymentUsecase では主に GetEmailByID を利用する。
	userRepo   userdom.RepositoryPort
	mailSender MailSenderForPayment
	mailFrom   string

	now func() time.Time
}

type NewPaymentUsecaseInput struct {
	PaymentRepo PaymentRepo

	CartRepo      CartRepoForPayment
	InventoryRepo InventoryRepoForPayment
	OrderRepo     OrderRepoForPayment

	// UserRepo は domain/user.RepositoryPort を渡す。
	// これにより GetEmailByID も user repository port の契約に含まれる。
	UserRepo   userdom.RepositoryPort
	MailSender MailSenderForPayment
	MailFrom   string

	Now func() time.Time
}

func NewPaymentUsecase(in NewPaymentUsecaseInput) *PaymentUsecase {
	now := in.Now
	if now == nil {
		now = time.Now
	}

	return &PaymentUsecase{
		repo:          in.PaymentRepo,
		cartRepo:      in.CartRepo,
		inventoryRepo: in.InventoryRepo,
		orderRepo:     in.OrderRepo,

		userRepo:   in.UserRepo,
		mailSender: in.MailSender,
		mailFrom:   in.MailFrom,

		now: now,
	}
}

// ============================================================
// Queries
// ============================================================

func (u *PaymentUsecase) GetByPaymentID(ctx context.Context, paymentID string) (*paymentdom.Payment, error) {
	if u == nil || u.repo == nil {
		return nil, paymentdom.ErrNotFound
	}
	if paymentID == "" {
		return nil, paymentdom.ErrInvalidPaymentID
	}

	return u.repo.GetByPaymentID(ctx, paymentID)
}

// ============================================================
// Commands
// ============================================================

func (u *PaymentUsecase) Create(ctx context.Context, p paymentdom.Payment) (*paymentdom.Payment, error) {
	if u == nil || u.repo == nil {
		return nil, paymentdom.ErrNotFound
	}

	in := paymentdom.CreatePaymentInput{
		PaymentID:             p.PaymentID,
		PaymentMethodID:       p.PaymentMethodID,
		StripeCustomerID:      p.StripeCustomerID,
		StripePaymentMethodID: p.StripePaymentMethodID,
		StripePaymentIntentID: p.StripePaymentIntentID,
		Amount:                p.Amount,
		Status:                p.Status,
		ErrorType:             p.ErrorType,
		ErrorCode:             p.ErrorCode,
		ErrorMsg:              p.ErrorMsg,
	}

	created, err := u.repo.Create(ctx, in)
	if err != nil {
		return nil, err
	}

	if created != nil && isPaidStatus(created.Status) {
		u.handlePostPaidBestEffort(ctx, created)
	}

	return created, nil
}

// Update partially updates an existing payment document.
//
// This method is used by PaymentFlowUsecase after Stripe PaymentIntent state changes.
//
// Payment records are not overwritten by Save.
// Payment records are not deleted.
// Updates are applied through UpdateByPaymentID only.
func (u *PaymentUsecase) Update(
	ctx context.Context,
	paymentID string,
	patch paymentdom.UpdatePaymentInput,
) (*paymentdom.Payment, error) {
	if u == nil || u.repo == nil {
		return nil, paymentdom.ErrNotFound
	}
	if paymentID == "" {
		return nil, paymentdom.ErrInvalidPaymentID
	}

	updated, err := u.repo.UpdateByPaymentID(ctx, paymentID, patch)
	if err != nil {
		return nil, err
	}

	if updated != nil && isPaidStatus(updated.Status) {
		u.handlePostPaidBestEffort(ctx, updated)
	}

	return updated, nil
}

// ============================================================
// Paid status
// ============================================================

func isPaidStatus(st paymentdom.PaymentStatus) bool {
	return st == paymentdom.StatusSucceeded
}

// ============================================================
// Post-paid flow
// ============================================================

// handlePostPaidBestEffort runs post-paid side effects in best-effort manner.
//
// 前提:
// - payment / order の docId は同じ
// - rootID = payment.PaymentID
// - payment.PaymentID は order.ID と同じ値である
//
// 処理順:
// 0) order.Paid=true 更新
// 1) 注文確定メール送信
// 2) inventory reserve
// 3) cart clear
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
		if getErr == nil {
			ord = &o
		}
	}

	// 0) order.Paid=true
	if u.orderRepo != nil {
		updatedOrder, mkErr := u.markOrderPaidTrue(ctx, rootID, ord)
		if mkErr == nil && updatedOrder != nil {
			ord = updatedOrder
		}
	}

	// 1) 注文確定メール送信
	if ord != nil && u.userRepo != nil && u.mailSender != nil && u.mailFrom != "" {
		_ = u.sendOrderConfirmationMail(ctx, *ord)
	}

	// 2) inventory reserve
	if u.inventoryRepo != nil && ord != nil {
		rawItems := extractOrderItems(*ord)
		agg := aggregateReserveItems(rawItems)

		for _, it := range agg {
			invID := it.InventoryID
			if invID == "" || it.ModelID == "" || it.Qty <= 0 {
				continue
			}

			_ = u.inventoryRepo.ReserveByOrder(ctx, invID, it.ModelID, rootID, it.Qty)
		}
	}

	// 3) cart clear
	if u.cartRepo != nil && ord != nil {
		cartID := ord.CartID
		if cartID != "" {
			_ = u.cartRepo.Clear(ctx, cartID)
		}
	}
}

// ============================================================
// order.Paid = true
// ============================================================

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

	updated, err := u.orderRepo.Update(ctx, current, nil)
	if err != nil {
		return nil, err
	}

	return &updated, nil
}

// ============================================================
// Mail
// ============================================================

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

// ============================================================
// Inventory reserve helpers
// ============================================================

type reserveItem struct {
	InventoryID string
	ModelID     string
	Qty         int
}

// extractOrderItems extracts valid order items for inventory reserve.
//
// Invalid items are skipped:
// - InventoryID empty
// - ModelID empty
// - Qty <= 0
func extractOrderItems(ord orderdom.Order) []reserveItem {
	if len(ord.Items) == 0 {
		return nil
	}

	out := make([]reserveItem, 0, len(ord.Items))
	for _, it := range ord.Items {
		if it.InventoryID == "" || it.ModelID == "" || it.Qty <= 0 {
			continue
		}

		out = append(out, reserveItem{
			InventoryID: it.InventoryID,
			ModelID:     it.ModelID,
			Qty:         it.Qty,
		})
	}

	if len(out) == 0 {
		return nil
	}

	return out
}

// aggregateReserveItems aggregates reserve qty by inventoryId + modelId.
//
// Output order is stable:
// 1. InventoryID asc
// 2. ModelID asc
func aggregateReserveItems(items []reserveItem) []reserveItem {
	if len(items) == 0 {
		return nil
	}

	type key struct {
		Inv string
		Mdl string
	}

	m := map[key]int{}
	for _, it := range items {
		inv := it.InventoryID
		mdl := it.ModelID

		if inv == "" || mdl == "" || it.Qty <= 0 {
			continue
		}

		m[key{Inv: inv, Mdl: mdl}] += it.Qty
	}

	out := make([]reserveItem, 0, len(m))
	for k, q := range m {
		if q <= 0 {
			continue
		}

		out = append(out, reserveItem{
			InventoryID: k.Inv,
			ModelID:     k.Mdl,
			Qty:         q,
		})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].InventoryID == out[j].InventoryID {
			return out[i].ModelID < out[j].ModelID
		}
		return out[i].InventoryID < out[j].InventoryID
	})

	return out
}
