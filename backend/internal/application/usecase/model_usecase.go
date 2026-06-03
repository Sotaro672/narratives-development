// backend/internal/application/usecase/model_usecase.go
package usecase

import (
	"context"

	modeldom "narratives/internal/domain/model"
)

// ------------------------------------------------------------
// ModelUsecase
// ------------------------------------------------------------
//
// Mall handler から model.RepositoryPort として渡したいので、
// ModelUsecase 自体が modeldom.RepositoryPort を実装する（= repo に委譲する）。
//
// NOTE:
// Product-level metadata は productBlueprint.CategoryFields に集約する方針。
//
// この usecase は category-specific model variation の操作を担当する。
// apparel では size / color / measurements を扱う。
// alcohol では volume のみを扱う。
// どの category で model variation を作成するかは、
// productBlueprintCategory/input_schema.go の schema を application/usecase 側で参照して判断する。
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

// HTTP の GET /models/variations/{variationId} 用の明示メソッド。
// このメソッドは 1 箇所のみ定義（DuplicateMethod 回避）。
func (u *ModelUsecase) GetModelVariationByID(
	ctx context.Context,
	variationID string,
) (modeldom.ModelVariation, error) {
	if u.repo == nil {
		return nil, modeldom.ErrNotFound
	}
	if variationID == "" {
		return nil, modeldom.ErrInvalidID
	}

	return u.repo.GetModelVariationByID(ctx, variationID)
}

// ------------------------------------------------------------
// RepositoryPort implementation (delegate to u.repo)
// ------------------------------------------------------------

func (u *ModelUsecase) ListVariations(
	ctx context.Context,
	filter modeldom.VariationFilter,
	page modeldom.Page,
) (modeldom.VariationPageResult, error) {
	if u.repo == nil {
		return modeldom.VariationPageResult{}, modeldom.ErrNotFound
	}

	return u.repo.ListVariations(ctx, filter, page)
}

func (u *ModelUsecase) GetModelVariations(
	ctx context.Context,
	productBlueprintID string,
) ([]modeldom.ModelVariation, error) {
	if u.repo == nil {
		return nil, modeldom.ErrNotFound
	}
	if productBlueprintID == "" {
		return nil, modeldom.ErrInvalidBlueprintID
	}

	return u.repo.GetModelVariations(ctx, productBlueprintID)
}

// CreateModelVariation creates a category-specific ModelVariation.
//
// NOTE:
//   - apparel では NewModelVariation.Apparel を使う。
//   - alcohol では NewModelVariation.Alcohol を使う。
//   - apparel.outerwear / apparel.shoes では Measurements は nil / 空でもよい。
//   - alcohol では Volume のみを variation field として扱う。
//   - measurements 必須カテゴリかどうかは usecase 側で category schema を参照して判定する。
func (u *ModelUsecase) CreateModelVariation(
	ctx context.Context,
	v modeldom.NewModelVariation,
) (modeldom.ModelVariation, error) {
	if u.repo == nil {
		return nil, modeldom.ErrNotFound
	}
	if err := v.Validate(); err != nil {
		return nil, err
	}

	created, err := u.repo.CreateModelVariation(ctx, v)
	if err != nil {
		return nil, err
	}

	return created, nil
}

// UpdateModelVariation updates a category-specific ModelVariation.
//
// NOTE:
//   - apparel では size / color / measurements 更新に対応する。
//   - alcohol では volume 更新に対応する。
//   - 履歴は保存しない。
func (u *ModelUsecase) UpdateModelVariation(
	ctx context.Context,
	variationID string,
	updates modeldom.ModelVariationUpdate,
) (modeldom.ModelVariation, error) {
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

func (u *ModelUsecase) DeleteModelVariation(
	ctx context.Context,
	variationID string,
) (modeldom.ModelVariation, error) {
	if u.repo == nil {
		return nil, modeldom.ErrNotFound
	}

	if variationID == "" {
		return nil, modeldom.ErrInvalidID
	}

	return u.repo.DeleteModelVariation(ctx, variationID)
}

// ReplaceModelVariations replaces category-specific ModelVariations.
//
// NOTE:
//   - 全要素が同じ ProductBlueprintID を持つ前提。
//   - ProductBlueprintID は NewModelVariation.ProductBlueprintID() から解決する。
//   - apparel では size / color / measurements を扱う。
//   - alcohol では volume のみを扱う。
//   - ModelVariation は interface なので、UpdatedAt はここで直接補正しない。
//     UpdatedAt は repository / domain 側で設定された値を正とする。
func (u *ModelUsecase) ReplaceModelVariations(
	ctx context.Context,
	vars []modeldom.NewModelVariation,
) ([]modeldom.ModelVariation, error) {
	if u.repo == nil {
		return nil, modeldom.ErrNotFound
	}

	if len(vars) == 0 {
		return u.repo.ReplaceModelVariations(ctx, vars)
	}

	productBlueprintID := vars[0].ProductBlueprintID()
	if productBlueprintID == "" {
		return nil, modeldom.ErrInvalidBlueprintID
	}

	for _, v := range vars {
		if err := v.Validate(); err != nil {
			return nil, err
		}
		if v.ProductBlueprintID() != productBlueprintID {
			return nil, modeldom.ErrProductMismatch
		}
	}

	updated, err := u.repo.ReplaceModelVariations(ctx, vars)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

func (u *ModelUsecase) GetSizeVariations(
	ctx context.Context,
	productBlueprintID string,
) ([]modeldom.SizeVariation, error) {
	if u.repo == nil {
		return nil, modeldom.ErrNotFound
	}
	if productBlueprintID == "" {
		return nil, modeldom.ErrInvalidBlueprintID
	}

	return u.repo.GetSizeVariations(ctx, productBlueprintID)
}

func (u *ModelUsecase) GetModelNumbers(
	ctx context.Context,
	productBlueprintID string,
) ([]modeldom.ModelNumber, error) {
	if u.repo == nil {
		return nil, modeldom.ErrNotFound
	}
	if productBlueprintID == "" {
		return nil, modeldom.ErrInvalidBlueprintID
	}

	return u.repo.GetModelNumbers(ctx, productBlueprintID)
}
