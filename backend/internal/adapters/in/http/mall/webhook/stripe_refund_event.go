// backend/internal/adapters/in/http/mall/webhook/stripe_refund_event.go
package mallHandler

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	paymentdom "narratives/internal/domain/payment"
)

type stripeRefundEventInput struct {
	PaymentID string
	RefundID  string

	StripeRefundID        string
	StripeChargeID        string
	StripePaymentIntentID string

	Status paymentdom.RefundStatus
	Amount int

	OccurredAt time.Time
}

func extractStripeRefundEventInput(event stripeEvent) (input stripeRefundEventInput, supported bool, err error) {
	eventType := strings.TrimSpace(event.Type)

	switch eventType {
	case "refund.created", "refund.failed", "refund.updated":
	default:
		return stripeRefundEventInput{}, false, nil
	}

	if strings.TrimSpace(event.ID) == "" {
		return stripeRefundEventInput{}, false, errors.New("Stripe event id is empty")
	}

	var refund stripeRefund
	if err := json.Unmarshal(event.Data.Object, &refund); err != nil {
		return stripeRefundEventInput{}, false, fmt.Errorf("decode Stripe Refund: %w", err)
	}
	if strings.TrimSpace(refund.Object) != "refund" {
		return stripeRefundEventInput{}, false, errors.New("Stripe refund object type is invalid")
	}

	stripeRefundID := strings.TrimSpace(refund.ID)
	if stripeRefundID == "" || !strings.HasPrefix(stripeRefundID, "re_") {
		return stripeRefundEventInput{}, false, errors.New("Stripe refund id is invalid")
	}

	paymentID := firstNonEmpty(refund.Metadata["paymentId"], refund.Metadata["orderId"])
	refundID := strings.TrimSpace(refund.Metadata["refundId"])
	if strings.Contains(refundID, "/") {
		return stripeRefundEventInput{}, false, errors.New("Stripe refund metadata refundId is invalid")
	}
	if paymentID == "" {
		return stripeRefundEventInput{}, false, nil
	}

	stripeChargeID := strings.TrimSpace(refund.Charge)
	if stripeChargeID == "" || !strings.HasPrefix(stripeChargeID, "ch_") {
		return stripeRefundEventInput{}, false, errors.New("Stripe refund charge id is invalid")
	}

	stripePaymentIntentID := strings.TrimSpace(refund.PaymentIntent)
	if stripePaymentIntentID == "" || !strings.HasPrefix(stripePaymentIntentID, "pi_") {
		return stripeRefundEventInput{}, false, errors.New("Stripe refund payment intent id is invalid")
	}
	if refund.Amount <= 0 {
		return stripeRefundEventInput{}, false, errors.New("Stripe refund amount is invalid")
	}

	refundStatus, ok := refundStatusFromStripe(refund.Status)
	if !ok {
		return stripeRefundEventInput{}, false, errors.New("Stripe refund status is invalid")
	}
	if eventType == "refund.failed" && refundStatus != paymentdom.RefundStatusFailed {
		return stripeRefundEventInput{}, false, errors.New("Stripe refund.failed status mismatch")
	}

	occurredUnix := event.Created
	if occurredUnix <= 0 {
		occurredUnix = refund.Created
	}
	if occurredUnix <= 0 {
		return stripeRefundEventInput{}, false, errors.New("Stripe refund event created timestamp is invalid")
	}

	return stripeRefundEventInput{
		PaymentID:             paymentID,
		RefundID:              refundID,
		StripeRefundID:        stripeRefundID,
		StripeChargeID:        stripeChargeID,
		StripePaymentIntentID: stripePaymentIntentID,
		Status:                refundStatus,
		Amount:                refund.Amount,
		OccurredAt:            time.Unix(occurredUnix, 0).UTC(),
	}, true, nil
}

func refundStatusFromStripe(status string) (paymentdom.RefundStatus, bool) {
	switch strings.TrimSpace(status) {
	case "pending":
		return paymentdom.RefundStatusPending, true
	case "requires_action":
		return paymentdom.RefundStatusRequiresAction, true
	case "succeeded":
		return paymentdom.RefundStatusSucceeded, true
	case "failed":
		return paymentdom.RefundStatusFailed, true
	case "canceled":
		return paymentdom.RefundStatusCanceled, true
	default:
		return "", false
	}
}
