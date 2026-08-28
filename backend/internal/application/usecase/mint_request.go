// backend/internal/application/usecase/mint_request.go
package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	mintdom "narratives/internal/domain/mint"
)

// UpdateRequestInfo は mint request を起票し、productId 単位の mint task を作成します.
//
// 処理:
// - mint request 作成
// - productId 単位の MintProductTask を作成
// - 最初の worker task を enqueue
// - HTTP には即時返却
func (u *MintUsecase) UpdateRequestInfo(
	ctx context.Context,
	productionID string,
	tokenBlueprintID string,
	scheduledBurnDate *string,
) error {
	if u == nil {
		return errors.New("mint usecase is nil")
	}

	if u.mintRepo == nil {
		return errors.New("mint repo is nil")
	}

	if u.mintTaskRepo == nil {
		return errors.New("mint task repo is nil")
	}

	if u.passedProductLister == nil {
		return errors.New("passedProductLister is nil")
	}

	if u.tbRepo == nil {
		return errors.New("tokenBlueprint repo is nil")
	}

	pid := productionID
	if pid == "" {
		return errors.New("productionID is empty")
	}

	tbID := tokenBlueprintID
	if tbID == "" {
		return errors.New("tokenBlueprintID is empty")
	}

	memberID := MemberIDFromContext(ctx)
	if memberID == "" {
		return errors.New("memberID not found in context")
	}

	now := time.Now().UTC()

	tb, err := u.tbRepo.GetByID(ctx, tbID)
	if err != nil {
		return err
	}

	if tb == nil {
		return errors.New("tokenBlueprint not found")
	}

	brandID := tb.BrandID
	if brandID == "" {
		return errors.New("brandID is empty on tokenBlueprint")
	}

	passedProductIDs, err := u.passedProductLister.
		ListPassedProductIDsByProductionID(ctx, pid)
	if err != nil {
		return err
	}

	if len(passedProductIDs) == 0 {
		return errors.New("no passed products for this production")
	}

	mintEntity, err := mintdom.NewMint(
		pid,
		brandID,
		tbID,
		passedProductIDs,
		memberID,
		now,
	)
	if err != nil {
		return err
	}

	mintEntity.ID = pid
	mintEntity.MintedAt = nil

	if scheduledBurnDate != nil {
		if s := *scheduledBurnDate; s != "" {
			t, err := time.ParseInLocation("2006-01-02", s, time.UTC)
			if err != nil {
				return errors.New(
					"invalid scheduledBurnDate format (expected YYYY-MM-DD)",
				)
			}

			utc := t.UTC()
			mintEntity.ScheduledBurnDate = &utc
		}
	}

	if err := mintEntity.MarkQueued(memberID); err != nil {
		return err
	}

	if _, err := u.mintRepo.Create(ctx, mintEntity); err != nil {
		return err
	}

	if _, err := u.mintTaskRepo.CreateTasks(
		ctx,
		pid,
		passedProductIDs,
	); err != nil {
		return fmt.Errorf("create mint product tasks: %w", err)
	}

	if u.mintTaskEnqueuer != nil {
		if err := u.mintTaskEnqueuer.EnqueueMintTask(ctx, pid); err != nil {
			return fmt.Errorf("enqueue mint task: %w", err)
		}
	}

	// handler 側を 202 Accepted + queued DTO に変更するのが理想です。
	return nil
}
