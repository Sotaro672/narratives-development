// backend/internal/application/port/transfer_order_repository.go
package port

import (
	"context"
	"time"

	orderdom "narratives/internal/domain/order"
)

// FindEligibleTransferItemInput identifies one transfer-eligible order item.
//
// AvatarID is the buyer-side avatar that owns the paid order.
// ProductID, ModelID, and TokenBlueprintID must identify the scanned product
// exactly.
type FindEligibleTransferItemInput struct {
	AvatarID         string
	ProductID        string
	ModelID          string
	TokenBlueprintID string
}

// TransferTargetItem is the minimal order-item state required by the transfer
// application flow.
type TransferTargetItem struct {
	OrderID   string
	ItemIndex int
	ItemType  orderdom.OrderItemType

	InventoryID string
	ModelID     string
	ResaleID    string

	ProductID          string
	ProductBlueprintID string
	TokenBlueprintID   string
	BrandID            string
}

// OrderRepoForTransfer provides exact lookup and state transitions for one
// transfer-eligible order item.
//
// Implementations must use the canonical orderTransferItems read model and
// must not scan Order documents in memory.
type OrderRepoForTransfer interface {
	// FindEligibleTransferItem returns orderdom.ErrNotFound when no paid,
	// untransferred item exactly matches the input.
	//
	// Return-requested items must remain discoverable so a valid scan can
	// record tokenTransferVerifiedAt before token transfer is blocked.
	FindEligibleTransferItem(
		ctx context.Context,
		in FindEligibleTransferItemInput,
	) (TransferTargetItem, error)

	MarkTokenTransferVerified(
		ctx context.Context,
		orderID string,
		itemIndex int,
		at time.Time,
	) (orderdom.OrderItemSnapshot, error)

	LockTransferItem(
		ctx context.Context,
		orderID string,
		itemIndex int,
		now time.Time,
	) error

	UnlockTransferItem(
		ctx context.Context,
		orderID string,
		itemIndex int,
	) error

	MarkTransferredItem(
		ctx context.Context,
		orderID string,
		itemIndex int,
		at time.Time,
	) error
}
