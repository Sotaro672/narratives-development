// backend/internal/application/port/stripe_payment_intent_gateway.go
package port

import "context"

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
	TransferGroup         string

	PaymentMethodID string

	OffSession bool
}

type CreateAndConfirmPaymentIntentResult struct {
	StripePaymentIntentID string
	StripeChargeID        string
	Status                string
	ClientSecret          string
	RequiresAction        bool

	ErrorType    string
	ErrorCode    string
	ErrorMessage string
}
