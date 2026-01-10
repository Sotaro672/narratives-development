// backend/internal/application/usecase/payment_flow_usecase.go
package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"

	paymentdom "narratives/internal/domain/payment"
)

// StripeWebhookTrigger is an outbound port.
//
// Case A（責務分離）では、PaymentFlowUsecase が
// 「payment 起票後に外部決済（の代わりに）webhook を叩く」ために使う。
// 本番では Stripe などの外部決済 → webhook が来る想定だが、
// 開発段階では self /mall/webhooks/stripe を叩いて擬似的に paid を進められる。
type StripeWebhookTrigger interface {
	TriggerPaid(ctx context.Context, invoiceID, billingAddressID string, amount int) error
}

// PaymentFlowUsecase orchestrates "payment creation -> (optional) trigger paid webhook" (Case A).
//
// ✅ 責務分離を守る:
// - /mall/me/orders   : OrderHandler   -> OrderUsecase（order起票）
// - /mall/me/invoices : InvoiceHandler -> InvoiceUsecase（invoice起票）
// - /mall/me/payments : PaymentHandler -> PaymentFlowUsecase（payment起票 + 決済開始）
//
// NOTE:
// - invoice の起票はここではやらない（必ず事前に /mall/me/invoices で作られている前提）
// - invoice.paid=true の更新は webhook (StripeWebhookHandler) 側で行う想定
type PaymentFlowUsecase struct {
	paymentUC *PaymentUsecase
	trigger   StripeWebhookTrigger
	now       func() time.Time
}

func NewPaymentFlowUsecase(paymentUC *PaymentUsecase, trigger StripeWebhookTrigger) *PaymentFlowUsecase {
	return &PaymentFlowUsecase{
		paymentUC: paymentUC,
		trigger:   trigger, // nil でも良い（本番運用で webhook が外から来る場合）
		now:       time.Now,
	}
}

var (
	ErrPaymentFlowPaymentUsecaseMissing = errors.New("payment_flow: payment usecase is not configured")
	ErrPaymentFlowInvoiceIDEmpty        = errors.New("payment_flow: invoiceId is empty")
	ErrPaymentFlowBillingAddrEmpty      = errors.New("payment_flow: billingAddressId is empty")
	ErrPaymentFlowAmountInvalid         = errors.New("payment_flow: amount is invalid")
)

// CreatePaymentAndStartInput is the app-level input for Case A payment start.
type CreatePaymentAndStartInput struct {
	// InvoiceID is required.
	// Your current design uses docId = orderId for invoice.
	// So invoiceId == orderId is acceptable/expected.
	InvoiceID string

	// BillingAddressID is required.
	// In your current data flow, billingAddressId == uid(userId).
	BillingAddressID string

	// Amount is optional.
	// If nil, it is computed from Prices + Tax + ShippingFee.
	// If provided, it overrides computed value.
	Amount *int

	// Optional for computing Amount (if Amount is nil)
	Prices      []int
	Tax         int
	ShippingFee int

	// Optional: initial status override (rarely needed)
	// If empty, defaultStatus will be used.
	Status paymentdom.PaymentStatus

	// Optional
	ErrorType *string
	CreatedAt *time.Time
}

// CreatePaymentAndStart does:
// 1) create payment record
// 2) (optional) trigger webhook to simulate paid (dev)
//   - webhook handler will: create/confirm payment -> invoice.paid=true (and future inventory update)
//
// This method returns created Payment (value).
func (u *PaymentFlowUsecase) CreatePaymentAndStart(ctx context.Context, in CreatePaymentAndStartInput) (paymentdom.Payment, error) {
	if u == nil || u.paymentUC == nil {
		return paymentdom.Payment{}, ErrPaymentFlowPaymentUsecaseMissing
	}

	invoiceID := strings.TrimSpace(in.InvoiceID)
	billingAddrID := strings.TrimSpace(in.BillingAddressID)
	if invoiceID == "" {
		return paymentdom.Payment{}, ErrPaymentFlowInvoiceIDEmpty
	}
	if billingAddrID == "" {
		return paymentdom.Payment{}, ErrPaymentFlowBillingAddrEmpty
	}

	// amount
	amount := computeAmount(in.Prices, in.Tax, in.ShippingFee)
	if in.Amount != nil {
		amount = *in.Amount
	}
	if amount < 0 {
		return paymentdom.Payment{}, ErrPaymentFlowAmountInvalid
	}

	// status default
	status := in.Status
	if strings.TrimSpace(string(status)) == "" {
		// backend policy: accept any non-empty, but pick a consistent default
		status = paymentdom.PaymentStatus("created")
	}

	createdAt := u.now().UTC()
	if in.CreatedAt != nil && !in.CreatedAt.IsZero() {
		createdAt = in.CreatedAt.UTC()
	}

	// build domain payment
	p, err := paymentdom.New(
		invoiceID,
		billingAddrID,
		amount,
		status,
		in.ErrorType,
		createdAt,
	)
	if err != nil {
		log.Printf("[payment_flow_uc] domain.New failed invoiceId=%s err=%v", _pfMaskID(invoiceID), err)
		return paymentdom.Payment{}, err
	}

	// 1) create payment record via PaymentUsecase
	created, err := callPaymentCreate(u.paymentUC, ctx, p)
	if err != nil {
		log.Printf("[payment_flow_uc] paymentUC.Create failed invoiceId=%s err=%v", _pfMaskID(invoiceID), err)
		return paymentdom.Payment{}, err
	}

	log.Printf("[payment_flow_uc] Create OK invoiceId=%s billingAddressId=%s amount=%d status=%s",
		_pfMaskID(created.InvoiceID), _pfMaskID(created.BillingAddressID), created.Amount, created.Status,
	)

	// 2) optional: trigger webhook (dev/self)
	if u.trigger != nil {
		if tErr := u.trigger.TriggerPaid(ctx, invoiceID, billingAddrID, amount); tErr != nil {
			// payment is already created; return payment with warning error
			log.Printf("[payment_flow_uc] WARN trigger webhook failed invoiceId=%s err=%v", _pfMaskID(invoiceID), tErr)
			return created, fmt.Errorf("payment_flow: webhook trigger failed: %w", tErr)
		}
		log.Printf("[payment_flow_uc] OK webhook triggered invoiceId=%s amount=%d", _pfMaskID(invoiceID), amount)
	}

	return created, nil
}

// ------------------------------------------------------------
// Reflection bridge (avoid hard dependency on CreatePaymentInput shape)
// ------------------------------------------------------------

// callPaymentCreate calls PaymentUsecase.Create with one of supported signatures:
//   - Create(ctx, payment.Payment) (payment.Payment, error)
//   - Create(ctx, *payment.Payment) (*payment.Payment, error)
//   - Create(ctx, <struct{InvoiceID,...}>) (<payment.Payment or *payment.Payment>, error)
//
// This keeps PaymentFlowUsecase stable even if PaymentUsecase input type changes.
func callPaymentCreate(paymentUC any, ctx context.Context, p paymentdom.Payment) (paymentdom.Payment, error) {
	if paymentUC == nil {
		return paymentdom.Payment{}, ErrPaymentFlowPaymentUsecaseMissing
	}

	rv := reflect.ValueOf(paymentUC)
	if !rv.IsValid() {
		return paymentdom.Payment{}, ErrPaymentFlowPaymentUsecaseMissing
	}

	m := rv.MethodByName("Create")
	if !m.IsValid() {
		return paymentdom.Payment{}, errors.New("payment_flow: payment usecase missing method Create")
	}

	// Expect: Create(ctx, in) -> (out, err)
	if m.Type().NumIn() != 2 || m.Type().NumOut() != 2 {
		return paymentdom.Payment{}, errors.New("payment_flow: payment usecase Create has invalid signature")
	}

	inType := m.Type().In(1)
	arg, err := buildCreateArg(inType, p)
	if err != nil {
		return paymentdom.Payment{}, err
	}

	outs := m.Call([]reflect.Value{reflect.ValueOf(ctx), arg})
	if len(outs) != 2 {
		return paymentdom.Payment{}, errors.New("payment_flow: payment usecase Create has invalid signature")
	}

	// err
	if !outs[1].IsNil() {
		if e, ok := outs[1].Interface().(error); ok {
			return paymentdom.Payment{}, e
		}
		return paymentdom.Payment{}, errors.New("payment_flow: payment usecase Create returned non-error")
	}

	// out -> paymentdom.Payment
	outPayment, convErr := coerceToPayment(outs[0])
	if convErr != nil {
		return paymentdom.Payment{}, convErr
	}
	return outPayment, nil
}

func buildCreateArg(t reflect.Type, p paymentdom.Payment) (reflect.Value, error) {
	// direct paymentdom.Payment
	if t.AssignableTo(reflect.TypeOf(paymentdom.Payment{})) {
		return reflect.ValueOf(p), nil
	}
	// *paymentdom.Payment
	if t.AssignableTo(reflect.TypeOf(&paymentdom.Payment{})) {
		pp := p
		return reflect.ValueOf(&pp), nil
	}

	// struct input: set fields by name best-effort
	if t.Kind() == reflect.Struct {
		v := reflect.New(t).Elem()

		set := func(name string, val any) {
			f := v.FieldByName(name)
			if !f.IsValid() || !f.CanSet() {
				return
			}
			x := reflect.ValueOf(val)
			if x.IsValid() && x.Type().AssignableTo(f.Type()) {
				f.Set(x)
				return
			}
			// common: status might be alias type
			if x.IsValid() && x.Type().ConvertibleTo(f.Type()) {
				f.Set(x.Convert(f.Type()))
				return
			}
		}

		set("InvoiceID", p.InvoiceID)
		set("BillingAddressID", p.BillingAddressID)
		set("Amount", p.Amount)
		set("Status", p.Status)
		set("ErrorType", p.ErrorType)
		set("CreatedAt", p.CreatedAt)

		return v, nil
	}

	return reflect.Value{}, errors.New("payment_flow: unsupported PaymentUsecase.Create input type")
}

func coerceToPayment(v reflect.Value) (paymentdom.Payment, error) {
	if !v.IsValid() {
		return paymentdom.Payment{}, errors.New("payment_flow: payment usecase Create returned invalid value")
	}

	// handle pointers
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return paymentdom.Payment{}, errors.New("payment_flow: payment usecase Create returned nil payment")
		}
		return coerceToPayment(v.Elem())
	}

	// direct type assertion
	if v.Type().AssignableTo(reflect.TypeOf(paymentdom.Payment{})) {
		return v.Interface().(paymentdom.Payment), nil
	}

	return paymentdom.Payment{}, errors.New("payment_flow: payment usecase Create returned unsupported payment type")
}

// ------------------------------------------------------------

func computeAmount(prices []int, tax int, shipping int) int {
	sum := 0
	for _, p := range prices {
		sum += p
	}
	sum += tax
	sum += shipping
	return sum
}

// local mask helper
func _pfMaskID(s string) string {
	t := strings.TrimSpace(s)
	if t == "" {
		return ""
	}
	if len(t) <= 10 {
		return t
	}
	return t[:4] + "***" + t[len(t)-4:]
}
