// backend\internal\application\usecase\invoice_usecase.go
package usecase

import (
	"context"
	"log"
	"strings"
	"time"

	common "narratives/internal/domain/common"
	invoicedom "narratives/internal/domain/invoice"
	paymentdom "narratives/internal/domain/payment"
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

// ✅ Payment 起票は PaymentUsecase に寄せる（小さい口だけ定義）
type PaymentCreator interface {
	Create(ctx context.Context, in paymentdom.CreatePaymentInput) (*paymentdom.Payment, error)
}

// InvoiceFilter provides basic filtering for listing invoices.
type InvoiceFilter struct {
	OrderID *string
	Paid    *bool
}

// InvoiceUsecase orchestrates invoice operations.
type InvoiceUsecase struct {
	repo           InvoiceRepo
	paymentCreator PaymentCreator
	now            func() time.Time
}

func NewInvoiceUsecase(repo InvoiceRepo, paymentCreator PaymentCreator) *InvoiceUsecase {
	return &InvoiceUsecase{
		repo:           repo,
		paymentCreator: paymentCreator,
		now:            time.Now,
	}
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

// ✅ 起票（Create）は「常に paid=false」で作る
type CreateInvoiceInput struct {
	OrderID string
	Prices  []int

	Tax         int
	ShippingFee int

	// ✅ payment起票に必要な情報（invoice起票と同時に生成する）
	BillingAddressID string
	PaymentStatus    paymentdom.PaymentStatus
	PaymentErrorType *string
}

func (u *InvoiceUsecase) Create(ctx context.Context, in CreateInvoiceInput) (invoicedom.Invoice, error) {
	orderID := strings.TrimSpace(in.OrderID)

	log.Printf("[invoice_uc] Create called orderId=%s prices_len=%d tax=%d ship=%d",
		orderID, len(in.Prices), in.Tax, in.ShippingFee,
	)

	inv, err := invoicedom.New(
		orderID,
		in.Prices,
		in.Tax,
		in.ShippingFee,
	)
	if err != nil {
		log.Printf("[invoice_uc] Create New failed orderId=%s err=%v", orderID, err)
		return invoicedom.Invoice{}, err
	}

	// ✅ paid は触らない（New の default=false のまま）
	out, err := u.repo.Create(ctx, inv)
	if err != nil {
		log.Printf("[invoice_uc] Create repo.Create failed orderId=%s err=%v", orderID, err)
		return invoicedom.Invoice{}, err
	}

	// ✅ payment 起票は PaymentUsecase に委譲（ここでは「依頼」だけ）
	if u.paymentCreator != nil {
		billingAddrID := strings.TrimSpace(in.BillingAddressID)
		if billingAddrID != "" {
			amount := 0
			for _, p := range in.Prices {
				amount += p
			}
			amount += in.Tax
			amount += in.ShippingFee

			st := in.PaymentStatus
			if strings.TrimSpace(string(st)) == "" {
				st = paymentdom.PaymentStatusPending
			}

			_, pErr := u.paymentCreator.Create(ctx, paymentdom.CreatePaymentInput{
				InvoiceID:        orderID, // ✅ docId=invoiceId (=orderId) 前提
				BillingAddressID: billingAddrID,
				Amount:           amount,
				Status:           st,
				ErrorType:        in.PaymentErrorType,
			})
			if pErr != nil {
				log.Printf("[invoice_uc] Create payment failed orderId=%s err=%v", orderID, pErr)
				return invoicedom.Invoice{}, pErr
			}
		}
	}

	log.Printf("[invoice_uc] Create OK orderId=%s paid=%t updatedAt_nil=%t",
		out.OrderID, out.Paid, out.UpdatedAt == nil,
	)
	return out, nil
}

func (u *InvoiceUsecase) DeleteByOrderID(ctx context.Context, orderID string) error {
	return u.repo.DeleteByOrderID(ctx, strings.TrimSpace(orderID))
}
