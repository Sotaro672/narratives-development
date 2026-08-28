// backend/internal/application/port/stripe_transfer_reversal_gateway.go
package port

import "context"

// StripeTransferReversalGateway reverses a completed Stripe Connect Transfer.
type StripeTransferReversalGateway interface {
	CreateTransferReversal(
		ctx context.Context,
		in CreateStripeTransferReversalInput,
	) (*CreateStripeTransferReversalResult, error)
}

type CreateStripeTransferReversalInput struct {
	StripeTransferID string

	Amount int

	IdempotencyKey string

	OrderID      string
	PaymentID    string
	SettlementID string
	CompanyID    string
	AccountID    string
}

type CreateStripeTransferReversalResult struct {
	StripeTransferReversalID string
}
