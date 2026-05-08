// backend/internal/application/inspection/usecase.go
package inspection

import (
	inspectiondom "narratives/internal/domain/inspection"
)

// ------------------------------------------------------------
// Usecase
// ------------------------------------------------------------

type InspectionUsecase struct {
	inspectionRepo inspectiondom.Repository
	productRepo    ProductInspectionRepo
	mintRepo       InspectionMintGetter // nil 許容
	modelRepo      ModelVariationGetter // nil 許容
}

// NewInspectionUsecase を唯一の出入り口にするため、必要な依存はすべてここで受け取る。
// mintRepo / modelRepo は不要なら nil を渡せる。
func NewInspectionUsecase(
	inspectionRepo inspectiondom.Repository,
	productRepo ProductInspectionRepo,
	mintRepo InspectionMintGetter, // nil 許容
	modelRepo ModelVariationGetter, // nil 許容
) *InspectionUsecase {
	return &InspectionUsecase{
		inspectionRepo: inspectionRepo,
		productRepo:    productRepo,
		mintRepo:       mintRepo,
		modelRepo:      modelRepo,
	}
}
