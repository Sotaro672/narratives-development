// backend/internal/application/usecase/payoutAccount_usecase.go
package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	applicationport "narratives/internal/application/port"
	payoutdom "narratives/internal/domain/payoutAccount"
)

const (
	defaultPayoutAccountCountry     = "JP"
	defaultPayoutAccountDisplayName = "AMOL Seller"
	defaultPayoutAccountEntityType  = "individual"

	payoutAccountCreateIdempotencyKeyPrefix = "payout-account-create:"
)

var (
	ErrPayoutAccountRepositoryMissing = errors.New(
		"payoutAccount: repository is not configured",
	)
	ErrPayoutAccountStripeGatewayMissing = errors.New(
		"payoutAccount: stripe gateway is not configured",
	)
	ErrPayoutAccountStripeResultEmpty = errors.New(
		"payoutAccount: stripe account result is empty",
	)
	ErrPayoutAccountStripeAccountMismatch = errors.New(
		"payoutAccount: stripe account mismatch",
	)
	ErrPayoutAccountStripeSessionEmpty = errors.New(
		"payoutAccount: stripe account session is empty",
	)
)

// CreatePayoutAccountSessionInput is supplied by the authenticated Mall handler.
//
// UserID must be obtained from UserAuthMiddleware rather than request JSON.
// DisplayName and ContactEmail are authentication/profile values used only
// when the Stripe Connected Account is created for the first time.
type CreatePayoutAccountSessionInput struct {
	UserID       string
	DisplayName  string
	ContactEmail string
}

type CreatePayoutAccountSessionResult struct {
	Account      payoutdom.PayoutAccount
	ClientSecret string
}

// PayoutAccountUsecase manages the resale seller's Stripe payout destination.
//
// Policy:
//   - one User has at most one PayoutAccount
//   - PayoutAccount.UserID is also the Firestore document ID
//   - one Stripe Connected Account is created per User
//   - repeated Account Session creation reuses the existing Connected Account
//   - Stripe is the source of truth for onboarding and bank account state
type PayoutAccountUsecase struct {
	repo    payoutdom.Repository
	gateway applicationport.StripePayoutAccountGateway

	now func() time.Time
}

func NewPayoutAccountUsecase(
	repo payoutdom.Repository,
	gateway applicationport.StripePayoutAccountGateway,
) *PayoutAccountUsecase {
	return &PayoutAccountUsecase{
		repo:    repo,
		gateway: gateway,
		now:     time.Now,
	}
}

// GetByUserID returns the user's PayoutAccount after synchronizing its current
// Stripe state.
//
// payoutdom.ErrNotFound is returned when the user has not registered a payout
// account yet. The HTTP handler may translate that case to {"data": null}.
func (u *PayoutAccountUsecase) GetByUserID(
	ctx context.Context,
	userID string,
) (*payoutdom.PayoutAccount, error) {
	if err := u.validateReady(); err != nil {
		return nil, err
	}

	if userID == "" {
		return nil, payoutdom.ErrInvalidUserID
	}

	account, err := u.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	synced, err := u.syncStripeState(ctx, account)
	if err != nil {
		return nil, err
	}

	return &synced, nil
}

// CreateAccountSession creates or continues Stripe Embedded Onboarding.
//
// First request:
//   - create Stripe Connected Account with a stable idempotency key
//   - persist payoutAccounts/{userId}
//   - create Account Session
//
// Later requests:
//   - reuse the stored StripeAccountID
//   - synchronize current Stripe state
//   - create a fresh short-lived Account Session
//
// Account Session client secrets are intentionally not persisted because they
// are short-lived credentials used only by Stripe Connect.js.
func (u *PayoutAccountUsecase) CreateAccountSession(
	ctx context.Context,
	in CreatePayoutAccountSessionInput,
) (CreatePayoutAccountSessionResult, error) {
	if err := u.validateReady(); err != nil {
		return CreatePayoutAccountSessionResult{}, err
	}

	userID := in.UserID
	if userID == "" {
		return CreatePayoutAccountSessionResult{},
			payoutdom.ErrInvalidUserID
	}

	account, err := u.repo.GetByUserID(ctx, userID)
	switch {
	case err == nil:
		account, err = u.syncStripeState(ctx, account)
		if err != nil {
			return CreatePayoutAccountSessionResult{}, err
		}

	case errors.Is(err, payoutdom.ErrNotFound):
		account, err = u.createPayoutAccount(
			ctx,
			userID,
			in.DisplayName,
			in.ContactEmail,
		)
		if err != nil {
			return CreatePayoutAccountSessionResult{}, err
		}

	default:
		return CreatePayoutAccountSessionResult{}, err
	}

	session, err := u.gateway.CreatePayoutAccountSession(
		ctx,
		applicationport.CreateStripePayoutAccountSessionInput{
			StripeAccountID: account.StripeAccountID,
		},
	)
	if err != nil {
		return CreatePayoutAccountSessionResult{}, err
	}

	if session == nil || session.ClientSecret == "" {
		return CreatePayoutAccountSessionResult{},
			ErrPayoutAccountStripeSessionEmpty
	}

	if session.AccountID == "" ||
		session.AccountID != account.StripeAccountID {
		return CreatePayoutAccountSessionResult{},
			ErrPayoutAccountStripeAccountMismatch
	}

	return CreatePayoutAccountSessionResult{
		Account:      account,
		ClientSecret: session.ClientSecret,
	}, nil
}

// createPayoutAccount creates the Stripe Connected Account and persists its
// association with the User.
//
// The Stripe create request uses a stable per-user idempotency key. Therefore
// concurrent/retried requests do not intentionally create multiple Stripe
// Connected Accounts.
//
// Mall resale sellers are created as individual legal entities because the
// resale flow is intended for personal second-hand sales rather than businesses.
//
// If Firestore reports ErrConflict, another request already persisted the
// canonical PayoutAccount, so that record is loaded and used instead.
func (u *PayoutAccountUsecase) createPayoutAccount(
	ctx context.Context,
	userID string,
	displayName string,
	contactEmail string,
) (payoutdom.PayoutAccount, error) {
	if displayName == "" {
		displayName = contactEmail
	}
	if displayName == "" {
		displayName = defaultPayoutAccountDisplayName
	}

	stripeAccount, err := u.gateway.CreatePayoutAccount(
		ctx,
		applicationport.CreateStripePayoutAccountInput{
			UserID:         userID,
			DisplayName:    displayName,
			ContactEmail:   contactEmail,
			Country:        defaultPayoutAccountCountry,
			EntityType:     defaultPayoutAccountEntityType,
			IdempotencyKey: payoutAccountCreateIdempotencyKeyPrefix + userID,
		},
	)
	if err != nil {
		return payoutdom.PayoutAccount{}, err
	}

	if stripeAccount == nil {
		return payoutdom.PayoutAccount{},
			ErrPayoutAccountStripeResultEmpty
	}

	stripeAccountID := stripeAccount.ID
	if stripeAccountID == "" ||
		!strings.HasPrefix(stripeAccountID, "acct_") {
		return payoutdom.PayoutAccount{},
			payoutdom.ErrInvalidStripeAccountID
	}

	now := u.now().UTC()

	account, err := payoutdom.New(
		userID,
		stripeAccountID,
		stripeAccount.DetailsSubmitted,
		stripeAccount.PayoutsEnabled,
		"",
		"",
		now,
		now,
	)
	if err != nil {
		return payoutdom.PayoutAccount{}, err
	}

	created, err := u.repo.Create(ctx, account)
	if err == nil {
		return created, nil
	}

	if !errors.Is(err, payoutdom.ErrConflict) {
		return payoutdom.PayoutAccount{}, err
	}

	// A concurrent request already persisted payoutAccounts/{userId}.
	// The persisted document is the canonical association.
	existing, getErr := u.repo.GetByUserID(ctx, userID)
	if getErr != nil {
		return payoutdom.PayoutAccount{}, getErr
	}

	return existing, nil
}

// syncStripeState refreshes the persisted application snapshot from Stripe.
//
// Stripe remains the source of truth for:
//   - onboarding completion
//   - transfer availability
//   - display-only bank name
//   - bank account last4
//
// Firestore is updated only when one of those values changed.
func (u *PayoutAccountUsecase) syncStripeState(
	ctx context.Context,
	account payoutdom.PayoutAccount,
) (payoutdom.PayoutAccount, error) {
	stripeAccount, err := u.gateway.GetPayoutAccount(
		ctx,
		account.StripeAccountID,
	)
	if err != nil {
		return payoutdom.PayoutAccount{}, err
	}

	if stripeAccount == nil {
		return payoutdom.PayoutAccount{},
			ErrPayoutAccountStripeResultEmpty
	}

	stripeAccountID := stripeAccount.ID
	if stripeAccountID == "" ||
		stripeAccountID != account.StripeAccountID {
		return payoutdom.PayoutAccount{},
			ErrPayoutAccountStripeAccountMismatch
	}

	bankName := ""
	bankLast4 := ""

	bankAccount, err := u.gateway.GetPayoutBankAccount(
		ctx,
		account.StripeAccountID,
	)
	if err != nil {
		return payoutdom.PayoutAccount{}, err
	}

	if bankAccount != nil {
		bankName = bankAccount.BankName
		bankLast4 = bankAccount.Last4
	}

	if account.DetailsSubmitted == stripeAccount.DetailsSubmitted &&
		account.PayoutsEnabled == stripeAccount.PayoutsEnabled &&
		account.BankName == bankName &&
		account.BankLast4 == bankLast4 {
		return account, nil
	}

	updated := account
	if err := updated.ApplyStripeState(
		stripeAccount.DetailsSubmitted,
		stripeAccount.PayoutsEnabled,
		bankName,
		bankLast4,
		u.now().UTC(),
	); err != nil {
		return payoutdom.PayoutAccount{}, err
	}

	return u.repo.Update(ctx, updated)
}

func (u *PayoutAccountUsecase) validateReady() error {
	if u == nil {
		return ErrPayoutAccountRepositoryMissing
	}
	if u.repo == nil {
		return ErrPayoutAccountRepositoryMissing
	}
	if u.gateway == nil {
		return ErrPayoutAccountStripeGatewayMissing
	}

	return nil
}
