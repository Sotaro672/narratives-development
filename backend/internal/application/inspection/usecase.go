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

func NewInspectionUsecase(
	inspectionRepo inspectiondom.Repository,
	productRepo ProductInspectionRepo,
) *InspectionUsecase {
	return &InspectionUsecase{
		inspectionRepo: inspectionRepo,
		productRepo:    productRepo,
	}
}

func NewInspectionUsecaseWithMint(
	inspectionRepo inspectiondom.Repository,
	productRepo ProductInspectionRepo,
	mintRepo InspectionMintGetter,
) *InspectionUsecase {
	u := NewInspectionUsecase(inspectionRepo, productRepo)
	u.mintRepo = mintRepo
	return u
}

func NewInspectionUsecaseWithModel(
	inspectionRepo inspectiondom.Repository,
	productRepo ProductInspectionRepo,
	modelRepo ModelVariationGetter,
) *InspectionUsecase {
	u := NewInspectionUsecase(inspectionRepo, productRepo)
	u.modelRepo = modelRepo
	return u
}

func NewInspectionUsecaseWithMintAndModel(
	inspectionRepo inspectiondom.Repository,
	productRepo ProductInspectionRepo,
	mintRepo InspectionMintGetter,
	modelRepo ModelVariationGetter,
) *InspectionUsecase {
	u := NewInspectionUsecaseWithMint(inspectionRepo, productRepo, mintRepo)
	u.modelRepo = modelRepo
	return u
}
