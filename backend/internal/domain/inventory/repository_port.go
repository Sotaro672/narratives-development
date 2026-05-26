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

	// atomic upsert (for mint -> inventory reflection)
	// - docId = productBlueprintId__tokenBlueprintId
	// - Stock[modelId].Products に productId を追記（UNION / add-only）
	// - Accumulation は Products の件数と整合するように正規化（= len(Products)）
	// - ReservedByOrder / ReservedCount は既存値を維持（この処理では触らない）
	//
	// NOTE:
	// - reserved 系の更新は、競合を避けるためトランザクションで行う専用操作
	//   （例: ReserveByOrder / ReleaseReservationAfterTransfer）に寄せること。
	UpsertByProductBlueprintAndToken(
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
	// Transfer settlement persistence operation
	// ------------------------------------------------------------

	// ReleaseReservationAfterTransfer atomically removes productID from inventory stock
	// and releases the reservation for orderID.
	//
	// Inventory update goal:
	// - Find the inventory document that contains the productID in Stock[*].Products
	// - Remove productID from Stock[modelId].Products
	// - Decrement reservation for orderID:
	//   - If ReservedByOrder[orderID] exists:
	//       - subtract removedCount (usually 1)
	//       - if result <= 0, delete the key
	// - Normalize:
	//   - Stock[modelId].Accumulation = len(Products)
	//   - Stock[modelId].ReservedCount = SUM(ReservedByOrder)
	//
	// Contract:
	// - Must be transactional.
	// - Must be idempotent:
	//   - If productID is not present, do nothing and return removedCount=0, nil.
	// - The repository can resolve inventoryID by scanning inventories,
	//   or you may implement a stronger index later.
	//
	// Params:
	// - productID: product ID to remove from stock
	// - orderID:   order ID whose reservation should be decremented
	// - now:       timestamp for audit/updatedAt
	ReleaseReservationAfterTransfer(
		ctx context.Context,
		productID string,
		orderID string,
		now time.Time,
	) (removedCount int, err error)
}
