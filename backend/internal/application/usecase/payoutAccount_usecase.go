// backend/internal/application/usecase/payoutAccount_usecase.go
package usecase

import (
	"context"
	"errors"
	"time"

	applicationport "narratives/internal/application/port"
	payoutdom "narratives/internal/domain/payoutAccount"
)

var (
	ErrPayoutAccountRepositoryMissing = errors.New(
		"payoutAccount: repository is not configured",
	)
	ErrPayoutAccountCipherMissing = errors.New(
		"payoutAccount: account number cipher is not configured",
	)
	ErrPayoutAccountClockMissing = errors.New(
		"payoutAccount: clock is not configured",
	)
	ErrPayoutAccountInvalidAccountNumber = errors.New(
		"payoutAccount: invalid account number",
	)
	ErrPayoutAccountEncryptionFailed = errors.New(
		"payoutAccount: account number encryption failed",
	)
	ErrPayoutAccountOwnershipMismatch = errors.New(
		"payoutAccount: user ownership mismatch",
	)
)

// RegisterPayoutAccountInput contains the bank destination registered by the
// authenticated Mall user.
//
// AccountNumber is sensitive transient data. It exists only long enough to be
// validated, encrypted, and reduced to BankLast4.
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

// PayoutAccountUsecase manages the bank account used to settle a Mall user's
// resale sales receivables.
//
// Policy:
//   - payoutAccounts/{userId} is the canonical persistence location.
//   - one User has at most one PayoutAccount.
//   - the absence of payoutAccounts/{userId} means unregistered.
//   - no payment-provider account is created.
//   - no onboarding status or payoutReady state exists.
//   - plaintext account numbers are never persisted.
//   - account numbers are encrypted before Repository.Create/Update.
//   - BankLast4 is persisted separately for display.
//   - registering again replaces the user's payout destination while preserving
//     CreatedAt.
type PayoutAccountUsecase struct {
	repo   payoutdom.Repository
	cipher applicationport.PayoutAccountCipher
	now    func() time.Time
}

func NewPayoutAccountUsecase(
	repo payoutdom.Repository,
	cipher applicationport.PayoutAccountCipher,
) *PayoutAccountUsecase {
	return &PayoutAccountUsecase{
		repo:   repo,
		cipher: cipher,
		now:    time.Now,
	}
}

// GetByUserID returns the registered payout bank account.
//
// payoutdom.ErrNotFound is returned when the user has not registered a payout
// destination yet.
func (u *PayoutAccountUsecase) GetByUserID(
	ctx context.Context,
	userID string,
) (*payoutdom.PayoutAccount, error) {
	if err := u.validateRepositoryReady(); err != nil {
		return nil, err
	}
	if !isValidPayoutUserID(userID) {
		return nil, payoutdom.ErrInvalidUserID
	}

	account, err := u.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if account.UserID != userID {
		return nil, ErrPayoutAccountOwnershipMismatch
	}
	if err := account.Validate(); err != nil {
		return nil, err
	}

	return &account, nil
}

// Register creates or replaces the authenticated user's payout bank account.
//
// The plaintext account number is never passed to the domain entity or
// repository. Only its encrypted representation and last four digits are
// persisted.
func (u *PayoutAccountUsecase) Register(
	ctx context.Context,
	in RegisterPayoutAccountInput,
) (*payoutdom.PayoutAccount, error) {
	if err := u.validateRegistrationReady(); err != nil {
		return nil, err
	}
	if !isValidPayoutUserID(in.UserID) {
		return nil, payoutdom.ErrInvalidUserID
	}
	if !isValidPayoutAccountNumber(in.AccountNumber) {
		return nil, ErrPayoutAccountInvalidAccountNumber
	}

	accountNumberCiphertext, err := u.cipher.Encrypt(
		ctx,
		in.UserID,
		in.AccountNumber,
	)
	if err != nil {
		return nil, ErrPayoutAccountEncryptionFailed
	}
	if accountNumberCiphertext == "" {
		return nil, ErrPayoutAccountEncryptionFailed
	}

	bankLast4 := in.AccountNumber[len(in.AccountNumber)-4:]
	now := u.now().UTC()

	existing, err := u.repo.GetByUserID(ctx, in.UserID)
	switch {
	case err == nil:
		return u.replaceBankAccount(
			ctx,
			existing,
			in,
			accountNumberCiphertext,
			bankLast4,
			now,
		)

	case errors.Is(err, payoutdom.ErrNotFound):
		account, err := payoutdom.New(
			in.UserID,
			in.BankCode,
			in.BankName,
			in.BranchCode,
			in.BranchName,
			in.AccountType,
			accountNumberCiphertext,
			bankLast4,
			in.AccountHolderName,
			now,
			now,
		)
		if err != nil {
			return nil, err
		}

		created, err := u.repo.Create(ctx, account)
		if err == nil {
			if created.UserID != in.UserID {
				return nil, ErrPayoutAccountOwnershipMismatch
			}

			return &created, nil
		}
		if !errors.Is(err, payoutdom.ErrConflict) {
			return nil, err
		}

		return u.replaceAfterCreateConflict(
			ctx,
			in,
			accountNumberCiphertext,
			bankLast4,
			now,
		)

	default:
		return nil, err
	}
}

func (u *PayoutAccountUsecase) replaceAfterCreateConflict(
	ctx context.Context,
	in RegisterPayoutAccountInput,
	accountNumberCiphertext string,
	bankLast4 string,
	now time.Time,
) (*payoutdom.PayoutAccount, error) {
	existing, err := u.repo.GetByUserID(ctx, in.UserID)
	if err != nil {
		return nil, err
	}

	return u.replaceBankAccount(
		ctx,
		existing,
		in,
		accountNumberCiphertext,
		bankLast4,
		now,
	)
}

func (u *PayoutAccountUsecase) replaceBankAccount(
	ctx context.Context,
	existing payoutdom.PayoutAccount,
	in RegisterPayoutAccountInput,
	accountNumberCiphertext string,
	bankLast4 string,
	now time.Time,
) (*payoutdom.PayoutAccount, error) {
	if existing.UserID != in.UserID {
		return nil, ErrPayoutAccountOwnershipMismatch
	}
	if err := existing.ApplyBankAccount(
		in.BankCode,
		in.BankName,
		in.BranchCode,
		in.BranchName,
		in.AccountType,
		accountNumberCiphertext,
		bankLast4,
		in.AccountHolderName,
		now,
	); err != nil {
		return nil, err
	}

	updated, err := u.repo.Update(ctx, existing)
	if err != nil {
		return nil, err
	}
	if updated.UserID != in.UserID {
		return nil, ErrPayoutAccountOwnershipMismatch
	}
	if err := updated.Validate(); err != nil {
		return nil, err
	}

	return &updated, nil
}

func (u *PayoutAccountUsecase) validateRepositoryReady() error {
	if u == nil || u.repo == nil {
		return ErrPayoutAccountRepositoryMissing
	}

	return nil
}

func (u *PayoutAccountUsecase) validateRegistrationReady() error {
	if err := u.validateRepositoryReady(); err != nil {
		return err
	}
	if u.cipher == nil {
		return ErrPayoutAccountCipherMissing
	}
	if u.now == nil {
		return ErrPayoutAccountClockMissing
	}

	return nil
}

func isValidPayoutUserID(value string) bool {
	if value == "" {
		return false
	}

	for _, r := range value {
		switch r {
		case ' ', '\t', '\r', '\n':
			return false
		}
	}

	return true
}

func isValidPayoutAccountNumber(value string) bool {
	if len(value) != 7 {
		return false
	}

	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}

	return true
}
