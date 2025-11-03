package account

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Errors (inlined from error.go)
var (
	ErrInvalidID       = errors.New("account: invalid id")
	ErrInvalidMemberID = errors.New("account: invalid memberId")
	// backward compatibility alias
	ErrInvalidBrandName     = ErrInvalidMemberID
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
	// backward compatibility alias
	MaxBrandNameLength = MaxMemberIDLength
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
		ID:            strings.TrimSpace(id),
		MemberID:      strings.TrimSpace(memberID),
		BankName:      strings.TrimSpace(bankName),
		BranchName:    strings.TrimSpace(branchName),
		AccountNumber: accountNumber,
		AccountType:   accountType,
		Currency:      strings.TrimSpace(currency),
		Status:        status,
		CreatedAt:     createdAt.UTC(),
		UpdatedAt:     updatedAt.UTC(),
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
	now = now.UTC()
	return New(id, memberID, bankName, branchName, accountNumber, accountType, currency, status, now, now)
}

func NewFromStringTimes(
	id, memberID, bankName, branchName string,
	accountNumber int,
	accountType AccountType,
	currency string,
	status AccountStatus,
	createdAt, updatedAt string,
) (Account, error) {
	ct, err := parseTime(createdAt, ErrInvalidCreatedAt)
	if err != nil {
		return Account{}, err
	}
	ut, err := parseTime(updatedAt, ErrInvalidUpdatedAt)
	if err != nil {
		return Account{}, err
	}
	return New(id, memberID, bankName, branchName, accountNumber, accountType, currency, status, ct, ut)
}

// ========================================
// Delete
// ========================================

// Delete は論理削除（ステータスを deleted にして UpdatedAt/DeletedAt を更新）を行います。
func (a *Account) Delete(now time.Time) error {
	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}
	now = now.UTC()
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
	// account number: 0..99,999,999
	if a.AccountNumber < MinAccountNumber || a.AccountNumber > MaxAccountNumber {
		return ErrInvalidAccountNumber
	}
	if !IsValidAccountType(a.AccountType) {
		return ErrInvalidAccountType
	}
	if strings.TrimSpace(a.Currency) == "" {
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

// AccountsTableDDL defines the SQL for the accounts table migration.
const AccountsTableDDL = `
-- Accounts DDL generated from domain/account entity.
CREATE TABLE IF NOT EXISTS accounts (
  id TEXT PRIMARY KEY,
  member_id VARCHAR(100) NOT NULL,
  account_holder_name VARCHAR(100) GENERATED ALWAYS AS (member_id) STORED,
  bank_name VARCHAR(50) NOT NULL,
  branch_name VARCHAR(50) NOT NULL,
  account_number INTEGER NOT NULL CHECK (account_number >= 0 AND account_number <= 99999999),
  account_type TEXT NOT NULL CHECK (account_type IN ('普通','当座')),
  currency TEXT NOT NULL DEFAULT '円',
  status TEXT NOT NULL CHECK (status IN ('active','inactive','suspended','deleted')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_by TEXT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_by TEXT NULL,
  deleted_at TIMESTAMPTZ NULL,
  deleted_by TEXT NULL,
  CHECK (updated_at >= created_at),
  CHECK (deleted_at IS NULL OR deleted_at >= created_at),
  CHECK (id LIKE 'account_%')
);

CREATE INDEX IF NOT EXISTS idx_accounts_member_id ON accounts(member_id);
CREATE INDEX IF NOT EXISTS idx_accounts_status ON accounts(status);
CREATE INDEX IF NOT EXISTS idx_accounts_account_number ON accounts(account_number);
CREATE INDEX IF NOT EXISTS idx_accounts_deleted_at ON accounts(deleted_at);
`

// 口座名義（表示用）: MemberID をそのまま利用します。
func (a Account) AccountHolderName() string {
	return a.MemberID
}

// Helpers (moved from error.go)

func parseTime(s string, classify error) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, classify
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), nil
	}
	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("%w: cannot parse %q", classify, s)
}
