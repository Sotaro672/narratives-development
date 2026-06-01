// backend/internal/application/query/console/inventory_query_ports.go
package query

import (
	"context"

	invdom "narratives/internal/domain/inventory"
	pbdom "narratives/internal/domain/productBlueprint"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

type inventoryReader interface {
	ListByProductBlueprintID(ctx context.Context, productBlueprintID string) ([]invdom.Mint, error)
	GetByID(ctx context.Context, inventoryID string) (invdom.Mint, error)
}

type inventoryProductBlueprintReader interface {
	ListByCompanyID(ctx context.Context, companyID string) ([]pbdom.ProductBlueprint, error)
	GetByID(ctx context.Context, id string) (pbdom.ProductBlueprint, error)
}

type inventoryTokenBlueprintReader interface {
	GetByID(ctx context.Context, id string) (*tbdom.TokenBlueprint, error)
}
