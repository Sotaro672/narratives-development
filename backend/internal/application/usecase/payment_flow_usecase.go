// backend/internal/application/usecase/payment_flow_usecase.go
package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	applicationport "narratives/internal/application/port"
	common "narratives/internal/domain/common"
	orderdom "narratives/internal/domain/order"
	paymentdom "narratives/internal/domain/payment"
)

// OrderReaderForPaymentFlow provides the server-side order source of truth
// required before starting a payment.
//
// Contract:
//   - When the requested order does not exist, GetByID must return an error
//     that is or wraps ErrPaymentFlowOrderNotFound.
//   - Other repository/infrastructure errors must be returned without being
//     converted to ErrPaymentFlowOrderNotFound.
type OrderReaderForPaymentFlow interface {
	GetByID(
		ctx context.Context,
		id string,
	) (orderdom.Order, error)
}

// OrderWriterForPaymentFlow is required when a successful payment must be
// synchronously reflected to Order.Paid before dispatch can continue.
type OrderWriterForPaymentFlow interface {
	Update(
		ctx context.Context,
		order orderdom.Order,
		opts *common.SaveOptions,
	) (orderdom.Order, error)
}

// PaymentFlowUsecase orchestrates:
//
//  1. Verify the authoritative Order.
//  2. Verify unpaid state and amount using the server-side Order.
//  3. Verify the payment method against the Order snapshot.
//  4. Create and confirm the Stripe PaymentIntent with a transfer group.
//  5. Verify that Stripe returned a non-empty PaymentIntent ID.
//  6. Create the payment record with the PaymentIntent, Charge, and transfer group.
//  7. Let PaymentUsecase run post-paid processing when status is succeeded.
//
// Dispatch payment additionally:
//   - resolves all payment information from the Order snapshot
//   - uses an off-session Stripe payment
//   - reuses an existing succeeded payment without charging again
//   - requires Order.Paid=true before dispatch may continue
//
// Responsibility separation:
//   - /mall/me/orders   : OrderHandler   -> OrderUsecase
//   - /mall/me/payments : PaymentHandler -> PaymentFlowUsecase
type PaymentFlowUsecase struct {
	paymentUC *PaymentUsecase

	orderReader          OrderReaderForPaymentFlow
	paymentIntentGateway applicationport.StripePaymentIntentGateway

	now func() time.Time
}

// NewPaymentFlowUsecase creates a PaymentFlowUsecase for a Stripe
// PaymentIntent-based payment flow.
func NewPaymentFlowUsecase(
	paymentUC *PaymentUsecase,
	orderReader OrderReaderForPaymentFlow,
	paymentIntentGateway applicationport.StripePaymentIntentGateway,
) *PaymentFlowUsecase {
	return &PaymentFlowUsecase{
		paymentUC:            paymentUC,
		orderReader:          orderReader,
		paymentIntentGateway: paymentIntentGateway,
		now:                  time.Now,
	}
}

var (
	ErrPaymentFlowPaymentUsecaseMissing = errors.New(
		"payment_flow: payment usecase is not configured",
	)
	ErrPaymentFlowOrderReaderMissing = errors.New(
		"payment_flow: order reader is not configured",
	)
	ErrPaymentFlowOrderWriterMissing = errors.New(
		"payment_flow: order writer is not configured",
	)

	// ErrPaymentFlowOrderNotFound is the application-layer identity used when
	// the authoritative Order does not exist.
	//
	// OrderReaderForPaymentFlow implementations must return this error, or an
	// error wrapping it, only for a genuine not-found result.
	ErrPaymentFlowOrderNotFound = errors.New(
		"payment_flow: order not found",
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
	ErrPaymentFlowPaymentMethodMismatch = errors.New(
		"payment_flow: payment method does not match order snapshot",
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
		"payment_flow: stripe payment intent ID is empty",
	)
	ErrPaymentFlowStripePaymentIntentFailed = errors.New(
		"payment_flow: stripe payment intent failed",
	)
	ErrPaymentFlowStripePaymentIntentCanceled = errors.New(
		"payment_flow: stripe payment intent canceled",
	)

	ErrPaymentFlowDispatchRequiresAction = errors.New(
		"payment_flow: dispatch payment requires customer action",
	)
	ErrPaymentFlowDispatchProcessing = errors.New(
		"payment_flow: dispatch payment is processing",
	)
	ErrPaymentFlowDispatchPending = errors.New(
		"payment_flow: dispatch payment is pending",
	)
	ErrPaymentFlowDispatchNotSucceeded = errors.New(
		"payment_flow: dispatch payment did not succeed",
	)
	ErrPaymentFlowDispatchPaymentMismatch = errors.New(
		"payment_flow: existing payment does not match order",
	)
	ErrPaymentFlowDispatchPaidStateInvalid = errors.New(
		"payment_flow: order is paid but succeeded payment is missing",
	)
	ErrPaymentFlowDispatchOrderPaidUpdateFailed = errors.New(
		"payment_flow: failed to persist paid order state",
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
//
// OffSession must only be set by a trusted server-side flow such as dispatch.
// Normal Mall payment requests must leave it false.
type CreatePaymentAndStartInput struct {
	UserID string

	PaymentID string

	PaymentMethodID string

	StripeCustomerID      string
	StripePaymentMethodID string

	Amount *int

	OffSession bool
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
	StripeChargeID        string
	TransferGroup         string
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
//  4. Verify the requested payment method against the Order snapshot.
//  5. Create and confirm the Stripe PaymentIntent with a transfer group.
//  6. Require a non-empty Stripe PaymentIntent ID.
//  7. Create the payment record with the Charge ID and transfer group.
//  8. Return ClientSecret when additional authentication is required.
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

	userID := in.UserID
	paymentID := in.PaymentID
	paymentMethodID := in.PaymentMethodID
	stripeCustomerID := in.StripeCustomerID
	stripePaymentMethodID :=
		in.StripePaymentMethodID

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
		// GetByIDが返したsentinelを%wで保持する。
		// ErrPaymentFlowOrderNotFoundならHandler側のerrors.Isで404となる。
		// その他のエラーは同じerror chainのまま500となる。
		return nil, fmt.Errorf(
			"payment_flow: get order %q: %w",
			paymentID,
			err,
		)
	}

	if order.ID != paymentID {
		return nil, ErrPaymentFlowOrderIDMismatch
	}

	if order.UserID != userID {
		return nil, ErrPaymentFlowOrderOwnerMismatch
	}

	if order.Paid {
		return nil, ErrPaymentFlowOrderAlreadyPaid
	}

	orderPaymentMethodID :=
		order.PaymentMethodSnapshot.PaymentMethodID

	orderStripeCustomerID :=
		order.PaymentMethodSnapshot.CustomerID

	orderStripePaymentMethodID :=
		order.PaymentMethodSnapshot.StripePaymentMethodID

	if orderPaymentMethodID == "" ||
		orderStripeCustomerID == "" ||
		orderStripePaymentMethodID == "" {
		return nil, ErrPaymentFlowPaymentMethodMismatch
	}

	if paymentMethodID != orderPaymentMethodID ||
		stripeCustomerID != orderStripeCustomerID ||
		stripePaymentMethodID != orderStripePaymentMethodID {
		return nil, ErrPaymentFlowPaymentMethodMismatch
	}

	orderAmount, err :=
		orderdom.CalculatePaymentAmount(
			order,
		)
	if err != nil {
		return nil, fmt.Errorf(
			"%w: %v",
			ErrPaymentFlowOrderAmountInvalid,
			err,
		)
	}

	if requestedAmount != orderAmount {
		return nil, ErrPaymentFlowOrderAmountMismatch
	}

	// Stripe and the payment document use the amount calculated from the
	// server-side Order, never the unverified client value.
	amount := orderAmount

	idempotencyKeyPrefix := "payment"
	if in.OffSession {
		idempotencyKeyPrefix =
			"dispatch-payment"
	}

	idempotencyKey := fmt.Sprintf(
		"%s:%s:%s:%d",
		idempotencyKeyPrefix,
		paymentID,
		paymentMethodID,
		amount,
	)

	transferGroup := fmt.Sprintf(
		"order:%s",
		paymentID,
	)

	pi, stripeErr :=
		u.paymentIntentGateway.CreateAndConfirmPaymentIntent(
			ctx,
			applicationport.CreateAndConfirmPaymentIntentInput{
				StripeCustomerID:      stripeCustomerID,
				StripePaymentMethodID: stripePaymentMethodID,
				Amount:                amount,
				Currency:              "jpy",
				IdempotencyKey:        idempotencyKey,
				Description: fmt.Sprintf(
					"AMOL payment paymentId=%s",
					paymentID,
				),
				TransferGroup:   transferGroup,
				PaymentMethodID: paymentMethodID,
				OffSession:      in.OffSession,
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

	stripePaymentIntentID :=
		pi.StripePaymentIntentID

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

	stripeChargeID := pi.StripeChargeID

	status := paymentdom.StatusPending
	requiresAction := pi.RequiresAction

	var errorType *string
	var errorCode *string
	var errorMessage *string
	var resultErr error

	if value := pi.ErrorType; value != "" {
		errorType = &value
	}

	if value := pi.ErrorCode; value != "" {
		errorCode = &value
	}

	if value := pi.ErrorMessage; value != "" {
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
		stripeStatus :=
			strings.ToLower(
				pi.Status,
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
		stripeChargeID,
		transferGroup,
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
		StripeChargeID:        created.StripeChargeID,
		TransferGroup:         created.TransferGroup,
		ClientSecret:          pi.ClientSecret,
		RequiresAction:        requiresAction,
		ErrorType:             created.ErrorType,
		ErrorCode:             created.ErrorCode,
		ErrorMessage:          created.ErrorMsg,
	}

	return result, resultErr
}

// EnsureOrderPaidForDispatch ensures that the Order has a succeeded Payment
// before shipment state is changed.
//
// All payment information is resolved from the persisted Order snapshot.
// Console callers must provide only orderID.
//
// The operation is idempotent:
//   - if a succeeded Payment already exists, Stripe is not called again
//   - if Order.Paid was not persisted after a succeeded Payment, it is repaired
//   - a non-succeeded existing Payment blocks another charge
func (u *PaymentFlowUsecase) EnsureOrderPaidForDispatch(
	ctx context.Context,
	orderID string,
) error {
	if u == nil || u.paymentUC == nil {
		return ErrPaymentFlowPaymentUsecaseMissing
	}

	if u.orderReader == nil {
		return ErrPaymentFlowOrderReaderMissing
	}

	if u.paymentIntentGateway == nil {
		return ErrPaymentFlowStripeGatewayMissing
	}

	if orderID == "" {
		return ErrPaymentFlowPaymentIDEmpty
	}

	order, err :=
		u.orderReader.GetByID(
			ctx,
			orderID,
		)
	if err != nil {
		return fmt.Errorf(
			"payment_flow: get order %q for dispatch: %w",
			orderID,
			err,
		)
	}

	if order.ID != orderID {
		return ErrPaymentFlowOrderIDMismatch
	}

	orderAmount, err :=
		orderdom.CalculatePaymentAmount(
			order,
		)
	if err != nil {
		return fmt.Errorf(
			"%w: %v",
			ErrPaymentFlowOrderAmountInvalid,
			err,
		)
	}

	paymentMethodID :=
		order.PaymentMethodSnapshot.PaymentMethodID

	stripeCustomerID :=
		order.PaymentMethodSnapshot.CustomerID

	stripePaymentMethodID :=
		order.PaymentMethodSnapshot.StripePaymentMethodID

	if paymentMethodID == "" {
		return ErrPaymentFlowPaymentMethodEmpty
	}

	if stripeCustomerID == "" {
		return ErrPaymentFlowStripeCustomerIDEmpty
	}

	if stripePaymentMethodID == "" {
		return ErrPaymentFlowStripePaymentMethodIDEmpty
	}

	existingPayment, err :=
		u.paymentUC.GetByPaymentID(
			ctx,
			orderID,
		)

	if err == nil && existingPayment != nil {
		if err :=
			validateDispatchPaymentMatchesOrder(
				existingPayment,
				order,
				orderAmount,
			); err != nil {
			return err
		}

		if existingPayment.Status !=
			paymentdom.StatusSucceeded {
			return dispatchPaymentStatusError(
				existingPayment.Status,
			)
		}

		return u.ensureOrderPaidState(
			ctx,
			orderID,
		)
	}

	if err != nil &&
		!errors.Is(
			err,
			paymentdom.ErrNotFound,
		) {
		return fmt.Errorf(
			"payment_flow: get payment %q for dispatch: %w",
			orderID,
			err,
		)
	}

	if order.Paid {
		return ErrPaymentFlowDispatchPaidStateInvalid
	}

	result, paymentErr :=
		u.CreatePaymentAndStartWithResult(
			ctx,
			CreatePaymentAndStartInput{
				UserID: order.UserID,

				PaymentID: orderID,

				PaymentMethodID: paymentMethodID,

				StripeCustomerID:      stripeCustomerID,
				StripePaymentMethodID: stripePaymentMethodID,

				Amount: &orderAmount,

				OffSession: true,
			},
		)

	if result == nil {
		if paymentErr != nil {
			return paymentErr
		}

		return ErrPaymentFlowDispatchNotSucceeded
	}

	if result.Status !=
		paymentdom.StatusSucceeded {
		statusErr :=
			dispatchPaymentStatusError(
				result.Status,
			)

		if paymentErr != nil {
			return fmt.Errorf(
				"%w: %v",
				statusErr,
				paymentErr,
			)
		}

		return statusErr
	}

	if paymentErr != nil {
		return paymentErr
	}

	return u.ensureOrderPaidState(
		ctx,
		orderID,
	)
}

func validateDispatchPaymentMatchesOrder(
	payment *paymentdom.Payment,
	order orderdom.Order,
	orderAmount int,
) error {
	if payment == nil {
		return ErrPaymentFlowDispatchPaymentMismatch
	}

	if payment.PaymentID !=
		order.ID {
		return ErrPaymentFlowDispatchPaymentMismatch
	}

	if payment.Amount != orderAmount {
		return ErrPaymentFlowDispatchPaymentMismatch
	}

	if payment.PaymentMethodID !=
		order.PaymentMethodSnapshot.PaymentMethodID {
		return ErrPaymentFlowDispatchPaymentMismatch
	}

	if payment.StripeCustomerID !=
		order.PaymentMethodSnapshot.CustomerID {
		return ErrPaymentFlowDispatchPaymentMismatch
	}

	if payment.StripePaymentMethodID !=
		order.PaymentMethodSnapshot.StripePaymentMethodID {
		return ErrPaymentFlowDispatchPaymentMismatch
	}

	transferGroup := fmt.Sprintf(
		"order:%s",
		order.ID,
	)

	if payment.TransferGroup != transferGroup {
		return ErrPaymentFlowDispatchPaymentMismatch
	}

	return nil
}

func dispatchPaymentStatusError(
	status paymentdom.PaymentStatus,
) error {
	switch status {
	case paymentdom.StatusRequiresAction:
		return ErrPaymentFlowDispatchRequiresAction

	case paymentdom.StatusProcessing:
		return ErrPaymentFlowDispatchProcessing

	case paymentdom.StatusPending:
		return ErrPaymentFlowDispatchPending

	case paymentdom.StatusFailed:
		return ErrPaymentFlowStripePaymentIntentFailed

	case paymentdom.StatusCanceled:
		return ErrPaymentFlowStripePaymentIntentCanceled

	case paymentdom.StatusSucceeded:
		return nil

	default:
		return ErrPaymentFlowDispatchNotSucceeded
	}
}

func (u *PaymentFlowUsecase) ensureOrderPaidState(
	ctx context.Context,
	orderID string,
) error {
	order, err :=
		u.orderReader.GetByID(
			ctx,
			orderID,
		)
	if err != nil {
		return fmt.Errorf(
			"payment_flow: reload order %q after payment: %w",
			orderID,
			err,
		)
	}

	if order.Paid {
		return nil
	}

	orderWriter, ok :=
		u.orderReader.(OrderWriterForPaymentFlow)
	if !ok || orderWriter == nil {
		return ErrPaymentFlowOrderWriterMissing
	}

	order.UpdatePaid(true)

	updated, err :=
		orderWriter.Update(
			ctx,
			order,
			nil,
		)
	if err != nil {
		return fmt.Errorf(
			"%w: %v",
			ErrPaymentFlowDispatchOrderPaidUpdateFailed,
			err,
		)
	}

	if !updated.Paid {
		return ErrPaymentFlowDispatchOrderPaidUpdateFailed
	}

	return nil
}
