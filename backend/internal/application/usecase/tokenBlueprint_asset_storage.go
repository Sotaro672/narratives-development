// backend/internal/application/usecase/tokenBlueprint_asset_storage.go
package usecase

import "context"

// TokenBlueprintAssetStorage manages external storage assets
// associated with a token blueprint.
type TokenBlueprintAssetStorage interface {
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
