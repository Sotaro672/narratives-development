// backend/internal/application/usecase/payment_usecase.go
package usecase

import (
	"context"
	"log"
	"reflect"
	"strings"
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
//
// NOTE:
// - invoice の読み取りと保存だけできればよい
// - Firestore 実装（InvoiceRepositoryFS）が同等のメソッドを持っていればそのまま注入できる
type InvoiceRepoForPayment interface {
	GetByOrderID(ctx context.Context, orderID string) (invoicedom.Invoice, error)
	Save(ctx context.Context, inv invoicedom.Invoice, opts *common.SaveOptions) (invoicedom.Invoice, error)
}

// PaymentUsecase orchestrates payment operations.
type PaymentUsecase struct {
	repo        PaymentRepo
	invoiceRepo InvoiceRepoForPayment
	now         func() time.Time
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

// ============================================================
// Queries
// ============================================================

func (u *PaymentUsecase) GetByID(ctx context.Context, id string) (*paymentdom.Payment, error) {
	return u.repo.GetByID(ctx, strings.TrimSpace(id))
}

// docId=invoiceId 前提なら「invoiceID=paymentID」なので GetByID と実質同じ。
// ただし domain port に合わせて残す。
func (u *PaymentUsecase) GetByInvoiceID(ctx context.Context, invoiceID string) ([]paymentdom.Payment, error) {
	return u.repo.GetByInvoiceID(ctx, strings.TrimSpace(invoiceID))
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
	in.InvoiceID = strings.TrimSpace(in.InvoiceID)
	in.BillingAddressID = strings.TrimSpace(in.BillingAddressID)

	p, err := u.repo.Create(ctx, in)
	if err != nil {
		return nil, err
	}

	// ✅ payment 起票後、成功扱いなら invoice.paid=true にする（best-effort）
	// - 現状は webhook から「全て支払い済み」で Create される前提なので、ここで paid を立てるのが自然
	// - 本番では "paid/succeeded" のときだけ立てる
	if p != nil && u.invoiceRepo != nil && isPaidStatus(p.Status) {
		if mkErr := u.markInvoicePaid(ctx, p.InvoiceID); mkErr != nil {
			// payment は既に作成されているため、ここで error を返すと呼び出し側が再試行しづらい。
			// まずはログだけ残して成功を返す（運用上は後で整合処理 or 冪等再処理を入れる）
			log.Printf("[payment_uc] WARN: invoice mark paid failed invoiceId=%s err=%v", p.InvoiceID, mkErr)
		}
	}

	return p, nil
}

func (u *PaymentUsecase) Update(ctx context.Context, id string, patch paymentdom.UpdatePaymentInput) (*paymentdom.Payment, error) {
	return u.repo.Update(ctx, strings.TrimSpace(id), patch)
}

func (u *PaymentUsecase) Delete(ctx context.Context, id string) error {
	return u.repo.Delete(ctx, strings.TrimSpace(id))
}

// Dev/Test helper
func (u *PaymentUsecase) Reset(ctx context.Context) error {
	return u.repo.Reset(ctx)
}

// ============================================================
// Internal helpers
// ============================================================

func isPaidStatus(st paymentdom.PaymentStatus) bool {
	s := strings.TrimSpace(string(st))
	if s == "" {
		return false
	}
	// 現状は mock で "paid" を使っている
	if strings.EqualFold(s, "paid") {
		return true
	}
	// 将来の provider に備えた許容（必要なければ消してOK）
	if strings.EqualFold(s, "succeeded") || strings.EqualFold(s, "success") {
		return true
	}
	return false
}

func (u *PaymentUsecase) markInvoicePaid(ctx context.Context, invoiceID string) error {
	invoiceID = strings.TrimSpace(invoiceID)
	if invoiceID == "" || u.invoiceRepo == nil {
		return nil
	}

	inv, err := u.invoiceRepo.GetByOrderID(ctx, invoiceID)
	if err != nil {
		return err
	}

	now := u.now().UTC()
	changed := setInvoicePaidBestEffort(&inv, now)
	if !changed {
		// 触れなかった（フィールドが無い等）場合も Save はしない
		return nil
	}

	_, err = u.invoiceRepo.Save(ctx, inv, nil)
	return err
}

// setInvoicePaidBestEffort tries to set:
// - inv.Paid = true
// - inv.UpdatedAt = &now (if exists and settable)
// It returns true if it set Paid or UpdatedAt.
func setInvoicePaidBestEffort(inv any, now time.Time) bool {
	if inv == nil {
		return false
	}

	rv := reflect.ValueOf(inv)
	if !rv.IsValid() {
		return false
	}
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return false
	}

	ev := rv.Elem()
	if !ev.IsValid() || ev.Kind() != reflect.Struct {
		return false
	}

	changed := false

	// Paid bool
	if f := ev.FieldByName("Paid"); f.IsValid() && f.CanSet() && f.Kind() == reflect.Bool {
		if f.Bool() == false {
			f.SetBool(true)
			changed = true
		}
	}

	// UpdatedAt *time.Time
	if f := ev.FieldByName("UpdatedAt"); f.IsValid() && f.CanSet() {
		// accept *time.Time only
		if f.Kind() == reflect.Pointer && f.Type().Elem() == reflect.TypeOf(time.Time{}) {
			t := now
			f.Set(reflect.ValueOf(&t))
			changed = true
		}
	}

	return changed
}
