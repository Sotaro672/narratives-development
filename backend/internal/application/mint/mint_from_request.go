// backend/internal/application/mint/mint_from_request.go
package mint

import (
	"context"
	"errors"
	"fmt"
	"log"
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

func normalizeProductIDs(raw []string) []string {
	if len(raw) == 0 {
		return []string{}
	}

	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))

	for _, id := range raw {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}

		seen[id] = struct{}{}
		out = append(out, id)
	}

	sort.Strings(out)
	return out
}

// ============================================================
// ★ 修復: POST /mint/requests/{mintRequestId}/mint 用
// ============================================================

// MintFromMintRequest runs onchain mint for an existing mint request (docId = mintRequestID).
func (u *MintUsecase) MintFromMintRequest(ctx context.Context, mintRequestID string) (*tokendom.MintResult, error) {
	start := time.Now()

	log.Printf(
		"[mint_usecase] MintFromMintRequest start mintRequestId=%q usecase_nil=%t tokenMinter_nil=%t mintRepo_nil=%t tbRepo_nil=%t inventoryUC_nil=%t",
		mintRequestID,
		u == nil,
		u == nil || u.tokenMinter == nil,
		u == nil || u.mintRepo == nil,
		u == nil || u.tbRepo == nil,
		u == nil || u.inventoryUC == nil,
	)

	if u == nil {
		log.Printf("[mint_usecase] MintFromMintRequest abort reason=usecase_nil elapsed=%s", time.Since(start))
		return nil, errors.New("mint usecase is nil")
	}
	if mintRequestID == "" {
		log.Printf("[mint_usecase] MintFromMintRequest abort reason=empty_mintRequestId elapsed=%s", time.Since(start))
		return nil, errors.New("mintRequestID is empty")
	}
	if u.tokenMinter == nil {
		log.Printf("[mint_usecase] MintFromMintRequest abort reason=token_minter_nil mintRequestId=%q elapsed=%s", mintRequestID, time.Since(start))
		return nil, errors.New("token minter is nil")
	}
	if u.mintRepo == nil {
		log.Printf("[mint_usecase] MintFromMintRequest abort reason=mintRepo_nil mintRequestId=%q elapsed=%s", mintRequestID, time.Since(start))
		return nil, errors.New("mint repo is nil")
	}
	if u.mintRepoAdapter == nil {
		log.Printf("[mint_usecase] MintFromMintRequest abort reason=mintRepoAdapter_nil mintRequestId=%q elapsed=%s", mintRequestID, time.Since(start))
		return nil, errors.New("mint repo adapter is nil")
	}
	if u.mintResultMapper == nil {
		log.Printf("[mint_usecase] MintFromMintRequest abort reason=mintResultMapper_nil mintRequestId=%q elapsed=%s", mintRequestID, time.Since(start))
		return nil, errors.New("mint result mapper is nil")
	}

	// 1) Mint を事前取得
	// NotFound 以外のエラー（例: inconsistent minted/mintedAt）は握り潰さず上位へ返す。
	mintEnt, loadErr := u.mintRepoAdapter.Load(ctx, mintRequestID)
	if loadErr != nil {
		log.Printf("[mint_usecase] MintFromMintRequest abort reason=mint_load_error mintRequestId=%q err=%v elapsed=%s", mintRequestID, loadErr, time.Since(start))
		return nil, loadErr
	}
	if mintEnt == nil {
		log.Printf("[mint_usecase] MintFromMintRequest abort reason=mint_not_found mintRequestId=%q elapsed=%s", mintRequestID, time.Since(start))
		return nil, mintdom.ErrNotFound
	}

	passedProductIDs := normalizeProductIDs(mintEnt.Products)

	log.Printf(
		"[mint_usecase] MintFromMintRequest loaded mint id=%q tokenBlueprintId=%q minted=%t products=%d createdBy=%q",
		mintEnt.ID,
		mintEnt.TokenBlueprintID,
		mintEnt.Minted,
		len(passedProductIDs),
		mintEnt.CreatedBy,
	)

	// actorID は mint.createdBy を優先
	actorID := mintEnt.CreatedBy
	if actorID == "" {
		actorID = appusecase.MemberIDFromContext(ctx)
	}
	if actorID == "" {
		log.Printf("[mint_usecase] MintFromMintRequest abort reason=actor_missing mintRequestId=%q elapsed=%s", mintRequestID, time.Since(start))
		return nil, errors.New("actorID is missing (mint.createdBy and context memberId are empty)")
	}
	log.Printf("[mint_usecase] MintFromMintRequest actorId=%q mintRequestId=%q", actorID, mintRequestID)

	// tokenBlueprintId は必須（以後の TB 更新・inventory 反映に必要）
	tbID := mintEnt.TokenBlueprintID
	if tbID == "" {
		log.Printf("[mint_usecase] MintFromMintRequest abort reason=empty_tokenBlueprintId mintRequestId=%q elapsed=%s", mintRequestID, time.Since(start))
		return nil, errors.New("tokenBlueprintID is empty on mint")
	}

	// productBlueprintId は必須（inventory docId に必要）
	// resolveProductBlueprintIDFromProduction は product_blueprint_resolver.go に分離済み
	pbID := u.resolveProductBlueprintIDFromProduction(ctx, mintRequestID)
	if pbID == "" {
		log.Printf("[mint_usecase] MintFromMintRequest abort reason=empty_productBlueprintId mintRequestId=%q tokenBlueprintId=%q elapsed=%s", mintRequestID, tbID, time.Since(start))
		return nil, errors.New("productBlueprintID is empty (cannot upsert inventory)")
	}

	// passedProducts が空なら何もする意味が無いのでエラー（期待値：後続の在庫が増えないのは異常）
	if len(passedProductIDs) == 0 {
		log.Printf("[mint_usecase] MintFromMintRequest abort reason=no_passed_products mintRequestId=%q tokenBlueprintId=%q productBlueprintId=%q elapsed=%s",
			mintRequestID, tbID, pbID, time.Since(start),
		)
		return nil, errors.New("no passed products for this mint request")
	}

	// 0) 二重mint防止（ただし inventory upsert は必ず実行する）
	var result *tokendom.MintResult
	var onchainErr error

	if mintEnt.Minted {
		// オンチェーンはスキップ、ただし inventory 反映は行う
		existing := u.mintResultMapper.FromMint(*mintEnt)
		log.Printf(
			"[mint_usecase] MintFromMintRequest skip onchain reason=already_minted mintRequestId=%q signature=%q mintAddress=%q elapsed=%s",
			mintRequestID,
			func() string {
				if existing == nil {
					return ""
				}
				return existing.Signature
			}(),
			func() string {
				if existing == nil {
					return ""
				}
				return existing.MintAddress
			}(),
			time.Since(start),
		)
		result = existing
	} else {
		// ============================================================
		// ★ 2) オンチェーンミント実行の「直前」に事前処理を挿入
		//   - Firebase Storage 移行後は GCS bucket/.keep は必須ではない
		//   - metadataUri を空なら生成して token_blueprints に永続化
		// ============================================================

		// 2-0) Ensure keep objects (optional)
		//
		// Firebase Storage 移行後:
		// - tokenBlueprint icon / contents は frontend が Firebase Storage へ直接 upload する
		// - backend は GCS bucket / .keep object を必須保証しない
		// - tbBucketEnsurer は旧GCS互換用の任意依存として扱う
		if u.tbBucketEnsurer != nil {
			log.Printf(
				"[mint_usecase] MintFromMintRequest ensure keep start mintRequestId=%q tokenBlueprintId=%q",
				mintRequestID, tbID,
			)

			prepStart := time.Now()
			if err := u.tbBucketEnsurer.EnsureKeepObjects(ctx, tbID); err != nil {
				log.Printf(
					"[mint_usecase] MintFromMintRequest abort reason=ensure_keep_failed mintRequestId=%q tokenBlueprintId=%q err=%v elapsed=%s totalElapsed=%s",
					mintRequestID, tbID, err, time.Since(prepStart), time.Since(start),
				)
				return nil, err
			}

			log.Printf(
				"[mint_usecase] MintFromMintRequest ensure keep ok mintRequestId=%q tokenBlueprintId=%q elapsed=%s",
				mintRequestID, tbID, time.Since(prepStart),
			)
		} else {
			log.Printf(
				"[mint_usecase] MintFromMintRequest skip ensure keep reason=tbBucketEnsurer_nil firebase_storage_mode mintRequestId=%q tokenBlueprintId=%q",
				mintRequestID, tbID,
			)
		}

		// 2-1) Ensure metadataUri (must exist before TokenUsecase mint)
		if u.tbMetadataEnsurer == nil {
			log.Printf(
				"[mint_usecase] MintFromMintRequest abort reason=tbMetadataEnsurer_nil mintRequestId=%q tokenBlueprintId=%q actorId=%q elapsed=%s",
				mintRequestID, tbID, actorID, time.Since(start),
			)
			return nil, fmt.Errorf("tokenBlueprint metadata ensurer is nil")
		}
		if u.tbRepo == nil {
			log.Printf(
				"[mint_usecase] MintFromMintRequest abort reason=tbRepo_nil mintRequestId=%q tokenBlueprintId=%q actorId=%q elapsed=%s",
				mintRequestID, tbID, actorID, time.Since(start),
			)
			return nil, fmt.Errorf("tokenBlueprint repo is nil")
		}

		log.Printf(
			"[mint_usecase] MintFromMintRequest ensure metadata start mintRequestId=%q tokenBlueprintId=%q actorId=%q",
			mintRequestID, tbID, actorID,
		)

		metaStart := time.Now()

		tb, err := u.tbRepo.GetByID(ctx, tbID)
		if err != nil {
			log.Printf(
				"[mint_usecase] MintFromMintRequest abort reason=tb_get_failed mintRequestId=%q tokenBlueprintId=%q actorId=%q err=%v elapsed=%s totalElapsed=%s",
				mintRequestID, tbID, actorID, err, time.Since(metaStart), time.Since(start),
			)
			return nil, err
		}
		if tb == nil {
			log.Printf(
				"[mint_usecase] MintFromMintRequest abort reason=tb_not_found mintRequestId=%q tokenBlueprintId=%q actorId=%q elapsed=%s totalElapsed=%s",
				mintRequestID, tbID, actorID, time.Since(metaStart), time.Since(start),
			)
			return nil, fmt.Errorf("tokenBlueprint not found (id=%s)", tbID)
		}

		updated, err := u.tbMetadataEnsurer.EnsureMetadataURI(ctx, tb, actorID)
		if err != nil {
			log.Printf(
				"[mint_usecase] MintFromMintRequest abort reason=ensure_metadata_failed mintRequestId=%q tokenBlueprintId=%q actorId=%q err=%v elapsed=%s totalElapsed=%s",
				mintRequestID, tbID, actorID, err, time.Since(metaStart), time.Since(start),
			)
			return nil, err
		}
		if updated == nil {
			updated = tb
		}

		uri := strings.TrimSpace(updated.MetadataURI)
		if uri == "" {
			log.Printf(
				"[mint_usecase] MintFromMintRequest abort reason=ensure_metadata_empty mintRequestId=%q tokenBlueprintId=%q actorId=%q elapsed=%s totalElapsed=%s",
				mintRequestID, tbID, actorID, time.Since(metaStart), time.Since(start),
			)
			return nil, fmt.Errorf("metadataUri is empty after ensure (tokenBlueprintId=%s)", tbID)
		}

		log.Printf(
			"[mint_usecase] MintFromMintRequest ensure metadata ok mintRequestId=%q tokenBlueprintId=%q uri=%q elapsed=%s",
			mintRequestID, tbID, uri, time.Since(metaStart),
		)

		// 2-2) オンチェーンミント実行
		log.Printf("[mint_usecase] MintFromMintRequest onchain mint start mintRequestId=%q", mintRequestID)

		onchainStart := time.Now()
		result, onchainErr = u.tokenMinter.MintFromMintRequest(ctx, mintRequestID)
		onchainElapsed := time.Since(onchainStart)

		if onchainErr != nil {
			log.Printf(
				"[mint_usecase] MintFromMintRequest onchain mint error mintRequestId=%q err=%v elapsed=%s totalElapsed=%s",
				mintRequestID, onchainErr, onchainElapsed, time.Since(start),
			)
			return nil, onchainErr
		}
		if result == nil {
			log.Printf(
				"[mint_usecase] MintFromMintRequest onchain mint invalid_result mintRequestId=%q elapsed=%s totalElapsed=%s",
				mintRequestID, onchainElapsed, time.Since(start),
			)
			return nil, fmt.Errorf("onchain mint succeeded but result is nil (mintRequestId=%s)", mintRequestID)
		}

		log.Printf(
			"[mint_usecase] MintFromMintRequest onchain mint ok mintRequestId=%q elapsed=%s signature=%q mintAddress=%q slot=%d",
			mintRequestID,
			onchainElapsed,
			result.Signature,
			result.MintAddress,
			result.Slot,
		)

		// 3) TokenBlueprint minted=true（未mint の場合のみ）
		tbStart := time.Now()
		errTB := u.markTokenBlueprintMinted(ctx, tbID, actorID)
		tbElapsed := time.Since(tbStart)

		if errTB != nil {
			log.Printf(
				"[mint_usecase] MintFromMintRequest markTokenBlueprintMinted error tokenBlueprintId=%q actorId=%q err=%v elapsed=%s",
				tbID, actorID, errTB, tbElapsed,
			)
		} else {
			log.Printf(
				"[mint_usecase] MintFromMintRequest markTokenBlueprintMinted ok tokenBlueprintId=%q actorId=%q elapsed=%s",
				tbID, actorID, tbElapsed,
			)
		}

		// 4) mints 側 minted/mintedAt + onChainTxSignature を更新（失敗してもログだけ）
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

				updStart := time.Now()
				updated, errUpd := updater.Update(ctx, m)
				updElapsed := time.Since(updStart)

				if errUpd != nil {
					log.Printf(
						"[mint_usecase] MintFromMintRequest mintRepo.Update error mintRequestId=%q err=%v elapsed=%s",
						mintRequestID, errUpd, updElapsed,
					)
				} else {
					log.Printf(
						"[mint_usecase] MintFromMintRequest mintRepo.Update ok mintRequestId=%q minted=%t mintedAt=%v elapsed=%s",
						mintRequestID, updated.Minted, updated.MintedAt, updElapsed,
					)
				}
			} else {
				log.Printf("[mint_usecase] MintFromMintRequest skip mintRepo.Update reason=no_update_method")
			}
		} else {
			log.Printf("[mint_usecase] MintFromMintRequest skip mintRepo.Update reason=mintRepo_nil")
		}
	}

	// 5) inventories Upsert（modelId ごとに UpsertFromMintByModel）
	// 期待値: productBlueprintId/tokenBlueprintId は必須。空ならスキップではなくエラーを返す。
	if u.inventoryUC == nil {
		log.Printf("[mint_usecase] MintFromMintRequest abort reason=inventoryUC_nil mintRequestId=%q elapsed=%s", mintRequestID, time.Since(start))
		return nil, errors.New("inventory usecase is nil (cannot upsert inventory)")
	}

	log.Printf(
		"[mint_usecase] MintFromMintRequest inventory upsert(by-model) start mintRequestId=%q tokenBlueprintId=%q productBlueprintId=%q passedProducts=%d",
		mintRequestID, tbID, pbID, len(passedProductIDs),
	)

	// loadInspectionBatchByProductionID は inspection_batch_loader.go に分離済み
	batch, berr := u.loadInspectionBatchByProductionID(ctx, mintRequestID)
	if berr != nil || batch == nil {
		log.Printf(
			"[mint_usecase] MintFromMintRequest abort reason=inspection_load_failed mintRequestId=%q err=%v elapsed=%s",
			mintRequestID, berr, time.Since(start),
		)
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
			continue
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
		log.Printf(
			"[mint_usecase] MintFromMintRequest abort reason=no_model_groups mintRequestId=%q passed=%d inspections=%d elapsed=%s",
			mintRequestID, len(passedProductIDs), len(batch.Inspections), time.Since(start),
		)
		return nil, errors.New("no model groups found from inspection batch for passed products")
	}

	for _, mid := range modelIDs {
		pids := normalizeProductIDs(byModel[mid])
		if len(pids) == 0 {
			continue
		}

		invStart := time.Now()
		invEnt, invErr := u.inventoryUC.UpsertFromMintByModel(ctx, tbID, pbID, mid, pids)
		invElapsed := time.Since(invStart)

		if invErr != nil {
			log.Printf(
				"[mint_usecase] MintFromMintRequest inventory upsert(by-model) error mintRequestId=%q tokenBlueprintId=%q productBlueprintId=%q modelId=%q products=%d err=%v elapsed=%s",
				mintRequestID, tbID, pbID, mid, len(pids), invErr, invElapsed,
			)
			return nil, invErr
		}

		accumulation := len(pids)

		// invdom を import しているため、明示的に参照（lint/IDEの推論用）
		var _ invdom.Mint = invEnt

		if invEnt.Stock != nil {
			if ms, ok := invEnt.Stock[mid]; ok {
				accumulation = ms.Accumulation
			}
		}

		log.Printf(
			"[mint_usecase] MintFromMintRequest inventory upsert(by-model) ok inventoryId=%q modelId=%q accumulation=%d elapsed=%s",
			invEnt.ID,
			mid,
			accumulation,
			invElapsed,
		)
	}

	log.Printf(
		"[mint_usecase] MintFromMintRequest done mintRequestId=%q elapsed=%s result_nil=%t onchain_err_nil=%t",
		mintRequestID, time.Since(start), result == nil, onchainErr == nil,
	)

	return result, nil
}
