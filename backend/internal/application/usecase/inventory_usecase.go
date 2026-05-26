// backend/internal/application/usecase/inventory_usecase.go

package usecase

import (
	"context"
	"errors"
	"time"

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
// 修正方針:
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
	if len(productIDs) == 0 {
		return invdom.Mint{}, invdom.ErrInvalidProducts
	}
	for _, productID := range productIDs {
		if productID == "" {
			return invdom.Mint{}, invdom.ErrInvalidProducts
		}
	}

	return uc.repo.UpsertByProductBlueprintAndToken(ctx, tbID, pbID, mID, productIDs)
}

// ============================================================
// Reserve by Order
// ============================================================

type ReserveByOrderItem struct {
	InventoryID string
	ModelID     string
	Qty         int
}

// ReserveByOrder adds or overwrites reservation quantity for each model.
// Actual stock mutation is delegated to repository because it must be transactional.
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

		if err := uc.repo.ReserveByOrder(ctx, invID, mid, oid, qty); err != nil {
			return err
		}
	}

	return nil
}

// ============================================================
// Release after transfer
// ============================================================

// ReleaseAfterTransfer removes the transferred product from inventory stock and releases its reservation.
// The usecase owns the application-level operation name, while the repository owns the transaction-safe mutation.
func (uc *InventoryUsecase) ReleaseAfterTransfer(
	ctx context.Context,
	productID string,
	orderID string,
	now time.Time,
) error {
	if uc == nil || uc.repo == nil {
		return errors.New("inventory usecase/repo is nil")
	}

	pid := productID
	oid := orderID

	if pid == "" {
		return errors.New("inventory transfer result: invalid productId")
	}
	if oid == "" {
		return errors.New("inventory transfer result: invalid orderId")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	_, err := uc.repo.ReleaseReservationAfterTransfer(ctx, pid, oid, now)
	return err
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

func (uc *InventoryUsecase) ListByProductBlueprintID(ctx context.Context, productBlueprintID string) ([]invdom.Mint, error) {
	if uc == nil || uc.repo == nil {
		return nil, errors.New("inventory usecase/repo is nil")
	}
	return uc.repo.ListByProductBlueprintID(ctx, productBlueprintID)
}
