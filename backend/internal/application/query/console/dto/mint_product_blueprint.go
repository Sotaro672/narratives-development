// backend/internal/application/query/console/dto/mint_product_blueprint.go
package dto

import pbpdom "narratives/internal/domain/productBlueprint"

type MintProductBlueprintDTO struct {
	pbpdom.ProductBlueprint
	BrandName string `json:"brandName"`
}
