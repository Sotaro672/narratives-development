// backend\internal\domain\account\entity.go
package account

import (
	"errors"
	"strings"
	"time"
)

// Errors (inlined from error.go)
var (
	ErrInvalidID              = errors.New("account: invalid id")
	ErrInvalidCompanyID       = errors.New("account: invalid companyId")
	ErrInvalidBrandID         = errors.New("account: invalid brandId")
	ErrInvalidStripeAccountID = errors.New("account: invalid stripeAccountId")
	ErrInvalidMemberID        = errors.New("account: invalid memberId")
	ErrInvalidBankName        = errors.New("account: invalid bankName")
	ErrInvalidBranchName      = errors.New("account: invalid branchName")
	ErrInvalidAccountNumber   = errors.New("account: invalid accountNumber")
	ErrInvalidAccountType     = errors.New("account: invalid accountType")
	ErrInvalidCurrency        = errors.New("account: invalid currency")
	ErrInvalidStatus          = errors.New("account: invalid status")
	ErrInvalidCreatedAt       = errors.New("account: invalid createdAt")
	ErrInvalidUpdatedAt       = errors.New("account: invalid updatedAt")
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
//
// Account は Company 配下の Brand に紐づく Stripe Connect 受取口座を表します。
//
// - 1 Company は複数 Account を持てる
// - 1 Brand は最大 1 Account を持つ
// - StripeAccountID は Stripe Connected Account の acct_xxx
//
// 1 Brand = 1 Account の一意制約は Repository 層で保証します。
type Account struct {
	ID              string        `json:"id" firestore:"-"`
	CompanyID       string        `json:"companyId" firestore:"companyId"`
	BrandID         string        `json:"brandId" firestore:"brandId"`
	StripeAccountID string        `json:"stripeAccountId" firestore:"stripeAccountId"`
	MemberID        string        `json:"memberId" firestore:"memberId"`
	BankName        string        `json:"bankName" firestore:"bankName"`
	BranchName      string        `json:"branchName" firestore:"branchName"`
	AccountNumber   int           `json:"accountNumber" firestore:"accountNumber"`
	AccountType     AccountType   `json:"accountType" firestore:"accountType"`
	Currency        string        `json:"currency" firestore:"currency"`
	Status          AccountStatus `json:"status" firestore:"status"`
	CreatedAt       time.Time     `json:"createdAt" firestore:"createdAt"`
	CreatedBy       *string       `json:"createdBy,omitempty" firestore:"createdBy,omitempty"`
	UpdatedAt       time.Time     `json:"updatedAt" firestore:"updatedAt"`
	UpdatedBy       *string       `json:"updatedBy,omitempty" firestore:"updatedBy,omitempty"`
	DeletedAt       *time.Time    `json:"deletedAt,omitempty" firestore:"deletedAt,omitempty"`
	DeletedBy       *string       `json:"deletedBy,omitempty" firestore:"deletedBy,omitempty"`
}

// Policy (sync with web-app/src/shared/types/account.ts)
var (
	AccountIDPrefix          = "account_"
	DefaultCurrency          = "円"
	MaxCompanyIDLength       = 100
	MaxBrandIDLength         = 100
	MaxStripeAccountIDLength = 255
	MaxBankNameLength        = 50
	MaxBranchNameLength      = 50

	// accountNumber: number (0..99,999,999)
	MinAccountNumber = 0
	MaxAccountNumber = 99_999_999

	// MemberID length limit (adjust as needed to match frontend rules).
	MaxMemberIDLength = 100
)

// Constructors

func New(
	id, companyID, brandID, stripeAccountID string,
	memberID, bankName, branchName string,
	accountNumber int,
	accountType AccountType,
	currency string,
	status AccountStatus,
	createdAt, updatedAt time.Time,
) (Account, error) {
	a := Account{
		ID:              id,
		CompanyID:       companyID,
		BrandID:         brandID,
		StripeAccountID: stripeAccountID,
		MemberID:        memberID,
		BankName:        bankName,
		BranchName:      branchName,
		AccountNumber:   accountNumber,
		AccountType:     accountType,
		Currency:        currency,
		Status:          status,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
	}
	if err := a.validate(); err != nil {
		return Account{}, err
	}
	return a, nil
}

func NewWithNow(
	id, companyID, brandID, stripeAccountID string,
	memberID, bankName, branchName string,
	accountNumber int,
	accountType AccountType,
	currency string,
	status AccountStatus,
	now time.Time,
) (Account, error) {
	return New(
		id,
		companyID,
		brandID,
		stripeAccountID,
		memberID,
		bankName,
		branchName,
		accountNumber,
		accountType,
		currency,
		status,
		now,
		now,
	)
}

// ========================================
// Stripe Connect
// ========================================

// SetStripeAccountID は Stripe Connected Account を Account に紐づけます。
//
// StripeAccountID は Stripe が発行する acct_xxx を想定します。
func (a *Account) SetStripeAccountID(
	stripeAccountID string,
	now time.Time,
	updatedBy *string,
) error {
	stripeAccountID = strings.TrimSpace(stripeAccountID)

	if stripeAccountID == "" ||
		!strings.HasPrefix(stripeAccountID, "acct_") ||
		(MaxStripeAccountIDLength > 0 &&
			len([]rune(stripeAccountID)) > MaxStripeAccountIDLength) {
		return ErrInvalidStripeAccountID
	}

	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}

	a.StripeAccountID = stripeAccountID
	a.UpdatedAt = now
	a.UpdatedBy = updatedBy

	return nil
}

// Activate は Stripe Connect 口座を AMOL 上で有効にします。
func (a *Account) Activate(
	now time.Time,
	updatedBy *string,
) error {
	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}

	a.Status = StatusActive
	a.UpdatedAt = now
	a.UpdatedBy = updatedBy

	return nil
}

// Suspend は Stripe Connect 口座を一時停止します。
func (a *Account) Suspend(
	now time.Time,
	updatedBy *string,
) error {
	if now.IsZero() {
		return ErrInvalidUpdatedAt
	}

	a.Status = StatusSuspended
	a.UpdatedAt = now
	a.UpdatedBy = updatedBy

	return nil
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
	if a.CompanyID == "" ||
		(MaxCompanyIDLength > 0 &&
			len([]rune(a.CompanyID)) > MaxCompanyIDLength) {
		return ErrInvalidCompanyID
	}
	if a.BrandID == "" ||
		(MaxBrandIDLength > 0 &&
			len([]rune(a.BrandID)) > MaxBrandIDLength) {
		return ErrInvalidBrandID
	}

	stripeAccountID := strings.TrimSpace(a.StripeAccountID)
	if stripeAccountID == "" ||
		!strings.HasPrefix(stripeAccountID, "acct_") ||
		(MaxStripeAccountIDLength > 0 &&
			len([]rune(stripeAccountID)) > MaxStripeAccountIDLength) {
		return ErrInvalidStripeAccountID
	}

	if a.MemberID == "" ||
		(MaxMemberIDLength > 0 &&
			len([]rune(a.MemberID)) > MaxMemberIDLength) {
		return ErrInvalidMemberID
	}

	// Stripe Connect の onboarding 中は銀行口座情報を
	// Stripe 側だけが保持しているため、AMOL 側では未取得を許容します。
	if a.BankName != "" &&
		MaxBankNameLength > 0 &&
		len([]rune(a.BankName)) > MaxBankNameLength {
		return ErrInvalidBankName
	}
	if a.BranchName != "" &&
		MaxBranchNameLength > 0 &&
		len([]rune(a.BranchName)) > MaxBranchNameLength {
		return ErrInvalidBranchName
	}
	if a.AccountNumber < MinAccountNumber ||
		a.AccountNumber > MaxAccountNumber {
		return ErrInvalidAccountNumber
	}
	if a.AccountType != "" &&
		!IsValidAccountType(a.AccountType) {
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
	if a.UpdatedAt.IsZero() ||
		a.UpdatedAt.Before(a.CreatedAt) {
		return ErrInvalidUpdatedAt
	}
	if a.DeletedAt != nil &&
		a.DeletedAt.Before(a.CreatedAt) {
		return ErrInvalidUpdatedAt
	}
	return nil
}

// 口座名義（表示用）: MemberID をそのまま利用します。
func (a Account) AccountHolderName() string {
	return a.MemberID
}
