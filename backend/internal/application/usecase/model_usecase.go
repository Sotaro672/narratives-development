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
// RepositoryPort implementation
// ------------------------------------------------------------

func (u *ModelUsecase) ListByProductBlueprintID(
	ctx context.Context,
	productBlueprintID string,
) ([]modeldom.ModelVariation, error) {
	if u.repo == nil {
		return nil, modeldom.ErrNotFound
	}
	if productBlueprintID == "" {
		return nil, modeldom.ErrInvalidBlueprintID
	}

	return u.repo.ListByProductBlueprintID(ctx, productBlueprintID)
}

// Create creates a category-specific ModelVariation.
//
// NOTE:
//   - apparel では NewModelVariation.Apparel を使う。
//   - alcohol では NewModelVariation.Alcohol を使う。
//   - apparel.outerwear / apparel.shoes では Measurements は nil / 空でもよい。
//   - alcohol では Volume のみを variation field として扱う。
//   - measurements 必須カテゴリかどうかは usecase 側で category schema を参照して判定する。
func (u *ModelUsecase) Create(
	ctx context.Context,
	v modeldom.NewModelVariation,
) (modeldom.ModelVariation, error) {
	if u.repo == nil {
		return nil, modeldom.ErrNotFound
	}
	if err := v.Validate(); err != nil {
		return nil, err
	}

	created, err := u.repo.Create(ctx, v)
	if err != nil {
		return nil, err
	}

	return created, nil
}

// Update updates a category-specific ModelVariation.
//
// NOTE:
//   - apparel では size / color / measurements 更新に対応する。
//   - alcohol では volume 更新に対応する。
//   - 一括差し替えは行わない。
//   - 削除が必要な variation は Delete を個別に呼び出す。
//   - 履歴は保存しない。
func (u *ModelUsecase) Update(
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

	updated, err := u.repo.Update(ctx, variationID, updates)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

// Delete physically deletes a category-specific ModelVariation.
//
// NOTE:
//   - repository は対象 document を物理削除する。
func (u *ModelUsecase) Delete(
	ctx context.Context,
	variationID string,
) error {
	if u.repo == nil {
		return modeldom.ErrNotFound
	}
	if variationID == "" {
		return modeldom.ErrInvalidID
	}

	return u.repo.Delete(ctx, variationID)
}
