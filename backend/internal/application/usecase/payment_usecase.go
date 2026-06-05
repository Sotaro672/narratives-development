// backend/internal/application/usecase/payment_usecase.go
package usecase

/*
責任と機能:
- PaymentUsecase の公開API（Queries/Commands）と依存注入（DI）を提供する。
- 実装詳細（paid後の後続処理、order更新、注文確定メール送信、inventory reserve）は別ファイルに委譲し、
  このファイルでは「ユースケースの入口」と「依存関係の保持」に集中する。
*/

import (
	"context"
	"time"

	common "narratives/internal/domain/common"
	orderdom "narratives/internal/domain/order"
	paymentdom "narratives/internal/domain/payment"
)

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

// Cart clear の最小ポート（PaymentUsecase 側）
// carts/{cartId} を空にする
type CartRepoForPayment interface {
	Clear(ctx context.Context, cartID string) error
}

// Inventory reserve の最小ポート（PaymentUsecase 側）
type InventoryRepoForPayment interface {
	// ReserveByOrder sets:
	// - stock[modelId].reservedByOrder[orderId] = qty
	// - reservedCount = sum(reservedByOrder) (repo側で正規化)
	ReserveByOrder(ctx context.Context, inventoryID string, modelID string, orderID string, qty int) error
}

// Order 更新・参照の最小ポート（PaymentUsecase 側）
type OrderRepoForPayment interface {
	GetByID(ctx context.Context, id string) (orderdom.Order, error)
	Update(ctx context.Context, o orderdom.Order, opts *common.SaveOptions) (orderdom.Order, error)
}

// userId からメールアドレス取得の最小ポート
type UserRepoForPayment interface {
	GetEmailByID(ctx context.Context, userID string) (string, error)
}

// 注文確定メール送信の最小ポート
//
// メール件名・本文の組み立ては application/usecase では行わず、
// adapter 側の OrderMailer に分離する。
type MailSenderForPayment interface {
	SendOrderConfirmation(ctx context.Context, from, to string, ord orderdom.Order) error
}

// PaymentUsecase orchestrates payment operations.
type PaymentUsecase struct {
	repo          PaymentRepo
	cartRepo      CartRepoForPayment
	inventoryRepo InventoryRepoForPayment
	orderRepo     OrderRepoForPayment

	userRepo   UserRepoForPayment
	mailSender MailSenderForPayment
	mailFrom   string

	now func() time.Time
}

type NewPaymentUsecaseInput struct {
	PaymentRepo PaymentRepo

	CartRepo      CartRepoForPayment
	InventoryRepo InventoryRepoForPayment
	OrderRepo     OrderRepoForPayment

	UserRepo   UserRepoForPayment
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
