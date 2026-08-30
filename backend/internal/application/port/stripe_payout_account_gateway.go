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

	AttachPayoutBankAccount(
		ctx context.Context,
		in AttachStripePayoutBankAccountInput,
	) (*StripePayoutBankAccountResult, error)

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

// AttachStripePayoutBankAccountInput contains the Stripe-generated bank account
// token that must be attached to the authenticated user's Connected Account.
//
// BankAccountToken is created directly in the browser by Stripe.js. Raw bank
// account numbers, routing numbers, bank codes, branch codes, and account holder
// names must never be supplied to this application port.
//
// StripeAccountID must be resolved from the authenticated AMOL user's persisted
// PayoutAccount and must never be accepted directly from the browser.
//
// IdempotencyKey is supplied by the application layer so retrying an attachment
// request does not create duplicate external accounts.
type AttachStripePayoutBankAccountInput struct {
	StripeAccountID  string
	BankAccountToken string
	IdempotencyKey   string
}

// CreateStripePayoutAccountSessionInput contains the Connected Account for
// which an Embedded Connect Account Session must be created.
//
// StripeAccountID must be resolved from the authenticated AMOL user's persisted
// PayoutAccount rather than accepted directly from the browser.
//
// Deprecated: the custom AMOL payout-account registration flow no longer uses
// Embedded Connect for bank account registration. Retained temporarily until
// the legacy onboarding flow is removed.
type CreateStripePayoutAccountSessionInput struct {
	StripeAccountID string
}

// StripePayoutAccountSessionResult contains the short-lived client secret used
// by Stripe Connect.js to render Embedded Components.
//
// ClientSecret must not be persisted by the application.
//
// Deprecated: retained temporarily until the legacy Embedded Connect onboarding
// flow is removed.
type StripePayoutAccountSessionResult struct {
	AccountID    string
	ClientSecret string
}

// StripePayoutBankAccountResult contains display-only bank information.
//
// Full account numbers, routing numbers, bank codes, branch codes, and account
// holder names must not be returned to or persisted by the application layer.
type StripePayoutBankAccountResult struct {
	BankName string
	Last4    string
}
