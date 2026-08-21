// backend/internal/application/query/console/inventory_query_ports.go
package query

import (
	"context"

	invdom "narratives/internal/domain/inventory"
)

type inventoryReader interface {
	ListByProductBlueprintID(ctx context.Context, productBlueprintID string) ([]invdom.Mint, error)
	GetByID(ctx context.Context, inventoryID string) (invdom.Mint, error)
}
