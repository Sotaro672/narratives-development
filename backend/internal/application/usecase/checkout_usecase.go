// backend/internal/application/usecase/checkout_usecase.go
package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	invoicedom "narratives/internal/domain/invoice"
)

// StripeWebhookTrigger is an outbound port.
// A段階では「自分の /mall/webhooks/stripe をHTTPで叩く」実装を注入する。
type StripeWebhookTrigger interface {
	TriggerPaid(ctx context.Context, invoiceID, billingAddressID string, amount int) error
}

// CheckoutUsecase orchestrates "invoice -> trigger payment" flow (A).
// - InvoiceUsecase は invoice 起票だけに責務を限定したまま
// - その直後に webhook を叩く “オーケストレーション” をここに置く
type CheckoutUsecase struct {
	invoiceUC *InvoiceUsecase
	trigger   StripeWebhookTrigger
	now       func() time.Time
}

func NewCheckoutUsecase(invoiceUC *InvoiceUsecase, trigger StripeWebhookTrigger) *CheckoutUsecase {
	return &CheckoutUsecase{
		invoiceUC: invoiceUC,
		trigger:   trigger,
		now:       time.Now,
	}
}

var (
	ErrCheckoutInvoiceUsecaseMissing = errors.New("checkout: invoice usecase is not configured")
	ErrCheckoutTriggerMissing        = errors.New("checkout: stripe webhook trigger is not configured")
	ErrCheckoutOrderIDEmpty          = errors.New("checkout: orderId is empty")
	ErrCheckoutBillingAddrEmpty      = errors.New("checkout: billingAddressId is empty")
)

// CreateInvoiceAndTriggerPaymentInput is the app-level input for A flow.
type CreateInvoiceAndTriggerPaymentInput struct {
	OrderID          string
	Prices           []int
	Tax              int
	ShippingFee      int
	BillingAddressID string

	// Amount is optional.
	// If 0, it will be computed from Prices + Tax + ShippingFee.
	// (Your StripeWebhookHandler treats missing amount as 0, but having amount is useful for future checks.)
	Amount *int
}

// CreateInvoiceAndTriggerPayment does:
// 1) create invoice (paid=false)
// 2) immediately trigger webhook (A) to create payment -> invoice.paid=true is handled by PaymentUsecase
func (u *CheckoutUsecase) CreateInvoiceAndTriggerPayment(ctx context.Context, in CreateInvoiceAndTriggerPaymentInput) (invoicedom.Invoice, error) {
	if u.invoiceUC == nil {
		return invoicedom.Invoice{}, ErrCheckoutInvoiceUsecaseMissing
	}
	if u.trigger == nil {
		return invoicedom.Invoice{}, ErrCheckoutTriggerMissing
	}

	orderID := strings.TrimSpace(in.OrderID)
	billingAddrID := strings.TrimSpace(in.BillingAddressID)
	if orderID == "" {
		return invoicedom.Invoice{}, ErrCheckoutOrderIDEmpty
	}
	if billingAddrID == "" {
		return invoicedom.Invoice{}, ErrCheckoutBillingAddrEmpty
	}

	// 1) invoice create
	inv, err := u.invoiceUC.Create(ctx, CreateInvoiceInput{
		OrderID:     orderID,
		Prices:      in.Prices,
		Tax:         in.Tax,
		ShippingFee: in.ShippingFee,
	})
	if err != nil {
		return invoicedom.Invoice{}, err
	}

	// 2) trigger webhook (A)
	amount := computeAmount(in.Prices, in.Tax, in.ShippingFee)
	if in.Amount != nil {
		amount = *in.Amount
	}
	if amount < 0 {
		amount = 0
	}

	// invoiceID は「docId=orderId」前提の実装に合わせて orderId を使う
	invoiceID := strings.TrimSpace(inv.OrderID)
	if invoiceID == "" {
		// 念のため：invoice domain が別フィールドを持つ場合に備える
		invoiceID = orderID
	}

	if tErr := u.trigger.TriggerPaid(ctx, invoiceID, billingAddrID, amount); tErr != nil {
		// invoice は既に作られているので、ここで error を返すと呼び出し側で扱いが必要になる。
		// ただ、異常を明確に伝えたいので invoice を返しつつ error を返す。
		log.Printf("[checkout_uc] WARN: trigger webhook failed orderId=%s invoiceId=%s err=%v",
			orderID, invoiceID, tErr,
		)
		return inv, fmt.Errorf("checkout: webhook trigger failed: %w", tErr)
	}

	log.Printf("[checkout_uc] OK: invoice created and webhook triggered orderId=%s invoiceId=%s amount=%d",
		orderID, invoiceID, amount,
	)

	return inv, nil
}

func computeAmount(prices []int, tax int, shipping int) int {
	sum := 0
	for _, p := range prices {
		sum += p
	}
	sum += tax
	sum += shipping
	return sum
}
