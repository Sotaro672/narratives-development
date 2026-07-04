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
	ShippingSnapshot *ShippingSnapshot

	// Item-based filters for list items.
	ModelID     string
	InventoryID string
	ListID      string

	// Item-based filters for resale items.
	ItemType           OrderItemType
	ResaleID           string
	ProductID          string
	ProductBlueprintID string
	TokenBlueprintID   string
	BrandID            string

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

// EligibleTransferItem is an order item eligible for token transfer verification.
//
// Common expected source condition:
// - order.avatarId == avatarID
// - order.paid == true
// - item.transferred == false
//
// list item condition:
// - item.type == "list" or legacy empty
// - item.modelId is not empty
// - item.inventoryId is not empty
//
// resale item condition:
// - item.type == "resale"
// - item.resaleId is not empty
// - item.productId is not empty
//
// ItemKey is the stable transfer lock/mark key used by transfer repositories.
// Recommended values:
// - list item:   "list:" + modelId
// - resale item: "resale:" + resaleId
type EligibleTransferItem struct {
	OrderID string

	ItemKey   string
	ItemType  OrderItemType
	ItemIndex int

	// list item identifiers
	ModelID     string
	InventoryID string
	ListID      string

	// resale item identifiers
	ResaleID string

	// product identifiers
	ProductID          string
	ProductBlueprintID string
	TokenBlueprintID   string
	BrandID            string
}

// Repository defines the persistence port for Order.
type Repository interface {
	// Queries
	GetByID(ctx context.Context, id string) (Order, error)
	ListByAvatarID(ctx context.Context, avatarID string, sort Sort, page Page) (PageResult, error)

	// ListTransferredByAvatarIDModelIDAndTransferredAt returns orders that contain
	// transferred list items matching avatarId, modelId, and transferredAt.
	//
	// Expected source condition:
	// - order.avatarId == avatarID
	// - order.paid == true
	// - item.modelId == modelID
	// - item.transferred == true
	// - item.transferredAt == transferredAt
	//
	// Repository implementation is responsible for item-level filtering.
	// Firestore cannot reliably query nested array map fields with this full condition,
	// so Firestore adapter may query by avatarId first and filter items in memory.
	//
	// NOTE:
	// This query is list-item oriented and intentionally remains modelId-based
	// for backward compatibility. Resale transfer history should use productId,
	// resaleId, or transfer records depending on the caller's requirement.
	ListTransferredByAvatarIDModelIDAndTransferredAt(
		ctx context.Context,
		avatarID string,
		modelID string,
		transferredAt time.Time,
		sort Sort,
		page Page,
	) (PageResult, error)

	// Transfer verification query.
	//
	// Implementations should return both list and resale items when eligible.
	// Legacy list items can still be returned with ItemKey empty if the caller
	// does not use this method for locking, but new transfer paths should prefer
	// ItemKey.
	ListEligibleTransferItemsByAvatarID(
		ctx context.Context,
		avatarID string,
	) ([]EligibleTransferItem, error)

	// Commands
	Create(ctx context.Context, o Order) (Order, error)
	Update(ctx context.Context, o Order, opts *UpdateOptions) (Order, error)
}

// Standard repository errors
var (
	ErrNotFound = errors.New("order: not found")
	ErrConflict = errors.New("order: conflict")
)
