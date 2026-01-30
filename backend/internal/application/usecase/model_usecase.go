package usecase

import (
	"context"
	"log"
	"strings"
	"time"

	modeldom "narratives/internal/domain/model"
)

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
//
// ✅ Mall handler から model.RepositoryPort として渡したいので、
//   ModelUsecase 自体が modeldom.RepositoryPort を実装する（= repo に委譲する）
// ------------------------------------------------------------

type ModelUsecase struct {
	repo        modeldom.RepositoryPort
	historyRepo ModelHistoryRepo
}

func NewModelUsecase(repo modeldom.RepositoryPort, historyRepo ModelHistoryRepo) *ModelUsecase {
	return &ModelUsecase{
		repo:        repo,
		historyRepo: historyRepo,
	}
}

// ------------------------------------------------------------
// internal helpers
// ------------------------------------------------------------

// 履歴スナップショット保存（失敗してもビジネス処理は失敗させない）
func (u *ModelUsecase) saveHistorySnapshot(ctx context.Context, blueprintID string) {
	if u.historyRepo == nil {
		log.Printf("[ModelUsecase] saveHistorySnapshot skipped: historyRepo is nil (blueprintID=%s)", blueprintID)
		return
	}
	if u.repo == nil {
		log.Printf("[ModelUsecase] saveHistorySnapshot skipped: repo is nil (blueprintID=%s)", blueprintID)
		return
	}

	blueprintID = strings.TrimSpace(blueprintID)
	if blueprintID == "" {
		log.Printf("[ModelUsecase] saveHistorySnapshot skipped: empty blueprintID")
		return
	}

	log.Printf("[ModelUsecase] saveHistorySnapshot start: blueprintID=%s", blueprintID)

	// ✅ RepositoryPort に追加した ListModelVariationsByProductBlueprintID を使って全件取得
	all, err := u.repo.ListModelVariationsByProductBlueprintID(ctx, blueprintID)
	if err != nil {
		log.Printf("[ModelUsecase] saveHistorySnapshot: list variations failed: %v", err)
		return
	}

	if err := u.historyRepo.SaveSnapshot(ctx, blueprintID, all); err != nil {
		log.Printf("[ModelUsecase] saveHistorySnapshot: save snapshot failed: %v", err)
		return
	}

	log.Printf("[ModelUsecase] saveHistorySnapshot done: blueprintID=%s count=%d", blueprintID, len(all))
}

// ------------------------------------------------------------
// Queries (compat / convenience)
// ------------------------------------------------------------

// GetByID is a legacy-style alias: variationID を受け取り、ModelVariation を返す
func (u *ModelUsecase) GetByID(ctx context.Context, id string) (*modeldom.ModelVariation, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, modeldom.ErrInvalidID
	}
	return u.GetModelVariationByID(ctx, id)
}

// ★ HTTP の GET /models/variations/{variationId} 用の明示メソッド
// ※ このメソッドは 1 箇所のみ定義（DuplicateMethod 回避）
func (u *ModelUsecase) GetModelVariationByID(ctx context.Context, variationID string) (*modeldom.ModelVariation, error) {
	if u.repo == nil {
		return nil, modeldom.ErrNotFound
	}
	variationID = strings.TrimSpace(variationID)
	if variationID == "" {
		return nil, modeldom.ErrInvalidID
	}
	return u.repo.GetModelVariationByID(ctx, variationID)
}

// 互換：既存コードが呼んでいる可能性があるため残す
func (u *ModelUsecase) GetModelDataByProductBlueprintID(ctx context.Context, productBlueprintID string) (*modeldom.ModelData, error) {
	return u.GetModelDataByBlueprintID(ctx, productBlueprintID)
}

// ✅ 新：productBlueprintID → variations 一覧（repo に委譲）
// （旧式の ListVariations + paging ループは削除）
func (u *ModelUsecase) ListModelVariationsByProductBlueprintID(ctx context.Context, productBlueprintID string) ([]modeldom.ModelVariation, error) {
	productBlueprintID = strings.TrimSpace(productBlueprintID)
	if productBlueprintID == "" {
		return nil, modeldom.ErrInvalidBlueprintID
	}
	if u.repo == nil {
		return nil, modeldom.ErrNotFound
	}

	return u.repo.ListModelVariationsByProductBlueprintID(ctx, productBlueprintID)
}

// ------------------------------------------------------------
// RepositoryPort implementation (delegate to u.repo)
// ------------------------------------------------------------

func (u *ModelUsecase) GetModelData(ctx context.Context, productID string) (*modeldom.ModelData, error) {
	if u.repo == nil {
		return nil, modeldom.ErrNotFound
	}
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return nil, modeldom.ErrInvalidProductID
	}
	return u.repo.GetModelData(ctx, productID)
}

func (u *ModelUsecase) GetModelDataByBlueprintID(ctx context.Context, productBlueprintID string) (*modeldom.ModelData, error) {
	if u.repo == nil {
		return nil, modeldom.ErrNotFound
	}
	productBlueprintID = strings.TrimSpace(productBlueprintID)
	if productBlueprintID == "" {
		return nil, modeldom.ErrInvalidBlueprintID
	}
	return u.repo.GetModelDataByBlueprintID(ctx, productBlueprintID)
}

func (u *ModelUsecase) UpdateModelData(ctx context.Context, productID string, updates modeldom.ModelDataUpdate) (*modeldom.ModelData, error) {
	if u.repo == nil {
		return nil, modeldom.ErrNotFound
	}
	productID = strings.TrimSpace(productID)
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
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return nil, modeldom.ErrInvalidProductID
	}
	return u.repo.GetModelVariations(ctx, productID)
}

// Create ModelVariation → 履歴保存のみ（version は扱わない）
func (u *ModelUsecase) CreateModelVariation(ctx context.Context, v modeldom.NewModelVariation) (*modeldom.ModelVariation, error) {
	if u.repo == nil {
		return nil, modeldom.ErrNotFound
	}

	created, err := u.repo.CreateModelVariation(ctx, v)
	if err != nil {
		return nil, err
	}

	// 履歴スナップショット保存
	u.saveHistorySnapshot(ctx, created.ProductBlueprintID)

	return created, nil
}

// Update ModelVariation → 履歴保存のみ（version は扱わない）
func (u *ModelUsecase) UpdateModelVariation(ctx context.Context, variationID string, updates modeldom.ModelVariationUpdate) (*modeldom.ModelVariation, error) {
	if u.repo == nil {
		return nil, modeldom.ErrNotFound
	}

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

func (u *ModelUsecase) DeleteModelVariation(ctx context.Context, variationID string) (*modeldom.ModelVariation, error) {
	if u.repo == nil {
		return nil, modeldom.ErrNotFound
	}

	variationID = strings.TrimSpace(variationID)
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

	// 代表となる blueprintID を決めて履歴を保存（全件同一前提）
	if len(updated) > 0 {
		blueprintID := updated[0].ProductBlueprintID
		u.saveHistorySnapshot(ctx, blueprintID)
	}

	return updated, nil
}

func (u *ModelUsecase) GetSizeVariations(ctx context.Context, productID string) ([]modeldom.SizeVariation, error) {
	if u.repo == nil {
		return nil, modeldom.ErrNotFound
	}
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return nil, modeldom.ErrInvalidProductID
	}
	return u.repo.GetSizeVariations(ctx, productID)
}

func (u *ModelUsecase) GetModelNumbers(ctx context.Context, productID string) ([]modeldom.ModelNumber, error) {
	if u.repo == nil {
		return nil, modeldom.ErrNotFound
	}
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return nil, modeldom.ErrInvalidProductID
	}
	return u.repo.GetModelNumbers(ctx, productID)
}
