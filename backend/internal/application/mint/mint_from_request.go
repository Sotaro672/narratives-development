// backend/internal/application/mint/mint_from_request.go
package mint

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	appusecase "narratives/internal/application/usecase"
	invdom "narratives/internal/domain/inventory"
	mintdom "narratives/internal/domain/mint"
	tokendom "narratives/internal/domain/token"
)

// NOTE:
// - resolveProductBlueprintIDFromProduction は product_blueprint_resolver.go に分離済み
// - loadInspectionBatchByProductionID は inspection_batch_loader.go に分離済み

func validateProductIDs(productIDs []string) error {
	seen := make(map[string]struct{}, len(productIDs))

	for _, id := range productIDs {
		if id == "" {
			return mintdom.ErrInvalidProducts
		}
		if _, ok := seen[id]; ok {
			return mintdom.ErrInvalidProducts
		}
		seen[id] = struct{}{}
	}

	return nil
}

// ============================================================
// POST /mint/requests/{mintRequestId}/mint 用
// ============================================================

// MintFromMintRequest runs onchain mint for an existing mint request (docId = mintRequestID).
func (u *MintUsecase) MintFromMintRequest(ctx context.Context, mintRequestID string) (*tokendom.MintResult, error) {
	if u == nil {
		return nil, errors.New("mint usecase is nil")
	}
	if mintRequestID == "" {
		return nil, errors.New("mintRequestID is empty")
	}
	if u.tokenMinter == nil {
		return nil, errors.New("token minter is nil")
	}
	if u.mintRepo == nil {
		return nil, errors.New("mint repo is nil")
	}
	if u.mintResultMapper == nil {
		return nil, errors.New("mint result mapper is nil")
	}

	mintEntValue, err := u.mintRepo.GetByID(ctx, mintRequestID)
	if err != nil {
		return nil, err
	}
	mintEnt := &mintEntValue

	passedProductIDs := mintEnt.Products
	if err := validateProductIDs(passedProductIDs); err != nil {
		return nil, err
	}

	actorID := mintEnt.CreatedBy
	if actorID == "" {
		actorID = appusecase.MemberIDFromContext(ctx)
	}
	if actorID == "" {
		return nil, errors.New("actorID is missing (mint.createdBy and context memberId are empty)")
	}

	tbID := mintEnt.TokenBlueprintID
	if tbID == "" {
		return nil, errors.New("tokenBlueprintID is empty on mint")
	}

	pbID := u.resolveProductBlueprintIDFromProduction(ctx, mintRequestID)
	if pbID == "" {
		return nil, errors.New("productBlueprintID is empty (cannot upsert inventory)")
	}

	if len(passedProductIDs) == 0 {
		return nil, errors.New("no passed products for this mint request")
	}

	var result *tokendom.MintResult

	if mintEnt.Minted {
		result = u.mintResultMapper.FromMint(*mintEnt)
	} else {
		if u.tbMetadataEnsurer == nil {
			return nil, fmt.Errorf("tokenBlueprint metadata ensurer is nil")
		}
		if u.tbRepo == nil {
			return nil, fmt.Errorf("tokenBlueprint repo is nil")
		}

		tb, err := u.tbRepo.GetByID(ctx, tbID)
		if err != nil {
			return nil, err
		}
		if tb == nil {
			return nil, fmt.Errorf("tokenBlueprint not found (id=%s)", tbID)
		}

		updated, err := u.tbMetadataEnsurer.EnsureMetadataURI(ctx, tb, actorID)
		if err != nil {
			return nil, err
		}
		if updated == nil {
			updated = tb
		}

		uri := strings.TrimSpace(updated.MetadataURI)
		if uri == "" {
			return nil, fmt.Errorf("metadataUri is empty after ensure (tokenBlueprintId=%s)", tbID)
		}

		result, err = u.tokenMinter.MintFromMintRequest(ctx, mintRequestID)
		if err != nil {
			return nil, err
		}
		if result == nil {
			return nil, fmt.Errorf("onchain mint succeeded but result is nil (mintRequestId=%s)", mintRequestID)
		}

		_ = u.markTokenBlueprintMinted(ctx, tbID, actorID)

		if u.mintRepo != nil {
			if updater, ok := any(u.mintRepo).(interface {
				Update(ctx context.Context, m mintdom.Mint) (mintdom.Mint, error)
			}); ok {
				now := time.Now().UTC()
				m := *mintEnt

				m.ID = mintRequestID
				m.Minted = true
				m.MintedAt = &now

				_ = u.mintResultMapper.ApplyOnchainResult(&m, result)

				_, _ = updater.Update(ctx, m)
			}
		}
	}

	if u.inventoryUC == nil {
		return nil, errors.New("inventory usecase is nil (cannot upsert inventory)")
	}

	batch, berr := u.loadInspectionBatchByProductionID(ctx, mintRequestID)
	if berr != nil || batch == nil {
		if berr != nil {
			return nil, berr
		}
		return nil, errors.New("inspection batch is nil")
	}

	passedSet := make(map[string]struct{}, len(passedProductIDs))
	for _, p := range passedProductIDs {
		passedSet[p] = struct{}{}
	}

	byModel := map[string][]string{}
	for _, it := range batch.Inspections {
		pid := it.ProductID
		if pid == "" {
			return nil, mintdom.ErrInvalidProducts
		}
		if _, ok := passedSet[pid]; !ok {
			continue
		}

		mid := it.ModelID
		if mid == "" {
			continue
		}

		byModel[mid] = append(byModel[mid], pid)
	}

	modelIDs := make([]string, 0, len(byModel))
	for mid := range byModel {
		modelIDs = append(modelIDs, mid)
	}
	sort.Strings(modelIDs)

	if len(modelIDs) == 0 {
		return nil, errors.New("no model groups found from inspection batch for passed products")
	}

	for _, mid := range modelIDs {
		pids := byModel[mid]
		if err := validateProductIDs(pids); err != nil {
			return nil, err
		}
		if len(pids) == 0 {
			continue
		}

		invEnt, invErr := u.inventoryUC.UpsertFromMintByModel(ctx, tbID, pbID, mid, pids)
		if invErr != nil {
			return nil, invErr
		}

		var _ invdom.Mint = invEnt
	}

	return result, nil
}
