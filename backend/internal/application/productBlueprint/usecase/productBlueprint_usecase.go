// backend/internal/application/productBlueprint/usecase/usecase.go
package productBlueprintUsecase

// ProductBlueprintUsecase is the application service for productBlueprint.
type ProductBlueprintUsecase struct {
	repo ProductBlueprintRepo

	// ProductBlueprint 起票時に productBlueprintReview 側も初期化するためのポート。
	// NewProductBlueprintUsecase を唯一の入口とし、外から With で差し込まない。
	reviewInit ProductBlueprintReviewInitializer
}

// NewProductBlueprintUsecase creates a ProductBlueprintUsecase.
func NewProductBlueprintUsecase(
	repo ProductBlueprintRepo,
	reviewInit ProductBlueprintReviewInitializer,
) *ProductBlueprintUsecase {
	return &ProductBlueprintUsecase{
		repo:       repo,
		reviewInit: reviewInit,
	}
}
