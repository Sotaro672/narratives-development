// backend/internal/application/usecase/payoutAccount_usecase.go

package usecase

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	payoutdom "narratives/internal/domain/payoutAccount"
)

const (
	defaultPayoutAccountCountry     = "JP"
	defaultPayoutAccountDisplayName = "AMOL Seller"

	payoutAccountCreateIdempotencyKeyPrefix = "payout-account-create:"

	StripePayoutAccountLinkUseCaseOnboarding StripePayoutAccountLinkUseCase = "account_onboarding"
	StripePayoutAccountLinkUseCaseUpdate     StripePayoutAccountLinkUseCase = "account_update"
)

var (
	ErrPayoutAccountRepositoryMissing = errors.New(
		"payoutAccount: repository is not configured",
	)
	ErrPayoutAccountStripeGatewayMissing = errors.New(
		"payoutAccount: stripe gateway is not configured",
	)
	ErrPayoutAccountAllowedReturnOriginMissing = errors.New(
		"payoutAccount: allowed return origin is not configured",
	)
	ErrPayoutAccountInvalidReturnURL = errors.New(
		"payoutAccount: invalid returnUrl",
	)
	ErrPayoutAccountInvalidRefreshURL = errors.New(
		"payoutAccount: invalid refreshUrl",
	)
	ErrPayoutAccountStripeResultEmpty = errors.New(
		"payoutAccount: stripe account result is empty",
	)
	ErrPayoutAccountStripeAccountMismatch = errors.New(
		"payoutAccount: stripe account mismatch",
	)
	ErrPayoutAccountStripeLinkEmpty = errors.New(
		"payoutAccount: stripe account link is empty",
	)
)

// StripePayoutAccountGateway defines the Stripe Connect operations required by
// PayoutAccountUsecase.
//
// The application layer does not depend on a concrete Stripe adapter.
// backend/internal/adapters/out/stripe/account_gateway.go implements this port.
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

// CreatePayoutAccountLinkInput is supplied by the authenticated Mall handler.
//
// UserID must be obtained from UserAuthMiddleware rather than request JSON.
// DisplayName and ContactEmail are authentication/profile values used only
// when the Stripe Connected Account is created for the first time.
type CreatePayoutAccountLinkInput struct {
	UserID       string
	DisplayName  string
	ContactEmail string
	ReturnURL    string
	RefreshURL   string
}

type CreatePayoutAccountLinkResult struct {
	Account       payoutdom.PayoutAccount
	OnboardingURL string
	ExpiresAt     time.Time
}

// PayoutAccountUsecase manages the resale seller's Stripe payout destination.
//
// Policy:
//   - one User has at most one PayoutAccount
//   - PayoutAccount.UserID is also the Firestore document ID
//   - one Stripe Connected Account is created per User
//   - repeated Account Link creation reuses the existing Connected Account
//   - Stripe is the source of truth for onboarding and bank account state
type PayoutAccountUsecase struct {
	repo    payoutdom.Repository
	gateway StripePayoutAccountGateway

	allowedReturnOrigin string
	now                 func() time.Time
}

func NewPayoutAccountUsecase(
	repo payoutdom.Repository,
	gateway StripePayoutAccountGateway,
	allowedReturnOrigin string,
) *PayoutAccountUsecase {
	return &PayoutAccountUsecase{
		repo:                repo,
		gateway:             gateway,
		allowedReturnOrigin: strings.TrimSpace(allowedReturnOrigin),
		now:                 time.Now,
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

	userID = strings.TrimSpace(userID)
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

// CreateAccountLink creates or continues Stripe hosted onboarding.
//
// First request:
//   - create Stripe Connected Account with a stable idempotency key
//   - persist payoutAccounts/{userId}
//   - create Account Link
//
// Later requests:
//   - reuse the stored StripeAccountID
//   - synchronize current Stripe state
//   - create a fresh single-use Account Link
//
// Account Link itself is intentionally not idempotent because Stripe links are
// single-use and should be regenerated for every request.
func (u *PayoutAccountUsecase) CreateAccountLink(
	ctx context.Context,
	in CreatePayoutAccountLinkInput,
) (CreatePayoutAccountLinkResult, error) {
	if err := u.validateReady(); err != nil {
		return CreatePayoutAccountLinkResult{}, err
	}

	userID := strings.TrimSpace(in.UserID)
	if userID == "" {
		return CreatePayoutAccountLinkResult{},
			payoutdom.ErrInvalidUserID
	}

	returnURL := strings.TrimSpace(in.ReturnURL)
	if err := u.validateCallbackURL(returnURL); err != nil {
		return CreatePayoutAccountLinkResult{},
			ErrPayoutAccountInvalidReturnURL
	}

	refreshURL := strings.TrimSpace(in.RefreshURL)
	if err := u.validateCallbackURL(refreshURL); err != nil {
		return CreatePayoutAccountLinkResult{},
			ErrPayoutAccountInvalidRefreshURL
	}

	account, err := u.repo.GetByUserID(ctx, userID)
	switch {
	case err == nil:
		account, err = u.syncStripeState(ctx, account)
		if err != nil {
			return CreatePayoutAccountLinkResult{}, err
		}

	case errors.Is(err, payoutdom.ErrNotFound):
		account, err = u.createPayoutAccount(
			ctx,
			userID,
			strings.TrimSpace(in.DisplayName),
			strings.TrimSpace(in.ContactEmail),
		)
		if err != nil {
			return CreatePayoutAccountLinkResult{}, err
		}

	default:
		return CreatePayoutAccountLinkResult{}, err
	}

	linkUseCase := StripePayoutAccountLinkUseCaseOnboarding
	if account.DetailsSubmitted {
		linkUseCase = StripePayoutAccountLinkUseCaseUpdate
	}

	link, err := u.gateway.CreatePayoutAccountLink(
		ctx,
		CreateStripePayoutAccountLinkInput{
			StripeAccountID: account.StripeAccountID,
			UseCase:         linkUseCase,
			ReturnURL:       returnURL,
			RefreshURL:      refreshURL,
		},
	)
	if err != nil {
		return CreatePayoutAccountLinkResult{}, err
	}

	if link == nil || strings.TrimSpace(link.URL) == "" {
		return CreatePayoutAccountLinkResult{},
			ErrPayoutAccountStripeLinkEmpty
	}

	linkAccountID := strings.TrimSpace(link.AccountID)
	if linkAccountID == "" ||
		linkAccountID != account.StripeAccountID {
		return CreatePayoutAccountLinkResult{},
			ErrPayoutAccountStripeAccountMismatch
	}

	return CreatePayoutAccountLinkResult{
		Account:       account,
		OnboardingURL: strings.TrimSpace(link.URL),
		ExpiresAt:     link.ExpiresAt,
	}, nil
}

// createPayoutAccount creates the Stripe Connected Account and persists its
// association with the User.
//
// The Stripe create request uses a stable per-user idempotency key. Therefore
// concurrent/retried requests do not intentionally create multiple Stripe
// Connected Accounts.
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
		CreateStripePayoutAccountInput{
			UserID:         userID,
			DisplayName:    displayName,
			ContactEmail:   contactEmail,
			Country:        defaultPayoutAccountCountry,
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

	stripeAccountID := strings.TrimSpace(stripeAccount.ID)
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

	stripeAccountID := strings.TrimSpace(stripeAccount.ID)
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
		bankName = strings.TrimSpace(bankAccount.BankName)
		bankLast4 = strings.TrimSpace(bankAccount.Last4)
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
	if strings.TrimSpace(u.allowedReturnOrigin) == "" {
		return ErrPayoutAccountAllowedReturnOriginMissing
	}

	allowedURL, err := url.Parse(u.allowedReturnOrigin)
	if err != nil ||
		allowedURL.Scheme == "" ||
		allowedURL.Host == "" ||
		(allowedURL.Scheme != "https" && allowedURL.Scheme != "http") {
		return ErrPayoutAccountAllowedReturnOriginMissing
	}

	return nil
}

// validateCallbackURL prevents the client from supplying an arbitrary external
// Stripe return/refresh destination.
//
// Only URLs whose scheme and host match allowedReturnOrigin are accepted.
// Path and query may differ so the frontend can use callback state such as:
//
//	/settings/payout-account?stripe=return
//	/settings/payout-account?stripe=refresh
func (u *PayoutAccountUsecase) validateCallbackURL(
	rawURL string,
) error {
	if rawURL == "" {
		return ErrPayoutAccountInvalidReturnURL
	}

	callbackURL, err := url.Parse(rawURL)
	if err != nil ||
		callbackURL.Scheme == "" ||
		callbackURL.Host == "" ||
		callbackURL.User != nil {
		return ErrPayoutAccountInvalidReturnURL
	}

	allowedURL, err := url.Parse(
		strings.TrimSpace(u.allowedReturnOrigin),
	)
	if err != nil ||
		allowedURL.Scheme == "" ||
		allowedURL.Host == "" {
		return ErrPayoutAccountAllowedReturnOriginMissing
	}

	if !strings.EqualFold(callbackURL.Scheme, allowedURL.Scheme) ||
		!strings.EqualFold(callbackURL.Host, allowedURL.Host) {
		return ErrPayoutAccountInvalidReturnURL
	}

	return nil
}
