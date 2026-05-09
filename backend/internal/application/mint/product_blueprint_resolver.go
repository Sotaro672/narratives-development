// backend/internal/application/mint/product_blueprint_resolver.go
package mint

import (
	"context"
)

// production から ProductBlueprintID を取り出す。
// prodRepo は GetProductBlueprintIDByProductionID(ctx, productionID) を持つ正規 repository とする。
func (u *MintUsecase) resolveProductBlueprintIDFromProduction(ctx context.Context, productionID string) string {
	if u == nil || u.prodRepo == nil {
		return ""
	}
	if productionID == "" {
		return ""
	}

	productBlueprintID, err := u.prodRepo.GetProductBlueprintIDByProductionID(ctx, productionID)
	if err != nil {
		return ""
	}

	return productBlueprintID
}
