package usecase

import (
	"context"
	"log"
	"strings"
	"time"

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

	// 全入れ替え
	ReplaceModelVariations(ctx context.Context, vars []modeldom.NewModelVariation) ([]modeldom.ModelVariation, error)

	// productBlueprintID ごとにモデル一覧
	ListModelVariationsByProductBlueprintID(ctx context.Context, productBlueprintID string) ([]modeldom.ModelVariation, error)
}

// ------------------------------------------------------------
// ModelHistoryRepo
// ------------------------------------------------------------
//   Firestore 保存パス（例）：
//   product_blueprints_history/{blueprintID}/models/{snapshotID}/variations/{variationID}
//   ※ここでは version を扱わず、実際のキー構成は実装側に委譲。
// ------------------------------------------------------------

type ModelHistoryRepo interface {
	SaveSnapshot(
		ctx context.Context,
		blueprintID string,
		variations []modeldom.ModelVariation,
	) error

	ListByProductBlueprintID(
		ctx context.Context,
		blueprintID string,
	) ([]modeldom.ModelVariation, error)
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
// internal helpers
// ------------------------------------------------------------

// 履歴スナップショット保存（失敗してもビジネス処理は失敗させない）
func (u *ModelUsecase) saveHistorySnapshot(
	ctx context.Context,
	blueprintID string,
) {
	if u.historyRepo == nil {
		log.Printf("[ModelUsecase] saveHistorySnapshot skipped: historyRepo is nil (blueprintID=%s)", blueprintID)
		return
	}
	blueprintID = strings.TrimSpace(blueprintID)
	if blueprintID == "" {
		log.Printf("[ModelUsecase] saveHistorySnapshot skipped: empty blueprintID")
		return
	}

	log.Printf("[ModelUsecase] saveHistorySnapshot start: blueprintID=%s", blueprintID)

	vars, err := u.repo.ListModelVariationsByProductBlueprintID(ctx, blueprintID)
	if err != nil {
		log.Printf("[ModelUsecase] saveHistorySnapshot: list variations failed: %v", err)
		return
	}

	if err := u.historyRepo.SaveSnapshot(ctx, blueprintID, vars); err != nil {
		log.Printf("[ModelUsecase] saveHistorySnapshot: save snapshot failed: %v", err)
		return
	}

	log.Printf("[ModelUsecase] saveHistorySnapshot done: blueprintID=%s count=%d", blueprintID, len(vars))
}

// ------------------------------------------------------------
// Queries
// ------------------------------------------------------------

// GetByID is a legacy-style alias: variationID を受け取り、ModelVariation を返す
func (u *ModelUsecase) GetByID(ctx context.Context, id string) (*modeldom.ModelVariation, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, modeldom.ErrInvalidID
	}
	return u.repo.GetModelVariationByID(ctx, id)
}

// ★追加：HTTP の GET /models/variations/{variationId} 用の明示メソッド
// （mintRequest の modelId (= variationId) から modelNumber/size/color を取得する用途）
func (u *ModelUsecase) GetModelVariationByID(
	ctx context.Context,
	variationID string,
) (*modeldom.ModelVariation, error) {
	return u.GetByID(ctx, variationID)
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

// Create ModelVariation → 履歴保存のみ（version は扱わない）
func (u *ModelUsecase) CreateModelVariation(
	ctx context.Context,
	v modeldom.NewModelVariation,
) (*modeldom.ModelVariation, error) {

	created, err := u.repo.CreateModelVariation(ctx, v)
	if err != nil {
		return nil, err
	}

	// 履歴スナップショット保存
	u.saveHistorySnapshot(ctx, created.ProductBlueprintID)

	return created, nil
}

// Update ModelVariation → 履歴保存のみ（version は扱わない）
func (u *ModelUsecase) UpdateModelVariation(
	ctx context.Context,
	variationID string,
	updates modeldom.ModelVariationUpdate,
) (*modeldom.ModelVariation, error) {

	variationID = strings.TrimSpace(variationID)
	if variationID == "" {
		return nil, modeldom.ErrInvalidID
	}

	updated, err := u.repo.UpdateModelVariation(ctx, variationID, updates)
	if err != nil {
		return nil, err
	}

	// 履歴スナップショット保存
	u.saveHistorySnapshot(ctx, updated.ProductBlueprintID)

	return updated, nil
}

// Delete
func (u *ModelUsecase) DeleteModelVariation(
	ctx context.Context,
	variationID string,
) (*modeldom.ModelVariation, error) {

	variationID = strings.TrimSpace(variationID)
	if variationID == "" {
		return nil, modeldom.ErrInvalidID
	}

	// ※必要なら削除前に saveHistorySnapshot を呼ぶこともできる
	return u.repo.DeleteModelVariation(ctx, variationID)
}

// Replace all variations
func (u *ModelUsecase) ReplaceModelVariations(
	ctx context.Context,
	vars []modeldom.NewModelVariation,
) ([]modeldom.ModelVariation, error) {

	updated, err := u.repo.ReplaceModelVariations(ctx, vars)
	if err != nil {
		return nil, err
	}

	// ReplaceVariations 後は UpdatedAt の補正のみ（version は扱わない）
	now := time.Now().UTC()
	for i := range updated {
		updated[i].UpdatedAt = now
	}

	// 代表となる blueprintID を決めて履歴を保存（全件同一前提）
	if len(updated) > 0 {
		blueprintID := updated[0].ProductBlueprintID
		u.saveHistorySnapshot(ctx, blueprintID)
	}

	return updated, nil
}
