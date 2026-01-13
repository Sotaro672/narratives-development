// backend/internal/application/usecase/payment_usecase.go
package usecase

/*
責任と機能:
- PaymentUsecase の公開API（Queries/Commands）と依存注入（DI）を提供する。
- 実装詳細（paid後の後続処理、invoice更新、inventory reserve、reflection util）は別ファイルに委譲し、
  このファイルでは「ユースケースの入口」と「依存関係の保持」に集中する。
*/

import (
	"context"
	"time"

	common "narratives/internal/domain/common"
	invoicedom "narratives/internal/domain/invoice"
	paymentdom "narratives/internal/domain/payment"
)

// PaymentRepo defines the minimal persistence port needed by PaymentUsecase.
// ✅ aligned with domain contract (payment.RepositoryPort / CreatePaymentInput)
type PaymentRepo interface {
	// Reads
	GetByID(ctx context.Context, id string) (*paymentdom.Payment, error)
	GetByInvoiceID(ctx context.Context, invoiceID string) ([]paymentdom.Payment, error)
	List(ctx context.Context, filter paymentdom.Filter, sort paymentdom.Sort, page paymentdom.Page) (paymentdom.PageResult, error)
	Count(ctx context.Context, filter paymentdom.Filter) (int, error)

	// Writes
	Create(ctx context.Context, in paymentdom.CreatePaymentInput) (*paymentdom.Payment, error)
	Update(ctx context.Context, id string, patch paymentdom.UpdatePaymentInput) (*paymentdom.Payment, error)
	Delete(ctx context.Context, id string) error

	// Dev/Test
	Reset(ctx context.Context) error
}

// ✅ Invoice paid を更新するための最小ポート（PaymentUsecase 側）
type InvoiceRepoForPayment interface {
	GetByOrderID(ctx context.Context, orderID string) (invoicedom.Invoice, error)
	Save(ctx context.Context, inv invoicedom.Invoice, opts *common.SaveOptions) (invoicedom.Invoice, error)
}

// ✅ Cart clear の最小ポート（PaymentUsecase 側）
// carts/{cartId} を空にする（cartId は avatarId と同義の運用）
type CartRepoForPayment interface {
	Clear(ctx context.Context, cartID string) error
}

// ✅ Inventory reserve の最小ポート（PaymentUsecase 側）
// payment paid と同タイミングで reservedByOrder / reservedCount を更新する
type InventoryRepoForPayment interface {
	// ReserveByOrder sets:
	// - stock[modelId].reservedByOrder[orderId] = qty
	// - reservedCount = sum(reservedByOrder) (repo側で正規化)
	ReserveByOrder(ctx context.Context, inventoryID string, modelID string, orderID string, qty int) error
}

// PaymentUsecase orchestrates payment operations.
type PaymentUsecase struct {
	repo          PaymentRepo
	invoiceRepo   InvoiceRepoForPayment
	cartRepo      CartRepoForPayment
	inventoryRepo InventoryRepoForPayment

	// ✅ order から cartId/avatarId/items を拾うため（型に依存しないよう any + reflection）
	orderRepo any

	now func() time.Time
}

func NewPaymentUsecase(repo PaymentRepo) *PaymentUsecase {
	return &PaymentUsecase{
		repo: repo,
		now:  time.Now,
	}
}

// ✅ optional injection: payment 起票後に invoice.paid を更新したい場合に注入する
func (u *PaymentUsecase) WithInvoiceRepoForPayment(repo InvoiceRepoForPayment) *PaymentUsecase {
	u.invoiceRepo = repo
	return u
}

// ✅ optional injection: paid 時に cart を空にしたい場合に注入する
func (u *PaymentUsecase) WithCartRepoForPayment(repo CartRepoForPayment) *PaymentUsecase {
	u.cartRepo = repo
	return u
}

// ✅ optional injection: paid 時に inventory の reservedByOrder / reservedCount を更新したい場合に注入する
func (u *PaymentUsecase) WithInventoryRepoForPayment(repo InventoryRepoForPayment) *PaymentUsecase {
	u.inventoryRepo = repo
	return u
}

// ✅ optional injection: payment に cartId が無い場合、order から拾うために注入する
// 期待するメソッド（どれか1つあればOK）:
// - GetByID(ctx, id) (T, error) or (*T, error)
// - Get(ctx, id) (T, error) or (*T, error)
// - FindByID(ctx, id) (T, error) or (*T, error)
func (u *PaymentUsecase) WithOrderRepoForPayment(repo any) *PaymentUsecase {
	u.orderRepo = repo
	return u
}

// ============================================================
// Queries
// ============================================================

func (u *PaymentUsecase) GetByID(ctx context.Context, id string) (*paymentdom.Payment, error) {
	return u.repo.GetByID(ctx, stringsTrimSpace(id))
}

// docId=invoiceId 前提なら「invoiceID=paymentID」なので GetByID と実質同じ。
// ただし domain port に合わせて残す。
func (u *PaymentUsecase) GetByInvoiceID(ctx context.Context, invoiceID string) ([]paymentdom.Payment, error) {
	return u.repo.GetByInvoiceID(ctx, stringsTrimSpace(invoiceID))
}

func (u *PaymentUsecase) List(ctx context.Context, filter paymentdom.Filter, sort paymentdom.Sort, page paymentdom.Page) (paymentdom.PageResult, error) {
	return u.repo.List(ctx, filter, sort, page)
}

func (u *PaymentUsecase) Count(ctx context.Context, filter paymentdom.Filter) (int, error) {
	return u.repo.Count(ctx, filter)
}

// ============================================================
// Commands
// ============================================================

func (u *PaymentUsecase) Create(ctx context.Context, in paymentdom.CreatePaymentInput) (*paymentdom.Payment, error) {
	in.InvoiceID = stringsTrimSpace(in.InvoiceID)
	in.BillingAddressID = stringsTrimSpace(in.BillingAddressID)

	p, err := u.repo.Create(ctx, in)
	if err != nil {
		return nil, err
	}

	// paid/succeeded のときだけ後続処理（best-effort）
	if p == nil || !isPaidStatus(p.Status) {
		return p, nil
	}

	u.handlePostPaidBestEffort(ctx, p)
	return p, nil
}

func (u *PaymentUsecase) Update(ctx context.Context, id string, patch paymentdom.UpdatePaymentInput) (*paymentdom.Payment, error) {
	return u.repo.Update(ctx, stringsTrimSpace(id), patch)
}

func (u *PaymentUsecase) Delete(ctx context.Context, id string) error {
	return u.repo.Delete(ctx, stringsTrimSpace(id))
}

// Dev/Test helper
func (u *PaymentUsecase) Reset(ctx context.Context) error {
	return u.repo.Reset(ctx)
}

// local tiny helper (avoid importing strings everywhere in this file)
func stringsTrimSpace(s string) string {
	// implemented in payment_reflect_util.go as trimSpace(...)
	return trimSpace(s)
}
