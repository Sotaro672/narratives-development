// backend\internal\domain\account\repository_port.go
package account

import (
	"context"
	"errors"
	"time"
)

// Patch (partial update). Nil fields are not updated.
type AccountPatch struct {
	CompanyID       *string
	StripeAccountID *string
	MemberID        *string
	BankName        *string
	BranchName      *string
	AccountNumber   *int
	AccountType     *AccountType
	Currency        *string
	Status          *AccountStatus
	UpdatedBy       *string
	DeletedAt       *time.Time
	DeletedBy       *string
}

// Representative errors (repository-level)
var (
	ErrNotFound = errors.New("account: not found")
	ErrConflict = errors.New("account: conflict")
)

// Repository port (contract)
type Repository interface {
	// ID
	NewID(ctx context.Context) (string, error)

	// Listing
	ListByCompanyID(ctx context.Context, companyID string) ([]Account, error)

	// Read
	GetByID(ctx context.Context, id string) (Account, error)

	// Write
	Create(ctx context.Context, a Account) (Account, error)
	Update(ctx context.Context, id string, patch AccountPatch) (Account, error)
}
