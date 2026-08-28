// backend/internal/application/port/payout_account_provider.go

package port

import (
	"context"

	payoutdom "narratives/internal/domain/payoutAccount"
)

// PayoutAccountProvider defines payout-account operations required by the
// application layer.
//
// application は特定の振込事業者や決済事業者のSDKへ依存せず、outbound
// adapter がこの port を実装します。
//
// AccountNumber is sensitive transient data. Implementations may use it when
// registering a payout destination, but must not return or persist the full
// account number through this port.
type PayoutAccountProvider interface {
	// Name returns the stable provider identifier persisted in PayoutAccount.
	//
	// Examples:
	//   - "mock"
	//   - future production provider identifier
	Name() string

	// Register registers or replaces a payout destination for the user.
	Register(
		ctx context.Context,
		in RegisterPayoutAccountInput,
	) (*RegisterPayoutAccountResult, error)

	// Get returns the latest provider-side availability state for an already
	// registered provider account.
	Get(
		ctx context.Context,
		providerAccountID string,
	) (*PayoutAccountProviderState, error)
}

// RegisterPayoutAccountInput contains the bank-account information required by
// a payout provider.
//
// AccountNumber must be treated as transient sensitive data:
//   - do not log it
//   - do not persist it in Firestore
//   - do not include it in returned errors
//   - do not retain it after the provider registration call completes
type RegisterPayoutAccountInput struct {
	UserID string

	BankCode   string
	BankName   string
	BranchCode string
	BranchName string

	AccountType       payoutdom.BankAccountType
	AccountNumber     string
	AccountHolderName string
}

// RegisterPayoutAccountResult contains only values safe and necessary for
// application/domain persistence.
//
// BankLast4 is the only account-number fragment allowed to leave the provider
// adapter after registration.
type RegisterPayoutAccountResult struct {
	ProviderAccountID string
	Status            payoutdom.Status
	PayoutReady       bool
	BankLast4         string
}

// PayoutAccountProviderState represents the latest provider-side state of an
// already registered payout destination.
type PayoutAccountProviderState struct {
	Status      payoutdom.Status
	PayoutReady bool
}
