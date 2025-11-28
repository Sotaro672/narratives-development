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

	// 単一の ModelVariation を ID で取得
	GetModelVariationByID(ctx context.Context, variationID string) (*modeldom.ModelVariation, error)

	// ModelVariation 作成
	CreateModelVariation(ctx context.Context, variation modeldom.NewModelVariation) (*modeldom.ModelVariation, error)

	// ModelVariation 更新
	UpdateModelVariation(ctx context.Context, variationID string, updates modeldom.ModelVariationUpdate) (*modeldom.ModelVariation, error)

	// ModelVariation 削除（具体的な削除方法は実装に委譲）
	DeleteModelVariation(ctx context.Context, variationID string) (*modeldom.ModelVariation, error)

	// まとめて入れ替える場合も、Repo 側で紐づけキーを解決する想定にしておく
	ReplaceModelVariations(ctx context.Context, vars []modeldom.NewModelVariation) ([]modeldom.ModelVariation, error)

	// ★ 追加: productBlueprintID ごとに ModelVariation 一覧を取得
	ListModelVariationsByProductBlueprintID(ctx context.Context, productBlueprintID string) ([]modeldom.ModelVariation, error)
}

// ------------------------------------------------------------
// ModelHistoryRepo
// ------------------------------------------------------------

type ModelHistoryRepo interface {
	// TODO: モデルバリエーション履歴保存用のメソッドを定義する
}

// ------------------------------------------------------------
// ModelUsecase
// ------------------------------------------------------------

type ModelUsecase struct {
	repo        ModelRepo
	historyRepo ModelHistoryRepo
}

func NewModelUsecase(repo ModelRepo, historyRepo ModelHistoryRepo) *ModelUsecase {
	return &ModelUsecase{
		repo:        repo,
		historyRepo: historyRepo,
	}
}

// ------------------------------------------------------------
// Queries
// ------------------------------------------------------------

// GetByID は /models/{id} から呼ばれる「単一 ModelVariation 取得」。
// Firestore 実装側では id を variation ドキュメント ID として扱う。
func (u *ModelUsecase) GetByID(ctx context.Context, id string) (*modeldom.ModelVariation, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, modeldom.ErrInvalidID
	}
	return u.repo.GetModelVariationByID(ctx, id)
}

func (u *ModelUsecase) GetModelData(ctx context.Context, id string) (*modeldom.ModelData, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, modeldom.ErrInvalidProductID
	}
	return u.repo.GetModelData(ctx, id)
}

func (u *ModelUsecase) GetModelDataByProductBlueprintID(
	ctx context.Context,
	productBlueprintID string,
) (*modeldom.ModelData, error) {
	productBlueprintID = strings.TrimSpace(productBlueprintID)
	if productBlueprintID == "" {
		return nil, modeldom.ErrInvalidBlueprintID
	}
	return u.repo.GetModelDataByBlueprintID(ctx, productBlueprintID)
}

// ★ 与えられた productBlueprintID に紐づく ModelVariation を list する
func (u *ModelUsecase) ListModelVariationsByProductBlueprintID(
	ctx context.Context,
	productBlueprintID string,
) ([]modeldom.ModelVariation, error) {
	productBlueprintID = strings.TrimSpace(productBlueprintID)
	if productBlueprintID == "" {
		return nil, modeldom.ErrInvalidBlueprintID
	}
	return u.repo.ListModelVariationsByProductBlueprintID(ctx, productBlueprintID)
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

// DeleteModelVariation は ModelVariation の削除を行うユースケース。
// 実際の削除方法（物理削除など）は repository 実装に委譲する。
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
