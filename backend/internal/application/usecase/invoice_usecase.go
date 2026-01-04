// backend/internal/application/usecase/invoice_usecase.go
package usecase

import (
	"context"
	"strings"

	common "narratives/internal/domain/common"
	invoicedom "narratives/internal/domain/invoice"
)

// InvoiceRepo is the persistence port required by InvoiceUsecase.
type InvoiceRepo interface {
	// Queries
	GetByOrderID(ctx context.Context, orderID string) (invoicedom.Invoice, error)
	Exists(ctx context.Context, orderID string) (bool, error)
	Count(ctx context.Context, filter InvoiceFilter) (int, error)
	List(ctx context.Context, filter InvoiceFilter, sort common.Sort, page common.Page) (common.PageResult[invoicedom.Invoice], error)
	ListByCursor(ctx context.Context, filter InvoiceFilter, sort common.Sort, cpage common.CursorPage) (common.CursorPageResult[invoicedom.Invoice], error)

	// Commands
	Create(ctx context.Context, inv invoicedom.Invoice) (invoicedom.Invoice, error)
	Save(ctx context.Context, inv invoicedom.Invoice, opts *common.SaveOptions) (invoicedom.Invoice, error)
	DeleteByOrderID(ctx context.Context, orderID string) error
}

// InvoiceFilter provides basic filtering for listing invoices.
type InvoiceFilter struct {
	OrderID *string
	Paid    *bool
}

// InvoiceUsecase orchestrates invoice operations.
type InvoiceUsecase struct {
	repo InvoiceRepo
}

func NewInvoiceUsecase(repo InvoiceRepo) *InvoiceUsecase {
	return &InvoiceUsecase{repo: repo}
}

// =======================
// Queries
// =======================

func (u *InvoiceUsecase) GetByOrderID(ctx context.Context, orderID string) (invoicedom.Invoice, error) {
	return u.repo.GetByOrderID(ctx, strings.TrimSpace(orderID))
}

// 互換: handler の GET /invoices/{id} は "orderId" を渡す前提
func (u *InvoiceUsecase) GetByID(ctx context.Context, id string) (invoicedom.Invoice, error) {
	return u.repo.GetByOrderID(ctx, strings.TrimSpace(id))
}

func (u *InvoiceUsecase) Exists(ctx context.Context, orderID string) (bool, error) {
	return u.repo.Exists(ctx, strings.TrimSpace(orderID))
}

func (u *InvoiceUsecase) Count(ctx context.Context, f InvoiceFilter) (int, error) {
	return u.repo.Count(ctx, f)
}

func (u *InvoiceUsecase) List(ctx context.Context, f InvoiceFilter, s common.Sort, p common.Page) (common.PageResult[invoicedom.Invoice], error) {
	return u.repo.List(ctx, f, s, p)
}

func (u *InvoiceUsecase) ListByCursor(ctx context.Context, f InvoiceFilter, s common.Sort, c common.CursorPage) (common.CursorPageResult[invoicedom.Invoice], error) {
	return u.repo.ListByCursor(ctx, f, s, c)
}

// =======================
// Commands
// =======================

type CreateInvoiceInput struct {
	OrderID string
	Prices  []int

	Tax         int
	ShippingFee int

	// ✅ default paid=false when nil
	Paid *bool
}

func (u *InvoiceUsecase) Create(ctx context.Context, in CreateInvoiceInput) (invoicedom.Invoice, error) {
	orderID := strings.TrimSpace(in.OrderID)

	inv, err := invoicedom.New(
		orderID,
		in.Prices,
		in.Tax,
		in.ShippingFee,
	)
	if err != nil {
		return invoicedom.Invoice{}, err
	}

	paid := false
	if in.Paid != nil {
		paid = *in.Paid
	}
	inv.Paid = paid // ✅ default false / allow true if explicitly specified

	return u.repo.Create(ctx, inv)
}

type UpdateInvoicePaidInput struct {
	OrderID string
	Paid    bool
}

func (u *InvoiceUsecase) UpdatePaid(ctx context.Context, in UpdateInvoicePaidInput) (invoicedom.Invoice, error) {
	orderID := strings.TrimSpace(in.OrderID)
	cur, err := u.repo.GetByOrderID(ctx, orderID)
	if err != nil {
		return invoicedom.Invoice{}, err
	}

	if err := cur.SetPaid(in.Paid); err != nil {
		return invoicedom.Invoice{}, err
	}

	return u.repo.Save(ctx, cur, nil)
}

func (u *InvoiceUsecase) DeleteByOrderID(ctx context.Context, orderID string) error {
	return u.repo.DeleteByOrderID(ctx, strings.TrimSpace(orderID))
}
