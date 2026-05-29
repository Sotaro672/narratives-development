// backend/internal/application/usecase/inventory_usecase.go
package usecase

import (
	"context"
	"errors"
	"sort"
	"time"

	invdom "narratives/internal/domain/inventory"
)

type ProductModelResolver interface {
	GetModelIDByProductID(ctx context.Context, productID string) (string, error)
}

type InventoryUsecase struct {
	repo invdom.RepositoryPort

	productModelResolver ProductModelResolver
}

func NewInventoryUsecase(repo invdom.RepositoryPort) *InventoryUsecase {
	return &InventoryUsecase{
		repo:                 repo,
		productModelResolver: nil,
	}
}

func (uc *InventoryUsecase) WithProductModelResolver(
	resolver ProductModelResolver,
) *InventoryUsecase {
	if uc == nil {
		return uc
	}

	uc.productModelResolver = resolver
	return uc
}

// ============================================================
// Upsert entry from Mint
// ============================================================
//
// - mint から在庫へ反映する入口
// - MintUsecase は passed productIDs だけを渡す
// - productID -> modelID の解決と modelID ごとの grouping は InventoryUsecase が担当する
func (uc *InventoryUsecase) UpsertFromMint(
	ctx context.Context,
	tokenBlueprintID string,
	productBlueprintID string,
	productIDs []string,
) ([]invdom.Mint, error) {
	if uc == nil || uc.repo == nil {
		return nil, errors.New("inventory usecase/repo is nil")
	}
	if uc.productModelResolver == nil {
		return nil, errors.New("inventory product model resolver is nil")
	}

	tbID := tokenBlueprintID
	pbID := productBlueprintID

	if tbID == "" {
		return nil, invdom.ErrInvalidTokenBlueprintID
	}
	if pbID == "" {
		return nil, invdom.ErrInvalidProductBlueprintID
	}
	if len(productIDs) == 0 {
		return nil, invdom.ErrInvalidProducts
	}

	seenProducts := make(map[string]struct{}, len(productIDs))
	byModel := make(map[string][]string)

	for _, productID := range productIDs {
		pid := productID
		if pid == "" {
			return nil, invdom.ErrInvalidProducts
		}

		if _, ok := seenProducts[pid]; ok {
			return nil, invdom.ErrInvalidProducts
		}
		seenProducts[pid] = struct{}{}

		modelID, err := uc.productModelResolver.GetModelIDByProductID(ctx, pid)
		if err != nil {
			return nil, err
		}
		if modelID == "" {
			return nil, invdom.ErrInvalidModelID
		}

		byModel[modelID] = append(byModel[modelID], pid)
	}

	modelIDs := make([]string, 0, len(byModel))
	for modelID := range byModel {
		modelIDs = append(modelIDs, modelID)
	}
	sort.Strings(modelIDs)

	out := make([]invdom.Mint, 0, len(modelIDs))
	for _, modelID := range modelIDs {
		groupedProductIDs := byModel[modelID]
		if len(groupedProductIDs) == 0 {
			continue
		}

		inv, err := uc.UpsertFromMintByModel(
			ctx,
			tbID,
			pbID,
			modelID,
			groupedProductIDs,
		)
		if err != nil {
			return nil, err
		}

		out = append(out, inv)
	}

	return out, nil
}

// ============================================================
// Upsert entry from Mint by Model
// ============================================================
//
// - modelID が既に解決済みの場合の低レベル入口
// - 在庫の蓄積は Stock（modelId -> {Products: ...}）で表現する前提
// - repo の atomic upsert（transaction + UNION）に委譲する
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

	return uc.repo.UpsertByModelAndToken(ctx, tbID, pbID, mID, productIDs)
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
// The caller must pass inventoryID and modelID from the order item reservation detail.
// The usecase owns the application-level operation name, while the repository owns the transaction-safe mutation.
func (uc *InventoryUsecase) ReleaseAfterTransfer(
	ctx context.Context,
	inventoryID string,
	modelID string,
	productID string,
	orderID string,
	now time.Time,
) error {
	if uc == nil || uc.repo == nil {
		return errors.New("inventory usecase/repo is nil")
	}

	invID := inventoryID
	mid := modelID
	pid := productID
	oid := orderID

	if invID == "" {
		return invdom.ErrInvalidMintID
	}
	if mid == "" {
		return invdom.ErrInvalidModelID
	}
	if pid == "" {
		return errors.New("inventory transfer result: invalid productId")
	}
	if oid == "" {
		return errors.New("inventory transfer result: invalid orderId")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	_, err := uc.repo.ReleaseReservationAfterTransfer(
		ctx,
		invID,
		mid,
		pid,
		oid,
		now,
	)
	return err
}
