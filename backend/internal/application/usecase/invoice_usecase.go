// backend\internal\application\usecase\invoice_usecase.go
package usecase

import (
	"context"
	"strings"

	invoicedom "narratives/internal/domain/invoice"
)

// InvoiceRepo defines the minimal persistence port needed by InvoiceUsecase.
type InvoiceRepo interface {
	GetByID(ctx context.Context, id string) (invoicedom.Invoice, error)
	Exists(ctx context.Context, id string) (bool, error)
	Create(ctx context.Context, v invoicedom.Invoice) (invoicedom.Invoice, error)
	Save(ctx context.Context, v invoicedom.Invoice) (invoicedom.Invoice, error)
	Delete(ctx context.Context, id string) error
}

// InvoiceUsecase orchestrates invoice operations.
type InvoiceUsecase struct {
	repo InvoiceRepo
}

func NewInvoiceUsecase(repo InvoiceRepo) *InvoiceUsecase {
	return &InvoiceUsecase{repo: repo}
}

// Queries

func (u *InvoiceUsecase) GetByID(ctx context.Context, id string) (invoicedom.Invoice, error) {
	return u.repo.GetByID(ctx, strings.TrimSpace(id))
}

func (u *InvoiceUsecase) Exists(ctx context.Context, id string) (bool, error) {
	return u.repo.Exists(ctx, strings.TrimSpace(id))
}

// Commands

func (u *InvoiceUsecase) Create(ctx context.Context, v invoicedom.Invoice) (invoicedom.Invoice, error) {
	return u.repo.Create(ctx, v)
}

func (u *InvoiceUsecase) Save(ctx context.Context, v invoicedom.Invoice) (invoicedom.Invoice, error) {
	return u.repo.Save(ctx, v)
}

func (u *InvoiceUsecase) Delete(ctx context.Context, id string) error {
	return u.repo.Delete(ctx, strings.TrimSpace(id))
}
