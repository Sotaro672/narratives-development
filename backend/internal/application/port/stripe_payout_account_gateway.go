// backend/internal/application/port/stripe_payout_account_gateway.go
package port

import (
	"context"
	"time"
)

const (
	StripePayoutAccountLinkUseCaseOnboarding StripePayoutAccountLinkUseCase = "account_onboarding"
	StripePayoutAccountLinkUseCaseUpdate     StripePayoutAccountLinkUseCase = "account_update"
)

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

	CreatePayoutAccountLink(
		ctx context.Context,
		in CreateStripePayoutAccountLinkInput,
	) (*StripePayoutAccountLinkResult, error)

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
type CreateStripePayoutAccountInput struct {
	UserID         string
	DisplayName    string
	ContactEmail   string
	Country        string
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

// StripePayoutBankAccountResult contains display-only bank information.
//
// Full account numbers, routing numbers, and branch numbers must not be
// returned to or persisted by the application layer.
type StripePayoutBankAccountResult struct {
	BankName string
	Last4    string
}

type StripePayoutAccountLinkUseCase string

type CreateStripePayoutAccountLinkInput struct {
	StripeAccountID string
	UseCase         StripePayoutAccountLinkUseCase
	ReturnURL       string
	RefreshURL      string
}

type StripePayoutAccountLinkResult struct {
	AccountID string
	URL       string
	ExpiresAt time.Time
}
