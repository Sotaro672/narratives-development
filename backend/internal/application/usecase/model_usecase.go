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
//   Firestore 保存パス：
//   product_blueprints_history/{blueprintID}/models/{version}/variations/{variationID}
// ------------------------------------------------------------

type ModelHistoryRepo interface {
	SaveSnapshot(
		ctx context.Context,
		blueprintID string,
		version int64,
		variations []modeldom.ModelVariation,
	) error

	ListByProductBlueprintIDAndVersion(
		ctx context.Context,
		blueprintID string,
		version int64,
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
	version int64,
) {
	if u.historyRepo == nil {
		log.Printf("[ModelUsecase] saveHistorySnapshot skipped: historyRepo is nil (blueprintID=%s version=%d)", blueprintID, version)
		return
	}
	blueprintID = strings.TrimSpace(blueprintID)
	if blueprintID == "" {
		log.Printf("[ModelUsecase] saveHistorySnapshot skipped: empty blueprintID")
		return
	}
	if version <= 0 {
		log.Printf("[ModelUsecase] saveHistorySnapshot: version<=0, fallback to 1 (blueprintID=%s)", blueprintID)
		version = 1
	}

	log.Printf("[ModelUsecase] saveHistorySnapshot start: blueprintID=%s version=%d", blueprintID, version)

	vars, err := u.repo.ListModelVariationsByProductBlueprintID(ctx, blueprintID)
	if err != nil {
		log.Printf("[ModelUsecase] saveHistorySnapshot: list variations failed: %v", err)
		return
	}

	if err := u.historyRepo.SaveSnapshot(ctx, blueprintID, version, vars); err != nil {
		log.Printf("[ModelUsecase] saveHistorySnapshot: save snapshot failed: %v", err)
		return
	}

	log.Printf("[ModelUsecase] saveHistorySnapshot done: blueprintID=%s version=%d count=%d", blueprintID, version, len(vars))
}

// ------------------------------------------------------------
// Queries
// ------------------------------------------------------------

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

// Create ModelVariation → Version = 1 固定 + 履歴保存
func (u *ModelUsecase) CreateModelVariation(
	ctx context.Context,
	v modeldom.NewModelVariation,
) (*modeldom.ModelVariation, error) {

	created, err := u.repo.CreateModelVariation(ctx, v)
	if err != nil {
		return nil, err
	}

	// version を 1 に設定（ドメイン上の初期値）
	if created.Version <= 0 {
		created.Version = 1
	}

	// 履歴スナップショット保存（productBlueprint の Version と揃えたい場合はここを調整）
	u.saveHistorySnapshot(ctx, created.ProductBlueprintID, created.Version)

	return created, nil
}

// Update ModelVariation → Version++ + 履歴保存
func (u *ModelUsecase) UpdateModelVariation(
	ctx context.Context,
	variationID string,
	updates modeldom.ModelVariationUpdate,
) (*modeldom.ModelVariation, error) {

	variationID = strings.TrimSpace(variationID)
	if variationID == "" {
		return nil, modeldom.ErrInvalidID
	}

	// 現状を取得して version をインクリメント
	current, err := u.repo.GetModelVariationByID(ctx, variationID)
	if err != nil {
		return nil, err
	}

	newVersion := current.Version + 1
	if newVersion <= 0 {
		newVersion = 1
	}

	updated, err := u.repo.UpdateModelVariation(ctx, variationID, updates)
	if err != nil {
		return nil, err
	}

	updated.Version = newVersion

	// 履歴スナップショット保存
	u.saveHistorySnapshot(ctx, updated.ProductBlueprintID, updated.Version)

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

	// ReplaceVariations 後の version 管理は ProductBlueprintUsecase 側で一括で行う想定。
	// ここでは最低限の初期値補正と UpdatedAt 更新のみを行う。
	now := time.Now().UTC()
	for i := range updated {
		if updated[i].Version <= 0 {
			updated[i].Version = 1
		}
		updated[i].UpdatedAt = now
	}

	// 代表となる blueprintID を決めて履歴を保存（全件同一前提）
	if len(updated) > 0 {
		blueprintID := updated[0].ProductBlueprintID
		u.saveHistorySnapshot(ctx, blueprintID, updated[0].Version)
	}

	return updated, nil
}
