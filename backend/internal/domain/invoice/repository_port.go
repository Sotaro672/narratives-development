// backend/internal/domain/invoice/repository_port.go
package invoice

import (
	"context"
	"errors"

	common "narratives/internal/domain/common"
)

// Filter aligns with entity fields (entity.go as source of truth).
type Filter struct {
	// Exact matches
	OrderID string

	// Optional
	// If nil, it means "no filter".
	Paid *bool
}

// Sort uses common.Sort; columns are constrained by constants below.
type Sort = common.Sort
type SortOrder = common.SortOrder

const (
	SortAsc  SortOrder = common.SortAsc
	SortDesc SortOrder = common.SortDesc
)

// Allowed sort columns (entity.go only has OrderID)
const (
	SortByOrderID string = "orderId"
)

// Paging aliases
type Page = common.Page
type PageResult = common.PageResult[Invoice]
type CursorPage = common.CursorPage
type CursorPageResult = common.CursorPageResult[Invoice]

// Save options (optional for adapters)
type SaveOptions = common.SaveOptions

// Repository defines the persistence port for Invoice.
type Repository interface {
	// Queries
	GetByOrderID(ctx context.Context, orderID string) (Invoice, error)
	List(ctx context.Context, filter Filter, sort Sort, page Page) (PageResult, error)
	ListByCursor(ctx context.Context, filter Filter, sort Sort, cpage CursorPage) (CursorPageResult, error)
	Count(ctx context.Context, filter Filter) (int, error)

	// Commands
	Create(ctx context.Context, inv Invoice) (Invoice, error)
	Save(ctx context.Context, inv Invoice, opts *SaveOptions) (Invoice, error)
	DeleteByOrderID(ctx context.Context, orderID string) error

	// Optional (testing/dev)
	Reset(ctx context.Context) error
}

// Standard repository errors
var (
	ErrNotFound = errors.New("invoice: not found")
	ErrConflict = errors.New("invoice: conflict")
)
