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
// This filter is used by console/query-side read models.
// The main Order repository port intentionally does not expose a generic List
// method. Mall order history must use ListByAvatarID.
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

// EligibleTransferItem is the canonical orderTransferItems read model returned
// for an order item that is currently eligible for token transfer.
//
// Source condition:
// - avatarId matches the requested avatar
// - paid == true
// - transferred == false
//
// Identity is always orderId + itemIndex. String-derived item keys are not
// supported.
type EligibleTransferItem struct {
	OrderID   string
	ItemType  OrderItemType
	ItemIndex int

	// list item identifiers
	ModelID     string
	InventoryID string
	ListID      string

	// resale item identifier
	ResaleID string

	// canonical product identifiers
	ProductID          string
	ProductBlueprintID string
	TokenBlueprintID   string
	BrandID            string
}

// Validate verifies that the projection item contains the canonical required
// identifiers for its explicit item type.
func (i EligibleTransferItem) Validate() error {
	if i.OrderID == "" || i.ItemIndex < 0 {
		return ErrInvalidItemSnapshot
	}

	switch i.ItemType {
	case OrderItemTypeList:
		if i.ModelID == "" ||
			i.InventoryID == "" ||
			i.ListID == "" ||
			i.ProductBlueprintID == "" ||
			i.TokenBlueprintID == "" {
			return ErrInvalidItemSnapshot
		}

	case OrderItemTypeResale:
		if i.ResaleID == "" ||
			i.ProductID == "" ||
			i.ProductBlueprintID == "" ||
			i.TokenBlueprintID == "" ||
			i.BrandID == "" {
			return ErrInvalidItemSnapshot
		}

	default:
		return ErrInvalidItemSnapshot
	}

	return nil
}

// Repository defines persistence for the Order aggregate.
//
// Create and Update must persist the Order and replace all corresponding
// orderTransferItems projection records atomically. A failed projection write
// must roll back the Order write, and a failed Order write must not change the
// projection.
type Repository interface {
	// Queries
	GetByID(
		ctx context.Context,
		id string,
	) (Order, error)

	ListByAvatarID(
		ctx context.Context,
		avatarID string,
		sort Sort,
		page Page,
	) (PageResult, error)

	// Commands
	Create(
		ctx context.Context,
		o Order,
	) (Order, error)

	Update(
		ctx context.Context,
		o Order,
		opts *UpdateOptions,
	) (Order, error)
}

// Standard repository errors
var (
	ErrNotFound = errors.New("order: not found")
	ErrConflict = errors.New("order: conflict")
)
