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
// Brand revenue snapshot
// ========================================

// BrandRevenueSnapshot identifies the immutable productBlueprint Brand payout
// destination captured when a resale Order item is created.
//
// This snapshot is separate from SellerSnapshot.
//
// For a Resale item:
//   - SellerSnapshot identifies the resale seller Avatar/User/PayoutAccount.
//   - BrandRevenueSnapshot identifies the original productBlueprint Brand
//     Account that receives the Brand share of the resale proceeds.
//
// StripeAccountID is snapshotted at Order creation so later Brand or Account
// changes do not alter the financial destination of an already-created Order.
type BrandRevenueSnapshot struct {
	BrandID         string `json:"brandId"`
	CompanyID       string `json:"companyId"`
	AccountID       string `json:"accountId"`
	StripeAccountID string `json:"stripeAccountId"`
}

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
//   - brandRevenueSnapshot
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

	// SellerSnapshot represents the actual seller of this Order item.
	// For List items this is the primary-sale Brand/Account seller.
	// For Resale items this is the Avatar/User/PayoutAccount resale seller.
	SellerSnapshot SellerSnapshot `json:"sellerSnapshot"`

	// BrandRevenueSnapshot is populated only for Resale items and identifies
	// the productBlueprint Brand Account entitled to the Brand share of resale proceeds.
	BrandRevenueSnapshot BrandRevenueSnapshot `json:"brandRevenueSnapshot,omitempty"`

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
