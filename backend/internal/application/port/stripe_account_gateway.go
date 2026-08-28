// backend/internal/application/port/stripe_account_gateway.go
package port

import (
	"context"
	"time"
)

// StripeAccountGateway defines the Stripe Connect operations
// required by the application layer.
//
// application は Stripe adapter の具体型へ依存せず、
// outbound adapter がこの port を実装します。
type StripeAccountGateway interface {
	CreateAccount(
		ctx context.Context,
		in CreateStripeAccountInput,
	) (*StripeAccountResult, error)

	GetAccount(
		ctx context.Context,
		stripeAccountID string,
	) (*StripeAccountResult, error)

	CreateOnboardingLink(
		ctx context.Context,
		in CreateStripeAccountLinkInput,
	) (*StripeAccountLinkResult, error)
}

// CreateStripeAccountInput represents information required
// to create a Stripe Connected Account.
type CreateStripeAccountInput struct {
	AccountID      string
	CompanyID      string
	DisplayName    string
	ContactEmail   string
	Country        string
	IdempotencyKey string
}

// StripeAccountResult represents the Stripe Connected Account state
// required by the application layer.
type StripeAccountResult struct {
	ID                      string
	DisplayName             string
	ContactEmail            string
	Country                 string
	Dashboard               string
	Livemode                bool
	Closed                  bool
	RecipientTransferStatus string
	CreatedAt               time.Time
}

// CreateStripeAccountLinkInput represents information required
// to create a Stripe hosted onboarding link.
//
// Account Link は single-use であり、都度新しく発行するため
// stable な Idempotency-Key は保持しません。
type CreateStripeAccountLinkInput struct {
	StripeAccountID string
	ReturnURL       string
	RefreshURL      string
}

// StripeAccountLinkResult represents a Stripe hosted onboarding link.
type StripeAccountLinkResult struct {
	AccountID string
	URL       string
	ExpiresAt time.Time
}
