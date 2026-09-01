// backend/internal/application/usecase/payment_usecase.go
package usecase

/*
責務:
- Paymentの取得・作成・部分更新を提供する。
- Stripe webhook eventによるPayment status同期を提供する。
- Refund stateの専用更新経路を提供する。
- succeeded状態ではOrder.Paidを冪等に整合させる。
- succeededへの初回遷移時だけbest-effortの支払い後処理を実行する。

前提:
- payment document ID = payment.PaymentID
- payment.PaymentID = order.ID
- paymentIdはpayment document fieldとして保存しない
- payment recordsは削除しない
- Stripe PaymentIntentはpayment record作成前に作成する
- StripePaymentIntentIDはpendingを含む全statusで必須
- TransferGroupはpendingを含む全statusで必須
- StripeChargeIDはStripe Charge作成前は空を許容する
- PaymentStatusとRefundStatusは独立して管理する
- Refund成功後もPaymentStatusはsucceededを維持する

Stripe状態同期:
- Stripe由来のstatus更新はApplyStripeEventを使用する。
- 一般的なUpdateからstatusを変更してはならない。
- Stripe event IDの重複判定とstatus遷移はRepositoryが
  Firestore Transaction内で原子的に処理する。
- PostPaidRequiredはPaymentが初めてsucceededへ遷移した
  1回だけtrueになる。

Refund状態更新:
- 一般的なUpdateからRefund stateを変更してはならない。
- Refund stateはUpdateRefundStateを使用する。
- StripeRefundID / RefundStatus / RefundedAmount / RefundedAtは
  1つの論理状態としてまとめて更新する。

支払い成功後の処理:
0) order.Paid=true更新（必須・冪等）
1) resale status=sold更新（best-effort）
2) resale購入通知コメント作成（best-effort）

注文受付時に行うべき処理:
- inventory reserve
- cart delete
- 注文受付メール送信

これらは発送時決済では遅すぎるため、PaymentUsecaseでは実行しない。
*/

import (
	"context"
	"errors"
	"sort"
	"time"

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
	ApplyStripePaymentEvent(ctx context.Context, in ApplyStripePaymentEventInput) (*ApplyStripePaymentEventResult, error)
}

// ApplyStripePaymentEventInput is the application-level input generated from
// a verified Stripe webhook event.
type ApplyStripePaymentEventInput struct {
	EventID string

	PaymentID string

	StripePaymentIntentID string
	StripeChargeID        string

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

// OrderRepoForPayment is the minimal port for reading/updating orders after
// payment.
type OrderRepoForPayment interface {
	GetByID(ctx context.Context, id string) (orderdom.Order, error)
	Update(ctx context.Context, order orderdom.Order, opts *common.SaveOptions) (orderdom.Order, error)
}

// ResaleRepoForPayment is the minimal port for updating resale status after
// payment.
//
// Resale order item:
// - order.Items[].Type == "resale"
// - order.Items[].ResaleID points to resales/{resaleId}
type ResaleRepoForPayment interface {
	GetByID(ctx context.Context, id string) (resaledom.Resale, error)
	Update(ctx context.Context, id string, item resaledom.Resale) (resaledom.Resale, error)
}

// ResalePurchaseCommentWriter creates a purchase notification comment after
// a resale has been successfully marked as sold.
type ResalePurchaseCommentWriter interface {
	CreatePurchaseComment(ctx context.Context, resaleID string, buyerAvatarID string) error
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
	ErrPaymentRefundUpdateRequiresRefundState = errors.New(
		"payment: refund update requires refund state application",
	)
	ErrPaymentOrderRepositoryMissing = errors.New(
		"payment: order repository is not configured",
	)
	ErrPaymentPaidOrderUnavailable = errors.New(
		"payment: paid order is unavailable",
	)
)

// ============================================================
// Usecase
// ============================================================

// PaymentUsecase orchestrates payment operations.
type PaymentUsecase struct {
	repo paymentdom.RepositoryPort

	stripeEventRepo StripePaymentEventRepository

	orderRepo                   OrderRepoForPayment
	resaleRepo                  ResaleRepoForPayment
	resalePurchaseCommentWriter ResalePurchaseCommentWriter

	now func() time.Time
}

type NewPaymentUsecaseInput struct {
	PaymentRepo paymentdom.RepositoryPort

	StripeEventRepo StripePaymentEventRepository

	OrderRepo                   OrderRepoForPayment
	ResaleRepo                  ResaleRepoForPayment
	ResalePurchaseCommentWriter ResalePurchaseCommentWriter

	Now func() time.Time
}

func NewPaymentUsecase(in NewPaymentUsecaseInput) *PaymentUsecase {
	now := in.Now
	if now == nil {
		now = time.Now
	}

	return &PaymentUsecase{
		repo:                        in.PaymentRepo,
		stripeEventRepo:             in.StripeEventRepo,
		orderRepo:                   in.OrderRepo,
		resaleRepo:                  in.ResaleRepo,
		resalePurchaseCommentWriter: in.ResalePurchaseCommentWriter,
		now:                         now,
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

// Create creates a Payment.
//
// StripePaymentIntentID and TransferGroup are required for every status,
// including pending.
//
// StripeChargeID may be empty until Stripe has created a Charge.
//
// A Payment created as succeeded synchronizes Order.Paid immediately.
// Non-critical post-paid side effects are then executed once from this creation
// path. Later succeeded webhook events still re-run the required Order.Paid
// synchronization idempotently.
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
	if payment.PaymentID == "" {
		return nil, paymentdom.ErrInvalidPaymentID
	}
	if payment.StripePaymentIntentID == "" {
		return nil, paymentdom.ErrInvalidStripePaymentIntent
	}
	if payment.TransferGroup == "" {
		return nil, paymentdom.ErrInvalidTransferGroup
	}
	if payment.StripeRefundID != "" ||
		payment.RefundedAmount != 0 ||
		payment.RefundedAt != nil ||
		(payment.RefundStatus != "" &&
			payment.RefundStatus != paymentdom.RefundStatusNone) {
		return nil, paymentdom.ErrInvalidRefundState
	}

	in := paymentdom.CreatePaymentInput{
		PaymentID:             payment.PaymentID,
		PaymentMethodID:       payment.PaymentMethodID,
		StripeCustomerID:      payment.StripeCustomerID,
		StripePaymentMethodID: payment.StripePaymentMethodID,
		StripePaymentIntentID: payment.StripePaymentIntentID,
		StripeChargeID:        payment.StripeChargeID,
		TransferGroup:         payment.TransferGroup,
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

	if created != nil && created.Status == paymentdom.StatusSucceeded {
		order, err := u.ensurePaidOrder(ctx, created)
		if err != nil {
			return nil, err
		}

		u.handlePostPaidBestEffort(ctx, order)
	}

	return created, nil
}

// Update partially updates an existing Payment.
//
// Stripe status must not be changed through this method. Stripe-originated
// status changes must use ApplyStripeEvent so that event deduplication,
// transition validation, and post-paid marker acquisition happen atomically.
//
// Refund state must also not be changed through this method.
// StripeRefundID, RefundStatus, RefundedAmount, and RefundedAt must be changed
// together through UpdateRefundState.
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
	if patch.Status != nil {
		return nil, ErrPaymentStatusUpdateRequiresStripeEvent
	}
	if patch.StripeRefundID != nil ||
		patch.RefundStatus != nil ||
		patch.RefundedAmount != nil ||
		patch.RefundedAt != nil {
		return nil, ErrPaymentRefundUpdateRequiresRefundState
	}
	if patch.StripePaymentIntentID != nil &&
		*patch.StripePaymentIntentID == "" {
		return nil, paymentdom.ErrInvalidStripePaymentIntent
	}
	if patch.StripeChargeID != nil &&
		*patch.StripeChargeID == "" {
		return nil, paymentdom.ErrInvalidStripeChargeID
	}
	if patch.TransferGroup != nil &&
		*patch.TransferGroup == "" {
		return nil, paymentdom.ErrInvalidTransferGroup
	}

	return u.repo.UpdateByPaymentID(
		ctx,
		paymentID,
		patch,
	)
}

// UpdatePaymentRefundStateInput represents one complete Payment refund state.
//
// Refund state is independent from PaymentStatus.
// A refunded Payment therefore remains StatusSucceeded.
type UpdatePaymentRefundStateInput struct {
	PaymentID string

	StripeRefundID string
	RefundStatus   paymentdom.RefundStatus
	RefundedAmount int
	RefundedAt     *time.Time
}

// UpdateRefundState applies one complete Payment refund state.
//
// StripeRefundID, RefundStatus, RefundedAmount, and RefundedAt are validated
// together through Payment.SetRefundState before Repository persistence.
//
// The current AMOL refund flow supports full refunds only. Therefore a
// succeeded refund requires RefundedAmount == Payment.Amount.
//
// PaymentStatus is intentionally not changed here. A successful refund keeps
// the original PaymentIntent lifecycle state as succeeded.
func (u *PaymentUsecase) UpdateRefundState(
	ctx context.Context,
	in UpdatePaymentRefundStateInput,
) (*paymentdom.Payment, error) {
	if u == nil || u.repo == nil {
		return nil, paymentdom.ErrNotFound
	}
	if in.PaymentID == "" {
		return nil, paymentdom.ErrInvalidPaymentID
	}
	if in.RefundStatus == paymentdom.RefundStatusNone ||
		!paymentdom.IsValidRefundStatus(in.RefundStatus) {
		return nil, paymentdom.ErrInvalidRefundStatus
	}
	if in.StripeRefundID == "" {
		return nil, paymentdom.ErrInvalidStripeRefundID
	}

	current, err := u.repo.GetByPaymentID(
		ctx,
		in.PaymentID,
	)
	if err != nil {
		return nil, err
	}
	if current == nil ||
		current.PaymentID != in.PaymentID {
		return nil, paymentdom.ErrNotFound
	}

	refundedAt := in.RefundedAt
	if refundedAt != nil {
		value := refundedAt.UTC()
		refundedAt = &value
	}

	next := *current
	if err := next.SetRefundState(
		in.StripeRefundID,
		in.RefundStatus,
		in.RefundedAmount,
		refundedAt,
	); err != nil {
		return nil, err
	}

	refundStatus := next.RefundStatus
	refundedAmount := next.RefundedAmount

	patch := paymentdom.UpdatePaymentInput{
		StripeRefundID: &next.StripeRefundID,
		RefundStatus:   &refundStatus,
		RefundedAmount: &refundedAmount,
		RefundedAt:     next.RefundedAt,
	}

	updated, err := u.repo.UpdateByPaymentID(
		ctx,
		in.PaymentID,
		patch,
	)
	if err != nil {
		return nil, err
	}
	if updated == nil ||
		updated.PaymentID != in.PaymentID {
		return nil, paymentdom.ErrNotFound
	}

	return updated, nil
}

// ApplyStripeEvent applies a verified Stripe webhook event.
//
// Event deduplication and status transition must be performed atomically by
// StripePaymentEventRepository.
//
// A duplicate event is returned as a successful no-op for Payment state.
//
// When the resulting Payment is succeeded, Order.Paid is ensured on every call.
// Best-effort post-paid processing is executed only when PostPaidRequired is true.
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

	result, err := u.stripeEventRepo.ApplyStripePaymentEvent(
		ctx,
		in,
	)
	if err != nil {
		return nil, err
	}
	if result == nil ||
		result.Payment == nil {
		return nil, ErrPaymentStripeEventResultEmpty
	}

	var paidOrder *orderdom.Order
	if result.Payment.Status == paymentdom.StatusSucceeded {
		paidOrder, err = u.ensurePaidOrder(
			ctx,
			result.Payment,
		)
		if err != nil {
			return nil, err
		}
	}

	if result.PostPaidRequired {
		u.handlePostPaidBestEffort(
			ctx,
			paidOrder,
		)
	}

	return result.Payment, nil
}

// ============================================================
// Required paid-state synchronization
// ============================================================

// ensurePaidOrder synchronizes the required application state for a
// succeeded Payment.
//
// This method is intentionally idempotent and may run for duplicate succeeded
// Stripe webhook events. Order.Paid is set to true if necessary.
//
// Trade creation belongs to the order-placement flow and is intentionally not
// performed by PaymentUsecase.
func (u *PaymentUsecase) ensurePaidOrder(
	ctx context.Context,
	payment *paymentdom.Payment,
) (*orderdom.Order, error) {
	if u == nil ||
		u.orderRepo == nil {
		return nil, ErrPaymentOrderRepositoryMissing
	}
	if payment == nil ||
		payment.Status != paymentdom.StatusSucceeded {
		return nil, nil
	}
	if payment.PaymentID == "" {
		return nil, paymentdom.ErrInvalidPaymentID
	}

	order, err := u.orderRepo.GetByID(
		ctx,
		payment.PaymentID,
	)
	if err != nil {
		return nil, err
	}

	updatedOrder, err := u.markOrderPaidTrue(
		ctx,
		order,
	)
	if err != nil {
		return nil, err
	}
	if updatedOrder == nil {
		return nil, ErrPaymentPaidOrderUnavailable
	}

	return updatedOrder, nil
}

// ============================================================
// Best-effort post-paid flow
// ============================================================

// handlePostPaidBestEffort runs non-critical post-paid side effects.
//
// Required Order.Paid synchronization must already have succeeded before this
// method is called.
//
// This method may only be called from:
//  1. A successful initial Create whose Repository transaction also stores
//     the post-paid execution marker.
//  2. ApplyStripeEvent when PostPaidRequired is true.
func (u *PaymentUsecase) handlePostPaidBestEffort(
	ctx context.Context,
	order *orderdom.Order,
) {
	if u == nil ||
		order == nil {
		return
	}

	if u.resaleRepo != nil {
		_ = u.markResalesSoldByOrder(
			ctx,
			*order,
		)
	}

	// Inventory reservation, cart deletion, and order-acceptance mail are
	// intentionally not executed here. With payment deferred until dispatch,
	// those operations must belong to the order-placement flow.
}

// ============================================================
// order.Paid = true
// ============================================================

func (u *PaymentUsecase) markOrderPaidTrue(
	ctx context.Context,
	order orderdom.Order,
) (*orderdom.Order, error) {
	if u == nil ||
		u.orderRepo == nil {
		return nil, ErrPaymentOrderRepositoryMissing
	}
	if order.ID == "" {
		return nil, orderdom.ErrInvalidID
	}

	if order.Paid {
		return &order, nil
	}

	order.Paid = true

	updated, err := u.orderRepo.Update(
		ctx,
		order,
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
	if u == nil ||
		u.resaleRepo == nil {
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

		_, err = u.resaleRepo.Update(
			ctx,
			resaleID,
			current,
		)
		if err != nil {
			continue
		}

		if u.resalePurchaseCommentWriter != nil {
			_ = u.resalePurchaseCommentWriter.CreatePurchaseComment(
				ctx,
				resaleID,
				order.AvatarID,
			)
		}
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
		if item.ResaleID == "" {
			continue
		}
		if _, exists := seen[item.ResaleID]; exists {
			continue
		}

		seen[item.ResaleID] = struct{}{}
		resaleIDs = append(
			resaleIDs,
			item.ResaleID,
		)
	}

	sort.Strings(resaleIDs)
	return resaleIDs
}
