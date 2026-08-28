// backend/internal/application/usecase/account_usecase.go
package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	applicationport "narratives/internal/application/port"
	accdom "narratives/internal/domain/account"
)

// AccountUsecase provides application-level operations for Account.
type AccountUsecase struct {
	repo           AccountRepo
	accountGateway applicationport.StripeAccountGateway
}

// AccountRepo is the minimal repository contract needed by this use case.
type AccountRepo interface {
	NewID(
		ctx context.Context,
	) (string, error)

	ListByCompanyID(
		ctx context.Context,
		companyID string,
	) ([]accdom.Account, error)

	GetByID(
		ctx context.Context,
		id string,
	) (accdom.Account, error)

	Create(
		ctx context.Context,
		a accdom.Account,
	) (accdom.Account, error)

	Update(
		ctx context.Context,
		id string,
		patch accdom.AccountPatch,
	) (accdom.Account, error)
}

// ConnectAccountInput contains the information required
// to create or continue Stripe onboarding for an Account.
//
// AccountID が空の場合は新規 Account を作成します。
// AccountID が指定されている場合は既存 Account の
// Stripe onboarding を継続します。
type ConnectAccountInput struct {
	AccountID    string
	DisplayName  string
	ContactEmail string
	Country      string
	ReturnURL    string
	RefreshURL   string
}

// ConnectAccountResult represents an Account
// and its Stripe hosted onboarding URL.
type ConnectAccountResult struct {
	Account       accdom.Account
	OnboardingURL string
	ExpiresAt     time.Time
}

// NewAccountUsecase creates an AccountUsecase.
func NewAccountUsecase(
	repo AccountRepo,
	accountGateway applicationport.StripeAccountGateway,
) *AccountUsecase {
	return &AccountUsecase{
		repo:           repo,
		accountGateway: accountGateway,
	}
}

// ListByCompanyID returns accounts belonging to the authenticated company.
func (u *AccountUsecase) ListByCompanyID(
	ctx context.Context,
) ([]accdom.Account, error) {
	if u == nil || u.repo == nil {
		return nil, errors.New("account: repository is nil")
	}

	companyID := CompanyIDFromContext(ctx)
	if companyID == "" {
		return nil, accdom.ErrInvalidCompanyID
	}

	return u.repo.ListByCompanyID(
		ctx,
		companyID,
	)
}

// GetByID returns an account by ID.
func (u *AccountUsecase) GetByID(
	ctx context.Context,
	id string,
) (accdom.Account, error) {
	if u == nil || u.repo == nil {
		return accdom.Account{}, errors.New("account: repository is nil")
	}

	id = strings.TrimSpace(
		id,
	)
	if id == "" {
		return accdom.Account{}, accdom.ErrInvalidID
	}

	companyID := CompanyIDFromContext(ctx)
	if companyID == "" {
		return accdom.Account{}, accdom.ErrInvalidCompanyID
	}

	a, err := u.repo.GetByID(
		ctx,
		id,
	)
	if err != nil {
		return accdom.Account{}, err
	}

	if a.CompanyID != companyID {
		return accdom.Account{}, accdom.ErrNotFound
	}

	return a, nil
}

// ConnectAccount creates a Stripe Connected Account when AccountID is empty,
// or continues onboarding for an existing Account when AccountID is specified.
//
// Account は Brand より先に作成できます。
// Brand との関連付けは Brand.AccountID 側で管理します。
func (u *AccountUsecase) ConnectAccount(
	ctx context.Context,
	in ConnectAccountInput,
) (ConnectAccountResult, error) {
	if u == nil || u.repo == nil {
		return ConnectAccountResult{},
			errors.New("account: repository is nil")
	}

	if u.accountGateway == nil {
		return ConnectAccountResult{},
			errors.New("account: stripe account gateway is nil")
	}

	companyID := CompanyIDFromContext(ctx)
	if companyID == "" {
		return ConnectAccountResult{},
			accdom.ErrInvalidCompanyID
	}

	memberID := MemberIDFromContext(ctx)
	if memberID == "" {
		return ConnectAccountResult{},
			accdom.ErrInvalidMemberID
	}

	returnURL := strings.TrimSpace(
		in.ReturnURL,
	)
	if returnURL == "" {
		return ConnectAccountResult{},
			errors.New("account: returnUrl is empty")
	}

	refreshURL := strings.TrimSpace(
		in.RefreshURL,
	)
	if refreshURL == "" {
		return ConnectAccountResult{},
			errors.New("account: refreshUrl is empty")
	}

	accountID := strings.TrimSpace(
		in.AccountID,
	)

	var (
		a   accdom.Account
		err error
	)

	if accountID != "" {
		a, err = u.repo.GetByID(
			ctx,
			accountID,
		)
		if err != nil {
			return ConnectAccountResult{},
				err
		}

		if a.CompanyID != companyID {
			return ConnectAccountResult{},
				accdom.ErrNotFound
		}

		if a.Status == accdom.StatusDeleted {
			return ConnectAccountResult{},
				accdom.ErrNotFound
		}
	} else {
		accountID, err = u.repo.NewID(
			ctx,
		)
		if err != nil {
			return ConnectAccountResult{},
				err
		}

		accountID = strings.TrimSpace(
			accountID,
		)
		if accountID == "" {
			return ConnectAccountResult{},
				accdom.ErrInvalidID
		}

		a, err = u.createStripeAccount(
			ctx,
			accountID,
			companyID,
			memberID,
			strings.TrimSpace(
				in.DisplayName,
			),
			strings.TrimSpace(
				in.ContactEmail,
			),
			strings.TrimSpace(
				in.Country,
			),
		)
		if err != nil {
			return ConnectAccountResult{},
				err
		}
	}

	stripeAccountID := strings.TrimSpace(
		a.StripeAccountID,
	)
	if stripeAccountID == "" {
		return ConnectAccountResult{},
			accdom.ErrInvalidStripeAccountID
	}

	link, err := u.accountGateway.CreateOnboardingLink(
		ctx,
		applicationport.CreateStripeAccountLinkInput{
			StripeAccountID: stripeAccountID,
			ReturnURL:       returnURL,
			RefreshURL:      refreshURL,
		},
	)
	if err != nil {
		return ConnectAccountResult{},
			err
	}

	if link == nil ||
		strings.TrimSpace(
			link.URL,
		) == "" {
		return ConnectAccountResult{},
			errors.New(
				"account: stripe onboarding link is empty",
			)
	}

	return ConnectAccountResult{
		Account: a,
		OnboardingURL: strings.TrimSpace(
			link.URL,
		),
		ExpiresAt: link.ExpiresAt,
	}, nil
}

// SyncStripeStatus retrieves the latest Stripe Connected Account state
// and reflects transfer capability status into Account.Status.
func (u *AccountUsecase) SyncStripeStatus(
	ctx context.Context,
	accountID string,
) (accdom.Account, error) {
	if u == nil || u.repo == nil {
		return accdom.Account{},
			errors.New("account: repository is nil")
	}

	if u.accountGateway == nil {
		return accdom.Account{},
			errors.New("account: stripe account gateway is nil")
	}

	companyID := CompanyIDFromContext(ctx)
	if companyID == "" {
		return accdom.Account{},
			accdom.ErrInvalidCompanyID
	}

	accountID = strings.TrimSpace(
		accountID,
	)
	if accountID == "" {
		return accdom.Account{},
			accdom.ErrInvalidID
	}

	a, err := u.repo.GetByID(
		ctx,
		accountID,
	)
	if err != nil {
		return accdom.Account{},
			err
	}

	if a.CompanyID != companyID {
		return accdom.Account{},
			accdom.ErrNotFound
	}

	if a.Status == accdom.StatusDeleted {
		return accdom.Account{},
			accdom.ErrNotFound
	}

	stripeAccountID := strings.TrimSpace(
		a.StripeAccountID,
	)
	if stripeAccountID == "" {
		return accdom.Account{},
			accdom.ErrInvalidStripeAccountID
	}

	stripeAccount, err := u.accountGateway.GetAccount(
		ctx,
		stripeAccountID,
	)
	if err != nil {
		return accdom.Account{},
			err
	}

	if stripeAccount == nil {
		return accdom.Account{},
			errors.New(
				"account: stripe account result is nil",
			)
	}

	nextStatus := accountStatusFromStripe(
		stripeAccount.RecipientTransferStatus,
		stripeAccount.Closed,
	)

	if a.Status == nextStatus {
		return a, nil
	}

	patch := accdom.AccountPatch{
		Status: &nextStatus,
	}

	memberID := MemberIDFromContext(ctx)
	if memberID != "" {
		patch.UpdatedBy =
			&memberID
	}

	return u.repo.Update(
		ctx,
		a.ID,
		patch,
	)
}

// Create creates a new account for the authenticated company.
func (u *AccountUsecase) Create(
	ctx context.Context,
	a accdom.Account,
) (accdom.Account, error) {
	if u == nil || u.repo == nil {
		return accdom.Account{}, errors.New("account: repository is nil")
	}

	companyID := CompanyIDFromContext(ctx)
	if companyID == "" {
		return accdom.Account{}, accdom.ErrInvalidCompanyID
	}

	if strings.TrimSpace(
		a.ID,
	) == "" {
		accountID, err := u.repo.NewID(
			ctx,
		)
		if err != nil {
			return accdom.Account{},
				err
		}

		accountID = strings.TrimSpace(
			accountID,
		)
		if accountID == "" {
			return accdom.Account{},
				accdom.ErrInvalidID
		}

		a.ID = accountID
	}

	if strings.TrimSpace(
		a.StripeAccountID,
	) == "" {
		return accdom.Account{},
			accdom.ErrInvalidStripeAccountID
	}

	if a.CompanyID != "" &&
		a.CompanyID != companyID {
		return accdom.Account{},
			accdom.ErrInvalidCompanyID
	}

	a.CompanyID = companyID

	return u.repo.Create(
		ctx,
		a,
	)
}

// Update updates an account belonging to the authenticated company.
func (u *AccountUsecase) Update(
	ctx context.Context,
	id string,
	patch accdom.AccountPatch,
) (accdom.Account, error) {
	if u == nil || u.repo == nil {
		return accdom.Account{}, errors.New("account: repository is nil")
	}

	id = strings.TrimSpace(
		id,
	)
	if id == "" {
		return accdom.Account{}, accdom.ErrInvalidID
	}

	companyID := CompanyIDFromContext(ctx)
	if companyID == "" {
		return accdom.Account{}, accdom.ErrInvalidCompanyID
	}

	current, err := u.repo.GetByID(
		ctx,
		id,
	)
	if err != nil {
		return accdom.Account{}, err
	}

	if current.CompanyID != companyID {
		return accdom.Account{}, accdom.ErrNotFound
	}

	if patch.CompanyID != nil &&
		*patch.CompanyID != companyID {
		return accdom.Account{},
			accdom.ErrInvalidCompanyID
	}

	// Account を別 Company へ移動することは許可しない。
	patch.CompanyID = nil

	updated, err := u.repo.Update(
		ctx,
		id,
		patch,
	)
	if err != nil {
		return accdom.Account{}, err
	}

	if updated.CompanyID != companyID {
		return accdom.Account{},
			accdom.ErrNotFound
	}

	return updated, nil
}

// ========================================
// Stripe Account creation
// ========================================

func (u *AccountUsecase) createStripeAccount(
	ctx context.Context,
	accountID string,
	companyID string,
	memberID string,
	displayName string,
	contactEmail string,
	country string,
) (accdom.Account, error) {
	if u.accountGateway == nil {
		return accdom.Account{},
			errors.New(
				"account: stripe account gateway is nil",
			)
	}

	accountID = strings.TrimSpace(
		accountID,
	)
	if accountID == "" {
		return accdom.Account{},
			accdom.ErrInvalidID
	}

	companyID = strings.TrimSpace(
		companyID,
	)
	if companyID == "" {
		return accdom.Account{},
			accdom.ErrInvalidCompanyID
	}

	memberID = strings.TrimSpace(
		memberID,
	)
	if memberID == "" {
		return accdom.Account{},
			accdom.ErrInvalidMemberID
	}

	displayName = strings.TrimSpace(
		displayName,
	)
	if displayName == "" {
		displayName = "AMOL"
	}

	stripeAccount, err := u.accountGateway.CreateAccount(
		ctx,
		applicationport.CreateStripeAccountInput{
			AccountID:    accountID,
			CompanyID:    companyID,
			DisplayName:  displayName,
			ContactEmail: contactEmail,
			Country:      country,
			IdempotencyKey: "stripe-connect-account:" +
				accountID,
		},
	)
	if err != nil {
		return accdom.Account{},
			err
	}

	if stripeAccount == nil {
		return accdom.Account{},
			errors.New(
				"account: stripe account result is nil",
			)
	}

	stripeAccountID := strings.TrimSpace(
		stripeAccount.ID,
	)
	if stripeAccountID == "" {
		return accdom.Account{},
			accdom.ErrInvalidStripeAccountID
	}

	now := time.Now().UTC()

	status := accountStatusFromStripe(
		stripeAccount.RecipientTransferStatus,
		stripeAccount.Closed,
	)

	a, err := accdom.NewWithNow(
		accountID,
		companyID,
		stripeAccountID,
		memberID,
		"",
		"",
		0,
		"",
		accdom.DefaultCurrency,
		status,
		now,
	)
	if err != nil {
		return accdom.Account{},
			err
	}

	a.CreatedBy =
		&memberID
	a.UpdatedBy =
		&memberID

	created, err := u.repo.Create(
		ctx,
		a,
	)
	if err == nil {
		return created, nil
	}

	// Stripe CreateAccount は AccountID ベースの
	// stable Idempotency-Key を使用するため、
	// Firestore Create が競合した場合は同じ AccountID の
	//既存 Account を再取得して利用できます。
	if errors.Is(
		err,
		accdom.ErrConflict,
	) {
		existing, getErr :=
			u.repo.GetByID(
				ctx,
				accountID,
			)
		if getErr != nil {
			return accdom.Account{},
				getErr
		}

		if existing.CompanyID != companyID {
			return accdom.Account{},
				accdom.ErrNotFound
		}

		return existing, nil
	}

	return accdom.Account{},
		err
}

// ========================================
// Stripe status mapping
// ========================================

func accountStatusFromStripe(
	recipientTransferStatus string,
	closed bool,
) accdom.AccountStatus {
	if closed {
		return accdom.StatusSuspended
	}

	switch strings.ToLower(
		strings.TrimSpace(
			recipientTransferStatus,
		),
	) {
	case "active":
		return accdom.StatusActive

	case "restricted",
		"unsupported":
		return accdom.StatusSuspended

	case "pending":
		return accdom.StatusInactive

	default:
		return accdom.StatusInactive
	}
}
