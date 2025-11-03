package transaction

import (
	"context"
	"errors"
	"time"
)

// Error contracts
var (
	ErrNotFound = errors.New("transaction: not found")
	ErrConflict = errors.New("transaction: conflict")
)

// DTO for API boundary (timestamp is ISO8601 string)
type TransactionDTO struct {
	ID          string          `json:"id"`
	AccountID   string          `json:"accountId"`
	BrandName   string          `json:"brandName"`
	Type        TransactionType `json:"type"`
	Amount      int             `json:"amount"`
	Currency    string          `json:"currency"`
	FromAccount string          `json:"fromAccount"`
	ToAccount   string          `json:"toAccount"`
	Timestamp   string          `json:"timestamp"` // ISO8601
	Description string          `json:"description"`
}

// Use case inputs

type CreateTransactionInput struct {
	AccountID   string          `json:"accountId"`
	BrandName   string          `json:"brandName"`
	Type        TransactionType `json:"type"`
	Amount      int             `json:"amount"`
	Currency    string          `json:"currency"`
	FromAccount string          `json:"fromAccount"`
	ToAccount   string          `json:"toAccount"`
	Timestamp   time.Time       `json:"timestamp"`
	Description string          `json:"description"`
}

type UpdateTransactionInput struct {
	AccountID   *string          `json:"accountId,omitempty"`
	BrandName   *string          `json:"brandName,omitempty"`
	Type        *TransactionType `json:"type,omitempty"`
	Amount      *int             `json:"amount,omitempty"`
	Currency    *string          `json:"currency,omitempty"`
	FromAccount *string          `json:"fromAccount,omitempty"`
	ToAccount   *string          `json:"toAccount,omitempty"`
	Timestamp   *time.Time       `json:"timestamp,omitempty"`
	Description *string          `json:"description,omitempty"`
}

// Search contracts

type TransactionFilters struct {
	AccountIDs     []string
	Brands         []string
	Types          []TransactionType
	Currencies     []string
	FromAccounts   []string
	ToAccounts     []string
	DescriptionLike string

	DateFrom  *time.Time
	DateTo    *time.Time
	AmountMin *int
	AmountMax *int
}

type TransactionSortColumn string

const (
	SortByDate      TransactionSortColumn = "timestamp"
	SortByAmount    TransactionSortColumn = "amount"
	SortByBrandName TransactionSortColumn = "brandName"
	SortByAccountID TransactionSortColumn = "accountId"
)

type SortOrder string

const (
	SortAsc  SortOrder = "asc"
	SortDesc SortOrder = "desc"
)

type TransactionSort struct {
	Column TransactionSortColumn
	Order  SortOrder // asc | desc
}

type TransactionPagination struct {
	Page     int
	PerPage  int
}

type TransactionSearchCriteria struct {
	SearchTerm string
	Filters    TransactionFilters
	Sort       TransactionSort
	Pagination *TransactionPagination
}

// Repository Port

type RepositoryPort interface {
	// Reads
	GetAllTransactions(ctx context.Context) ([]*Transaction, error)
	GetTransactionByID(ctx context.Context, id string) (*Transaction, error)
	GetTransactionsByBrand(ctx context.Context, brandName string) ([]*Transaction, error)
	GetTransactionsByAccount(ctx context.Context, accountID string) ([]*Transaction, error)

	// Search
	SearchTransactions(ctx context.Context, criteria TransactionSearchCriteria) (txs []*Transaction, total int, err error)

	// Mutations
	CreateTransaction(ctx context.Context, in CreateTransactionInput) (*Transaction, error)
	UpdateTransaction(ctx context.Context, id string, in UpdateTransactionInput) (*Transaction, error)
	DeleteTransaction(ctx context.Context, id string) error

	// Maintenance
	ResetTransactions(ctx context.Context) error

	// Optional transaction boundary
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}
