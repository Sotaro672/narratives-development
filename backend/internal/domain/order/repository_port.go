package order

import (
	"context"
	"errors"
	"time"

	common "narratives/internal/domain/common"
)

// Filter aligns with entity fields (no LastUpdate).
type Filter struct {
	// Exact matches
	ID     string
	UserID string
	CartID string

	// ✅ Snapshot-based (replaces ShippingAddressID / BillingAddressID)
	// If nil, it means "no filter".
	// NOTE: Adapters may choose to support only some fields (e.g., BillingSnapshot.Last4 only).
	ShippingSnapshot *ShippingSnapshot
	BillingSnapshot  *BillingSnapshot

	ListID    string
	InvoiceID string
	PaymentID string

	// Time ranges
	CreatedFrom    *time.Time
	CreatedTo      *time.Time
	UpdatedFrom    *time.Time
	UpdatedTo      *time.Time
	TransferedFrom *time.Time // spelling per TS
	TransferedTo   *time.Time // spelling per TS
}

// Sort uses common.Sort; columns are constrained by constants below.
type Sort = common.Sort
type SortOrder = common.SortOrder

const (
	SortAsc  SortOrder = common.SortAsc
	SortDesc SortOrder = common.SortDesc
)

// Allowed sort columns
const (
	SortByCreatedAt      string = "createdAt"
	SortByUpdatedAt      string = "updatedAt"
	SortByTransferedDate string = "transferedDate" // spelling per TS
)

// Paging aliases
type Page = common.Page
type PageResult = common.PageResult[Order]
type CursorPage = common.CursorPage
type CursorPageResult = common.CursorPageResult[Order]

// Save options (optional for adapters)
type SaveOptions = common.SaveOptions

// Repository defines the persistence port for Order.
type Repository interface {
	// Queries
	GetByID(ctx context.Context, id string) (Order, error)
	List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult, error)
	ListByCursor(ctx context.Context, filter Filter, sort Sort, cpage CursorPage) (CursorPageResult, error)
	Count(ctx context.Context, filter Filter) (int, error)

	// Commands
	Create(ctx context.Context, o Order) (Order, error)
	Save(ctx context.Context, o Order, opts *SaveOptions) (Order, error)
	Delete(ctx context.Context, id string) error

	// Optional (testing/dev)
	Reset(ctx context.Context) error
}

// Standard repository errors
var (
	ErrNotFound = errors.New("order: not found")
	ErrConflict = errors.New("order: conflict")
)
