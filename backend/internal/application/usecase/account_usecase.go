// backend/internal/application/usecase/account_usecase.go
package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	accdom "narratives/internal/domain/account"
	branddom "narratives/internal/domain/brand"
)

// AccountUsecase provides application-level operations for Account.
type AccountUsecase struct {
	repo           AccountRepo
	brandRepo      AccountBrandRepo
	accountGateway StripeAccountGateway
}

// AccountRepo is the minimal repository contract needed by this use case.
type AccountRepo interface {
	ListByCompanyID(
		ctx context.Context,
		companyID string,
	) ([]accdom.Account, error)

	GetByID(
		ctx context.Context,
		id string,
	) (accdom.Account, error)

	GetByBrandID(
		ctx context.Context,
		brandID string,
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

// AccountBrandRepo is the minimal Brand repository contract
// required for Stripe account connection.
type AccountBrandRepo interface {
	GetByID(
		ctx context.Context,
		id string,
	) (branddom.Brand, error)
}

// StripeAccountGateway defines the Stripe Connect operations
// required by AccountUsecase.
//
// application/usecase は Stripe adapter の具体型へ依存せず、
// outbound adapter がこの Port を実装します。
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
	CompanyID      string
	BrandID        string
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
type CreateStripeAccountLinkInput struct {
	StripeAccountID string
	ReturnURL       string
	RefreshURL      string
	IdempotencyKey  string
}

// StripeAccountLinkResult represents a Stripe hosted onboarding link.
type StripeAccountLinkResult struct {
	AccountID string
	URL       string
	ExpiresAt time.Time
}

// ConnectBrandAccountInput contains the information required
// to create or continue Stripe onboarding for a Brand.
type ConnectBrandAccountInput struct {
	BrandID      string
	ContactEmail string
	Country      string
	ReturnURL    string
	RefreshURL   string
}

// ConnectBrandAccountResult represents a connected account
// and its Stripe hosted onboarding URL.
type ConnectBrandAccountResult struct {
	Account       accdom.Account
	OnboardingURL string
	ExpiresAt     time.Time
}

// NewAccountUsecase creates an AccountUsecase.
func NewAccountUsecase(
	repo AccountRepo,
	brandRepo AccountBrandRepo,
	accountGateway StripeAccountGateway,
) *AccountUsecase {
	return &AccountUsecase{
		repo:           repo,
		brandRepo:      brandRepo,
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

// GetByBrandID returns the account connected to a Brand.
// 1 Brand = 1 Account を前提とする。
func (u *AccountUsecase) GetByBrandID(
	ctx context.Context,
	brandID string,
) (accdom.Account, error) {
	if u == nil || u.repo == nil {
		return accdom.Account{}, errors.New("account: repository is nil")
	}

	brandID = strings.TrimSpace(
		brandID,
	)
	if brandID == "" {
		return accdom.Account{}, accdom.ErrInvalidBrandID
	}

	companyID := CompanyIDFromContext(ctx)
	if companyID == "" {
		return accdom.Account{}, accdom.ErrInvalidCompanyID
	}

	a, err := u.repo.GetByBrandID(
		ctx,
		brandID,
	)
	if err != nil {
		return accdom.Account{}, err
	}

	if a.CompanyID != companyID {
		return accdom.Account{}, accdom.ErrNotFound
	}

	return a, nil
}

// ConnectBrandAccount creates a Stripe Connected Account when one does not
// exist for the Brand and returns a Stripe hosted onboarding URL.
//
// 既に Brand に Account が存在する場合は新しい Stripe Account を作らず、
// 既存 StripeAccountID に対して新しい onboarding link を発行します。
func (u *AccountUsecase) ConnectBrandAccount(
	ctx context.Context,
	in ConnectBrandAccountInput,
) (ConnectBrandAccountResult, error) {
	if u == nil || u.repo == nil {
		return ConnectBrandAccountResult{},
			errors.New("account: repository is nil")
	}

	if u.brandRepo == nil {
		return ConnectBrandAccountResult{},
			errors.New("account: brand repository is nil")
	}

	if u.accountGateway == nil {
		return ConnectBrandAccountResult{},
			errors.New("account: stripe account gateway is nil")
	}

	companyID := CompanyIDFromContext(ctx)
	if companyID == "" {
		return ConnectBrandAccountResult{},
			accdom.ErrInvalidCompanyID
	}

	memberID := MemberIDFromContext(ctx)
	if memberID == "" {
		return ConnectBrandAccountResult{},
			accdom.ErrInvalidMemberID
	}

	brandID := strings.TrimSpace(
		in.BrandID,
	)
	if brandID == "" {
		return ConnectBrandAccountResult{},
			accdom.ErrInvalidBrandID
	}

	returnURL := strings.TrimSpace(
		in.ReturnURL,
	)
	if returnURL == "" {
		return ConnectBrandAccountResult{},
			errors.New("account: returnUrl is empty")
	}

	refreshURL := strings.TrimSpace(
		in.RefreshURL,
	)
	if refreshURL == "" {
		return ConnectBrandAccountResult{},
			errors.New("account: refreshUrl is empty")
	}

	brand, err := u.brandRepo.GetByID(
		ctx,
		brandID,
	)
	if err != nil {
		if errors.Is(
			err,
			branddom.ErrNotFound,
		) {
			return ConnectBrandAccountResult{},
				accdom.ErrNotFound
		}

		return ConnectBrandAccountResult{},
			err
	}

	if brand.CompanyID != companyID {
		return ConnectBrandAccountResult{},
			accdom.ErrNotFound
	}

	a, err := u.repo.GetByBrandID(
		ctx,
		brandID,
	)
	switch {
	case err == nil:
		if a.CompanyID != companyID {
			return ConnectBrandAccountResult{},
				accdom.ErrNotFound
		}

	case errors.Is(
		err,
		accdom.ErrNotFound,
	):
		a, err = u.createStripeAccount(
			ctx,
			companyID,
			memberID,
			brand,
			strings.TrimSpace(
				in.ContactEmail,
			),
			strings.TrimSpace(
				in.Country,
			),
		)
		if err != nil {
			return ConnectBrandAccountResult{},
				err
		}

	default:
		return ConnectBrandAccountResult{},
			err
	}

	if strings.TrimSpace(
		a.StripeAccountID,
	) == "" {
		return ConnectBrandAccountResult{},
			accdom.ErrInvalidStripeAccountID
	}

	link, err := u.accountGateway.CreateOnboardingLink(
		ctx,
		CreateStripeAccountLinkInput{
			StripeAccountID: a.StripeAccountID,
			ReturnURL:       returnURL,
			RefreshURL:      refreshURL,
			IdempotencyKey: "stripe-connect-account-link:" +
				a.ID,
		},
	)
	if err != nil {
		return ConnectBrandAccountResult{},
			err
	}

	if link == nil ||
		strings.TrimSpace(
			link.URL,
		) == "" {
		return ConnectBrandAccountResult{},
			errors.New(
				"account: stripe onboarding link is empty",
			)
	}

	return ConnectBrandAccountResult{
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
	brandID string,
) (accdom.Account, error) {
	if u == nil || u.repo == nil {
		return accdom.Account{},
			errors.New("account: repository is nil")
	}

	if u.brandRepo == nil {
		return accdom.Account{},
			errors.New("account: brand repository is nil")
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

	brandID = strings.TrimSpace(
		brandID,
	)
	if brandID == "" {
		return accdom.Account{},
			accdom.ErrInvalidBrandID
	}

	brand, err := u.brandRepo.GetByID(
		ctx,
		brandID,
	)
	if err != nil {
		if errors.Is(
			err,
			branddom.ErrNotFound,
		) {
			return accdom.Account{},
				accdom.ErrNotFound
		}

		return accdom.Account{},
			err
	}

	if brand.CompanyID != companyID {
		return accdom.Account{},
			accdom.ErrNotFound
	}

	a, err := u.repo.GetByBrandID(
		ctx,
		brandID,
	)
	if err != nil {
		return accdom.Account{},
			err
	}

	if a.CompanyID != companyID {
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

	if a.BrandID == "" {
		return accdom.Account{}, accdom.ErrInvalidBrandID
	}

	if a.StripeAccountID == "" {
		return accdom.Account{}, accdom.ErrInvalidStripeAccountID
	}

	if a.CompanyID != "" &&
		a.CompanyID != companyID {
		return accdom.Account{}, accdom.ErrInvalidCompanyID
	}

	if u.brandRepo != nil {
		brand, err := u.brandRepo.GetByID(
			ctx,
			a.BrandID,
		)
		if err != nil {
			if errors.Is(
				err,
				branddom.ErrNotFound,
			) {
				return accdom.Account{},
					accdom.ErrNotFound
			}

			return accdom.Account{},
				err
		}

		if brand.CompanyID != companyID {
			return accdom.Account{},
				accdom.ErrNotFound
		}
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
		return accdom.Account{}, accdom.ErrInvalidCompanyID
	}

	if patch.BrandID != nil &&
		*patch.BrandID != current.BrandID {
		if u.brandRepo == nil {
			return accdom.Account{},
				errors.New(
					"account: brand repository is nil",
				)
		}

		brandID := strings.TrimSpace(
			*patch.BrandID,
		)
		if brandID == "" {
			return accdom.Account{},
				accdom.ErrInvalidBrandID
		}

		brand, err := u.brandRepo.GetByID(
			ctx,
			brandID,
		)
		if err != nil {
			if errors.Is(
				err,
				branddom.ErrNotFound,
			) {
				return accdom.Account{},
					accdom.ErrNotFound
			}

			return accdom.Account{},
				err
		}

		if brand.CompanyID != companyID {
			return accdom.Account{},
				accdom.ErrNotFound
		}
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
		return accdom.Account{}, accdom.ErrNotFound
	}

	return updated, nil
}

// ========================================
// Stripe Account creation
// ========================================

func (u *AccountUsecase) createStripeAccount(
	ctx context.Context,
	companyID string,
	memberID string,
	brand branddom.Brand,
	contactEmail string,
	country string,
) (accdom.Account, error) {
	if u.accountGateway == nil {
		return accdom.Account{},
			errors.New(
				"account: stripe account gateway is nil",
			)
	}

	stripeAccount, err := u.accountGateway.CreateAccount(
		ctx,
		CreateStripeAccountInput{
			CompanyID:    companyID,
			BrandID:      brand.ID,
			DisplayName:  brand.Name,
			ContactEmail: contactEmail,
			Country:      country,
			IdempotencyKey: "stripe-connect-account:" +
				brand.ID,
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

	// BrandID から決定的に AccountID を生成することで、
	// 同一 Brand に対する並行作成でも同じ Firestore DocID を使用します。
	accountID :=
		accdom.AccountIDPrefix +
			brand.ID

	a, err := accdom.NewWithNow(
		accountID,
		companyID,
		brand.ID,
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

	// Stripe CreateAccount は BrandID ベースの Idempotency-Key を使用するため、
	// 並行リクエストで Firestore Create が競合しても、
	// 既存 Account を再取得して利用できます。
	if errors.Is(
		err,
		accdom.ErrConflict,
	) {
		existing, getErr :=
			u.repo.GetByBrandID(
				ctx,
				brand.ID,
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
