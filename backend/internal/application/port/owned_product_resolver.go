// backend/internal/application/port/owned_product_resolver.go
package port

import "context"

// OwnedProductResolver determines whether an avatar currently owns a token
// belonging to the specified product blueprint.
type OwnedProductResolver interface {
	HasOwnedProductBlueprint(
		ctx context.Context,
		avatarID string,
		productBlueprintID string,
	) (bool, error)
}
