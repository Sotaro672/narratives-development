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
	// productID などのキーで ModelData を取得
	GetModelData(ctx context.Context, id string) (*modeldom.ModelData, error)
	// ProductBlueprintID ベースで ModelData を取得
	GetModelDataByBlueprintID(ctx context.Context, blueprintID string) (*modeldom.ModelData, error)
	// id（productID / blueprintID など）をキーとして ModelData を更新
	UpdateModelData(ctx context.Context, id string, updates modeldom.ModelDataUpdate) (*modeldom.ModelData, error)

	GetModelVariationByID(ctx context.Context, variationID string) (*modeldom.ModelVariation, error)

	// ★ 新規作成では productId は使わないので削除
	CreateModelVariation(ctx context.Context, variation modeldom.NewModelVariation) (*modeldom.ModelVariation, error)

	UpdateModelVariation(ctx context.Context, variationID string, updates modeldom.ModelVariationUpdate) (*modeldom.ModelVariation, error)
	DeleteModelVariation(ctx context.Context, variationID string) (*modeldom.ModelVariation, error)

	// まとめて入れ替える場合も、Repo 側で紐づけキーを解決する想定にしておく
	ReplaceModelVariations(ctx context.Context, vars []modeldom.NewModelVariation) ([]modeldom.ModelVariation, error)
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
// Firestore 実装側では id を productID / productBlueprintID などとして扱う想定。
func (u *ModelUsecase) GetByID(ctx context.Context, id string) (*modeldom.ModelData, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, modeldom.ErrInvalidProductID
	}
	return u.repo.GetModelData(ctx, id)
}

func (u *ModelUsecase) GetModelData(ctx context.Context, id string) (*modeldom.ModelData, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, modeldom.ErrInvalidProductID
	}
	return u.repo.GetModelData(ctx, id)
}

func (u *ModelUsecase) GetModelDataByProductBlueprintID(ctx context.Context, productBlueprintID string) (*modeldom.ModelData, error) {
	productBlueprintID = strings.TrimSpace(productBlueprintID)
	if productBlueprintID == "" {
		return nil, modeldom.ErrInvalidBlueprintID
	}
	return u.repo.GetModelDataByBlueprintID(ctx, productBlueprintID)
}

// ------------------------------------------------------------
// Commands
// ------------------------------------------------------------

func (u *ModelUsecase) UpdateModelData(
	ctx context.Context,
	id string,
	updates modeldom.ModelDataUpdate,
) (*modeldom.ModelData, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, modeldom.ErrInvalidProductID
	}
	return u.repo.UpdateModelData(ctx, id, updates)
}

// ★ CreateModelVariation から productId を排除
//   - 紐付けは v.ProductBlueprintID など、NewModelVariation 側の情報で行う想定。
func (u *ModelUsecase) CreateModelVariation(
	ctx context.Context,
	v modeldom.NewModelVariation,
) (*modeldom.ModelVariation, error) {
	// 必要ならここで v 内のフィールドを Trim することも可能
	return u.repo.CreateModelVariation(ctx, v)
}

func (u *ModelUsecase) UpdateModelVariation(
	ctx context.Context,
	variationID string,
	updates modeldom.ModelVariationUpdate,
) (*modeldom.ModelVariation, error) {
	variationID = strings.TrimSpace(variationID)
	if variationID == "" {
		return nil, modeldom.ErrInvalidID
	}
	return u.repo.UpdateModelVariation(ctx, variationID, updates)
}

func (u *ModelUsecase) DeleteModelVariation(
	ctx context.Context,
	variationID string,
) (*modeldom.ModelVariation, error) {
	variationID = strings.TrimSpace(variationID)
	if variationID == "" {
		return nil, modeldom.ErrInvalidID
	}
	return u.repo.DeleteModelVariation(ctx, variationID)
}

// まとめて variations を差し替えるケース
func (u *ModelUsecase) ReplaceModelVariations(
	ctx context.Context,
	vars []modeldom.NewModelVariation,
) ([]modeldom.ModelVariation, error) {
	return u.repo.ReplaceModelVariations(ctx, vars)
}
