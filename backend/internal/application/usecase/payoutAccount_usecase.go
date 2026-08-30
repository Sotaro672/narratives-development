// backend/internal/application/usecase/payoutAccount_usecase.go
package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	applicationport "narratives/internal/application/port"
	avatardom "narratives/internal/domain/avatar"
	payoutdom "narratives/internal/domain/payoutAccount"
)

const (
	payoutAccountCountryJapan                = "JP"
	payoutAccountEntityTypeIndividual        = "individual"
	payoutAccountIdempotencyPrefix           = "mall-payout-account:"
	payoutAccountBankAttachIdempotencyPrefix = "mall-payout-bank-account:"
)

var (
	ErrPayoutAccountRepositoryMissing = errors.New(
		"payoutAccount: repository is not configured",
	)
	ErrPayoutAccountStripeGatewayMissing = errors.New(
		"payoutAccount: Stripe gateway is not configured",
	)
	ErrPayoutAccountAvatarRepositoryMissing = errors.New(
		"payoutAccount: avatar repository is not configured",
	)
	ErrPayoutAccountAuthUserReaderMissing = errors.New(
		"payoutAccount: auth user reader is not configured",
	)
	ErrPayoutAccountStripeResultEmpty = errors.New(
		"payoutAccount: Stripe account result is empty",
	)
	ErrPayoutAccountBankResultEmpty = errors.New(
		"payoutAccount: Stripe bank account result is empty",
	)
	ErrPayoutAccountSessionResultEmpty = errors.New(
		"payoutAccount: Stripe account session result is empty",
	)
	ErrPayoutAccountStripeAccountMismatch = errors.New(
		"payoutAccount: Stripe account mismatch",
	)
	ErrPayoutAccountProviderMismatch = errors.New(
		"payoutAccount: provider mismatch",
	)
	ErrPayoutAccountBankAccountTokenInvalid = errors.New(
		"payoutAccount: invalid bank account token",
	)
	ErrPayoutAccountDirectRegistrationDisabled = errors.New(
		"payoutAccount: direct bank account registration is disabled",
	)

	// Legacy aliases kept temporarily so old callers continue to compile while
	// the token-based Stripe bank account registration flow replaces legacy
	// direct bank registration.
	ErrPayoutAccountProviderMissing     = ErrPayoutAccountStripeGatewayMissing
	ErrPayoutAccountProviderResultEmpty = ErrPayoutAccountStripeResultEmpty
	ErrPayoutAccountProviderStateEmpty  = ErrPayoutAccountStripeResultEmpty
)

// RegisterPayoutAccountInput is retained temporarily for compatibility with
// legacy callers.
//
// Direct bank-account registration through AMOL is disabled. Full bank account
// information must never be sent to the AMOL backend.
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

// PayoutAccountSession contains only the short-lived client secret that may be
// returned to the authenticated browser.
//
// Deprecated: retained temporarily until the legacy Embedded Connect flow is
// removed.
type PayoutAccountSession struct {
	ClientSecret string `json:"clientSecret"`
}

// PayoutAccountUsecase manages the Stripe Connected Account used as a Mall
// user's resale payout destination.
//
// Policy:
//   - one User has at most one persisted PayoutAccount
//   - payoutAccounts/{userId} is the canonical persistence location
//   - Provider is always "stripe" for the production resale payout flow
//   - ProviderAccountID is backend-only
//   - bank account details are tokenized directly by Stripe.js
//   - AMOL receives only the resulting Stripe bank account token
//   - full bank account numbers never pass through this usecase
//   - Stripe account creation uses a stable per-user idempotency key
//   - bank attachment uses a token-hash-based idempotency key
//   - PayoutReady is synchronized from Stripe before payout account reads
type PayoutAccountUsecase struct {
	repo payoutdom.Repository

	stripeGateway  applicationport.StripePayoutAccountGateway
	avatarRepo     avatardom.Repository
	authUserReader applicationport.AuthUserReader

	now func() time.Time
}

// NewPayoutAccountUsecase is retained temporarily so legacy callers compile
// until all DI paths are migrated to NewStripePayoutAccountUsecase.
//
// The legacy provider is intentionally not stored or used.
func NewPayoutAccountUsecase(
	repo payoutdom.Repository,
	_ applicationport.PayoutAccountProvider,
) *PayoutAccountUsecase {
	return &PayoutAccountUsecase{
		repo: repo,
		now:  time.Now,
	}
}

// NewStripePayoutAccountUsecase creates the production Stripe-backed payout
// account usecase.
func NewStripePayoutAccountUsecase(
	repo payoutdom.Repository,
	stripeGateway applicationport.StripePayoutAccountGateway,
	avatarRepo avatardom.Repository,
	authUserReader applicationport.AuthUserReader,
) *PayoutAccountUsecase {
	return &PayoutAccountUsecase{
		repo:           repo,
		stripeGateway:  stripeGateway,
		avatarRepo:     avatarRepo,
		authUserReader: authUserReader,
		now:            time.Now,
	}
}

// GetByUserID returns the persisted payout account after synchronizing the
// current Stripe payout state and display-safe bank metadata.
//
// payoutdom.ErrNotFound is returned when no payout account exists yet.
func (u *PayoutAccountUsecase) GetByUserID(
	ctx context.Context,
	userID string,
) (*payoutdom.PayoutAccount, error) {
	if err := u.validateStripeReady(); err != nil {
		return nil, err
	}

	if !isValidPayoutUserID(userID) {
		return nil, payoutdom.ErrInvalidUserID
	}

	account, err := u.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if account.Provider != payoutdom.ProviderStripe {
		return nil, ErrPayoutAccountProviderMismatch
	}

	synced, err := u.syncStripeState(ctx, account)
	if err != nil {
		return nil, err
	}

	return &synced, nil
}

// EnsureStripeAccount returns the user's existing Stripe Connected Account or
// creates and persists one when none exists.
//
// A legacy non-Stripe PayoutAccount is migrated in place. CreatedAt remains
// unchanged and the Stripe Connected Account becomes the new provider snapshot.
func (u *PayoutAccountUsecase) EnsureStripeAccount(
	ctx context.Context,
	userID string,
) (*payoutdom.PayoutAccount, error) {
	if err := u.validateStripeReady(); err != nil {
		return nil, err
	}

	if !isValidPayoutUserID(userID) {
		return nil, payoutdom.ErrInvalidUserID
	}

	existing, getErr := u.repo.GetByUserID(ctx, userID)
	existingFound := false

	switch {
	case getErr == nil:
		existingFound = true

		if existing.Provider == payoutdom.ProviderStripe {
			synced, err := u.syncStripeState(ctx, existing)
			if err != nil {
				return nil, err
			}

			return &synced, nil
		}

	case errors.Is(getErr, payoutdom.ErrNotFound):
		// Create a Stripe Connected Account below.

	default:
		return nil, getErr
	}

	avatar, err := u.avatarRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if avatar.UserID != userID {
		return nil, payoutdom.ErrInvalidUserID
	}

	displayName := avatar.AvatarName
	if displayName == "" {
		return nil, avatardom.ErrInvalidAvatarName
	}

	contactEmail, err := u.authUserReader.GetEmailByUID(ctx, userID)
	if err != nil {
		return nil, err
	}

	result, err := u.stripeGateway.CreatePayoutAccount(
		ctx,
		applicationport.CreateStripePayoutAccountInput{
			UserID:         userID,
			DisplayName:    displayName,
			ContactEmail:   contactEmail,
			Country:        payoutAccountCountryJapan,
			EntityType:     payoutAccountEntityTypeIndividual,
			IdempotencyKey: payoutAccountIdempotencyPrefix + userID,
		},
	)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrPayoutAccountStripeResultEmpty
	}

	stripeAccountID := result.ID
	if !isValidPayoutStripeAccountID(stripeAccountID) {
		return nil, payoutdom.ErrInvalidProviderAccountID
	}

	status, payoutReady := payoutAccountStateFromStripe(*result)
	now := u.now().UTC()

	var saved payoutdom.PayoutAccount

	if existingFound {
		updated := existing

		if err := updated.ApplyRegistration(
			payoutdom.ProviderStripe,
			stripeAccountID,
			status,
			payoutReady,
			"",
			"",
			"",
			"",
			"",
			"",
			"",
			now,
		); err != nil {
			return nil, err
		}

		saved, err = u.repo.Update(ctx, updated)
		if err != nil {
			return nil, err
		}
	} else {
		account, err := payoutdom.New(
			userID,
			payoutdom.ProviderStripe,
			stripeAccountID,
			status,
			payoutReady,
			"",
			"",
			"",
			"",
			"",
			"",
			"",
			now,
			now,
		)
		if err != nil {
			return nil, err
		}

		saved, err = u.repo.Create(ctx, account)
		if err != nil {
			if !errors.Is(err, payoutdom.ErrConflict) {
				return nil, err
			}

			saved, err = u.applyStripeAccountAfterConflict(
				ctx,
				userID,
				stripeAccountID,
				status,
				payoutReady,
				now,
			)
			if err != nil {
				return nil, err
			}
		}
	}

	synced, err := u.syncStripeState(ctx, saved)
	if err != nil {
		return nil, err
	}

	return &synced, nil
}

// RegisterBankAccount attaches a Stripe.js-created single-use bank account
// token to the authenticated user's persisted Connected Account.
//
// The browser supplies only bankAccountToken. StripeAccountID is always resolved
// on the backend from the authenticated user's PayoutAccount.
//
// Raw bank account numbers, routing numbers, bank codes, branch codes, account
// types, and account holder names must never pass through this method.
func (u *PayoutAccountUsecase) RegisterBankAccount(
	ctx context.Context,
	userID string,
	bankAccountToken string,
) (*payoutdom.PayoutAccount, error) {
	if err := u.validateStripeReady(); err != nil {
		return nil, err
	}

	if !isValidPayoutUserID(userID) {
		return nil, payoutdom.ErrInvalidUserID
	}
	if !isValidPayoutBankAccountToken(bankAccountToken) {
		return nil, ErrPayoutAccountBankAccountTokenInvalid
	}

	account, err := u.EnsureStripeAccount(ctx, userID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, ErrPayoutAccountStripeResultEmpty
	}

	if account.Provider != payoutdom.ProviderStripe {
		return nil, ErrPayoutAccountProviderMismatch
	}

	stripeAccountID := account.ProviderAccountID
	if !isValidPayoutStripeAccountID(stripeAccountID) {
		return nil, payoutdom.ErrInvalidProviderAccountID
	}

	result, err := u.stripeGateway.AttachPayoutBankAccount(
		ctx,
		applicationport.AttachStripePayoutBankAccountInput{
			StripeAccountID:  stripeAccountID,
			BankAccountToken: bankAccountToken,
			IdempotencyKey: payoutAccountBankAttachIdempotencyKey(
				userID,
				bankAccountToken,
			),
		},
	)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrPayoutAccountBankResultEmpty
	}

	synced, err := u.syncStripeState(ctx, *account)
	if err != nil {
		return nil, err
	}

	return &synced, nil
}

// CreateAccountSession creates a short-lived Stripe Account Session for the
// authenticated user's Connected Account.
//
// Deprecated: retained temporarily until the legacy Embedded Connect flow is
// removed.
func (u *PayoutAccountUsecase) CreateAccountSession(
	ctx context.Context,
	userID string,
) (*PayoutAccountSession, error) {
	account, err := u.EnsureStripeAccount(ctx, userID)
	if err != nil {
		return nil, err
	}

	stripeAccountID := account.ProviderAccountID
	if !isValidPayoutStripeAccountID(stripeAccountID) {
		return nil, payoutdom.ErrInvalidProviderAccountID
	}

	result, err := u.stripeGateway.CreatePayoutAccountSession(
		ctx,
		applicationport.CreateStripePayoutAccountSessionInput{
			StripeAccountID: stripeAccountID,
		},
	)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrPayoutAccountSessionResultEmpty
	}

	if result.AccountID != stripeAccountID {
		return nil, ErrPayoutAccountStripeAccountMismatch
	}

	clientSecret := result.ClientSecret
	if clientSecret == "" {
		return nil, ErrPayoutAccountSessionResultEmpty
	}

	return &PayoutAccountSession{
		ClientSecret: clientSecret,
	}, nil
}

// Register is retained only until all legacy direct-registration callers are
// removed.
//
// AMOL must never receive full bank account numbers. The supported registration
// flow is Stripe.js tokenization followed by RegisterBankAccount.
func (u *PayoutAccountUsecase) Register(
	ctx context.Context,
	in RegisterPayoutAccountInput,
) (*payoutdom.PayoutAccount, error) {
	_ = ctx
	_ = in

	return nil, ErrPayoutAccountDirectRegistrationDisabled
}

func (u *PayoutAccountUsecase) applyStripeAccountAfterConflict(
	ctx context.Context,
	userID string,
	stripeAccountID string,
	status payoutdom.Status,
	payoutReady bool,
	now time.Time,
) (payoutdom.PayoutAccount, error) {
	existing, err := u.repo.GetByUserID(ctx, userID)
	if err != nil {
		return payoutdom.PayoutAccount{}, err
	}

	if existing.Provider == payoutdom.ProviderStripe {
		if existing.ProviderAccountID != stripeAccountID {
			return payoutdom.PayoutAccount{}, ErrPayoutAccountStripeAccountMismatch
		}

		return existing, nil
	}

	if err := existing.ApplyRegistration(
		payoutdom.ProviderStripe,
		stripeAccountID,
		status,
		payoutReady,
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		now,
	); err != nil {
		return payoutdom.PayoutAccount{}, err
	}

	return u.repo.Update(ctx, existing)
}

// syncStripeState refreshes payout availability and display-safe bank metadata
// directly from Stripe.
//
// Stripe exposes only BankName and Last4 through the current gateway. AMOL does
// not infer or manufacture bank/branch codes, account type, or holder name.
func (u *PayoutAccountUsecase) syncStripeState(
	ctx context.Context,
	account payoutdom.PayoutAccount,
) (payoutdom.PayoutAccount, error) {
	if account.Provider != payoutdom.ProviderStripe {
		return payoutdom.PayoutAccount{}, ErrPayoutAccountProviderMismatch
	}

	stripeAccountID := account.ProviderAccountID
	if !isValidPayoutStripeAccountID(stripeAccountID) {
		return payoutdom.PayoutAccount{}, payoutdom.ErrInvalidProviderAccountID
	}

	state, err := u.stripeGateway.GetPayoutAccount(ctx, stripeAccountID)
	if err != nil {
		return payoutdom.PayoutAccount{}, err
	}
	if state == nil {
		return payoutdom.PayoutAccount{}, ErrPayoutAccountStripeResultEmpty
	}
	if state.ID != stripeAccountID {
		return payoutdom.PayoutAccount{}, ErrPayoutAccountStripeAccountMismatch
	}

	status, payoutReady := payoutAccountStateFromStripe(*state)

	bank, err := u.stripeGateway.GetPayoutBankAccount(ctx, stripeAccountID)
	if err != nil {
		return payoutdom.PayoutAccount{}, err
	}

	bankName := ""
	bankLast4 := ""

	if bank != nil {
		bankName = bank.BankName
		bankLast4 = bank.Last4
	}

	stateChanged :=
		account.Status != status ||
			account.PayoutReady != payoutReady

	bankChanged :=
		account.BankCode != "" ||
			account.BankName != bankName ||
			account.BranchCode != "" ||
			account.BranchName != "" ||
			account.AccountType != "" ||
			account.BankLast4 != bankLast4 ||
			account.AccountHolderName != ""

	if !stateChanged && !bankChanged {
		return account, nil
	}

	updated := account
	now := u.now().UTC()

	if stateChanged {
		if err := updated.ApplyProviderState(
			status,
			payoutReady,
			now,
		); err != nil {
			return payoutdom.PayoutAccount{}, err
		}
	}

	if bankChanged {
		if err := updated.ApplyBankSnapshot(
			"",
			bankName,
			"",
			"",
			"",
			bankLast4,
			"",
			now,
		); err != nil {
			return payoutdom.PayoutAccount{}, err
		}
	}

	return u.repo.Update(ctx, updated)
}

func payoutAccountStateFromStripe(
	result applicationport.StripePayoutAccountResult,
) (payoutdom.Status, bool) {
	if result.PayoutsEnabled {
		return payoutdom.StatusRegistered, true
	}

	if result.DetailsSubmitted {
		return payoutdom.StatusRestricted, false
	}

	return payoutdom.StatusPending, false
}

func payoutAccountBankAttachIdempotencyKey(
	userID string,
	bankAccountToken string,
) string {
	sum := sha256.Sum256([]byte(bankAccountToken))

	return payoutAccountBankAttachIdempotencyPrefix +
		userID +
		":" +
		hex.EncodeToString(sum[:])
}

func isValidPayoutUserID(value string) bool {
	return value != "" &&
		!strings.ContainsAny(value, " \t\r\n")
}

func isValidPayoutStripeAccountID(value string) bool {
	return value != "" &&
		!strings.ContainsAny(value, " \t\r\n") &&
		strings.HasPrefix(value, "acct_")
}

func isValidPayoutBankAccountToken(value string) bool {
	if value == "" || strings.ContainsAny(value, " \t\r\n") {
		return false
	}

	return strings.HasPrefix(value, "btok_") ||
		strings.HasPrefix(value, "tok_")
}

func (u *PayoutAccountUsecase) validateStripeReady() error {
	if u == nil || u.repo == nil {
		return ErrPayoutAccountRepositoryMissing
	}
	if u.stripeGateway == nil {
		return ErrPayoutAccountStripeGatewayMissing
	}
	if u.avatarRepo == nil {
		return ErrPayoutAccountAvatarRepositoryMissing
	}
	if u.authUserReader == nil {
		return ErrPayoutAccountAuthUserReaderMissing
	}

	return nil
}
