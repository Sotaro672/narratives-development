package account

import (
    "context"
    "errors"
    "time"

    common "narratives/internal/domain/common"
)

// Patch (partial update). Nil fields are not updated.
type AccountPatch struct {
    BankName      *string
    BranchName    *string
    AccountNumber *int
    AccountType   *AccountType
    Currency      *string
    Status        *AccountStatus
    UpdatedBy     *string
    DeletedAt     *time.Time
    DeletedBy     *string
}

// Filter for listing/searching accounts.
type Filter struct {
    // Free-text search on id, bankName, branchName, currency
    SearchQuery string

    // Exact-match filters
    MemberID *string
    Currency *string

    // Enum filters
    Statuses []AccountStatus
    Types    []AccountType

    // Ranges
    AccountNumberMin *int
    AccountNumberMax *int

    // Time ranges
    CreatedFrom *time.Time
    CreatedTo   *time.Time
    UpdatedFrom *time.Time
    UpdatedTo   *time.Time

    // Soft delete tri-state:
    // nil: all, true: only deleted (DeletedAt IS NOT NULL), false: only active (DeletedAt IS NULL)
    Deleted *bool
}

// Sort options (use with common.Sort)
type SortColumn string

const (
    SortByCreatedAt    SortColumn = "createdAt"
    SortByUpdatedAt    SortColumn = "updatedAt"
    SortByBankName     SortColumn = "bankName"
    SortByAccountNumber SortColumn = "accountNumber"
)

// Common aliases (domain-only, infra-agnostic)
type Sort = common.Sort
type SortOrder = common.SortOrder
type Page = common.Page
type PageResult[T any] = common.PageResult[T]
type CursorPage = common.CursorPage
type CursorPageResult[T any] = common.CursorPageResult[T]
type SaveOptions = common.SaveOptions

const (
    SortAsc  = common.SortAsc
    SortDesc = common.SortDesc
)

// Representative errors (repository-level)
var (
    ErrNotFound = errors.New("account: not found")
    ErrConflict = errors.New("account: conflict")
)

// Repository port (contract)
type Repository interface {
    // Listing
    List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult[Account], error)
    ListByCursor(ctx context.Context, filter Filter, sort Sort, cpage CursorPage) (CursorPageResult[Account], error)

    // Read
    GetByID(ctx context.Context, id string) (Account, error)
    Exists(ctx context.Context, id string) (bool, error)
    Count(ctx context.Context, filter Filter) (int, error)

    // Write
    Create(ctx context.Context, a Account) (Account, error)
    Update(ctx context.Context, id string, patch AccountPatch) (Account, error)
    Delete(ctx context.Context, id string) error

    // Optional upsert/save
    Save(ctx context.Context, a Account, opts *SaveOptions) (Account, error)
}
