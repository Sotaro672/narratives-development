// backend/internal/application/port/stripe_refund_gateway.go
package port

import (
	"context"
	"time"

	paymentdom "narratives/internal/domain/payment"
)

// StripeRefundGateway executes a Stripe Charge refund.
//
// AMOL uses Separate Charges and Transfers. Refunding the platform Charge does
// not automatically reverse seller Transfers, so transferred Settlements must
// also be processed through the transfer reversal port.
type StripeRefundGateway interface {
	CreateRefund(
		ctx context.Context,
		in CreateStripeRefundInput,
	) (*CreateStripeRefundResult, error)
}

type CreateStripeRefundInput struct {
	StripeChargeID string
	Amount         int
	IdempotencyKey string
	PaymentID      string
	RefundID       string
}

type CreateStripeRefundResult struct {
	StripeRefundID string
	Status         paymentdom.RefundStatus
	CreatedAt      time.Time
}
