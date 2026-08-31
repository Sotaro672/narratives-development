// backend/internal/domain/payoutAccount/entity.go

package payoutAccount

import (
	"errors"
	"time"
)

type BankAccountType string

const (
	BankAccountTypeOrdinary BankAccountType = "ordinary"
	BankAccountTypeCurrent  BankAccountType = "current"
)

var (
	ErrInvalidUserID                  = errors.New("payoutAccount: invalid userId")
	ErrInvalidBankCode                = errors.New("payoutAccount: invalid bankCode")
	ErrInvalidBankName                = errors.New("payoutAccount: invalid bankName")
	ErrInvalidBranchCode              = errors.New("payoutAccount: invalid branchCode")
	ErrInvalidBranchName              = errors.New("payoutAccount: invalid branchName")
	ErrInvalidBankAccountType         = errors.New("payoutAccount: invalid bank account type")
	ErrInvalidAccountNumberCiphertext = errors.New("payoutAccount: invalid accountNumberCiphertext")
	ErrInvalidBankLast4               = errors.New("payoutAccount: invalid bankLast4")
	ErrInvalidAccountHolderName       = errors.New("payoutAccount: invalid accountHolderName")
	ErrInvalidCreatedAt               = errors.New("payoutAccount: invalid createdAt")
	ErrInvalidUpdatedAt               = errors.New("payoutAccount: invalid updatedAt")
)

var (
	MaxUserIDLength                  = 128
	MaxBankNameLength                = 100
	MaxBranchNameLength              = 100
	MaxAccountNumberCiphertextLength = 8192
	MaxAccountHolderNameLength       = 128
)

// PayoutAccount represents the bank account registered by a Mall user for
// receiving resale proceeds.
//
// Persistence policy:
//   - Firestore document path: payoutAccounts/{userId}
//   - One User has at most one PayoutAccount.
//   - The absence of payoutAccounts/{userId} represents an unregistered user.
//   - The full plaintext bank account number must never be persisted.
//   - AccountNumberCiphertext contains only the encrypted account number.
//   - AccountNumberCiphertext must never be exposed to the browser.
//   - BankLast4 is persisted separately for display purposes.
//
// PayoutAccount does not represent a payment-provider account and therefore
// contains no provider, providerAccountId, onboarding status, or payoutReady
// state.
type PayoutAccount struct {
	UserID string `json:"userId" firestore:"userId"`

	BankCode   string `json:"bankCode" firestore:"bankCode"`
	BankName   string `json:"bankName" firestore:"bankName"`
	BranchCode string `json:"branchCode" firestore:"branchCode"`
	BranchName string `json:"branchName" firestore:"branchName"`

	AccountType             BankAccountType `json:"accountType" firestore:"accountType"`
	AccountNumberCiphertext string          `json:"-" firestore:"accountNumberCiphertext"`
	BankLast4               string          `json:"bankLast4" firestore:"bankLast4"`
	AccountHolderName       string          `json:"accountHolderName" firestore:"accountHolderName"`

	CreatedAt time.Time `json:"createdAt" firestore:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt" firestore:"updatedAt"`
}

// New creates a persisted PayoutAccount.
//
// accountNumberCiphertext must contain only an encrypted representation of the
// bank account number. The plaintext account number must be encrypted before
// this constructor is called.
func New(
	userID string,
	bankCode string,
	bankName string,
	branchCode string,
	branchName string,
	accountType BankAccountType,
	accountNumberCiphertext string,
	bankLast4 string,
	accountHolderName string,
	createdAt time.Time,
	updatedAt time.Time,
) (PayoutAccount, error) {
	account := PayoutAccount{
		UserID:                  userID,
		BankCode:                bankCode,
		BankName:                bankName,
		BranchCode:              branchCode,
		BranchName:              branchName,
		AccountType:             accountType,
		AccountNumberCiphertext: accountNumberCiphertext,
		BankLast4:               bankLast4,
		AccountHolderName:       accountHolderName,
		CreatedAt:               createdAt.UTC(),
		UpdatedAt:               updatedAt.UTC(),
	}

	if err := account.Validate(); err != nil {
		return PayoutAccount{}, err
	}

	return account, nil
}

// NewWithNow creates a PayoutAccount using the same timestamp for CreatedAt and
// UpdatedAt.
func NewWithNow(
	userID string,
	bankCode string,
	bankName string,
	branchCode string,
	branchName string,
	accountType BankAccountType,
	accountNumberCiphertext string,
	bankLast4 string,
	accountHolderName string,
	now time.Time,
) (PayoutAccount, error) {
	return New(
		userID,
		bankCode,
		bankName,
		branchCode,
		branchName,
		accountType,
		accountNumberCiphertext,
		bankLast4,
		accountHolderName,
		now,
		now,
	)
}

// ApplyBankAccount replaces the registered payout destination.
//
// accountNumberCiphertext must already be encrypted before this method is
// called. The plaintext account number must never be passed to this entity.
func (a *PayoutAccount) ApplyBankAccount(
	bankCode string,
	bankName string,
	branchCode string,
	branchName string,
	accountType BankAccountType,
	accountNumberCiphertext string,
	bankLast4 string,
	accountHolderName string,
	now time.Time,
) error {
	if a == nil {
		return ErrInvalidUserID
	}

	if err := validateBankCode(bankCode); err != nil {
		return err
	}
	if err := validateBankName(bankName); err != nil {
		return err
	}
	if err := validateBranchCode(branchCode); err != nil {
		return err
	}
	if err := validateBranchName(branchName); err != nil {
		return err
	}
	if err := validateBankAccountType(accountType); err != nil {
		return err
	}
	if err := validateAccountNumberCiphertext(accountNumberCiphertext); err != nil {
		return err
	}
	if err := validateBankLast4(bankLast4); err != nil {
		return err
	}
	if err := validateAccountHolderName(accountHolderName); err != nil {
		return err
	}
	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}

	a.BankCode = bankCode
	a.BankName = bankName
	a.BranchCode = branchCode
	a.BranchName = branchName
	a.AccountType = accountType
	a.AccountNumberCiphertext = accountNumberCiphertext
	a.BankLast4 = bankLast4
	a.AccountHolderName = accountHolderName
	a.UpdatedAt = now.UTC()

	return nil
}

// Validate validates the persisted PayoutAccount state.
//
// Input normalization is intentionally not performed in the domain layer.
// Callers must normalize request values before constructing or updating the
// entity.
func (a PayoutAccount) Validate() error {
	if err := validateUserID(a.UserID); err != nil {
		return err
	}
	if err := validateBankCode(a.BankCode); err != nil {
		return err
	}
	if err := validateBankName(a.BankName); err != nil {
		return err
	}
	if err := validateBranchCode(a.BranchCode); err != nil {
		return err
	}
	if err := validateBranchName(a.BranchName); err != nil {
		return err
	}
	if err := validateBankAccountType(a.AccountType); err != nil {
		return err
	}
	if err := validateAccountNumberCiphertext(a.AccountNumberCiphertext); err != nil {
		return err
	}
	if err := validateBankLast4(a.BankLast4); err != nil {
		return err
	}
	if err := validateAccountHolderName(a.AccountHolderName); err != nil {
		return err
	}
	if a.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if a.UpdatedAt.IsZero() || a.UpdatedAt.Before(a.CreatedAt) {
		return ErrInvalidUpdatedAt
	}

	return nil
}

func validateUserID(userID string) error {
	if userID == "" {
		return ErrInvalidUserID
	}
	if MaxUserIDLength > 0 && len([]rune(userID)) > MaxUserIDLength {
		return ErrInvalidUserID
	}

	return nil
}

func validateBankCode(bankCode string) error {
	if !isFixedDigits(bankCode, 4) {
		return ErrInvalidBankCode
	}

	return nil
}

func validateBankName(bankName string) error {
	if bankName == "" {
		return ErrInvalidBankName
	}
	if MaxBankNameLength > 0 && len([]rune(bankName)) > MaxBankNameLength {
		return ErrInvalidBankName
	}

	return nil
}

func validateBranchCode(branchCode string) error {
	if !isFixedDigits(branchCode, 3) {
		return ErrInvalidBranchCode
	}

	return nil
}

func validateBranchName(branchName string) error {
	if branchName == "" {
		return ErrInvalidBranchName
	}
	if MaxBranchNameLength > 0 && len([]rune(branchName)) > MaxBranchNameLength {
		return ErrInvalidBranchName
	}

	return nil
}

func validateBankAccountType(accountType BankAccountType) error {
	switch accountType {
	case BankAccountTypeOrdinary, BankAccountTypeCurrent:
		return nil
	default:
		return ErrInvalidBankAccountType
	}
}

func validateAccountNumberCiphertext(accountNumberCiphertext string) error {
	if accountNumberCiphertext == "" {
		return ErrInvalidAccountNumberCiphertext
	}
	if MaxAccountNumberCiphertextLength > 0 &&
		len(accountNumberCiphertext) > MaxAccountNumberCiphertextLength {
		return ErrInvalidAccountNumberCiphertext
	}

	return nil
}

func validateBankLast4(bankLast4 string) error {
	if !isFixedDigits(bankLast4, 4) {
		return ErrInvalidBankLast4
	}

	return nil
}

func validateAccountHolderName(accountHolderName string) error {
	if accountHolderName == "" {
		return ErrInvalidAccountHolderName
	}
	if MaxAccountHolderNameLength > 0 &&
		len([]rune(accountHolderName)) > MaxAccountHolderNameLength {
		return ErrInvalidAccountHolderName
	}

	return nil
}

func isFixedDigits(value string, length int) bool {
	if len(value) != length {
		return false
	}

	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}

	return true
}
