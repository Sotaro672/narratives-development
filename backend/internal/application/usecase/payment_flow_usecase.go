// backend/internal/application/usecase/payment_flow_usecase.go
package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	orderdom "narratives/internal/domain/order"
	paymentdom "narratives/internal/domain/payment"
)

// OrderReaderForPaymentFlow provides the server-side order source of truth
// required before starting a payment.
type OrderReaderForPaymentFlow interface {
	GetByID(
		ctx context.Context,
		id string,
	) (orderdom.Order, error)
}

// StripePaymentIntentGateway is an outbound port for real Stripe payment
// execution.
//
// A purchase payment uses PaymentIntent rather than SetupIntent.
// The backend creates the PaymentIntent with a Stripe secret key and confirms
// it using the registered Stripe PaymentMethod.
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
//
//  1. Verify the authenticated user and order ownership.
//  2. Verify unpaid state and amount using the server-side Order.
//  3. Create and confirm the Stripe PaymentIntent.
//  4. Verify that Stripe returned a non-empty PaymentIntent ID.
//  5. Create the payment record with that PaymentIntent ID.
//  6. Let PaymentUsecase run post-paid processing when status is succeeded.
//
// Responsibility separation:
// - /mall/me/orders   : OrderHandler   -> OrderUsecase
// - /mall/me/payments : PaymentHandler -> PaymentFlowUsecase
type PaymentFlowUsecase struct {
	paymentUC *PaymentUsecase

	orderReader          OrderReaderForPaymentFlow
	paymentIntentGateway StripePaymentIntentGateway

	now func() time.Time
}

// NewPaymentFlowUsecase creates a PaymentFlowUsecase for a Stripe
// PaymentIntent-based payment flow.
func NewPaymentFlowUsecase(
	paymentUC *PaymentUsecase,
	orderReader OrderReaderForPaymentFlow,
	paymentIntentGateway StripePaymentIntentGateway,
) *PaymentFlowUsecase {
	return &PaymentFlowUsecase{
		paymentUC:            paymentUC,
		orderReader:          orderReader,
		paymentIntentGateway: paymentIntentGateway,
		now:                  time.Now,
	}
}

func (u *PaymentFlowUsecase) SetOrderReader(
	orderReader OrderReaderForPaymentFlow,
) {
	if u == nil {
		return
	}

	u.orderReader = orderReader
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
	ErrPaymentFlowPaymentUsecaseMissing = errors.New(
		"payment_flow: payment usecase is not configured",
	)
	ErrPaymentFlowOrderReaderMissing = errors.New(
		"payment_flow: order reader is not configured",
	)
	ErrPaymentFlowPaymentIDEmpty = errors.New(
		"payment_flow: paymentId is empty",
	)
	ErrPaymentFlowUserIDEmpty = errors.New(
		"payment_flow: userId is empty",
	)
	ErrPaymentFlowPaymentMethodEmpty = errors.New(
		"payment_flow: paymentMethodId is empty",
	)
	ErrPaymentFlowAmountInvalid = errors.New(
		"payment_flow: amount is invalid",
	)
	ErrPaymentFlowOrderIDMismatch = errors.New(
		"payment_flow: invalid order id",
	)
	ErrPaymentFlowOrderOwnerMismatch = errors.New(
		"payment_flow: invalid order owner",
	)
	ErrPaymentFlowOrderAlreadyPaid = errors.New(
		"payment_flow: invalid order state: already paid",
	)
	ErrPaymentFlowOrderAmountInvalid = errors.New(
		"payment_flow: invalid order amount",
	)
	ErrPaymentFlowOrderAmountMismatch = errors.New(
		"payment_flow: invalid amount: order total mismatch",
	)

	ErrPaymentFlowStripeGatewayMissing = errors.New(
		"payment_flow: stripe payment intent gateway is not configured",
	)
	ErrPaymentFlowStripeCustomerIDEmpty = errors.New(
		"payment_flow: stripeCustomerId is empty",
	)
	ErrPaymentFlowStripePaymentMethodIDEmpty = errors.New(
		"payment_flow: stripePaymentMethodId is empty",
	)
	ErrPaymentFlowStripePaymentIntentIDEmpty = errors.New(
		"payment_flow: stripePaymentIntentId is empty",
	)
	ErrPaymentFlowStripePaymentIntentFailed = errors.New(
		"payment_flow: stripe payment intent failed",
	)
	ErrPaymentFlowStripePaymentIntentCanceled = errors.New(
		"payment_flow: stripe payment intent canceled",
	)
)

// CreatePaymentAndStartInput is the application-level input for starting a
// payment.
//
// UserID must be obtained from the authenticated request context.
// PaymentID must be the same value as order.ID.
// PaymentID is used as the Firestore payment document ID.
//
// Amount is the client-requested amount. PaymentFlowUsecase compares it with
// the total calculated from the server-side Order and uses the server-side
// total for payment processing.
type CreatePaymentAndStartInput struct {
	UserID string

	PaymentID string

	PaymentMethodID string

	StripeCustomerID      string
	StripePaymentMethodID string

	Amount *int
}

// CreatePaymentAndStartResult is the response-friendly result.
//
// If RequiresAction is true, the frontend uses ClientSecret to complete
// additional Stripe authentication.
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

// CreatePaymentAndStartWithResult performs the complete payment start flow:
//
//  1. Validate the authenticated user.
//  2. Read and validate the server-side Order.
//  3. Compare the requested amount with the authoritative order total.
//  4. Create and confirm a Stripe PaymentIntent.
//  5. Require a non-empty Stripe PaymentIntent ID.
//  6. Create the payment record with the latest Stripe status.
//  7. Return ClientSecret when additional authentication is required.
//
// No payment document is created before Stripe returns a PaymentIntent ID.
func (u *PaymentFlowUsecase) CreatePaymentAndStartWithResult(
	ctx context.Context,
	in CreatePaymentAndStartInput,
) (*CreatePaymentAndStartResult, error) {
	if u == nil || u.paymentUC == nil {
		return nil, ErrPaymentFlowPaymentUsecaseMissing
	}

	if u.orderReader == nil {
		return nil, ErrPaymentFlowOrderReaderMissing
	}

	if u.paymentIntentGateway == nil {
		return nil, ErrPaymentFlowStripeGatewayMissing
	}

	userID := strings.TrimSpace(in.UserID)
	paymentID := strings.TrimSpace(in.PaymentID)
	paymentMethodID := strings.TrimSpace(in.PaymentMethodID)
	stripeCustomerID := strings.TrimSpace(in.StripeCustomerID)
	stripePaymentMethodID := strings.TrimSpace(
		in.StripePaymentMethodID,
	)

	if userID == "" {
		return nil, ErrPaymentFlowUserIDEmpty
	}

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

	requestedAmount := 0
	if in.Amount != nil {
		requestedAmount = *in.Amount
	}

	if requestedAmount <= 0 {
		return nil, ErrPaymentFlowAmountInvalid
	}

	order, err := u.orderReader.GetByID(
		ctx,
		paymentID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"payment_flow: get order %q: %w",
			paymentID,
			err,
		)
	}

	if strings.TrimSpace(order.ID) != paymentID {
		return nil, ErrPaymentFlowOrderIDMismatch
	}

	if strings.TrimSpace(order.UserID) != userID {
		return nil, ErrPaymentFlowOrderOwnerMismatch
	}

	if order.Paid {
		return nil, ErrPaymentFlowOrderAlreadyPaid
	}

	orderAmount, err := calculatePaymentOrderAmount(order)
	if err != nil {
		return nil, err
	}

	if requestedAmount != orderAmount {
		return nil, ErrPaymentFlowOrderAmountMismatch
	}

	// Stripe and the payment document use the amount calculated from the
	// server-side Order, never the unverified client value.
	amount := orderAmount

	idempotencyKey := fmt.Sprintf(
		"payment:%s:%s:%d",
		paymentID,
		paymentMethodID,
		amount,
	)

	pi, stripeErr :=
		u.paymentIntentGateway.CreateAndConfirmPaymentIntent(
			ctx,
			CreateAndConfirmPaymentIntentInput{
				StripeCustomerID:      stripeCustomerID,
				StripePaymentMethodID: stripePaymentMethodID,
				Amount:                amount,
				Currency:              "jpy",
				IdempotencyKey:        idempotencyKey,
				Description: fmt.Sprintf(
					"AMOL payment paymentId=%s",
					paymentID,
				),
				PaymentMethodID: paymentMethodID,
			},
		)

	// Without a Stripe result there is no PaymentIntent ID that can satisfy
	// the Payment domain invariant. Therefore, no payment document is
	// created.
	if pi == nil {
		if stripeErr != nil {
			return nil, fmt.Errorf(
				"payment_flow: create and confirm Stripe PaymentIntent: %w",
				stripeErr,
			)
		}

		return nil, errors.New(
			"payment_flow: stripe payment intent result is nil",
		)
	}

	stripePaymentIntentID := strings.TrimSpace(
		pi.StripePaymentIntentID,
	)
	if stripePaymentIntentID == "" {
		if stripeErr != nil {
			return nil, fmt.Errorf(
				"%w: %v",
				ErrPaymentFlowStripePaymentIntentIDEmpty,
				stripeErr,
			)
		}

		return nil, ErrPaymentFlowStripePaymentIntentIDEmpty
	}

	status := paymentdom.StatusPending
	requiresAction := pi.RequiresAction

	var errorType *string
	var errorCode *string
	var errorMessage *string
	var resultErr error

	if value := strings.TrimSpace(pi.ErrorType); value != "" {
		errorType = &value
	}

	if value := strings.TrimSpace(pi.ErrorCode); value != "" {
		errorCode = &value
	}

	if value := strings.TrimSpace(pi.ErrorMessage); value != "" {
		errorMessage = &value
	}

	// If Stripe returned both a PaymentIntent ID and an error, the
	// PaymentIntent exists. Record it as failed so that the attempt remains
	// traceable while preserving the non-empty PaymentIntent ID invariant.
	if stripeErr != nil {
		status = paymentdom.StatusFailed

		message := stripeErr.Error()
		errorMessage = &message

		resultErr = fmt.Errorf(
			"payment_flow: Stripe PaymentIntent failed: %w",
			stripeErr,
		)
	} else {
		stripeStatus := strings.ToLower(
			strings.TrimSpace(pi.Status),
		)

		switch stripeStatus {
		case "succeeded":
			status = paymentdom.StatusSucceeded
			requiresAction = false

		case "requires_action", "requires_source_action":
			status = paymentdom.StatusRequiresAction
			requiresAction = true

		case "processing":
			status = paymentdom.StatusProcessing

		case "requires_confirmation", "requires_payment_method":
			status = paymentdom.StatusPending

		case "canceled":
			status = paymentdom.StatusCanceled
			requiresAction = false

			if errorMessage == nil {
				message := "Stripe PaymentIntent was canceled"
				errorMessage = &message
			}

			resultErr = ErrPaymentFlowStripePaymentIntentCanceled

		default:
			status = paymentdom.StatusFailed
			requiresAction = false

			if errorMessage == nil {
				message := fmt.Sprintf(
					"Stripe PaymentIntent status is unsupported or failed: %s",
					stripeStatus,
				)
				errorMessage = &message
			}

			resultErr = ErrPaymentFlowStripePaymentIntentFailed
		}
	}

	createdAt := time.Now().UTC()
	if u.now != nil {
		createdAt = u.now().UTC()
	}

	payment, err := paymentdom.New(
		paymentID,
		paymentMethodID,
		stripeCustomerID,
		stripePaymentMethodID,
		stripePaymentIntentID,
		amount,
		status,
		errorType,
		errorCode,
		errorMessage,
		createdAt,
	)
	if err != nil {
		return nil, err
	}

	created, err := u.paymentUC.Create(
		ctx,
		payment,
	)
	if err != nil {
		// The Stripe PaymentIntent already exists at this point.
		// The deterministic idempotency key allows a retry to obtain the
		// same PaymentIntent instead of creating another one.
		return nil, fmt.Errorf(
			"payment_flow: create payment record after Stripe PaymentIntent %q: %w",
			stripePaymentIntentID,
			err,
		)
	}

	if created == nil {
		return nil, errors.New(
			"payment_flow: created payment is nil",
		)
	}

	result := &CreatePaymentAndStartResult{
		Payment:               *created,
		PaymentID:             created.PaymentID,
		Status:                created.Status,
		StripePaymentIntentID: created.StripePaymentIntentID,
		ClientSecret:          pi.ClientSecret,
		RequiresAction:        requiresAction,
		ErrorType:             created.ErrorType,
		ErrorCode:             created.ErrorCode,
		ErrorMessage:          created.ErrorMsg,
	}

	return result, resultErr
}

// calculatePaymentOrderAmount calculates the authoritative payment amount
// from the server-side Order snapshot.
func calculatePaymentOrderAmount(
	order orderdom.Order,
) (int, error) {
	if len(order.Items) == 0 {
		return 0, ErrPaymentFlowOrderAmountInvalid
	}

	maxInt := int(^uint(0) >> 1)
	total := 0

	for _, item := range order.Items {
		if item.Price < 0 || item.Qty <= 0 {
			return 0, ErrPaymentFlowOrderAmountInvalid
		}

		if item.Price > maxInt/item.Qty {
			return 0, ErrPaymentFlowOrderAmountInvalid
		}

		lineAmount := item.Price * item.Qty
		if total > maxInt-lineAmount {
			return 0, ErrPaymentFlowOrderAmountInvalid
		}

		total += lineAmount
	}

	if total <= 0 {
		return 0, ErrPaymentFlowOrderAmountInvalid
	}

	return total, nil
}
