// backend/internal/application/query/console/list_helper.go
package query

import (
	"context"

	pbpdom "narratives/internal/domain/productBlueprint"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

// ============================================================
// Shared Ports (read-only) - used by list detail / list management
// ============================================================

type ProductBlueprintGetter interface {
	GetByID(ctx context.Context, id string) (pbpdom.ProductBlueprint, error)
}

type TokenBlueprintGetter interface {
	GetByID(ctx context.Context, id string) (*tbdom.TokenBlueprint, error)
}
