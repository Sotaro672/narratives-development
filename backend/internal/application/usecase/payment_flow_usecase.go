// backend/internal/application/usecase/payment_flow_usecase.go
package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	paymentdom "narratives/internal/domain/payment"
)

// StripePaymentIntentGateway is an outbound port for real Stripe payment execution.
//
// 支払ボタン押下後の購入決済では、SetupIntent ではなく PaymentIntent を使う。
// backend が Stripe secret key を使って PaymentIntent を作成し、
// 登録済み payment method で confirm する。
type StripePaymentIntentGateway interface {
	CreateAndConfirmPaymentIntent(
		ctx context.Context,
		in CreateAndConfirmPaymentIntentInput,
	) (*CreateAndConfirmPaymentIntentResult, error)
}

type CreateAndConfirmPaymentIntentInput struct {
	StripeCustomerID      string
	StripePaymentMethodID string
	Amount                int
	Currency              string
	IdempotencyKey        string
	Description           string

	PaymentMethodID string
}

type CreateAndConfirmPaymentIntentResult struct {
	StripePaymentIntentID string
	Status                string
	ClientSecret          string
	RequiresAction        bool

	ErrorType    string
	ErrorCode    string
	ErrorMessage string
}

// PaymentFlowUsecase orchestrates:
// 1. payment record 作成
// 2. Stripe PaymentIntent 作成/confirm
// 3. succeeded の場合、paid 後処理
//
// 責務分離:
// - /mall/me/orders   : OrderHandler   -> OrderUsecase
// - /mall/me/payments : PaymentHandler -> PaymentFlowUsecase
type PaymentFlowUsecase struct {
	paymentUC *PaymentUsecase

	paymentIntentGateway StripePaymentIntentGateway

	now func() time.Time
}

// NewPaymentFlowUsecase creates PaymentFlowUsecase for Stripe PaymentIntent based payment flow.
func NewPaymentFlowUsecase(
	paymentUC *PaymentUsecase,
	paymentIntentGateway StripePaymentIntentGateway,
) *PaymentFlowUsecase {
	return &PaymentFlowUsecase{
		paymentUC:            paymentUC,
		paymentIntentGateway: paymentIntentGateway,
		now:                  time.Now,
	}
}

func (u *PaymentFlowUsecase) SetPaymentIntentGateway(
	paymentIntentGateway StripePaymentIntentGateway,
) {
	if u == nil {
		return
	}
	u.paymentIntentGateway = paymentIntentGateway
}

var (
	ErrPaymentFlowPaymentUsecaseMissing = errors.New("payment_flow: payment usecase is not configured")
	ErrPaymentFlowPaymentIDEmpty        = errors.New("payment_flow: paymentId is empty")
	ErrPaymentFlowPaymentMethodEmpty    = errors.New("payment_flow: paymentMethodId is empty")
	ErrPaymentFlowAmountInvalid         = errors.New("payment_flow: amount is invalid")

	ErrPaymentFlowStripeGatewayMissing        = errors.New("payment_flow: stripe payment intent gateway is not configured")
	ErrPaymentFlowStripeCustomerIDEmpty       = errors.New("payment_flow: stripeCustomerId is empty")
	ErrPaymentFlowStripePaymentMethodIDEmpty  = errors.New("payment_flow: stripePaymentMethodId is empty")
	ErrPaymentFlowStripePaymentIntentIDEmpty  = errors.New("payment_flow: stripePaymentIntentId is empty")
	ErrPaymentFlowStripePaymentIntentFailed   = errors.New("payment_flow: stripe payment intent failed")
	ErrPaymentFlowStripePaymentIntentCanceled = errors.New("payment_flow: stripe payment intent canceled")
)

// CreatePaymentAndStartInput is the app-level input for payment start.
//
// PaymentID must be the same value as order.ID.
// It is used as the Firestore payment document ID.
//
// 正の payment record schema:
//
//	amount
//	createdAt
//	paymentMethodId
//	status
//	stripeCustomerId
//	stripePaymentIntentId
//	stripePaymentMethodId
type CreatePaymentAndStartInput struct {
	PaymentID string

	PaymentMethodID string

	StripeCustomerID      string
	StripePaymentMethodID string

	Amount *int
}

// CreatePaymentAndStartResult is the response-friendly result.
//
// Handler はこの結果を JSON へ変換して frontend に返す。
// requiresAction=true の場合、frontend は clientSecret を使って
// stripe.confirmCardPayment(clientSecret) を呼ぶ。
type CreatePaymentAndStartResult struct {
	Payment paymentdom.Payment

	PaymentID string

	Status paymentdom.PaymentStatus

	StripePaymentIntentID string
	ClientSecret          string
	RequiresAction        bool

	ErrorType    *string
	ErrorCode    *string
	ErrorMessage *string
}

// CreatePaymentAndStartWithResult does:
// 1. create payment record as PENDING
// 2. execute Stripe PaymentIntent create + confirm
// 3. persist latest payment status
// 4. if succeeded, run post-paid side effects
// 5. if requires_action, return clientSecret to frontend
func (u *PaymentFlowUsecase) CreatePaymentAndStartWithResult(
	ctx context.Context,
	in CreatePaymentAndStartInput,
) (*CreatePaymentAndStartResult, error) {
	if u == nil || u.paymentUC == nil {
		return nil, ErrPaymentFlowPaymentUsecaseMissing
	}

	if u.paymentIntentGateway == nil {
		return nil, ErrPaymentFlowStripeGatewayMissing
	}

	paymentID := strings.TrimSpace(in.PaymentID)
	paymentMethodID := strings.TrimSpace(in.PaymentMethodID)
	stripeCustomerID := strings.TrimSpace(in.StripeCustomerID)
	stripePaymentMethodID := strings.TrimSpace(in.StripePaymentMethodID)

	if paymentID == "" {
		return nil, ErrPaymentFlowPaymentIDEmpty
	}

	if paymentMethodID == "" {
		return nil, ErrPaymentFlowPaymentMethodEmpty
	}

	if stripeCustomerID == "" {
		return nil, ErrPaymentFlowStripeCustomerIDEmpty
	}

	if stripePaymentMethodID == "" {
		return nil, ErrPaymentFlowStripePaymentMethodIDEmpty
	}

	amount := 0
	if in.Amount != nil {
		amount = *in.Amount
	}

	if amount <= 0 {
		return nil, ErrPaymentFlowAmountInvalid
	}

	createdAt := u.now().UTC()

	p, err := paymentdom.New(
		paymentID,
		paymentMethodID,
		stripeCustomerID,
		stripePaymentMethodID,
		"",
		amount,
		paymentdom.StatusPending,
		nil,
		nil,
		nil,
		createdAt,
	)
	if err != nil {
		log.Printf(
			"[payment_flow_uc] domain.New failed paymentId=%s paymentMethodId=%s err=%v",
			paymentID,
			paymentMethodID,
			err,
		)
		return nil, err
	}

	created, err := u.paymentUC.Create(ctx, p)
	if err != nil {
		log.Printf(
			"[payment_flow_uc] paymentUC.Create failed paymentId=%s paymentMethodId=%s err=%v",
			paymentID,
			paymentMethodID,
			err,
		)
		return nil, err
	}

	if created == nil {
		return nil, errors.New("payment_flow: created payment is nil")
	}

	log.Printf(
		"[payment_flow_uc] Create OK paymentId=%s paymentMethodId=%s amount=%d status=%s",
		created.PaymentID,
		created.PaymentMethodID,
		created.Amount,
		created.Status,
	)

	return u.startStripePaymentIntent(ctx, *created)
}

func (u *PaymentFlowUsecase) startStripePaymentIntent(
	ctx context.Context,
	created paymentdom.Payment,
) (*CreatePaymentAndStartResult, error) {
	if u == nil || u.paymentIntentGateway == nil {
		return nil, ErrPaymentFlowStripeGatewayMissing
	}

	paymentID := strings.TrimSpace(created.PaymentID)
	paymentMethodID := strings.TrimSpace(created.PaymentMethodID)
	stripeCustomerID := strings.TrimSpace(created.StripeCustomerID)
	stripePaymentMethodID := strings.TrimSpace(created.StripePaymentMethodID)

	if paymentID == "" {
		return nil, ErrPaymentFlowPaymentIDEmpty
	}

	if paymentMethodID == "" {
		return nil, ErrPaymentFlowPaymentMethodEmpty
	}

	if stripeCustomerID == "" {
		return nil, ErrPaymentFlowStripeCustomerIDEmpty
	}

	if stripePaymentMethodID == "" {
		return nil, ErrPaymentFlowStripePaymentMethodIDEmpty
	}

	if created.Amount <= 0 {
		return nil, ErrPaymentFlowAmountInvalid
	}

	idempotencyKey := fmt.Sprintf(
		"payment:%s:%s:%d",
		paymentID,
		paymentMethodID,
		created.Amount,
	)

	pi, err := u.paymentIntentGateway.CreateAndConfirmPaymentIntent(
		ctx,
		CreateAndConfirmPaymentIntentInput{
			StripeCustomerID:      stripeCustomerID,
			StripePaymentMethodID: stripePaymentMethodID,
			Amount:                created.Amount,
			Currency:              "jpy",
			IdempotencyKey:        idempotencyKey,
			Description:           fmt.Sprintf("AMOL payment paymentId=%s", paymentID),
			PaymentMethodID:       paymentMethodID,
		},
	)
	if err != nil {
		log.Printf("[payment_flow_uc] Stripe PaymentIntent failed paymentId=%s err=%v", paymentID, err)

		failed := created
		failed.Status = paymentdom.StatusFailed

		errMsg := err.Error()
		failed.ErrorMsg = &errMsg

		_ = u.paymentUC.Update(ctx, failed)

		return &CreatePaymentAndStartResult{
			Payment:      failed,
			PaymentID:    failed.PaymentID,
			Status:       failed.Status,
			ErrorMessage: &errMsg,
		}, err
	}

	if pi == nil {
		return nil, errors.New("payment_flow: stripe payment intent result is nil")
	}

	stripePaymentIntentID := strings.TrimSpace(pi.StripePaymentIntentID)
	if stripePaymentIntentID == "" {
		return nil, ErrPaymentFlowStripePaymentIntentIDEmpty
	}

	stripeStatus := strings.TrimSpace(strings.ToLower(pi.Status))

	switch stripeStatus {
	case "succeeded":
		succeeded := created
		succeeded.Status = paymentdom.StatusSucceeded
		succeeded.StripePaymentIntentID = stripePaymentIntentID

		if err := u.paymentUC.Update(ctx, succeeded); err != nil {
			return nil, err
		}

		log.Printf(
			"[payment_flow_uc] Stripe PaymentIntent succeeded paymentId=%s paymentIntentId=%s amount=%d",
			succeeded.PaymentID,
			stripePaymentIntentID,
			succeeded.Amount,
		)

		u.paymentUC.handlePostPaidBestEffort(ctx, &succeeded)

		return &CreatePaymentAndStartResult{
			Payment:               succeeded,
			PaymentID:             succeeded.PaymentID,
			Status:                succeeded.Status,
			StripePaymentIntentID: stripePaymentIntentID,
			ClientSecret:          strings.TrimSpace(pi.ClientSecret),
			RequiresAction:        false,
		}, nil

	case "requires_action", "requires_source_action":
		pending := created
		pending.Status = paymentdom.StatusRequiresAction
		pending.StripePaymentIntentID = stripePaymentIntentID

		if err := u.paymentUC.Update(ctx, pending); err != nil {
			return nil, err
		}

		log.Printf(
			"[payment_flow_uc] Stripe PaymentIntent requires_action paymentId=%s paymentIntentId=%s",
			pending.PaymentID,
			stripePaymentIntentID,
		)

		return &CreatePaymentAndStartResult{
			Payment:               pending,
			PaymentID:             pending.PaymentID,
			Status:                pending.Status,
			StripePaymentIntentID: stripePaymentIntentID,
			ClientSecret:          strings.TrimSpace(pi.ClientSecret),
			RequiresAction:        true,
		}, nil

	case "processing":
		processing := created
		processing.Status = paymentdom.StatusProcessing
		processing.StripePaymentIntentID = stripePaymentIntentID

		if err := u.paymentUC.Update(ctx, processing); err != nil {
			return nil, err
		}

		return &CreatePaymentAndStartResult{
			Payment:               processing,
			PaymentID:             processing.PaymentID,
			Status:                processing.Status,
			StripePaymentIntentID: stripePaymentIntentID,
			ClientSecret:          strings.TrimSpace(pi.ClientSecret),
			RequiresAction:        pi.RequiresAction,
		}, nil

	case "requires_confirmation", "requires_payment_method":
		pending := created
		pending.Status = paymentdom.StatusPending
		pending.StripePaymentIntentID = stripePaymentIntentID

		if err := u.paymentUC.Update(ctx, pending); err != nil {
			return nil, err
		}

		return &CreatePaymentAndStartResult{
			Payment:               pending,
			PaymentID:             pending.PaymentID,
			Status:                pending.Status,
			StripePaymentIntentID: stripePaymentIntentID,
			ClientSecret:          strings.TrimSpace(pi.ClientSecret),
			RequiresAction:        pi.RequiresAction,
		}, nil

	case "canceled":
		canceled := created
		canceled.Status = paymentdom.StatusCanceled
		canceled.StripePaymentIntentID = stripePaymentIntentID

		errorMsg := "Stripe PaymentIntent was canceled"
		canceled.ErrorMsg = &errorMsg

		if err := u.paymentUC.Update(ctx, canceled); err != nil {
			return nil, err
		}

		return &CreatePaymentAndStartResult{
			Payment:               canceled,
			PaymentID:             canceled.PaymentID,
			Status:                canceled.Status,
			StripePaymentIntentID: stripePaymentIntentID,
			ErrorMessage:          &errorMsg,
		}, ErrPaymentFlowStripePaymentIntentCanceled

	default:
		failed := created
		failed.Status = paymentdom.StatusFailed
		failed.StripePaymentIntentID = stripePaymentIntentID

		msg := fmt.Sprintf("Stripe PaymentIntent status is unsupported or failed: %s", stripeStatus)
		if pi.ErrorMessage != "" {
			msg = pi.ErrorMessage
		}

		failed.ErrorMsg = &msg

		if pi.ErrorType != "" {
			v := pi.ErrorType
			failed.ErrorType = &v
		}
		if pi.ErrorCode != "" {
			v := pi.ErrorCode
			failed.ErrorCode = &v
		}

		if err := u.paymentUC.Update(ctx, failed); err != nil {
			return nil, err
		}

		return &CreatePaymentAndStartResult{
			Payment:               failed,
			PaymentID:             failed.PaymentID,
			Status:                failed.Status,
			StripePaymentIntentID: stripePaymentIntentID,
			ErrorType:             failed.ErrorType,
			ErrorCode:             failed.ErrorCode,
			ErrorMessage:          failed.ErrorMsg,
		}, ErrPaymentFlowStripePaymentIntentFailed
	}
}
