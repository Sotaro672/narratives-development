// backend/internal/application/productBlueprint/usecase/usecase.go
package productBlueprintUsecase

// ProductBlueprintUsecase is the application service for productBlueprint.
type ProductBlueprintUsecase struct {
	repo ProductBlueprintRepo

	// ✅ ProductBlueprint 起票時に productBlueprintReview 側も初期化するためのポート
	// NOTE: NewProductBlueprintUsecase が唯一の入口となるよう、外から With で差し込まない。
	reviewInit ProductBlueprintReviewInitializer
}

// ✅ NewProductBlueprintUsecase を唯一の出入り口にするため、reviewInit をコンストラクタ引数にする。
// - reviewInit が nil の場合は初期化をスキップ（既存互換）
// - 「必ず作りたい」場合は DI 側で non-nil を渡す
func NewProductBlueprintUsecase(
	repo ProductBlueprintRepo,
	reviewInit ProductBlueprintReviewInitializer,
) *ProductBlueprintUsecase {
	return &ProductBlueprintUsecase{
		repo:       repo,
		reviewInit: reviewInit,
	}
}
