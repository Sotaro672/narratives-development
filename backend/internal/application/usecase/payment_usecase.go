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
1) resale status=sold 更新（best-effort）
2) 注文確定メール送信
3) inventory reserve 更新（best-effort）
4) cart delete（best-effort）
*/

import (
	"context"
	"sort"
	"strings"
	"time"

	cartdom "narratives/internal/domain/cart"
	common "narratives/internal/domain/common"
	orderdom "narratives/internal/domain/order"
	paymentdom "narratives/internal/domain/payment"
	resaledom "narratives/internal/domain/resale"
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
	GetByPaymentID(
		ctx context.Context,
		paymentID string,
	) (*paymentdom.Payment, error)

	// Writes
	Create(
		ctx context.Context,
		in paymentdom.CreatePaymentInput,
	) (*paymentdom.Payment, error)

	UpdateByPaymentID(
		ctx context.Context,
		paymentID string,
		patch paymentdom.UpdatePaymentInput,
	) (*paymentdom.Payment, error)
}

// InventoryRepoForPayment is the minimal port for inventory reserve after payment.
type InventoryRepoForPayment interface {
	// ReserveByOrder sets:
	// - stock[modelId].reservedByOrder[orderId] = qty
	// - reservedCount = sum(reservedByOrder) (repo側で正規化)
	ReserveByOrder(
		ctx context.Context,
		inventoryID string,
		modelID string,
		orderID string,
		qty int,
	) error
}

// OrderRepoForPayment is the minimal port for reading/updating orders after payment.
type OrderRepoForPayment interface {
	GetByID(
		ctx context.Context,
		id string,
	) (orderdom.Order, error)

	Update(
		ctx context.Context,
		order orderdom.Order,
		opts *common.SaveOptions,
	) (orderdom.Order, error)
}

// ResaleRepoForPayment is the minimal port for updating resale status after payment.
//
// resale order item:
// - order.Items[].Type == "resale"
// - order.Items[].ResaleID points to resales/{resaleId}
//
// PaymentUsecase marks those resale listings as sold after payment succeeded.
type ResaleRepoForPayment interface {
	GetByID(
		ctx context.Context,
		id string,
	) (resaledom.Resale, error)

	Update(
		ctx context.Context,
		id string,
		item resaledom.Resale,
	) (resaledom.Resale, error)
}

// AuthUserEmailGetter is the minimal port for reading email from Firebase Authentication.
//
// Firestore users table does not own email.
// PaymentUsecase uses this port only for sending order confirmation mail.
type AuthUserEmailGetter interface {
	GetEmailByUID(
		ctx context.Context,
		uid string,
	) (string, error)
}

// MailSenderForPayment is the minimal port for sending order confirmation mail.
//
// メール件名・本文の組み立ては application/usecase では行わず、
// adapter 側の OrderMailer に分離する。
type MailSenderForPayment interface {
	SendOrderConfirmation(
		ctx context.Context,
		from string,
		to string,
		order orderdom.Order,
	) error
}

// ============================================================
// Usecase
// ============================================================

// PaymentUsecase orchestrates payment operations.
type PaymentUsecase struct {
	repo          PaymentRepo
	cartRepo      cartdom.Repository
	inventoryRepo InventoryRepoForPayment
	orderRepo     OrderRepoForPayment
	resaleRepo    ResaleRepoForPayment

	// authUserGetter は Firebase Authentication から
	// uid に紐づく email を取得する。
	// Firestore users table に email は持たせない。
	authUserGetter AuthUserEmailGetter
	mailSender     MailSenderForPayment
	mailFrom       string

	now func() time.Time
}

type NewPaymentUsecaseInput struct {
	PaymentRepo PaymentRepo

	CartRepo      cartdom.Repository
	InventoryRepo InventoryRepoForPayment
	OrderRepo     OrderRepoForPayment
	ResaleRepo    ResaleRepoForPayment

	// AuthUserGetter は Firebase Authentication から
	// email を取得するための port。
	// 注文確定メール送信時に ord.UserID を uid として利用する。
	AuthUserGetter AuthUserEmailGetter
	MailSender     MailSenderForPayment
	MailFrom       string

	Now func() time.Time
}

func NewPaymentUsecase(
	in NewPaymentUsecaseInput,
) *PaymentUsecase {
	now := in.Now
	if now == nil {
		now = time.Now
	}

	return &PaymentUsecase{
		repo:          in.PaymentRepo,
		cartRepo:      in.CartRepo,
		inventoryRepo: in.InventoryRepo,
		orderRepo:     in.OrderRepo,
		resaleRepo:    in.ResaleRepo,

		authUserGetter: in.AuthUserGetter,
		mailSender:     in.MailSender,
		mailFrom:       in.MailFrom,

		now: now,
	}
}

// ============================================================
// Queries
// ============================================================

func (u *PaymentUsecase) GetByPaymentID(
	ctx context.Context,
	paymentID string,
) (*paymentdom.Payment, error) {
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

func (u *PaymentUsecase) Create(
	ctx context.Context,
	payment paymentdom.Payment,
) (*paymentdom.Payment, error) {
	if u == nil || u.repo == nil {
		return nil, paymentdom.ErrNotFound
	}

	in := paymentdom.CreatePaymentInput{
		PaymentID:             payment.PaymentID,
		PaymentMethodID:       payment.PaymentMethodID,
		StripeCustomerID:      payment.StripeCustomerID,
		StripePaymentMethodID: payment.StripePaymentMethodID,
		StripePaymentIntentID: payment.StripePaymentIntentID,
		Amount:                payment.Amount,
		Status:                payment.Status,
		ErrorType:             payment.ErrorType,
		ErrorCode:             payment.ErrorCode,
		ErrorMsg:              payment.ErrorMsg,
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
// This method is used by PaymentFlowUsecase after Stripe
// PaymentIntent state changes.
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

	updated, err := u.repo.UpdateByPaymentID(
		ctx,
		paymentID,
		patch,
	)
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

func isPaidStatus(status paymentdom.PaymentStatus) bool {
	return status == paymentdom.StatusSucceeded
}

// ============================================================
// Post-paid flow
// ============================================================

// handlePostPaidBestEffort runs post-paid side effects
// in best-effort manner.
//
// 前提:
// - payment / order の docId は同じ
// - rootID = payment.PaymentID
// - payment.PaymentID は order.ID と同じ値である
//
// 処理順:
// 0) order.Paid=true 更新
// 1) resale status=sold
// 2) 注文確定メール送信
// 3) inventory reserve
// 4) cart delete
func (u *PaymentUsecase) handlePostPaidBestEffort(
	ctx context.Context,
	payment *paymentdom.Payment,
) {
	if u == nil || payment == nil {
		return
	}

	rootID := payment.PaymentID
	if rootID == "" {
		return
	}

	var order *orderdom.Order

	if u.orderRepo != nil {
		foundOrder, err := u.orderRepo.GetByID(
			ctx,
			rootID,
		)
		if err == nil {
			order = &foundOrder
		}
	}

	// 0) order.Paid=true
	if u.orderRepo != nil {
		updatedOrder, err := u.markOrderPaidTrue(
			ctx,
			rootID,
			order,
		)
		if err == nil && updatedOrder != nil {
			order = updatedOrder
		}
	}

	// 1) resale status=sold
	if u.resaleRepo != nil && order != nil {
		_ = u.markResalesSoldByOrder(ctx, *order)
	}

	// 2) 注文確定メール送信
	if order != nil &&
		u.authUserGetter != nil &&
		u.mailSender != nil &&
		u.mailFrom != "" {
		_ = u.sendOrderConfirmationMail(ctx, *order)
	}

	// 3) inventory reserve
	if u.inventoryRepo != nil && order != nil {
		rawItems := extractOrderItems(*order)
		aggregatedItems := aggregateReserveItems(rawItems)

		for _, item := range aggregatedItems {
			inventoryID := item.InventoryID
			if inventoryID == "" ||
				item.ModelID == "" ||
				item.Qty <= 0 {
				continue
			}

			_ = u.inventoryRepo.ReserveByOrder(
				ctx,
				inventoryID,
				item.ModelID,
				rootID,
				item.Qty,
			)
		}
	}

	// 4) cart delete
	if u.cartRepo != nil && order != nil {
		cartID := order.CartID
		if cartID != "" {
			_ = u.cartRepo.DeleteByAvatarID(
				ctx,
				cartID,
			)
		}
	}
}

// ============================================================
// order.Paid = true
// ============================================================

func (u *PaymentUsecase) markOrderPaidTrue(
	ctx context.Context,
	orderID string,
	order *orderdom.Order,
) (*orderdom.Order, error) {
	if u == nil || u.orderRepo == nil {
		return order, nil
	}
	if orderID == "" {
		return order, nil
	}

	var current orderdom.Order

	if order != nil {
		current = *order
	} else {
		fetched, err := u.orderRepo.GetByID(
			ctx,
			orderID,
		)
		if err != nil {
			return nil, err
		}

		current = fetched
	}

	if current.Paid {
		return &current, nil
	}

	current.Paid = true

	updated, err := u.orderRepo.Update(
		ctx,
		current,
		nil,
	)
	if err != nil {
		return nil, err
	}

	return &updated, nil
}

// ============================================================
// resale.Status = sold
// ============================================================

func (u *PaymentUsecase) markResalesSoldByOrder(
	ctx context.Context,
	order orderdom.Order,
) error {
	if u == nil || u.resaleRepo == nil {
		return nil
	}

	resaleIDs := extractResaleIDsFromOrder(order)
	if len(resaleIDs) == 0 {
		return nil
	}

	now := u.now().UTC()

	for _, resaleID := range resaleIDs {
		current, err := u.resaleRepo.GetByID(
			ctx,
			resaleID,
		)
		if err != nil {
			continue
		}

		if current.Status == resaledom.StatusSold {
			continue
		}

		if err := current.MarkSold(now); err != nil {
			continue
		}

		_, _ = u.resaleRepo.Update(
			ctx,
			resaleID,
			current,
		)
	}

	return nil
}

func extractResaleIDsFromOrder(
	order orderdom.Order,
) []string {
	if len(order.Items) == 0 {
		return nil
	}

	seen := map[string]struct{}{}
	resaleIDs := make(
		[]string,
		0,
		len(order.Items),
	)

	for _, item := range order.Items {
		if item.Type != orderdom.OrderItemTypeResale {
			continue
		}

		resaleID := strings.TrimSpace(item.ResaleID)
		if resaleID == "" {
			continue
		}

		if _, exists := seen[resaleID]; exists {
			continue
		}

		seen[resaleID] = struct{}{}
		resaleIDs = append(resaleIDs, resaleID)
	}

	sort.Strings(resaleIDs)

	return resaleIDs
}

// ============================================================
// Mail
// ============================================================

func (u *PaymentUsecase) sendOrderConfirmationMail(
	ctx context.Context,
	order orderdom.Order,
) error {
	if u == nil ||
		u.authUserGetter == nil ||
		u.mailSender == nil ||
		u.mailFrom == "" {
		return nil
	}

	if order.ID == "" || order.UserID == "" {
		return nil
	}

	to, err := u.authUserGetter.GetEmailByUID(
		ctx,
		order.UserID,
	)
	if err != nil {
		return err
	}
	if to == "" {
		return nil
	}

	return u.mailSender.SendOrderConfirmation(
		ctx,
		u.mailFrom,
		to,
		order,
	)
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
func extractOrderItems(
	order orderdom.Order,
) []reserveItem {
	if len(order.Items) == 0 {
		return nil
	}

	items := make(
		[]reserveItem,
		0,
		len(order.Items),
	)

	for _, item := range order.Items {
		if item.InventoryID == "" ||
			item.ModelID == "" ||
			item.Qty <= 0 {
			continue
		}

		items = append(items, reserveItem{
			InventoryID: item.InventoryID,
			ModelID:     item.ModelID,
			Qty:         item.Qty,
		})
	}

	if len(items) == 0 {
		return nil
	}

	return items
}

// aggregateReserveItems aggregates reserve qty by
// inventoryId + modelId.
//
// Output order is stable:
// 1. InventoryID asc
// 2. ModelID asc
func aggregateReserveItems(
	items []reserveItem,
) []reserveItem {
	if len(items) == 0 {
		return nil
	}

	type key struct {
		InventoryID string
		ModelID     string
	}

	quantities := map[key]int{}

	for _, item := range items {
		inventoryID := item.InventoryID
		modelID := item.ModelID

		if inventoryID == "" ||
			modelID == "" ||
			item.Qty <= 0 {
			continue
		}

		itemKey := key{
			InventoryID: inventoryID,
			ModelID:     modelID,
		}

		quantities[itemKey] += item.Qty
	}

	aggregated := make(
		[]reserveItem,
		0,
		len(quantities),
	)

	for itemKey, quantity := range quantities {
		if quantity <= 0 {
			continue
		}

		aggregated = append(aggregated, reserveItem{
			InventoryID: itemKey.InventoryID,
			ModelID:     itemKey.ModelID,
			Qty:         quantity,
		})
	}

	sort.Slice(aggregated, func(i, j int) bool {
		if aggregated[i].InventoryID ==
			aggregated[j].InventoryID {
			return aggregated[i].ModelID <
				aggregated[j].ModelID
		}

		return aggregated[i].InventoryID <
			aggregated[j].InventoryID
	})

	return aggregated
}
