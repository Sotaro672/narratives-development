// backend/internal/application/usecase/mint_parent_progress.go
package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	mintdom "narratives/internal/domain/mint"
	tokendom "narratives/internal/domain/token"
)

func (u *MintUsecase) updateParentAndMaybeEnqueueNext(
	ctx context.Context,
	mintEnt *mintdom.Mint,
	reqTBID string,
	actorID string,
	latestSignature string,
) error {
	if u == nil {
		return errors.New("mint usecase is nil")
	}

	if mintEnt == nil {
		return errors.New("mint entity is nil")
	}

	if u.mintTaskRepo == nil {
		return errors.New("mint task repo is nil")
	}

	if u.mintRepo == nil {
		return errors.New("mint repo is nil")
	}

	tasks, err := u.mintTaskRepo.ListByMintID(
		ctx,
		mintEnt.ID,
	)
	if err != nil {
		return fmt.Errorf(
			"list mint product tasks: %w",
			err,
		)
	}

	total := len(tasks)
	if total == 0 {
		return errors.New("mint has no product tasks")
	}

	mintedCount := 0
	fatalCount := 0
	retryableCount := 0

	for _, task := range tasks {
		switch task.Status {
		case mintdom.MintProductTaskStatusMinted:
			mintedCount++

		case mintdom.MintProductTaskStatusFailedFatal:
			fatalCount++

		case mintdom.MintProductTaskStatusPending,
			mintdom.MintProductTaskStatusFailedRetryable:
			retryableCount++
		}
	}

	if mintedCount == total {
		if err := mintEnt.MarkMinted(
			time.Now().UTC(),
			latestSignature,
		); err != nil {
			return err
		}

		if _, err := u.mintRepo.Update(ctx, *mintEnt); err != nil {
			return fmt.Errorf(
				"mark parent minted: %w",
				err,
			)
		}

		if u.tbMintMarker != nil && reqTBID != "" {
			_, _ = u.tbMintMarker.MarkTokenBlueprintMinted(
				ctx,
				reqTBID,
				actorID,
			)
		}

		return nil
	}

	if fatalCount > 0 && mintedCount+fatalCount == total {
		if err := mintEnt.MarkFailedFatal(); err != nil {
			return err
		}

		if _, err := u.mintRepo.Update(ctx, *mintEnt); err != nil {
			return fmt.Errorf(
				"mark parent failed fatal: %w",
				err,
			)
		}

		return nil
	}

	if mintedCount > 0 {
		if err := mintEnt.MarkPartiallyMinted(); err != nil {
			return err
		}
	} else {
		if err := mintEnt.MarkMinting(); err != nil {
			return err
		}
	}

	if _, err := u.mintRepo.Update(ctx, *mintEnt); err != nil {
		return fmt.Errorf(
			"update parent mint progress: %w",
			err,
		)
	}

	if retryableCount > 0 && u.mintTaskEnqueuer != nil {
		if err := u.mintTaskEnqueuer.EnqueueMintTask(
			ctx,
			mintEnt.ID,
		); err != nil {
			return fmt.Errorf(
				"enqueue next mint task: %w",
				err,
			)
		}
	}

	return nil
}

func (u *MintUsecase) finalizeMintIfAllTasksCompleted(
	ctx context.Context,
	mintEnt *mintdom.Mint,
	reqTBID string,
	actorID string,
) (*tokendom.MintResult, error) {
	if u == nil {
		return nil, errors.New("mint usecase is nil")
	}

	if mintEnt == nil {
		return nil, errors.New("mint entity is nil")
	}

	tasks, err := u.mintTaskRepo.ListByMintID(
		ctx,
		mintEnt.ID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"list mint product tasks: %w",
			err,
		)
	}

	if len(tasks) == 0 {
		return nil, mintdom.ErrMintProductTaskNotFound
	}

	latestSignature := ""
	allMinted := true

	for _, task := range tasks {
		if task.Status != mintdom.MintProductTaskStatusMinted {
			allMinted = false
			break
		}

		if task.Signature != "" {
			latestSignature = task.Signature
		}
	}

	if !allMinted {
		return nil, mintdom.ErrMintProductTaskNotFound
	}

	if err := mintEnt.MarkMinted(
		time.Now().UTC(),
		latestSignature,
	); err != nil {
		return nil, err
	}

	if _, err := u.mintRepo.Update(ctx, *mintEnt); err != nil {
		return nil, fmt.Errorf(
			"mark parent minted: %w",
			err,
		)
	}

	if u.tbMintMarker != nil && reqTBID != "" {
		_, _ = u.tbMintMarker.MarkTokenBlueprintMinted(
			ctx,
			reqTBID,
			actorID,
		)
	}

	return u.mintResultMapper.FromMint(*mintEnt), nil
}
