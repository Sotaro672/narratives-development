// backend/internal/domain/payoutAccount/entity.go

package payoutAccount

import (
	"errors"
	"strings"
	"time"
)

type Status string

const (
	StatusPending    Status = "pending"
	StatusRegistered Status = "registered"
	StatusRestricted Status = "restricted"
)

type BankAccountType string

const (
	BankAccountTypeOrdinary BankAccountType = "ordinary"
	BankAccountTypeCurrent  BankAccountType = "current"
)

var (
	ErrInvalidUserID            = errors.New("payoutAccount: invalid userId")
	ErrInvalidProvider          = errors.New("payoutAccount: invalid provider")
	ErrInvalidProviderAccountID = errors.New("payoutAccount: invalid providerAccountId")
	ErrInvalidStatus            = errors.New("payoutAccount: invalid status")
	ErrInvalidBankCode          = errors.New("payoutAccount: invalid bankCode")
	ErrInvalidBankName          = errors.New("payoutAccount: invalid bankName")
	ErrInvalidBranchCode        = errors.New("payoutAccount: invalid branchCode")
	ErrInvalidBranchName        = errors.New("payoutAccount: invalid branchName")
	ErrInvalidBankAccountType   = errors.New("payoutAccount: invalid bank account type")
	ErrInvalidBankLast4         = errors.New("payoutAccount: invalid bankLast4")
	ErrInvalidAccountHolderName = errors.New("payoutAccount: invalid accountHolderName")
	ErrInvalidCreatedAt         = errors.New("payoutAccount: invalid createdAt")
	ErrInvalidUpdatedAt         = errors.New("payoutAccount: invalid updatedAt")
)

var (
	MaxUserIDLength            = 128
	MaxProviderLength          = 64
	MaxProviderAccountIDLength = 255
	MaxBankNameLength          = 100
	MaxBranchNameLength        = 100
	MaxAccountHolderNameLength = 128
)

// PayoutAccount represents the payout destination registered by a Mall user.
//
// Persistence policy:
//   - Firestore document path: payoutAccounts/{userId}
//   - One User has at most one PayoutAccount.
//   - ProviderAccountID is an internal provider-side identifier and must not be
//     exposed to the browser.
//   - Full bank account numbers are never persisted.
//   - BankLast4 is the only persisted account-number fragment.
//
// Registration state:
//   - StatusRegistered means the user completed AMOL's registration flow.
//   - PayoutReady means the external payout provider has confirmed that the
//     destination can actually receive payouts.
//   - StatusRegistered and PayoutReady=false is valid, especially while using
//     a mock provider or while provider-side confirmation is pending.
//
// An unregistered user is normally represented by the absence of
// payoutAccounts/{userId}, rather than a persisted "unregistered" status.
type PayoutAccount struct {
	UserID string `json:"userId" firestore:"userId"`

	Provider          string `json:"provider" firestore:"provider"`
	ProviderAccountID string `json:"-" firestore:"providerAccountId"`

	Status      Status `json:"status" firestore:"status"`
	PayoutReady bool   `json:"payoutReady" firestore:"payoutReady"`

	BankCode   string `json:"bankCode" firestore:"bankCode"`
	BankName   string `json:"bankName" firestore:"bankName"`
	BranchCode string `json:"branchCode" firestore:"branchCode"`
	BranchName string `json:"branchName" firestore:"branchName"`

	AccountType       BankAccountType `json:"accountType" firestore:"accountType"`
	BankLast4         string          `json:"bankLast4" firestore:"bankLast4"`
	AccountHolderName string          `json:"accountHolderName" firestore:"accountHolderName"`

	CreatedAt time.Time `json:"createdAt" firestore:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt" firestore:"updatedAt"`
}

// New creates a persisted PayoutAccount.
//
// accountNumber itself must never be supplied to this entity. Only bankLast4 is
// accepted for persistence.
func New(
	userID string,
	provider string,
	providerAccountID string,
	status Status,
	payoutReady bool,
	bankCode string,
	bankName string,
	branchCode string,
	branchName string,
	accountType BankAccountType,
	bankLast4 string,
	accountHolderName string,
	createdAt time.Time,
	updatedAt time.Time,
) (PayoutAccount, error) {
	account := PayoutAccount{
		UserID:            strings.TrimSpace(userID),
		Provider:          strings.TrimSpace(provider),
		ProviderAccountID: strings.TrimSpace(providerAccountID),
		Status:            status,
		PayoutReady:       payoutReady,
		BankCode:          strings.TrimSpace(bankCode),
		BankName:          strings.TrimSpace(bankName),
		BranchCode:        strings.TrimSpace(branchCode),
		BranchName:        strings.TrimSpace(branchName),
		AccountType:       accountType,
		BankLast4:         strings.TrimSpace(bankLast4),
		AccountHolderName: strings.TrimSpace(accountHolderName),
		CreatedAt:         createdAt.UTC(),
		UpdatedAt:         updatedAt.UTC(),
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
	provider string,
	providerAccountID string,
	status Status,
	payoutReady bool,
	bankCode string,
	bankName string,
	branchCode string,
	branchName string,
	accountType BankAccountType,
	bankLast4 string,
	accountHolderName string,
	now time.Time,
) (PayoutAccount, error) {
	return New(
		userID,
		provider,
		providerAccountID,
		status,
		payoutReady,
		bankCode,
		bankName,
		branchCode,
		branchName,
		accountType,
		bankLast4,
		accountHolderName,
		now,
		now,
	)
}

// ApplyRegistration replaces the provider and bank-account snapshot after a
// successful registration or account-change operation.
//
// Only bankLast4 is persisted. The full account number must be consumed by the
// provider before this method is called and must not be passed into the domain.
func (a *PayoutAccount) ApplyRegistration(
	provider string,
	providerAccountID string,
	status Status,
	payoutReady bool,
	bankCode string,
	bankName string,
	branchCode string,
	branchName string,
	accountType BankAccountType,
	bankLast4 string,
	accountHolderName string,
	now time.Time,
) error {
	if a == nil {
		return ErrInvalidUserID
	}

	provider = strings.TrimSpace(provider)
	providerAccountID = strings.TrimSpace(providerAccountID)
	bankCode = strings.TrimSpace(bankCode)
	bankName = strings.TrimSpace(bankName)
	branchCode = strings.TrimSpace(branchCode)
	branchName = strings.TrimSpace(branchName)
	bankLast4 = strings.TrimSpace(bankLast4)
	accountHolderName = strings.TrimSpace(accountHolderName)

	if err := validateProvider(provider); err != nil {
		return err
	}
	if err := validateProviderAccountID(providerAccountID); err != nil {
		return err
	}
	if err := validateStatus(status); err != nil {
		return err
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
	if err := validateBankLast4(bankLast4); err != nil {
		return err
	}
	if err := validateAccountHolderName(accountHolderName); err != nil {
		return err
	}
	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}

	a.Provider = provider
	a.ProviderAccountID = providerAccountID
	a.Status = status
	a.PayoutReady = payoutReady
	a.BankCode = bankCode
	a.BankName = bankName
	a.BranchCode = branchCode
	a.BranchName = branchName
	a.AccountType = accountType
	a.BankLast4 = bankLast4
	a.AccountHolderName = accountHolderName
	a.UpdatedAt = now.UTC()

	return nil
}

// ApplyProviderState updates only provider-side availability state.
//
// This can be used later when an external provider asynchronously changes the
// payout destination from pending/restricted to available.
func (a *PayoutAccount) ApplyProviderState(status Status, payoutReady bool, now time.Time) error {
	if a == nil {
		return ErrInvalidUserID
	}
	if err := validateStatus(status); err != nil {
		return err
	}
	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}

	a.Status = status
	a.PayoutReady = payoutReady
	a.UpdatedAt = now.UTC()

	return nil
}

// Validate validates the persisted PayoutAccount state.
func (a PayoutAccount) Validate() error {
	if err := validateUserID(a.UserID); err != nil {
		return err
	}
	if err := validateProvider(a.Provider); err != nil {
		return err
	}
	if err := validateProviderAccountID(a.ProviderAccountID); err != nil {
		return err
	}
	if err := validateStatus(a.Status); err != nil {
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
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ErrInvalidUserID
	}
	if MaxUserIDLength > 0 && len([]rune(userID)) > MaxUserIDLength {
		return ErrInvalidUserID
	}

	return nil
}

func validateProvider(provider string) error {
	provider = strings.TrimSpace(provider)
	if provider == "" {
		return ErrInvalidProvider
	}
	if MaxProviderLength > 0 && len([]rune(provider)) > MaxProviderLength {
		return ErrInvalidProvider
	}

	return nil
}

func validateProviderAccountID(providerAccountID string) error {
	providerAccountID = strings.TrimSpace(providerAccountID)
	if providerAccountID == "" {
		return ErrInvalidProviderAccountID
	}
	if MaxProviderAccountIDLength > 0 && len([]rune(providerAccountID)) > MaxProviderAccountIDLength {
		return ErrInvalidProviderAccountID
	}

	return nil
}

func validateStatus(status Status) error {
	switch status {
	case StatusPending, StatusRegistered, StatusRestricted:
		return nil
	default:
		return ErrInvalidStatus
	}
}

func validateBankCode(bankCode string) error {
	if !isFixedDigits(strings.TrimSpace(bankCode), 4) {
		return ErrInvalidBankCode
	}

	return nil
}

func validateBankName(bankName string) error {
	bankName = strings.TrimSpace(bankName)
	if bankName == "" {
		return ErrInvalidBankName
	}
	if MaxBankNameLength > 0 && len([]rune(bankName)) > MaxBankNameLength {
		return ErrInvalidBankName
	}

	return nil
}

func validateBranchCode(branchCode string) error {
	if !isFixedDigits(strings.TrimSpace(branchCode), 3) {
		return ErrInvalidBranchCode
	}

	return nil
}

func validateBranchName(branchName string) error {
	branchName = strings.TrimSpace(branchName)
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

func validateBankLast4(bankLast4 string) error {
	if !isFixedDigits(strings.TrimSpace(bankLast4), 4) {
		return ErrInvalidBankLast4
	}

	return nil
}

func validateAccountHolderName(accountHolderName string) error {
	accountHolderName = strings.TrimSpace(accountHolderName)
	if accountHolderName == "" {
		return ErrInvalidAccountHolderName
	}
	if MaxAccountHolderNameLength > 0 && len([]rune(accountHolderName)) > MaxAccountHolderNameLength {
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
