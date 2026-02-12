// backend/internal/application/productBlueprint/usecase/product_blueprint_usecase.go
package productBlueprintUsecase

// ProductBlueprintUsecase orchestrates productBlueprint operations.
type ProductBlueprintUsecase struct {
	repo ProductBlueprintRepo
}

func NewProductBlueprintUsecase(repo ProductBlueprintRepo) *ProductBlueprintUsecase {
	return &ProductBlueprintUsecase{
		repo: repo,
	}
}
