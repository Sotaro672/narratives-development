// backend/internal/application/port/stripe_payout_account_gateway.go
package port

import "context"

// StripePayoutAccountGateway defines the Stripe Connect operations required by
// the PayoutAccount application flow.
//
// The application layer does not depend on a concrete Stripe adapter.
type StripePayoutAccountGateway interface {
	CreatePayoutAccount(
		ctx context.Context,
		in CreateStripePayoutAccountInput,
	) (*StripePayoutAccountResult, error)

	GetPayoutAccount(
		ctx context.Context,
		stripeAccountID string,
	) (*StripePayoutAccountResult, error)

	CreatePayoutAccountSession(
		ctx context.Context,
		in CreateStripePayoutAccountSessionInput,
	) (*StripePayoutAccountSessionResult, error)

	GetPayoutBankAccount(
		ctx context.Context,
		stripeAccountID string,
	) (*StripePayoutBankAccountResult, error)
}

// CreateStripePayoutAccountInput contains the values required to create the
// Connected Account used as the resale seller's payout destination.
//
// UserID is stored in Stripe metadata so the Connected Account can be traced
// back to its AMOL owner without using AvatarID as the KYC identity.
//
// EntityType identifies the legal entity represented by the Connected Account.
// Mall resale sellers are non-business individuals, so the application layer
// supplies "individual".
type CreateStripePayoutAccountInput struct {
	UserID         string
	DisplayName    string
	ContactEmail   string
	Country        string
	EntityType     string
	IdempotencyKey string
}

// StripePayoutAccountResult is the Stripe account state required by AMOL.
//
// DetailsSubmitted is true when the currently required onboarding information
// has been submitted.
//
// PayoutsEnabled is true only when the account can receive Stripe transfers.
type StripePayoutAccountResult struct {
	ID               string
	DetailsSubmitted bool
	PayoutsEnabled   bool
}

// CreateStripePayoutAccountSessionInput contains the Connected Account for
// which an Embedded Connect Account Session must be created.
//
// StripeAccountID must be resolved from the authenticated AMOL user's persisted
// PayoutAccount rather than accepted directly from the browser.
type CreateStripePayoutAccountSessionInput struct {
	StripeAccountID string
}

// StripePayoutAccountSessionResult contains the short-lived client secret used
// by Stripe Connect.js to render Embedded Components.
//
// ClientSecret must not be persisted by the application.
type StripePayoutAccountSessionResult struct {
	AccountID    string
	ClientSecret string
}

// StripePayoutBankAccountResult contains display-only bank information.
//
// Full account numbers, routing numbers, and branch numbers must not be
// returned to or persisted by the application layer.
type StripePayoutBankAccountResult struct {
	BankName string
	Last4    string
}
