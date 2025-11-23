package usecase

import (
	"context"
	"strings"

	modeldom "narratives/internal/domain/model"
)

// ------------------------------------------------------------
// ModelRepo
// ------------------------------------------------------------

type ModelRepo interface {
	GetModelData(ctx context.Context, productID string) (*modeldom.ModelData, error)
	GetModelDataByBlueprintID(ctx context.Context, blueprintID string) (*modeldom.ModelData, error)
	UpdateModelData(ctx context.Context, productID string, updates modeldom.ModelDataUpdate) (*modeldom.ModelData, error)

	GetModelVariationByID(ctx context.Context, variationID string) (*modeldom.ModelVariation, error)
	CreateModelVariation(ctx context.Context, productID string, variation modeldom.NewModelVariation) (*modeldom.ModelVariation, error)
	UpdateModelVariation(ctx context.Context, variationID string, updates modeldom.ModelVariationUpdate) (*modeldom.ModelVariation, error)
	DeleteModelVariation(ctx context.Context, variationID string) (*modeldom.ModelVariation, error)

	ReplaceModelVariations(ctx context.Context, productID string, vars []modeldom.NewModelVariation) ([]modeldom.ModelVariation, error)
}

// ------------------------------------------------------------
// ModelUsecase
// ------------------------------------------------------------

type ModelUsecase struct {
	repo ModelRepo
}

func NewModelUsecase(repo ModelRepo) *ModelUsecase {
	return &ModelUsecase{repo: repo}
}

// ------------------------------------------------------------
// Queries
// ------------------------------------------------------------

// GetByID は /models/{id} から呼ばれる ID 指定の単一取得。
// Firestore では GetModelData が productID ベースの単一取得なので、それをそのまま使う。
func (u *ModelUsecase) GetByID(ctx context.Context, id string) (*modeldom.ModelData, error) {
	id = strings.TrimSpace(id)
	return u.repo.GetModelData(ctx, id)
}

func (u *ModelUsecase) GetModelData(ctx context.Context, productID string) (*modeldom.ModelData, error) {
	return u.repo.GetModelData(ctx, productID)
}

func (u *ModelUsecase) GetModelDataByBlueprintID(ctx context.Context, blueprintID string) (*modeldom.ModelData, error) {
	return u.repo.GetModelDataByBlueprintID(ctx, blueprintID)
}

// ------------------------------------------------------------
// Commands
// ------------------------------------------------------------

func (u *ModelUsecase) UpdateModelData(ctx context.Context, productID string, updates modeldom.ModelDataUpdate) (*modeldom.ModelData, error) {
	return u.repo.UpdateModelData(ctx, productID, updates)
}

func (u *ModelUsecase) CreateModelVariation(ctx context.Context, productID string, v modeldom.NewModelVariation) (*modeldom.ModelVariation, error) {
	return u.repo.CreateModelVariation(ctx, productID, v)
}

func (u *ModelUsecase) UpdateModelVariation(ctx context.Context, variationID string, updates modeldom.ModelVariationUpdate) (*modeldom.ModelVariation, error) {
	return u.repo.UpdateModelVariation(ctx, variationID, updates)
}

func (u *ModelUsecase) DeleteModelVariation(ctx context.Context, variationID string) (*modeldom.ModelVariation, error) {
	return u.repo.DeleteModelVariation(ctx, variationID)
}

func (u *ModelUsecase) ReplaceModelVariations(ctx context.Context, productID string, vars []modeldom.NewModelVariation) ([]modeldom.ModelVariation, error) {
	return u.repo.ReplaceModelVariations(ctx, productID, vars)
}
