// backend/internal/application/mint/update_request_info.go
package mint

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	appusecase "narratives/internal/application/usecase"
	inspectiondom "narratives/internal/domain/inspection"
	mintdom "narratives/internal/domain/mint"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

// ============================================================
// ★ 修復: /mint/inspections/{id}/request を「従来どおりミントまで実行」に戻す
// ============================================================

func (u *MintUsecase) UpdateRequestInfo(
	ctx context.Context,
	productionID string,
	tokenBlueprintID string,
	scheduledBurnDate *string,
) (inspectiondom.InspectionBatch, error) {

	var empty inspectiondom.InspectionBatch

	if u == nil {
		return empty, errors.New("mint usecase is nil")
	}
	if u.inspRepo == nil {
		return empty, errors.New("inspection repo is nil")
	}
	if u.mintRepo == nil {
		return empty, errors.New("mint repo is nil")
	}
	if u.passedProductLister == nil {
		return empty, errors.New("passedProductLister is nil")
	}
	if u.tbRepo == nil {
		return empty, errors.New("tokenBlueprint repo is nil")
	}

	pid := strings.TrimSpace(productionID)
	if pid == "" {
		return empty, errors.New("productionID is empty")
	}

	tbID := strings.TrimSpace(tokenBlueprintID)
	if tbID == "" {
		return empty, errors.New("tokenBlueprintID is empty")
	}

	memberID := strings.TrimSpace(appusecase.MemberIDFromContext(ctx))
	if memberID == "" {
		return empty, errors.New("memberID not found in context")
	}

	now := time.Now().UTC()

	tb, err := u.tbRepo.GetByID(ctx, tbID)
	if err != nil {
		return empty, err
	}
	brandID := strings.TrimSpace(tb.BrandID)
	if brandID == "" {
		return empty, errors.New("brandID is empty on tokenBlueprint")
	}

	passedProductIDs, err := u.passedProductLister.ListPassedProductIDsByProductionID(ctx, pid)
	if err != nil {
		return empty, err
	}
	if len(passedProductIDs) == 0 {
		return empty, errors.New("no passed products for this production")
	}

	// mint entity 作成
	mintEntity, err := mintdom.NewMint(
		pid,
		brandID,
		tbID,
		passedProductIDs,
		memberID,
		now,
	)
	if err != nil {
		return empty, err
	}

	// ★ Policy A: docId = productionId（必ず揃える）
	mintEntity.ID = pid
	// setIfExistsString は mint_result_mapper.go 側の共通関数を利用（DuplicateDecl 回避）
	setIfExistsString(&mintEntity, "InspectionID", pid)

	// minted は request 作成時は必ず false（念のため）
	mintEntity.Minted = false
	mintEntity.MintedAt = nil

	if scheduledBurnDate != nil {
		if s := strings.TrimSpace(*scheduledBurnDate); s != "" {
			t, err := time.ParseInLocation("2006-01-02", s, time.UTC)
			if err != nil {
				return empty, errors.New("invalid scheduledBurnDate format (expected YYYY-MM-DD)")
			}
			utc := t.UTC()
			mintEntity.ScheduledBurnDate = &utc
		}
	}

	// Create は Policy A の docId 固定（productionId）で保存される
	savedMint, err := u.mintRepo.Create(ctx, mintEntity)
	if err != nil {
		return empty, err
	}

	mid := strings.TrimSpace(savedMint.ID)
	if mid == "" {
		return empty, errors.New("saved mintID is empty")
	}

	// Inspection に mintId を紐付け（= productionId と同値になる想定）
	batch, err := u.inspRepo.UpdateMintID(ctx, pid, &mid)
	if err != nil {
		return empty, err
	}

	log.Printf("[mint_usecase] UpdateRequestInfo done (request created) productionId=%q mintId=%q tokenBlueprintId=%q passedProducts=%d",
		pid, mid, tbID, len(passedProductIDs),
	)

	// ✅ 自動 mint は best-effort（失敗しても request 作成は成功扱いで返す）
	if u.tokenMinter == nil {
		log.Printf("[mint_usecase] UpdateRequestInfo auto MintFromMintRequest skipped reason=token_minter_nil productionId=%q", pid)
		return batch, nil
	}

	if _, err := u.MintFromMintRequest(ctx, pid); err != nil {
		log.Printf("[mint_usecase] UpdateRequestInfo auto MintFromMintRequest failed productionId=%q err=%v", pid, err)
		// ★ ここで return err しない（500 を止める）
	}

	return batch, nil
}

// 参照を明示（未使用import回避のため）
// ※ 上の処理では tbdom は tbRepo.Update 入力型でのみ参照される可能性があるため残す
var _ = tbdom.UpdateTokenBlueprintInput{}
