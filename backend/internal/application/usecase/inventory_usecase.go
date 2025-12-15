// backend/internal/application/usecase/inventory_usecase.go
package usecase

import (
	"context"
	"strings"
	"time"

	invdom "narratives/internal/domain/inventory"
)

// ============================================================
// InventoryUsecase
// ============================================================
//
// Inventory (mints) を扱うアプリケーション層ユースケース。
// - RepositoryPort を介して永続化層に依存する（Hexagonal）
// - 入力値の最低限バリデーションを行い、ドメイン整合性は domain.NewMint / validate に委譲する
type InventoryUsecase struct {
	repo invdom.RepositoryPort
}

func NewInventoryUsecase(repo invdom.RepositoryPort) *InventoryUsecase {
	return &InventoryUsecase{repo: repo}
}

// ============================================================
// Commands
// ============================================================

// CreateMint creates a new inventory Mint record.
//
// NOTE:
// - createdAt / updatedAt は createdAt を基準に初期化します。
// - products は productIDs から map へ変換され、mintAddress は "" 初期化です。
func (uc *InventoryUsecase) CreateMint(
	ctx context.Context,
	tokenBlueprintID string,
	productBlueprintID string,
	productIDs []string,
	accumulation int,
) (invdom.Mint, error) {
	now := time.Now().UTC()

	m, err := invdom.NewMint(
		"", // IDはrepo側で採番してよい
		tokenBlueprintID,
		productBlueprintID,
		productIDs,
		accumulation,
		now,
	)
	if err != nil {
		return invdom.Mint{}, err
	}

	return uc.repo.Create(ctx, m)
}

// UpdateMint updates an existing Mint.
// - UpdatedAt は repo 実装で更新しても良いが、ここでも更新して渡す。
func (uc *InventoryUsecase) UpdateMint(
	ctx context.Context,
	m invdom.Mint,
) (invdom.Mint, error) {
	// 最低限の必須チェック（ドメイン validate にも委譲）
	if strings.TrimSpace(m.ID) == "" {
		return invdom.Mint{}, invdom.ErrInvalidMintID
	}
	m.UpdatedAt = time.Now().UTC()
	return uc.repo.Update(ctx, m)
}

// DeleteMint deletes a Mint by id.
func (uc *InventoryUsecase) DeleteMint(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return invdom.ErrInvalidMintID
	}
	return uc.repo.Delete(ctx, id)
}

// ============================================================
// Queries
// ============================================================

func (uc *InventoryUsecase) GetMintByID(ctx context.Context, id string) (invdom.Mint, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return invdom.Mint{}, invdom.ErrInvalidMintID
	}
	return uc.repo.GetByID(ctx, id)
}

func (uc *InventoryUsecase) ListByTokenBlueprintID(ctx context.Context, tokenBlueprintID string) ([]invdom.Mint, error) {
	return uc.repo.ListByTokenBlueprintID(ctx, tokenBlueprintID)
}

func (uc *InventoryUsecase) ListByProductBlueprintID(ctx context.Context, productBlueprintID string) ([]invdom.Mint, error) {
	return uc.repo.ListByProductBlueprintID(ctx, productBlueprintID)
}

func (uc *InventoryUsecase) ListByTokenAndProductBlueprintID(ctx context.Context, tokenBlueprintID, productBlueprintID string) ([]invdom.Mint, error) {
	return uc.repo.ListByTokenAndProductBlueprintID(ctx, tokenBlueprintID, productBlueprintID)
}
