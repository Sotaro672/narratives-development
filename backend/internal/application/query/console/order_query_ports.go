// backend/internal/application/query/console/order_query_ports.go
package query

import (
	"context"

	resolver "narratives/internal/application/resolver"
	avatardom "narratives/internal/domain/avatar"
)

// ============================================================
// Order shared ports
// ============================================================

// InventoryBlueprintResolver resolves productBlueprintId/tokenBlueprintId
// from inventoryId.
type InventoryBlueprintResolver interface {
	ResolveBlueprintIDsByInventoryID(
		ctx context.Context,
		inventoryID string,
	) (
		productBlueprintID string,
		tokenBlueprintID string,
		err error,
	)
}

// ListReadableIDReader resolves listId to readableId.
type ListReadableIDReader interface {
	GetReadableIDByID(
		ctx context.Context,
		id string,
	) (string, error)
}

// AvatarGetter resolves avatarId to Avatar.
type AvatarGetter interface {
	GetByID(
		ctx context.Context,
		id string,
	) (avatardom.Avatar, error)
}

// ModelResolver resolves modelId(variationID) to display fields.
type ModelResolver interface {
	ResolveModelResolved(
		ctx context.Context,
		variationID string,
	) resolver.ModelResolved
}
