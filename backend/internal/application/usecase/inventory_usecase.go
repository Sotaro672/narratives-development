// backend/internal/application/usecase/inventory_usecase.go
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

// UpsertFromMint
func (uc *InventoryUsecase) UpsertFromMint(
	ctx context.Context,
	tokenBlueprintID string,
	productBlueprintID string,
	productIDs []string,
) (invdom.Mint, error) {
	if uc == nil || uc.repo == nil {
		return invdom.Mint{}, errors.New("inventory usecase/repo is nil")
	}

	tbID := strings.TrimSpace(tokenBlueprintID)
	pbID := strings.TrimSpace(productBlueprintID)
	if tbID == "" {
		return invdom.Mint{}, invdom.ErrInvalidTokenBlueprintID
	}
	if pbID == "" {
		return invdom.Mint{}, invdom.ErrInvalidProductBlueprintID
	}

	ids := normalizeIDs(productIDs)
	if len(ids) == 0 {
		return invdom.Mint{}, invdom.ErrInvalidProducts
	}

	docID := buildInventoryDocID(tbID, pbID)

	existing, err := uc.repo.GetByID(ctx, docID)
	if err != nil {
		if errors.Is(err, invdom.ErrNotFound) {
			now := time.Now().UTC()
			ent, err := invdom.NewMint(
				docID,
				tbID,
				pbID,
				ids,
				len(ids),
				now,
			)
			if err != nil {
				return invdom.Mint{}, err
			}
			return uc.repo.Create(ctx, ent)
		}
		return invdom.Mint{}, err
	}

	// 既存あり → products に未登録分だけ追加
	current := normalizeIDs(existing.Products)
	seen := map[string]struct{}{}
	for _, p := range current {
		seen[p] = struct{}{}
	}

	added := 0
	for _, pid := range ids {
		if _, ok := seen[pid]; ok {
			continue
		}
		seen[pid] = struct{}{}
		current = append(current, pid)
		added++
	}
	sort.Strings(current)

	existing.Products = current

	updated, err := uc.repo.Update(ctx, existing)
	if err != nil {
		return invdom.Mint{}, err
	}

	if added > 0 {
		updated, err = uc.repo.IncrementAccumulation(ctx, docID, added)
		if err != nil {
			return invdom.Mint{}, err
		}
	}

	return updated, nil
}

func (uc *InventoryUsecase) CreateMint(
	ctx context.Context,
	tokenBlueprintID string,
	productBlueprintID string,
	productIDs []string,
	accumulation int,
) (invdom.Mint, error) {
	now := time.Now().UTC()

	m, err := invdom.NewMint(
		"",
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

func (uc *InventoryUsecase) UpdateMint(
	ctx context.Context,
	m invdom.Mint,
) (invdom.Mint, error) {
	if strings.TrimSpace(m.ID) == "" {
		return invdom.Mint{}, invdom.ErrInvalidMintID
	}
	m.UpdatedAt = time.Now().UTC()
	m.Products = normalizeIDs(m.Products)
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

func (uc *InventoryUsecase) ListByTokenAndProductBlueprintID(ctx context.Context, tokenBlueprintID, productBlueprintID string) ([]invdom.Mint, error) {
	return uc.repo.ListByTokenAndProductBlueprintID(ctx, tokenBlueprintID, productBlueprintID)
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

func buildInventoryDocID(tokenBlueprintID, productBlueprintID string) string {
	sanitize := func(s string) string {
		s = strings.TrimSpace(s)
		s = strings.ReplaceAll(s, "/", "_")
		return s
	}
	return sanitize(tokenBlueprintID) + "__" + sanitize(productBlueprintID)
}
