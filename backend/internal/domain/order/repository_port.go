// backend/internal/domain/order/repository_port.go
package order

import (
	"context"
	"errors"
	"time"

	common "narratives/internal/domain/common"
)

// Filter aligns with entity fields (entity.go as source of truth).
//
// NOTE:
// This filter is used by console/query-side read models.
// The main Order repository port intentionally does not expose a generic List method.
// Mall order history must use ListByAvatarID to avoid scanning all orders.
type Filter struct {
	// Exact matches
	ID       string
	UserID   string
	AvatarID string
	CartID   string

	// Order-level state
	Paid *bool

	// Snapshot-based (optional)
	// If nil, it means "no filter".
	ShippingSnapshot      *ShippingSnapshot
	PaymentMethodSnapshot *PaymentMethodSnapshot

	// Item-based filters (optional)
	ModelID      string
	InventoryID  string
	ListID       string
	IsCanceled   *bool
	IsDispatched *bool
	Transferred  *bool

	// Time ranges
	CreatedFrom *time.Time
	CreatedTo   *time.Time
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
	SortByCreatedAt string = "createdAt"
)

// Paging aliases
type Page = common.Page
type PageResult = common.PageResult[Order]

// Update options (optional for adapters)
type UpdateOptions = common.SaveOptions

// Repository defines the persistence port for Order.
type Repository interface {
	// Queries
	GetByID(ctx context.Context, id string) (Order, error)
	ListByAvatarID(ctx context.Context, avatarID string, sort Sort, page Page) (PageResult, error)

	// Commands
	Create(ctx context.Context, o Order) (Order, error)
	Update(ctx context.Context, o Order, opts *UpdateOptions) (Order, error)
}

// Standard repository errors
var (
	ErrNotFound = errors.New("order: not found")
	ErrConflict = errors.New("order: conflict")
)
