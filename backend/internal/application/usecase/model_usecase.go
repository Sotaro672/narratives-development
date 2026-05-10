// backend\internal\application\usecase\model_usecase.go
package usecase

import (
	"context"
	"time"

	modeldom "narratives/internal/domain/model"
)

// ------------------------------------------------------------
// ModelUsecase
// ------------------------------------------------------------
//
// ✅ Mall handler から model.RepositoryPort として渡したいので、
//   ModelUsecase 自体が modeldom.RepositoryPort を実装する（= repo に委譲する）
// ------------------------------------------------------------

type ModelUsecase struct {
	repo modeldom.RepositoryPort
}

func NewModelUsecase(repo modeldom.RepositoryPort) *ModelUsecase {
	return &ModelUsecase{
		repo: repo,
	}
}

// ------------------------------------------------------------
// Queries
// ------------------------------------------------------------

// ★ HTTP の GET /models/variations/{variationId} 用の明示メソッド
// ※ このメソッドは 1 箇所のみ定義（DuplicateMethod 回避）
func (u *ModelUsecase) GetModelVariationByID(ctx context.Context, variationID string) (*modeldom.ModelVariation, error) {
	if u.repo == nil {
		return nil, modeldom.ErrNotFound
	}
	if variationID == "" {
		return nil, modeldom.ErrInvalidID
	}
	return u.repo.GetModelVariationByID(ctx, variationID)
}

// （呼び出し側は GetModelVariations(ctx, productBlueprintID) を利用）

// ------------------------------------------------------------
// RepositoryPort implementation (delegate to u.repo)
// ------------------------------------------------------------

func (u *ModelUsecase) GetModelData(ctx context.Context, productID string) (*modeldom.ModelData, error) {
	if u.repo == nil {
		return nil, modeldom.ErrNotFound
	}
	if productID == "" {
		return nil, modeldom.ErrInvalidProductID
	}
	return u.repo.GetModelData(ctx, productID)
}

func (u *ModelUsecase) GetModelDataByBlueprintID(ctx context.Context, productBlueprintID string) (*modeldom.ModelData, error) {
	if u.repo == nil {
		return nil, modeldom.ErrNotFound
	}
	if productBlueprintID == "" {
		return nil, modeldom.ErrInvalidBlueprintID
	}
	return u.repo.GetModelDataByBlueprintID(ctx, productBlueprintID)
}

func (u *ModelUsecase) UpdateModelData(ctx context.Context, productID string, updates modeldom.ModelDataUpdate) (*modeldom.ModelData, error) {
	if u.repo == nil {
		return nil, modeldom.ErrNotFound
	}
	if productID == "" {
		return nil, modeldom.ErrInvalidProductID
	}
	return u.repo.UpdateModelData(ctx, productID, updates)
}

func (u *ModelUsecase) ListVariations(ctx context.Context, filter modeldom.VariationFilter, page modeldom.Page) (modeldom.VariationPageResult, error) {
	if u.repo == nil {
		return modeldom.VariationPageResult{}, modeldom.ErrNotFound
	}
	return u.repo.ListVariations(ctx, filter, page)
}

func (u *ModelUsecase) GetModelVariations(ctx context.Context, productID string) ([]modeldom.ModelVariation, error) {
	if u.repo == nil {
		return nil, modeldom.ErrNotFound
	}
	if productID == "" {
		return nil, modeldom.ErrInvalidProductID
	}
	return u.repo.GetModelVariations(ctx, productID)
}

// Create ModelVariation（履歴は保存しない）
func (u *ModelUsecase) CreateModelVariation(ctx context.Context, v modeldom.NewModelVariation) (*modeldom.ModelVariation, error) {
	if u.repo == nil {
		return nil, modeldom.ErrNotFound
	}

	created, err := u.repo.CreateModelVariation(ctx, v)
	if err != nil {
		return nil, err
	}

	return created, nil
}

// Update ModelVariation（履歴は保存しない）
func (u *ModelUsecase) UpdateModelVariation(ctx context.Context, variationID string, updates modeldom.ModelVariationUpdate) (*modeldom.ModelVariation, error) {
	if u.repo == nil {
		return nil, modeldom.ErrNotFound
	}

	if variationID == "" {
		return nil, modeldom.ErrInvalidID
	}

	updated, err := u.repo.UpdateModelVariation(ctx, variationID, updates)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

func (u *ModelUsecase) DeleteModelVariation(ctx context.Context, variationID string) (*modeldom.ModelVariation, error) {
	if u.repo == nil {
		return nil, modeldom.ErrNotFound
	}

	if variationID == "" {
		return nil, modeldom.ErrInvalidID
	}

	return u.repo.DeleteModelVariation(ctx, variationID)
}

func (u *ModelUsecase) ReplaceModelVariations(ctx context.Context, vars []modeldom.NewModelVariation) ([]modeldom.ModelVariation, error) {
	if u.repo == nil {
		return nil, modeldom.ErrNotFound
	}

	updated, err := u.repo.ReplaceModelVariations(ctx, vars)
	if err != nil {
		return nil, err
	}

	// ReplaceVariations 後は UpdatedAt の補正のみ（version は扱わない）
	now := time.Now().UTC()
	for i := range updated {
		updated[i].UpdatedAt = now
	}

	return updated, nil
}

func (u *ModelUsecase) GetSizeVariations(ctx context.Context, productID string) ([]modeldom.SizeVariation, error) {
	if u.repo == nil {
		return nil, modeldom.ErrNotFound
	}
	if productID == "" {
		return nil, modeldom.ErrInvalidProductID
	}
	return u.repo.GetSizeVariations(ctx, productID)
}

func (u *ModelUsecase) GetModelNumbers(ctx context.Context, productID string) ([]modeldom.ModelNumber, error) {
	if u.repo == nil {
		return nil, modeldom.ErrNotFound
	}
	if productID == "" {
		return nil, modeldom.ErrInvalidProductID
	}
	return u.repo.GetModelNumbers(ctx, productID)
}
