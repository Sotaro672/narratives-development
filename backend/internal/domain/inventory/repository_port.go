// backend/internal/domain/inventory/repository_port.go
package inventory

import (
	"context"
	"time"
)

// RepositoryPort is output port for inventories persistence.
type RepositoryPort interface {
	GetByID(ctx context.Context, id string) (Mint, error)

	// Queries
	ListByProductBlueprintID(ctx context.Context, productBlueprintID string) ([]Mint, error)

	// ------------------------------------------------------------
	// inventoryId -> (productBlueprintId, tokenBlueprintId)
	// ------------------------------------------------------------
	//
	// ResolveBlueprintIDsByInventoryID returns the pair of blueprint IDs for a given inventory document ID.
	//
	// Expected behavior:
	// - If the inventory does not exist: return ErrNotFound
	// - If inventoryID is empty/invalid: return ErrInvalidMintID
	// - Otherwise: return (productBlueprintID, tokenBlueprintID, nil)
	//
	// NOTE:
	// - Implementation may parse inventoryID if it follows BuildMintID convention (productBlueprintId__tokenBlueprintId),
	//   but it MUST be safe and correct even if the ID format changes; therefore reading the document is acceptable.
	ResolveBlueprintIDsByInventoryID(
		ctx context.Context,
		inventoryID string,
	) (productBlueprintID string, tokenBlueprintID string, err error)

	// ------------------------------------------------------------
	// ShippingAddress assignment
	// ------------------------------------------------------------
	//
	// SetShippingAddressID sets the shippingAddress document ID
	// used as the inventory storage location.
	//
	// Contract:
	// - inventoryID must identify an existing inventory.
	// - shippingAddressID must not be empty.
	// - This operation updates only shippingAddressId and updatedAt.
	// - ShippingAddress existence and company ownership validation
	//   are responsibilities of the application/usecase layer.
	// - If the inventory does not exist: return ErrNotFound.
	// - If inventoryID is empty/invalid: return ErrInvalidMintID.
	SetShippingAddressID(
		ctx context.Context,
		inventoryID string,
		shippingAddressID string,
		now time.Time,
	) error

	// ClearShippingAddressIDByShippingAddressID clears the storage location
	// from every inventory that currently references shippingAddressID.
	//
	// Contract:
	// - shippingAddressID must not be empty.
	// - Only inventories whose shippingAddressId equals shippingAddressID are updated.
	// - shippingAddressId is removed/unset and updatedAt is updated.
	// - Inventory documents themselves must not be deleted.
	// - Inventories that reference another shippingAddressID must not be changed.
	// - If no inventory references shippingAddressID, return nil.
	// - The persistence implementation should perform the updates safely for
	//   all matching inventory documents.
	ClearShippingAddressIDByShippingAddressID(
		ctx context.Context,
		shippingAddressID string,
		now time.Time,
	) error

	// ------------------------------------------------------------
	// Transportation assignment
	// ------------------------------------------------------------
	//
	// SetTransportation sets the transportation configuration
	// used when shipping products from the inventory.
	//
	// Contract:
	// - inventoryID must identify an existing inventory.
	// - transportationOption must be a valid TransportationOption.
	// - transportationID is required only when transportationOption is custom.
	// - transportationID must be empty for yamato / sagawa / post.
	// - This operation updates only transportationOption,
	//   transportationId and updatedAt.
	// - TransportationFeeSetting existence and company ownership validation
	//   are responsibilities of the application/usecase layer.
	// - If the inventory does not exist: return ErrNotFound.
	// - If inventoryID is empty/invalid: return ErrInvalidMintID.
	SetTransportation(
		ctx context.Context,
		inventoryID string,
		transportationOption TransportationOption,
		transportationID string,
		now time.Time,
	) error

	// atomic upsert (for mint -> inventory reflection)
	// - docId = productBlueprintId__tokenBlueprintId
	// - Stock[modelId].Products に productId を追記（UNION / add-only）
	// - Accumulation は Products の件数と整合するように正規化（= len(Products)）
	// - ReservedByOrder / ReservedCount は既存値を維持（この処理では触らない）
	//
	// NOTE:
	// - reserved 系の更新は、競合を避けるためトランザクションで行う専用操作
	//   （例: ReserveByOrder / ReleaseReservationByOrder / ReleaseReservationAfterTransfer）に寄せること。
	UpsertByModelAndToken(
		ctx context.Context,
		tokenBlueprintID string,
		productBlueprintID string,
		modelID string,
		productIDs []string,
	) (Mint, error)

	// ReserveByOrder atomically updates reservation fields for a given model in an inventory document.
	// - Stock[modelId].ReservedByOrder[orderId] = qty (set/overwrite; idempotent)
	// - ReservedCount is normalized as SUM(ReservedByOrder)
	//
	// NOTE:
	// - This operation must be transactional to avoid lost updates with concurrent upserts.
	ReserveByOrder(
		ctx context.Context,
		inventoryID string,
		modelID string,
		orderID string,
		qty int,
	) error

	// ------------------------------------------------------------
	// Order cancellation reservation release
	// ------------------------------------------------------------

	// ReleaseReservationByOrder atomically releases the reservation for orderID
	// without removing any product from physical inventory.
	//
	// Inventory update goal:
	// - Use inventoryID and modelID from the canceled order item.
	// - Delete Stock[modelID].ReservedByOrder[orderID].
	// - Do not modify Stock[modelID].Products.
	// - Do not modify physical inventory quantity.
	// - Normalize:
	//   - Stock[modelID].Accumulation = len(Products)
	//   - Stock[modelID].ReservedCount = SUM(ReservedByOrder)
	//
	// Contract:
	// - Must be transactional.
	// - Must be idempotent:
	//   - If ReservedByOrder[orderID] does not exist, do nothing and return nil.
	// - Must not remove product IDs from Products.
	// - Must not scan inventories by orderID.
	//
	// Params:
	// - inventoryID: inventory document ID from order item
	// - modelID:     model ID from order item
	// - orderID:     canceled order ID whose reservation should be removed
	// - now:         timestamp for audit/updatedAt
	ReleaseReservationByOrder(
		ctx context.Context,
		inventoryID string,
		modelID string,
		orderID string,
		now time.Time,
	) error

	// ------------------------------------------------------------
	// Transfer settlement persistence operation
	// ------------------------------------------------------------

	// ReleaseReservationAfterTransfer atomically removes productID from the specified inventory stock
	// and releases the reservation for orderID.
	//
	// Inventory update goal:
	// - Use inventoryID and modelID from order item reservation detail.
	// - Remove productID from Stock[modelID].Products.
	// - Decrement reservation for orderID:
	//   - If ReservedByOrder[orderID] exists:
	//       - subtract removedCount, usually 1
	//       - if result <= 0, delete the key
	// - Normalize:
	//   - Stock[modelID].Accumulation = len(Products)
	//   - Stock[modelID].ReservedCount = SUM(ReservedByOrder)
	//
	// Contract:
	// - Must be transactional.
	// - Must be idempotent:
	//   - If productID is not present in Stock[modelID].Products, do nothing and return removedCount=0, nil.
	// - Must not scan inventories by productID.
	//
	// Params:
	// - inventoryID: inventory document ID from order item
	// - modelID:     model ID from order item
	// - productID:   product ID to remove from stock
	// - orderID:     order ID whose reservation should be decremented
	// - now:         timestamp for audit/updatedAt
	ReleaseReservationAfterTransfer(
		ctx context.Context,
		inventoryID string,
		modelID string,
		productID string,
		orderID string,
		now time.Time,
	) (removedCount int, err error)
}
