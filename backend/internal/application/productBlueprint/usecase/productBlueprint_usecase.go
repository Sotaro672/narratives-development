// backend/internal/application/productBlueprint/usecase/product_blueprint_usecase.go
package productBlueprintUsecase

import (
	productbpdom "narratives/internal/domain/productBlueprint"
)

// ProductBlueprintUsecase orchestrates productBlueprint operations.
type ProductBlueprintUsecase struct {
	repo        ProductBlueprintRepo
	historyRepo productbpdom.ProductBlueprintHistoryRepo
}

func NewProductBlueprintUsecase(
	repo ProductBlueprintRepo,
	historyRepo productbpdom.ProductBlueprintHistoryRepo,
) *ProductBlueprintUsecase {
	return &ProductBlueprintUsecase{
		repo:        repo,
		historyRepo: historyRepo,
	}
}
