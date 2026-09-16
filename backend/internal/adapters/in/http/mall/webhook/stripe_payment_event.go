// backend/internal/adapters/in/http/mall/webhook/stripe_payment_event.go
package mallHandler

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	usecase "narratives/internal/application/usecase"
	paymentdom "narratives/internal/domain/payment"
)

func extractStripePaymentEventInput(event stripeEvent) (input usecase.ApplyStripePaymentEventInput, supported bool, err error) {
	eventID := strings.TrimSpace(event.ID)
	if eventID == "" {
		return usecase.ApplyStripePaymentEventInput{}, false, errors.New("Stripe event id is empty")
	}

	eventType := strings.TrimSpace(event.Type)
	paymentStatus, supported := paymentStatusFromStripeEventType(eventType)
	if !supported {
		return usecase.ApplyStripePaymentEventInput{}, false, nil
	}

	var paymentIntent stripePaymentIntent
	if err := json.Unmarshal(event.Data.Object, &paymentIntent); err != nil {
		return usecase.ApplyStripePaymentEventInput{}, false, fmt.Errorf("decode Stripe PaymentIntent: %w", err)
	}

	paymentID := firstNonEmpty(paymentIntent.Metadata["paymentId"], paymentIntent.Metadata["orderId"])
	stripePaymentIntentID := strings.TrimSpace(paymentIntent.ID)
	if paymentID == "" || stripePaymentIntentID == "" {
		return usecase.ApplyStripePaymentEventInput{}, false, nil
	}

	occurredUnix := event.Created
	if occurredUnix <= 0 {
		occurredUnix = paymentIntent.Created
	}
	if occurredUnix <= 0 {
		return usecase.ApplyStripePaymentEventInput{}, false, errors.New("Stripe event created timestamp is invalid")
	}

	errorType, errorCode, errorMessage := stripePaymentErrorFields(eventType, paymentIntent)

	return usecase.ApplyStripePaymentEventInput{
		EventID:               eventID,
		PaymentID:             paymentID,
		StripePaymentIntentID: stripePaymentIntentID,
		StripeChargeID:        strings.TrimSpace(paymentIntent.LatestCharge),
		Status:                paymentStatus,
		ErrorType:             errorType,
		ErrorCode:             errorCode,
		ErrorMsg:              errorMessage,
		OccurredAt:            time.Unix(occurredUnix, 0).UTC(),
	}, true, nil
}

func paymentStatusFromStripeEventType(eventType string) (paymentdom.PaymentStatus, bool) {
	switch eventType {
	case "payment_intent.succeeded":
		return paymentdom.StatusSucceeded, true
	case "payment_intent.requires_action":
		return paymentdom.StatusRequiresAction, true
	case "payment_intent.processing":
		return paymentdom.StatusProcessing, true
	case "payment_intent.payment_failed":
		return paymentdom.StatusFailed, true
	case "payment_intent.canceled":
		return paymentdom.StatusCanceled, true
	default:
		return "", false
	}
}

func stripePaymentErrorFields(
	eventType string,
	paymentIntent stripePaymentIntent,
) (errorType *string, errorCode *string, errorMessage *string) {
	switch eventType {
	case "payment_intent.payment_failed":
		if paymentIntent.LastPaymentError != nil {
			errorType = optionalNonEmptyString(paymentIntent.LastPaymentError.Type)
			errorCode = optionalNonEmptyString(paymentIntent.LastPaymentError.Code)
			errorMessage = optionalNonEmptyString(paymentIntent.LastPaymentError.Message)
		}

		if errorMessage == nil {
			value := "Stripe PaymentIntent payment failed"
			errorMessage = &value
		}

	case "payment_intent.canceled":
		value := "canceled"
		errorType = &value

		reason := strings.TrimSpace(paymentIntent.CancellationReason)
		if reason != "" {
			errorCode = &reason
			message := fmt.Sprintf("Stripe PaymentIntent was canceled: %s", reason)
			errorMessage = &message
		} else {
			message := "Stripe PaymentIntent was canceled"
			errorMessage = &message
		}
	}

	return errorType, errorCode, errorMessage
}

func optionalNonEmptyString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	return &value
}
