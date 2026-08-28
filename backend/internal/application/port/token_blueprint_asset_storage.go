// backend/internal/application/port/token_blueprint_asset_storage.go
package port

import "context"

// TokenBlueprintAssetStorage manages external storage assets
// associated with a token blueprint.
type TokenBlueprintAssetStorage interface {
	// Exists reports whether the specified Firebase Storage object exists.
	Exists(ctx context.Context, objectPath string) (bool, error)

	// DeleteAll physically deletes all objects under:
	//
	// token-blueprints/{companyId}/{tokenBlueprintId}/
	//
	// This includes icon and contents assets.
	DeleteAll(
		ctx context.Context,
		companyID string,
		tokenBlueprintID string,
	) error
}
