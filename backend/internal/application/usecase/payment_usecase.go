// backend/internal/application/usecase/payment_usecase.go
package usecase

/*
責務:
- Paymentの取得・作成・部分更新を提供する。
- Stripe webhook eventによるPayment status同期を提供する。
- succeededへの初回遷移時だけ支払い後処理を実行する。

前提:
- payment document ID = payment.PaymentID
- payment.PaymentID = order.ID
- paymentIdはpayment document fieldとして保存しない
- payment recordsは削除しない
- Stripe PaymentIntentはpayment record作成前に作成する
- StripePaymentIntentIDはpendingを含む全statusで必須

Stripe状態同期:
- Stripe由来のstatus更新はApplyStripeEventを使用する。
- 一般的なUpdateからstatusを変更してはならない。
- Stripe event IDの重複判定とstatus遷移はRepositoryが
  Firestore Transaction内で原子的に処理する。
- PostPaidRequiredはPaymentが初めてsucceededへ遷移した
  1回だけtrueになる。

支払い成功後の処理:
0) order.Paid=true更新
1) resale status=sold更新（best-effort）
2) 注文確認メール送信（best-effort）
3) inventory reserve更新（best-effort）
4) cart delete（best-effort）
*/

import (
	"context"
	"errors"
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

// StripePaymentEventRepository applies Stripe events atomically.
//
// Implementations must perform the following operations in one transaction:
//
//  1. Check whether EventID has already been processed.
//  2. Read the current Payment.
//  3. Verify that StripePaymentIntentID matches the Payment.
//  4. Apply the valid status transition.
//  5. Record EventID as processed.
//  6. Set the post-paid execution marker only when the Payment becomes
//     succeeded for the first time.
//
// PostPaidRequired must be true only for the single caller that acquires the
// post-paid execution marker.
type StripePaymentEventRepository interface {
	ApplyStripePaymentEvent(
		ctx context.Context,
		in ApplyStripePaymentEventInput,
	) (*ApplyStripePaymentEventResult, error)
}

// ApplyStripePaymentEventInput is the application-level input generated from
// a verified Stripe webhook event.
type ApplyStripePaymentEventInput struct {
	EventID string

	PaymentID string

	StripePaymentIntentID string

	Status paymentdom.PaymentStatus

	ErrorType *string
	ErrorCode *string
	ErrorMsg  *string

	OccurredAt time.Time
}

// ApplyStripePaymentEventResult describes the atomic event application result.
type ApplyStripePaymentEventResult struct {
	Payment *paymentdom.Payment

	// EventApplied is false when EventID has already been processed.
	EventApplied bool

	// StatusChanged is true when the stored Payment status changed.
	StatusChanged bool

	// PostPaidRequired is true only when this application acquired the
	// first-succeeded post-paid execution marker.
	PostPaidRequired bool
}

// InventoryRepoForPayment is the minimal port for inventory reserve after
// payment.
type InventoryRepoForPayment interface {
	// ReserveByOrder sets:
	// - stock[modelId].reservedByOrder[orderId] = qty
	// - reservedCount = sum(reservedByOrder), normalized by repository
	ReserveByOrder(
		ctx context.Context,
		inventoryID string,
		modelID string,
		orderID string,
		qty int,
	) error
}

// OrderRepoForPayment is the minimal port for reading/updating orders after
// payment.
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

// ResaleRepoForPayment is the minimal port for updating resale status after
// payment.
//
// Resale order item:
// - order.Items[].Type == "resale"
// - order.Items[].ResaleID points to resales/{resaleId}
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

// AuthUserEmailGetter is the minimal port for reading email from Firebase
// Authentication.
//
// The Firestore users collection does not own email.
// PaymentUsecase uses this port only for sending order confirmation mail.
type AuthUserEmailGetter interface {
	GetEmailByUID(
		ctx context.Context,
		uid string,
	) (string, error)
}

// MailSenderForPayment is the minimal port for sending order confirmation
// mail.
type MailSenderForPayment interface {
	SendOrderConfirmation(
		ctx context.Context,
		from string,
		to string,
		order orderdom.Order,
	) error
}

// ============================================================
// Errors
// ============================================================

var (
	ErrPaymentStripeEventRepositoryMissing = errors.New(
		"payment: stripe payment event repository is not configured",
	)
	ErrPaymentStripeEventIDEmpty = errors.New(
		"payment: stripe event id is empty",
	)
	ErrPaymentStripeEventOccurredAtInvalid = errors.New(
		"payment: stripe event occurredAt is invalid",
	)
	ErrPaymentStatusUpdateRequiresStripeEvent = errors.New(
		"payment: status update requires Stripe event application",
	)
	ErrPaymentStripeEventResultEmpty = errors.New(
		"payment: stripe event application result is empty",
	)
)

// ============================================================
// Usecase
// ============================================================

// PaymentUsecase orchestrates payment operations.
type PaymentUsecase struct {
	repo paymentdom.RepositoryPort

	stripeEventRepo StripePaymentEventRepository

	cartRepo      cartdom.Repository
	inventoryRepo InventoryRepoForPayment
	orderRepo     OrderRepoForPayment
	resaleRepo    ResaleRepoForPayment

	// authUserGetter gets the email associated with a UID from Firebase
	// Authentication. Email is not stored in the Firestore users collection.
	authUserGetter AuthUserEmailGetter
	mailSender     MailSenderForPayment
	mailFrom       string

	now func() time.Time
}

type NewPaymentUsecaseInput struct {
	PaymentRepo paymentdom.RepositoryPort

	// StripeEventRepo may be omitted when PaymentRepo also implements
	// StripePaymentEventRepository.
	StripeEventRepo StripePaymentEventRepository

	CartRepo      cartdom.Repository
	InventoryRepo InventoryRepoForPayment
	OrderRepo     OrderRepoForPayment
	ResaleRepo    ResaleRepoForPayment

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

	stripeEventRepo := in.StripeEventRepo
	if stripeEventRepo == nil && in.PaymentRepo != nil {
		if repository, ok :=
			in.PaymentRepo.(StripePaymentEventRepository); ok {
			stripeEventRepo = repository
		}
	}

	return &PaymentUsecase{
		repo:            in.PaymentRepo,
		stripeEventRepo: stripeEventRepo,

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

	paymentID = strings.TrimSpace(paymentID)
	if paymentID == "" {
		return nil, paymentdom.ErrInvalidPaymentID
	}

	return u.repo.GetByPaymentID(ctx, paymentID)
}

// ============================================================
// Commands
// ============================================================

// Create creates a Payment.
//
// StripePaymentIntentID is required for every status, including pending.
//
// A Payment created as succeeded runs post-paid processing once from this
// creation path. Later succeeded webhook events must not run the processing
// again because ApplyStripePaymentEvent must return PostPaidRequired=false
// for an already-succeeded Payment.
//
// The repository implementation must persist the post-paid execution marker
// when it creates a Payment whose initial status is succeeded.
func (u *PaymentUsecase) Create(
	ctx context.Context,
	payment paymentdom.Payment,
) (*paymentdom.Payment, error) {
	if u == nil || u.repo == nil {
		return nil, paymentdom.ErrNotFound
	}

	if strings.TrimSpace(payment.PaymentID) == "" {
		return nil, paymentdom.ErrInvalidPaymentID
	}

	if strings.TrimSpace(
		payment.StripePaymentIntentID,
	) == "" {
		return nil, paymentdom.ErrInvalidStripePaymentIntent
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

	if created != nil &&
		created.Status == paymentdom.StatusSucceeded {
		u.handlePostPaidBestEffort(ctx, created)
	}

	return created, nil
}

// Update partially updates an existing Payment.
//
// Stripe status must not be changed through this method. Stripe-originated
// status changes must use ApplyStripeEvent so that event deduplication,
// transition validation, and post-paid marker acquisition happen atomically.
func (u *PaymentUsecase) Update(
	ctx context.Context,
	paymentID string,
	patch paymentdom.UpdatePaymentInput,
) (*paymentdom.Payment, error) {
	if u == nil || u.repo == nil {
		return nil, paymentdom.ErrNotFound
	}

	paymentID = strings.TrimSpace(paymentID)
	if paymentID == "" {
		return nil, paymentdom.ErrInvalidPaymentID
	}

	if patch.Status != nil {
		return nil, ErrPaymentStatusUpdateRequiresStripeEvent
	}

	if patch.StripePaymentIntentID != nil &&
		strings.TrimSpace(*patch.StripePaymentIntentID) == "" {
		return nil, paymentdom.ErrInvalidStripePaymentIntent
	}

	return u.repo.UpdateByPaymentID(
		ctx,
		paymentID,
		patch,
	)
}

// ApplyStripeEvent applies a verified Stripe webhook event.
//
// Event deduplication and status transition must be performed atomically by
// StripePaymentEventRepository.
//
// A duplicate event is returned as a successful no-op.
// Post-paid processing is executed only when PostPaidRequired is true.
func (u *PaymentUsecase) ApplyStripeEvent(
	ctx context.Context,
	in ApplyStripePaymentEventInput,
) (*paymentdom.Payment, error) {
	if u == nil || u.repo == nil {
		return nil, paymentdom.ErrNotFound
	}

	if u.stripeEventRepo == nil {
		return nil, ErrPaymentStripeEventRepositoryMissing
	}

	in.EventID = strings.TrimSpace(in.EventID)
	in.PaymentID = strings.TrimSpace(in.PaymentID)
	in.StripePaymentIntentID = strings.TrimSpace(
		in.StripePaymentIntentID,
	)

	if in.EventID == "" {
		return nil, ErrPaymentStripeEventIDEmpty
	}

	if in.PaymentID == "" {
		return nil, paymentdom.ErrInvalidPaymentID
	}

	if in.StripePaymentIntentID == "" {
		return nil, paymentdom.ErrInvalidStripePaymentIntent
	}

	if !paymentdom.IsValidStatus(in.Status) {
		return nil, paymentdom.ErrInvalidStatus
	}

	if in.OccurredAt.IsZero() {
		return nil, ErrPaymentStripeEventOccurredAtInvalid
	}

	in.OccurredAt = in.OccurredAt.UTC()

	result, err :=
		u.stripeEventRepo.ApplyStripePaymentEvent(
			ctx,
			in,
		)
	if err != nil {
		return nil, err
	}

	if result == nil || result.Payment == nil {
		return nil, ErrPaymentStripeEventResultEmpty
	}

	if result.PostPaidRequired {
		u.handlePostPaidBestEffort(
			ctx,
			result.Payment,
		)
	}

	return result.Payment, nil
}

// ============================================================
// Post-paid flow
// ============================================================

// handlePostPaidBestEffort runs post-paid side effects.
//
// This method may only be called from:
//
//  1. A successful initial Create whose Repository transaction also stores
//     the post-paid execution marker.
//  2. ApplyStripeEvent when PostPaidRequired is true.
//
// The Repository guarantees that PostPaidRequired is acquired once.
func (u *PaymentUsecase) handlePostPaidBestEffort(
	ctx context.Context,
	payment *paymentdom.Payment,
) {
	if u == nil || payment == nil {
		return
	}

	if payment.Status != paymentdom.StatusSucceeded {
		return
	}

	rootID := strings.TrimSpace(payment.PaymentID)
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
		_ = u.markResalesSoldByOrder(
			ctx,
			*order,
		)
	}

	// 2) send order confirmation email
	if order != nil &&
		u.authUserGetter != nil &&
		u.mailSender != nil &&
		u.mailFrom != "" {
		_ = u.sendOrderConfirmationMail(
			ctx,
			*order,
		)
	}

	// 3) inventory reserve
	if u.inventoryRepo != nil && order != nil {
		rawItems := extractOrderItems(*order)
		aggregatedItems := aggregateReserveItems(rawItems)

		for _, item := range aggregatedItems {
			if item.InventoryID == "" ||
				item.ModelID == "" ||
				item.Qty <= 0 {
				continue
			}

			_ = u.inventoryRepo.ReserveByOrder(
				ctx,
				item.InventoryID,
				item.ModelID,
				rootID,
				item.Qty,
			)
		}
	}

	// 4) cart delete
	if u.cartRepo != nil && order != nil {
		cartID := strings.TrimSpace(order.CartID)
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

	orderID = strings.TrimSpace(orderID)
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

	now := time.Now().UTC()
	if u.now != nil {
		now = u.now().UTC()
	}

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
		resaleIDs = append(
			resaleIDs,
			resaleID,
		)
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

	if strings.TrimSpace(order.ID) == "" ||
		strings.TrimSpace(order.UserID) == "" {
		return nil
	}

	to, err := u.authUserGetter.GetEmailByUID(
		ctx,
		order.UserID,
	)
	if err != nil {
		return err
	}

	to = strings.TrimSpace(to)
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
		inventoryID := strings.TrimSpace(
			item.InventoryID,
		)
		modelID := strings.TrimSpace(item.ModelID)

		if inventoryID == "" ||
			modelID == "" ||
			item.Qty <= 0 {
			continue
		}

		items = append(items, reserveItem{
			InventoryID: inventoryID,
			ModelID:     modelID,
			Qty:         item.Qty,
		})
	}

	if len(items) == 0 {
		return nil
	}

	return items
}

// aggregateReserveItems aggregates reserve quantity by
// inventoryId + modelId.
//
// Output order is stable:
// 1. InventoryID ascending
// 2. ModelID ascending
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
		inventoryID := strings.TrimSpace(
			item.InventoryID,
		)
		modelID := strings.TrimSpace(item.ModelID)

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

		aggregated = append(
			aggregated,
			reserveItem{
				InventoryID: itemKey.InventoryID,
				ModelID:     itemKey.ModelID,
				Qty:         quantity,
			},
		)
	}

	sort.Slice(
		aggregated,
		func(i, j int) bool {
			if aggregated[i].InventoryID ==
				aggregated[j].InventoryID {
				return aggregated[i].ModelID <
					aggregated[j].ModelID
			}

			return aggregated[i].InventoryID <
				aggregated[j].InventoryID
		},
	)

	return aggregated
}
