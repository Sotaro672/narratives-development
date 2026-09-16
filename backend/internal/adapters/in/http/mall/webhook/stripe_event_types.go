// backend/internal/adapters/in/http/mall/webhook/stripe_event_types.go
package mallHandler

import (
	"encoding/json"
	"strings"
)

type stripeEvent struct {
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Created int64           `json:"created"`
	Data    stripeEventData `json:"data"`
}

type stripeEventData struct {
	Object json.RawMessage `json:"object"`
}

type stripeRefund struct {
	ID            string            `json:"id"`
	Object        string            `json:"object"`
	Amount        int               `json:"amount"`
	Charge        string            `json:"charge"`
	Created       int64             `json:"created"`
	Status        string            `json:"status"`
	PaymentIntent string            `json:"payment_intent"`
	Metadata      map[string]string `json:"metadata"`
}

type stripePaymentIntent struct {
	ID                 string              `json:"id"`
	Status             string              `json:"status"`
	Created            int64               `json:"created"`
	Metadata           map[string]string   `json:"metadata"`
	LatestCharge       string              `json:"latest_charge"`
	LastPaymentError   *stripePaymentError `json:"last_payment_error"`
	CancellationReason string              `json:"cancellation_reason"`
}

type stripePaymentError struct {
	Type    string `json:"type"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}

	return ""
}
