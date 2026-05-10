// backend/internal/application/usecase/inventory_usecase.go

package usecase

import (
	"context"
	"errors"
	"log"
	"sort"
	"strings"

	invdom "narratives/internal/domain/inventory"
)

type InventoryUsecase struct {
	repo invdom.RepositoryPort
}

func NewInventoryUsecase(repo invdom.RepositoryPort) *InventoryUsecase {
	return &InventoryUsecase{repo: repo}
}

// ============================================================
// Upsert entry from Mint by Model
// ============================================================
//
// - mint から在庫へ反映する唯一の入口
// - 在庫の蓄積は Stock（modelId -> {Products: ...}）で表現する前提
//
// ✅ 修正方針:
//   - 既存 model の追加反映が反射経由の Get->merge->Update で失敗し得るため、
//     repo の atomic upsert（transaction + UNION）に委譲する。
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

	tbID := tokenBlueprintID
	pbID := productBlueprintID
	mID := modelID

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

	// docId をここで確定（repo 側の sanitize と揃える）
	inventoryID := buildInventoryID(pbID, tbID)

	log.Printf(
		"[inventory_uc] UpsertFromMintByModel start inventoryId=%q tokenBlueprintId=%q productBlueprintId=%q modelId=%q products=%d",
		inventoryID, tbID, pbID, mID, len(ids),
	)

	// ✅ repo の atomic upsert に委譲（既存 model でも UNION で確実に追記される）
	updated, err := uc.repo.UpsertByProductBlueprintAndToken(ctx, tbID, pbID, mID, ids)
	if err != nil {
		log.Printf(
			"[inventory_uc] UpsertFromMintByModel upsert error inventoryId=%q tokenBlueprintId=%q productBlueprintId=%q modelId=%q err=%v",
			inventoryID, tbID, pbID, mID, err,
		)
		return invdom.Mint{}, err
	}

	log.Printf("[inventory_uc] UpsertFromMintByModel upsert ok inventoryId=%q", inventoryID)
	return updated, nil
}

// ============================================================
// ✅ NEW: Reserve by Order (payment success -> invoice.paid=true と同時に呼ぶ想定)
// ============================================================

type ReserveByOrderItem struct {
	InventoryID string
	ModelID     string
	Qty         int
}

// ReserveByOrder adds (orderID -> qty) into Stock[modelId].ReservedByOrder
// and updates ReservedCount accordingly.
func (uc *InventoryUsecase) ReserveByOrder(ctx context.Context, orderID string, items []ReserveByOrderItem) error {
	if uc == nil || uc.repo == nil {
		return errors.New("inventory usecase/repo is nil")
	}

	oid := orderID
	if oid == "" {
		return errors.New("inventory reserve: invalid orderId")
	}
	if len(items) == 0 {
		// 何もしない（呼び出し側が「対象なし」でも安全）
		return nil
	}

	for _, it := range items {
		invID := it.InventoryID
		mid := it.ModelID
		qty := it.Qty
		if invID == "" || mid == "" || qty <= 0 {
			return errors.New("inventory reserve: invalid item")
		}

		m, err := uc.repo.GetByID(ctx, invID)
		if err != nil {
			return err
		}

		if err := reserveStockByModelOrder(&m, mid, oid, qty); err != nil {
			return err
		}

		if _, err := uc.repo.Update(ctx, m); err != nil {
			return err
		}
	}

	return nil
}

// ============================================================
// CRUD
// ============================================================

func (uc *InventoryUsecase) Create(ctx context.Context, m invdom.Mint) (invdom.Mint, error) {
	if uc == nil || uc.repo == nil {
		return invdom.Mint{}, errors.New("inventory usecase/repo is nil")
	}
	return uc.repo.Create(ctx, m)
}

func (uc *InventoryUsecase) Update(ctx context.Context, m invdom.Mint) (invdom.Mint, error) {
	if uc == nil || uc.repo == nil {
		return invdom.Mint{}, errors.New("inventory usecase/repo is nil")
	}
	if m.ID == "" {
		return invdom.Mint{}, invdom.ErrInvalidMintID
	}
	return uc.repo.Update(ctx, m)
}

func (uc *InventoryUsecase) Delete(ctx context.Context, id string) error {
	if uc == nil || uc.repo == nil {
		return errors.New("inventory usecase/repo is nil")
	}
	if id == "" {
		return invdom.ErrInvalidMintID
	}
	return uc.repo.Delete(ctx, id)
}

// ============================================================
// Queries
// ============================================================

func (uc *InventoryUsecase) GetByID(ctx context.Context, id string) (invdom.Mint, error) {
	if uc == nil || uc.repo == nil {
		return invdom.Mint{}, errors.New("inventory usecase/repo is nil")
	}
	if id == "" {
		return invdom.Mint{}, invdom.ErrInvalidMintID
	}
	return uc.repo.GetByID(ctx, id)
}
func (uc *InventoryUsecase) ListByTokenBlueprintID(ctx context.Context, tokenBlueprintID string) ([]invdom.Mint, error) {
	if uc == nil || uc.repo == nil {
		return nil, errors.New("inventory usecase/repo is nil")
	}
	return uc.repo.ListByTokenBlueprintID(ctx, tokenBlueprintID)
}

func (uc *InventoryUsecase) ListByProductBlueprintID(ctx context.Context, productBlueprintID string) ([]invdom.Mint, error) {
	if uc == nil || uc.repo == nil {
		return nil, errors.New("inventory usecase/repo is nil")
	}
	return uc.repo.ListByProductBlueprintID(ctx, productBlueprintID)
}

func (uc *InventoryUsecase) ListByModelID(ctx context.Context, modelID string) ([]invdom.Mint, error) {
	if uc == nil || uc.repo == nil {
		return nil, errors.New("inventory usecase/repo is nil")
	}
	return uc.repo.ListByModelID(ctx, modelID)
}

func (uc *InventoryUsecase) ListByTokenAndModelID(ctx context.Context, tokenBlueprintID, modelID string) ([]invdom.Mint, error) {
	if uc == nil || uc.repo == nil {
		return nil, errors.New("inventory usecase/repo is nil")
	}
	return uc.repo.ListByTokenAndModelID(ctx, tokenBlueprintID, modelID)
}

// ============================================================
// Helpers
// ============================================================

func buildInventoryID(productBlueprintID, tokenBlueprintID string) string {
	sanitize := func(s string) string {
		// Firestore docId に "/" が入ると階層扱いになるので repo と揃えて潰す
		s = strings.ReplaceAll(s, "/", "_")
		return s
	}
	pb := sanitize(productBlueprintID)
	tb := sanitize(tokenBlueprintID)
	return pb + "__" + tb
}

func normalizeIDs(raw []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(raw))
	for _, s := range raw {
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

// reserveStockByModelOrder updates reservation fields on Stock[modelID]:
// - ReservedByOrder[orderID] += qty
// - ReservedCount = sum(ReservedByOrder)
//
// It does NOT change Products / Accumulation.
func reserveStockByModelOrder(m *invdom.Mint, modelID, orderID string, qty int) error {
	if m == nil {
		return errors.New("mint is nil")
	}
	if modelID == "" {
		return invdom.ErrInvalidModelID
	}
	if orderID == "" {
		return errors.New("inventory reserve: invalid orderId")
	}
	if qty <= 0 {
		return errors.New("inventory reserve: invalid qty")
	}
	if m.Stock == nil {
		return errors.New("inventory.Mint.Stock is nil (no stock to reserve)")
	}

	ms, ok := m.Stock[modelID]
	if !ok {
		return errors.New("inventory reserve: model stock not found")
	}

	if ms.ReservedByOrder == nil {
		ms.ReservedByOrder = map[string]int{}
	}

	ms.ReservedByOrder[orderID] += qty

	sum := 0
	for _, n := range ms.ReservedByOrder {
		if n <= 0 {
			return errors.New("inventory reserve: invalid reserved qty")
		}
		sum += n
	}
	ms.ReservedCount = sum

	m.Stock[modelID] = ms

	return nil
}
