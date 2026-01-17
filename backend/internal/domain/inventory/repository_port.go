// backend/internal/domain/inventory/repository_port.go
package inventory

import "context"

// RepositoryPort is output port for inventories persistence.
type RepositoryPort interface {
	Create(ctx context.Context, m Mint) (Mint, error)
	GetByID(ctx context.Context, id string) (Mint, error)
	Update(ctx context.Context, m Mint) (Mint, error)
	Delete(ctx context.Context, id string) error

	// Queries
	ListByTokenBlueprintID(ctx context.Context, tokenBlueprintID string) ([]Mint, error)
	ListByProductBlueprintID(ctx context.Context, productBlueprintID string) ([]Mint, error)

	// stock の modelIds 補助フィールドで検索する想定
	ListByModelID(ctx context.Context, modelID string) ([]Mint, error)
	ListByTokenAndModelID(ctx context.Context, tokenBlueprintID, modelID string) ([]Mint, error)

	// atomic upsert (for mint -> inventory reflection)
	// - docId = productBlueprintId__tokenBlueprintId
	// - Stock[modelId].Products に productId を追記（UNION / add-only）
	// - Accumulation は Products の件数と整合するように正規化（= len(Products)）
	// - ReservedByOrder / ReservedCount は既存値を維持（この処理では触らない）
	//
	// NOTE:
	// - reserved 系の更新は、競合を避けるためトランザクションで行う専用操作
	//   （例: ReserveByOrder / UnreserveByOrder）に寄せること。
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
}
