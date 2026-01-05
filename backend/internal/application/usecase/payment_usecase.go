package usecase

import (
	"context"
	"strings"

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

// PaymentUsecase orchestrates payment operations.
type PaymentUsecase struct {
	repo PaymentRepo
}

func NewPaymentUsecase(repo PaymentRepo) *PaymentUsecase {
	return &PaymentUsecase{repo: repo}
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
	return u.repo.Create(ctx, in)
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
