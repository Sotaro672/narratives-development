// backend/internal/domain/order/item.go
package order

import "time"

// ========================================
// Item types
// ========================================

// OrderItemType identifies what kind of item is stored in Order.Items.
type OrderItemType string

type ReturnRequestKind string

const (
	OrderItemTypeList   OrderItemType = "list"
	OrderItemTypeResale OrderItemType = "resale"
)

const (
	ReturnRequestKindUnopened ReturnRequestKind = "unopened"
	ReturnRequestKindOpened   ReturnRequestKind = "opened"
)

// ========================================
// Item snapshot
// ========================================

// OrderItemSnapshot is stored inside Order.Items.
//
// List item:
//   - type: "list"
//   - modelId, inventoryId, listId
//   - productBlueprintId, tokenBlueprintId
//   - sellerSnapshot
//   - productBlueprintCategoryPath, consumptionTaxRate
//   - qty, price
//
// Resale item:
//   - type: "resale"
//   - resaleId, productId
//   - productBlueprintId, tokenBlueprintId, brandId
//   - sellerSnapshot
//   - productBlueprintCategoryPath, consumptionTaxRate
//   - qty=1, price
//
// Token transfer, cancellation, dispatch, and return state is maintained per item.
// Stripe Connect settlement state is maintained separately from this snapshot.
type OrderItemSnapshot struct {
	Type OrderItemType `json:"type"`

	// List item identifiers.
	ModelID     string `json:"modelId,omitempty"`
	InventoryID string `json:"inventoryId,omitempty"`
	ListID      string `json:"listId,omitempty"`

	// Resale item identifier.
	ResaleID string `json:"resaleId,omitempty"`

	// Product identifiers.
	ProductID          string `json:"productId,omitempty"`
	ProductBlueprintID string `json:"productBlueprintId,omitempty"`
	TokenBlueprintID   string `json:"tokenBlueprintId,omitempty"`
	BrandID            string `json:"brandId,omitempty"`

	SellerSnapshot SellerSnapshot `json:"sellerSnapshot"`

	ProductBlueprintCategoryPath []string `json:"productBlueprintCategoryPath"`

	ConsumptionTaxRate int `json:"consumptionTaxRate"`

	Qty   int `json:"qty"`
	Price int `json:"price"`

	IsCancelled  bool `json:"isCancelled"`
	IsDispatched bool `json:"isDispatched"`

	IsReturnRequested bool              `json:"isReturnRequested"`
	ReturnRequestKind ReturnRequestKind `json:"returnRequestKind,omitempty"`
	ReturnRequestedAt *time.Time        `json:"returnRequestedAt,omitempty"`

	IsReturnCompleted bool       `json:"isReturnCompleted"`
	ReturnCompletedAt *time.Time `json:"returnCompletedAt,omitempty"`

	TokenTransferVerifiedAt *time.Time `json:"tokenTransferVerifiedAt,omitempty"`

	Transferred   bool       `json:"transferred"`
	TransferredAt *time.Time `json:"transferredAt,omitempty"`
}
