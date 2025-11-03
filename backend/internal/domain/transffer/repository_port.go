package transffer

import (
	"context"
	"errors"
	"time"
)

// Backward-compat alias for legacy service signatures.
type Transfer = Transffer

// ========================================
// Create/Update inputs (contract only)
// ========================================

type CreateTransfferInput struct {
	MintAddress string    `json:"mintAddress"`
	FromAddress string    `json:"fromAddress"`
	ToAddress   string    `json:"toAddress"`
	RequestedAt time.Time `json:"requestedAt"` // usually now (UTC)
}

type UpdateTransfferInput struct {
	Status        *TransfferStatus    `json:"status,omitempty"`        // requested | fulfilled | error
	ErrorType     *TransfferErrorType `json:"errorType,omitempty"`     // when status=error
	TransfferedAt *time.Time          `json:"transfferedAt,omitempty"` // when status=fulfilled
}

// ========================================
// Query contracts (filters/sort/paging)
// ========================================

type Filter struct {
	// identifiers
	ID          string
	MintAddress string
	FromAddress string
	ToAddress   string

	// status/error
	Statuses   []TransfferStatus
	ErrorTypes []TransfferErrorType
	HasError   *bool // nil: all, true: only errors, false: only non-errors

	// time ranges
	RequestedFrom   *time.Time
	RequestedTo     *time.Time
	TransfferedFrom *time.Time
	TransfferedTo   *time.Time
}

type Sort struct {
	Column SortColumn
	Order  SortOrder
}

type SortColumn string

const (
	SortByRequestedAt   SortColumn = "requestedAt"
	SortByTransfferedAt SortColumn = "transfferedAt"
	SortByStatus        SortColumn = "status"
)

type SortOrder string

const (
	SortAsc  SortOrder = "asc"
	SortDesc SortOrder = "desc"
)

type Page struct {
	Number  int
	PerPage int
}

type PageResult struct {
	Items      []Transffer
	TotalCount int
	TotalPages int
	Page       int
	PerPage    int
}

// ========================================
// Repository Port (interfaces only)
// ========================================

type RepositoryPort interface {
	// Generic queries
	GetByID(ctx context.Context, id string) (*Transffer, error)
	List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult, error)
	Count(ctx context.Context, filter Filter) (int, error)

	// Mutations
	Create(ctx context.Context, in CreateTransfferInput) (*Transffer, error)
	Update(ctx context.Context, id string, in UpdateTransfferInput) (*Transffer, error)
	Delete(ctx context.Context, id string) error

	// Transaction boundary (optional)
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error

	// Maintenance (optional)
	Reset(ctx context.Context) error

	// Convenience methods (for existing service compatibility)
	GetAllTransfers(ctx context.Context) ([]*Transfer, error)
	GetTransferByID(ctx context.Context, id string) (*Transfer, error)
	GetTransfersByFromAddress(ctx context.Context, fromAddress string) ([]*Transfer, error)
	GetTransfersByToAddress(ctx context.Context, toAddress string) ([]*Transfer, error)
	GetTransfersByMintAddress(ctx context.Context, mintAddress string) ([]*Transfer, error)
	GetTransfersByStatus(ctx context.Context, status string) ([]*Transfer, error)
	CreateTransfer(ctx context.Context, in CreateTransfferInput) (*Transfer, error)
	UpdateTransfer(ctx context.Context, id string, in UpdateTransfferInput) (*Transfer, error)
	DeleteTransfer(ctx context.Context, id string) error
	ResetTransfers(ctx context.Context) error
}

// Common repository errors (contract)
var (
	ErrNotFound = errors.New("transffer: not found")
	ErrConflict = errors.New("transffer: conflict")
)
