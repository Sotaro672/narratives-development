package usecase

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	invdom "narratives/internal/domain/inventory"
)

type InventoryUsecase struct {
	repo invdom.RepositoryPort
}

func NewInventoryUsecase(repo invdom.RepositoryPort) *InventoryUsecase {
	return &InventoryUsecase{repo: repo}
}

// UpsertFromMintByModel
// docId = modelId__tokenBlueprintId
// products / accumulation は modelId ごとに更新される
func (uc *InventoryUsecase) UpsertFromMintByModel(
	ctx context.Context,
	tokenBlueprintID string,
	productBlueprintID string,
	modelID string,
	productIDs []string,
) (invdom.Mint, error) {
	if uc == nil || uc.repo == nil {
		return invdom.Mint{}, errors.New("inventory usecase/repo is nil")
	}

	tbID := strings.TrimSpace(tokenBlueprintID)
	pbID := strings.TrimSpace(productBlueprintID)
	mID := strings.TrimSpace(modelID)

	if tbID == "" {
		return invdom.Mint{}, invdom.ErrInvalidTokenBlueprintID
	}
	if pbID == "" {
		return invdom.Mint{}, invdom.ErrInvalidProductBlueprintID
	}
	if mID == "" {
		return invdom.Mint{}, invdom.ErrInvalidModelID
	}

	ids := normalizeIDs(productIDs)
	if len(ids) == 0 {
		return invdom.Mint{}, invdom.ErrInvalidProducts
	}

	// ✅ repo が atomic upsert を持つ前提で使う（最も安全）
	return uc.repo.UpsertByModelAndToken(ctx, tbID, pbID, mID, ids)
}

// 互換：既存コードが UpsertFromMint を呼んでいる場合に備えて残す
// ※ 旧仕様( tokenBlueprint + productBlueprint ) ではなく、modelID が必要なのでエラーに寄せる
func (uc *InventoryUsecase) UpsertFromMint(
	ctx context.Context,
	tokenBlueprintID string,
	productBlueprintID string,
	productIDs []string,
) (invdom.Mint, error) {
	return invdom.Mint{}, errors.New("UpsertFromMint is deprecated: use UpsertFromMintByModel(tokenBlueprintID, productBlueprintID, modelID, productIDs)")
}

func (uc *InventoryUsecase) CreateMint(
	ctx context.Context,
	tokenBlueprintID string,
	productBlueprintID string,
	modelID string,
	productIDs []string,
	accumulation int,
) (invdom.Mint, error) {
	now := time.Now().UTC()

	m, err := invdom.NewMint(
		"",
		tokenBlueprintID,
		productBlueprintID,
		modelID,
		productIDs,
		accumulation,
		now,
	)
	if err != nil {
		return invdom.Mint{}, err
	}

	return uc.repo.Create(ctx, m)
}

func (uc *InventoryUsecase) UpdateMint(
	ctx context.Context,
	m invdom.Mint,
) (invdom.Mint, error) {
	if strings.TrimSpace(m.ID) == "" {
		return invdom.Mint{}, invdom.ErrInvalidMintID
	}
	m.UpdatedAt = time.Now().UTC()
	m.Products = normalizeIDs(m.Products)
	if m.Accumulation <= 0 {
		m.Accumulation = len(m.Products)
	}
	return uc.repo.Update(ctx, m)
}

func (uc *InventoryUsecase) DeleteMint(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return invdom.ErrInvalidMintID
	}
	return uc.repo.Delete(ctx, id)
}

// Queries
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

func (uc *InventoryUsecase) ListByModelID(ctx context.Context, modelID string) ([]invdom.Mint, error) {
	return uc.repo.ListByModelID(ctx, modelID)
}

func (uc *InventoryUsecase) ListByTokenAndModelID(ctx context.Context, tokenBlueprintID, modelID string) ([]invdom.Mint, error) {
	return uc.repo.ListByTokenAndModelID(ctx, tokenBlueprintID, modelID)
}

// Helpers (local)
func normalizeIDs(raw []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(raw))
	for _, s := range raw {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
