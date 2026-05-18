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
	invdom "narratives/internal/domain/inventory" // inventory 連携
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

	// 1) Mint を事前取得
	// NotFound 以外のエラー（例: inconsistent minted/mintedAt）は握り潰さず上位へ返す。
	mintEntValue, err := u.mintRepo.GetByID(ctx, mintRequestID)
	if err != nil {
		return nil, err
	}
	mintEnt := &mintEntValue

	passedProductIDs := mintEnt.Products
	if err := validateProductIDs(passedProductIDs); err != nil {
		return nil, err
	}

	// actorID は mint.createdBy を優先
	actorID := mintEnt.CreatedBy
	if actorID == "" {
		actorID = appusecase.MemberIDFromContext(ctx)
	}
	if actorID == "" {
		return nil, errors.New("actorID is missing (mint.createdBy and context memberId are empty)")
	}

	// tokenBlueprintId は必須（以後の TB 更新・inventory 反映に必要）
	tbID := mintEnt.TokenBlueprintID
	if tbID == "" {
		return nil, errors.New("tokenBlueprintID is empty on mint")
	}

	// productBlueprintId は必須（inventory docId に必要）
	// resolveProductBlueprintIDFromProduction は product_blueprint_resolver.go に分離済み
	pbID := u.resolveProductBlueprintIDFromProduction(ctx, mintRequestID)
	if pbID == "" {
		return nil, errors.New("productBlueprintID is empty (cannot upsert inventory)")
	}

	// passedProducts が空なら何もする意味が無いのでエラー（期待値：後続の在庫が増えないのは異常）
	if len(passedProductIDs) == 0 {
		return nil, errors.New("no passed products for this mint request")
	}

	// 0) 二重mint防止（ただし inventory upsert は必ず実行する）
	var result *tokendom.MintResult

	if mintEnt.Minted {
		// オンチェーンはスキップ、ただし inventory 反映は行う
		result = u.mintResultMapper.FromMint(*mintEnt)
	} else {
		// ============================================================
		// 2) オンチェーンミント実行の「直前」に事前処理を挿入
		// - Firebase Storage 移行後は GCS bucket/.keep は必須ではない
		// - metadataUri を空なら生成して token_blueprints に永続化
		// ============================================================

		// 2-0) Ensure keep objects (optional)
		//
		// Firebase Storage 移行後:
		// - tokenBlueprint icon / contents は frontend が Firebase Storage へ直接 upload する
		// - backend は GCS bucket / .keep object を必須保証しない
		// - tbBucketEnsurer は旧GCS互換用の任意依存として扱う
		if u.tbBucketEnsurer != nil {
			if err := u.tbBucketEnsurer.EnsureKeepObjects(ctx, tbID); err != nil {
				return nil, err
			}
		}

		// 2-1) Ensure metadataUri (must exist before TokenUsecase mint)
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

		// 2-2) オンチェーンミント実行
		result, err = u.tokenMinter.MintFromMintRequest(ctx, mintRequestID)
		if err != nil {
			return nil, err
		}
		if result == nil {
			return nil, fmt.Errorf("onchain mint succeeded but result is nil (mintRequestId=%s)", mintRequestID)
		}

		// 3) TokenBlueprint minted=true（未mint の場合のみ）
		_ = u.markTokenBlueprintMinted(ctx, tbID, actorID)

		// 4) mints 側 minted/mintedAt + onChainTxSignature を更新（失敗しても握り潰す）
		if u.mintRepo != nil {
			if updater, ok := any(u.mintRepo).(interface {
				Update(ctx context.Context, m mintdom.Mint) (mintdom.Mint, error)
			}); ok {
				now := time.Now().UTC()
				m := *mintEnt

				// Policy A: docId を必ず mintRequestId に揃える
				m.ID = mintRequestID
				m.Minted = true
				m.MintedAt = &now

				// 署名は Firestore 実データの onChainTxSignature / Go の OnChainTxSignature に一本化する。
				_ = u.mintResultMapper.ApplyOnchainResult(&m, result)

				_, _ = updater.Update(ctx, m)
			}
		}
	}

	// 5) inventories Upsert（modelId ごとに UpsertFromMintByModel）
	// 期待値: productBlueprintId/tokenBlueprintId は必須。空ならスキップではなくエラーを返す。
	if u.inventoryUC == nil {
		return nil, errors.New("inventory usecase is nil (cannot upsert inventory)")
	}

	// loadInspectionBatchByProductionID は inspection_batch_loader.go に分離済み
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
			// modelId がないデータは upsert できない
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

		// invdom を import しているため、明示的に参照（lint/IDEの推論用）
		var _ invdom.Mint = invEnt
	}

	return result, nil
}
