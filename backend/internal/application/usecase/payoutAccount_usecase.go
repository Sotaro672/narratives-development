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

var (
	ErrPayoutAccountRepositoryMissing   = errors.New("payoutAccount: repository is not configured")
	ErrPayoutAccountProviderMissing     = errors.New("payoutAccount: provider is not configured")
	ErrPayoutAccountProviderResultEmpty = errors.New("payoutAccount: provider registration result is empty")
	ErrPayoutAccountProviderStateEmpty  = errors.New("payoutAccount: provider state is empty")
	ErrPayoutAccountProviderMismatch    = errors.New("payoutAccount: provider mismatch")
)

// RegisterPayoutAccountInput contains payout destination information submitted
// by the authenticated Mall user.
//
// UserID must be obtained from UserAuthMiddleware rather than request JSON.
//
// AccountNumber is transient sensitive data. It is passed to the configured
// PayoutAccountProvider only during registration and must never be persisted
// by this usecase or repository.
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

// PayoutAccountUsecase manages a Mall user's payout destination.
//
// Policy:
//   - one User has at most one persisted PayoutAccount
//   - PayoutAccount.UserID is also the Firestore document ID
//   - application code does not depend on a specific payout vendor
//   - full bank account numbers are never persisted
//   - ProviderAccountID is backend-only
//   - StatusRegistered means AMOL registration completed
//   - PayoutReady independently represents actual payout availability
type PayoutAccountUsecase struct {
	repo     payoutdom.Repository
	provider applicationport.PayoutAccountProvider

	now func() time.Time
}

func NewPayoutAccountUsecase(
	repo payoutdom.Repository,
	provider applicationport.PayoutAccountProvider,
) *PayoutAccountUsecase {
	return &PayoutAccountUsecase{
		repo:     repo,
		provider: provider,
		now:      time.Now,
	}
}

// GetByUserID returns the user's persisted PayoutAccount after synchronizing
// provider-side availability state.
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

	synced, err := u.syncProviderState(ctx, account)
	if err != nil {
		return nil, err
	}

	return &synced, nil
}

// Register registers or replaces the authenticated user's payout destination.
//
// The full AccountNumber is passed only to PayoutAccountProvider.Register.
// Only BankLast4 returned by the provider is persisted.
//
// Existing payoutAccounts/{userId} documents are updated in place so CreatedAt
// remains unchanged. A missing document is created.
//
// If a concurrent request creates the document first, the canonical document is
// loaded and updated with the registration result from this request.
func (u *PayoutAccountUsecase) Register(
	ctx context.Context,
	in RegisterPayoutAccountInput,
) (*payoutdom.PayoutAccount, error) {
	if err := u.validateReady(); err != nil {
		return nil, err
	}

	in.UserID = strings.TrimSpace(in.UserID)
	in.BankCode = strings.TrimSpace(in.BankCode)
	in.BankName = strings.TrimSpace(in.BankName)
	in.BranchCode = strings.TrimSpace(in.BranchCode)
	in.BranchName = strings.TrimSpace(in.BranchName)
	in.AccountNumber = strings.TrimSpace(in.AccountNumber)
	in.AccountHolderName = strings.TrimSpace(in.AccountHolderName)

	if in.UserID == "" {
		return nil, payoutdom.ErrInvalidUserID
	}

	providerName := strings.TrimSpace(u.provider.Name())
	if providerName == "" {
		return nil, payoutdom.ErrInvalidProvider
	}

	result, err := u.provider.Register(
		ctx,
		applicationport.RegisterPayoutAccountInput{
			UserID:            in.UserID,
			BankCode:          in.BankCode,
			BankName:          in.BankName,
			BranchCode:        in.BranchCode,
			BranchName:        in.BranchName,
			AccountType:       in.AccountType,
			AccountNumber:     in.AccountNumber,
			AccountHolderName: in.AccountHolderName,
		},
	)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrPayoutAccountProviderResultEmpty
	}

	now := u.now().UTC()

	existing, err := u.repo.GetByUserID(ctx, in.UserID)
	switch {
	case err == nil:
		updated := existing

		if err := updated.ApplyRegistration(
			providerName,
			result.ProviderAccountID,
			result.Status,
			result.PayoutReady,
			in.BankCode,
			in.BankName,
			in.BranchCode,
			in.BranchName,
			in.AccountType,
			result.BankLast4,
			in.AccountHolderName,
			now,
		); err != nil {
			return nil, err
		}

		saved, err := u.repo.Update(ctx, updated)
		if err != nil {
			return nil, err
		}

		return &saved, nil

	case errors.Is(err, payoutdom.ErrNotFound):
		account, err := payoutdom.New(
			in.UserID,
			providerName,
			result.ProviderAccountID,
			result.Status,
			result.PayoutReady,
			in.BankCode,
			in.BankName,
			in.BranchCode,
			in.BranchName,
			in.AccountType,
			result.BankLast4,
			in.AccountHolderName,
			now,
			now,
		)
		if err != nil {
			return nil, err
		}

		created, err := u.repo.Create(ctx, account)
		if err == nil {
			return &created, nil
		}
		if !errors.Is(err, payoutdom.ErrConflict) {
			return nil, err
		}

		return u.applyRegistrationAfterConflict(ctx, in, providerName, result, now)

	default:
		return nil, err
	}
}

// applyRegistrationAfterConflict handles the case where another request created
// payoutAccounts/{userId} between GetByUserID and Create.
//
// The persisted document remains canonical, while the latest successful
// provider registration result is applied to it.
func (u *PayoutAccountUsecase) applyRegistrationAfterConflict(
	ctx context.Context,
	in RegisterPayoutAccountInput,
	providerName string,
	result *applicationport.RegisterPayoutAccountResult,
	now time.Time,
) (*payoutdom.PayoutAccount, error) {
	existing, err := u.repo.GetByUserID(ctx, in.UserID)
	if err != nil {
		return nil, err
	}

	if err := existing.ApplyRegistration(
		providerName,
		result.ProviderAccountID,
		result.Status,
		result.PayoutReady,
		in.BankCode,
		in.BankName,
		in.BranchCode,
		in.BranchName,
		in.AccountType,
		result.BankLast4,
		in.AccountHolderName,
		now,
	); err != nil {
		return nil, err
	}

	updated, err := u.repo.Update(ctx, existing)
	if err != nil {
		return nil, err
	}

	return &updated, nil
}

// syncProviderState refreshes only provider-owned availability state.
//
// Bank metadata is not overwritten here because it is the snapshot registered
// through AMOL. If bank information changes, the user must perform a new
// registration operation.
//
// A persisted account belonging to another provider is not sent to the currently
// configured provider because ProviderAccountID namespaces are provider-specific.
func (u *PayoutAccountUsecase) syncProviderState(
	ctx context.Context,
	account payoutdom.PayoutAccount,
) (payoutdom.PayoutAccount, error) {
	providerName := strings.TrimSpace(u.provider.Name())
	if providerName == "" {
		return payoutdom.PayoutAccount{}, payoutdom.ErrInvalidProvider
	}
	if account.Provider != providerName {
		return payoutdom.PayoutAccount{}, ErrPayoutAccountProviderMismatch
	}

	state, err := u.provider.Get(ctx, account.ProviderAccountID)
	if err != nil {
		return payoutdom.PayoutAccount{}, err
	}
	if state == nil {
		return payoutdom.PayoutAccount{}, ErrPayoutAccountProviderStateEmpty
	}

	if account.Status == state.Status && account.PayoutReady == state.PayoutReady {
		return account, nil
	}

	updated := account
	if err := updated.ApplyProviderState(
		state.Status,
		state.PayoutReady,
		u.now().UTC(),
	); err != nil {
		return payoutdom.PayoutAccount{}, err
	}

	return u.repo.Update(ctx, updated)
}

func (u *PayoutAccountUsecase) validateReady() error {
	if u == nil || u.repo == nil {
		return ErrPayoutAccountRepositoryMissing
	}
	if u.provider == nil {
		return ErrPayoutAccountProviderMissing
	}

	return nil
}
