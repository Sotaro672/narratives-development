// backend/internal/application/mint/update_request_info.go
package mint

import (
	"context"
	"errors"
	"time"

	appusecase "narratives/internal/application/usecase"
	inspectiondom "narratives/internal/domain/inspection"
	mintdom "narratives/internal/domain/mint"
)

// ============================================================
// /mint/inspections/{id}/request を「従来どおりミントまで実行」に戻す
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

	pid := productionID
	if pid == "" {
		return empty, errors.New("productionID is empty")
	}

	tbID := tokenBlueprintID
	if tbID == "" {
		return empty, errors.New("tokenBlueprintID is empty")
	}

	memberID := appusecase.MemberIDFromContext(ctx)
	if memberID == "" {
		return empty, errors.New("memberID not found in context")
	}

	now := time.Now().UTC()

	tb, err := u.tbRepo.GetByID(ctx, tbID)
	if err != nil {
		return empty, err
	}
	if tb == nil {
		return empty, errors.New("tokenBlueprint not found")
	}

	brandID := tb.BrandID
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

	// Policy A: production / inspection / mint の docId は同一値として扱う。
	mintEntity.ID = pid

	// minted は request 作成時は必ず false
	mintEntity.Minted = false
	mintEntity.MintedAt = nil

	if scheduledBurnDate != nil {
		if s := *scheduledBurnDate; s != "" {
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

	mid := savedMint.ID
	if mid == "" {
		return empty, errors.New("saved mintID is empty")
	}

	// Inspection に mintId を紐付け（= productionId と同値になる想定）
	batch, err := u.inspRepo.UpdateMintID(ctx, pid, &mid)
	if err != nil {
		return empty, err
	}

	// 自動 mint は best-effort（失敗しても request 作成は成功扱いで返す）
	if u.tokenMinter == nil {
		return batch, nil
	}

	if _, err := u.MintFromMintRequest(ctx, pid); err != nil {
		// ここで return err しない（500 を止める）
		return batch, nil
	}

	return batch, nil
}
