// backend\internal\domain\account\entity.go
package account

import (
	"errors"
	"time"
)

// Errors (inlined from error.go)
var (
	ErrInvalidID            = errors.New("account: invalid id")
	ErrInvalidMemberID      = errors.New("account: invalid memberId")
	ErrInvalidBankName      = errors.New("account: invalid bankName")
	ErrInvalidBranchName    = errors.New("account: invalid branchName")
	ErrInvalidAccountNumber = errors.New("account: invalid accountNumber")
	ErrInvalidAccountType   = errors.New("account: invalid accountType")
	ErrInvalidCurrency      = errors.New("account: invalid currency")
	ErrInvalidStatus        = errors.New("account: invalid status")
	ErrInvalidCreatedAt     = errors.New("account: invalid createdAt")
	ErrInvalidUpdatedAt     = errors.New("account: invalid updatedAt")
)

// Enums (mirror TS)
// AccountStatus: "active" | "inactive" | "suspended" | "deleted"
type AccountStatus string

const (
	StatusActive    AccountStatus = "active"
	StatusInactive  AccountStatus = "inactive"
	StatusSuspended AccountStatus = "suspended"
	StatusDeleted   AccountStatus = "deleted"
)

func IsValidStatus(s AccountStatus) bool {
	switch s {
	case StatusActive, StatusInactive, StatusSuspended, StatusDeleted:
		return true
	default:
		return false
	}
}

// AccountType: "普通" | "当座"
type AccountType string

const (
	TypeFutsu AccountType = "普通"
	TypeToza  AccountType = "当座"
)

func IsValidAccountType(t AccountType) bool {
	switch t {
	case TypeFutsu, TypeToza:
		return true
	default:
		return false
	}
}

// Entity (mirror TS BankAccount)
type Account struct {
	ID            string
	MemberID      string
	BankName      string
	BranchName    string
	AccountNumber int
	AccountType   AccountType
	Currency      string
	Status        AccountStatus
	CreatedAt     time.Time
	CreatedBy     *string
	UpdatedAt     time.Time
	UpdatedBy     *string
	DeletedAt     *time.Time
	DeletedBy     *string
}

// Policy (sync with web-app/src/shared/types/account.ts)
var (
	AccountIDPrefix     = "account_"
	DefaultCurrency     = "円"
	MaxBankNameLength   = 50
	MaxBranchNameLength = 50

	// accountNumber: number (0..99,999,999)
	MinAccountNumber = 0
	MaxAccountNumber = 99_999_999

	// MemberID length limit (adjust as needed to match frontend rules).
	MaxMemberIDLength = 100
)

// Constructors

func New(
	id, memberID, bankName, branchName string,
	accountNumber int,
	accountType AccountType,
	currency string,
	status AccountStatus,
	createdAt, updatedAt time.Time,
) (Account, error) {
	a := Account{
		ID:            id,
		MemberID:      memberID,
		BankName:      bankName,
		BranchName:    branchName,
		AccountNumber: accountNumber,
		AccountType:   accountType,
		Currency:      currency,
		Status:        status,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
	}
	if err := a.validate(); err != nil {
		return Account{}, err
	}
	return a, nil
}

func NewWithNow(
	id, memberID, bankName, branchName string,
	accountNumber int,
	accountType AccountType,
	currency string,
	status AccountStatus,
	now time.Time,
) (Account, error) {
	return New(id, memberID, bankName, branchName, accountNumber, accountType, currency, status, now, now)
}

// ========================================
// Delete
// ========================================

// Delete は論理削除（ステータスを deleted にして UpdatedAt/DeletedAt を更新）を行います。
func (a *Account) Delete(now time.Time) error {
	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}
	a.Status = StatusDeleted
	a.UpdatedAt = now
	a.DeletedAt = &now
	return nil
}

// Validation

func (a Account) validate() error {
	if a.ID == "" {
		return ErrInvalidID
	}
	if a.MemberID == "" || (MaxMemberIDLength > 0 && len([]rune(a.MemberID)) > MaxMemberIDLength) {
		return ErrInvalidMemberID
	}
	if a.BankName == "" || (MaxBankNameLength > 0 && len([]rune(a.BankName)) > MaxBankNameLength) {
		return ErrInvalidBankName
	}
	if a.BranchName == "" || (MaxBranchNameLength > 0 && len([]rune(a.BranchName)) > MaxBranchNameLength) {
		return ErrInvalidBranchName
	}
	if a.AccountNumber < MinAccountNumber || a.AccountNumber > MaxAccountNumber {
		return ErrInvalidAccountNumber
	}
	if !IsValidAccountType(a.AccountType) {
		return ErrInvalidAccountType
	}
	if a.Currency == "" {
		return ErrInvalidCurrency
	}
	if !IsValidStatus(a.Status) {
		return ErrInvalidStatus
	}
	if a.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	if a.UpdatedAt.IsZero() || a.UpdatedAt.Before(a.CreatedAt) {
		return ErrInvalidUpdatedAt
	}
	if a.DeletedAt != nil && a.DeletedAt.Before(a.CreatedAt) {
		return ErrInvalidUpdatedAt
	}
	return nil
}

// 口座名義（表示用）: MemberID をそのまま利用します。
func (a Account) AccountHolderName() string {
	return a.MemberID
}
