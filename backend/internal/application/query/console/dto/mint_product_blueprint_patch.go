// backend/internal/application/query/console/dto/mint_product_blueprint_patch.go
package dto

import pbpdom "narratives/internal/domain/productBlueprint"

type MintProductBlueprintPatchDTO struct {
	pbpdom.Patch
	BrandName string `json:"brandName"`
}
