// backend/internal/application/usecase/mint_task_state.go
package usecase

import (
	"context"
	"errors"
	"strings"

	mintdom "narratives/internal/domain/mint"
)

func (u *MintUsecase) recordMintedProduct(
	ctx context.Context,
	mintID string,
	minted MintedTokenForUsecase,
) error {
	if u == nil {
		return errors.New("mint usecase is nil")
	}

	if u.mintProductMintRecord == nil {
		return errors.New("mint product recorder is nil")
	}

	return u.mintProductMintRecord.RecordProductAsMinted(
		ctx,
		mintID,
		minted,
	)
}

func (u *MintUsecase) markTaskFailed(
	ctx context.Context,
	mintID string,
	productID string,
	err error,
) error {
	if u == nil || u.mintTaskRepo == nil {
		return errors.New("mint task repo is nil")
	}

	message := ""
	if err != nil {
		message = err.Error()
	}

	if isRetryableMintError(err) {
		_, updateErr := u.mintTaskRepo.MarkFailedRetryable(
			ctx,
			mintID,
			productID,
			message,
		)

		return updateErr
	}

	_, updateErr := u.mintTaskRepo.MarkFailedFatal(
		ctx,
		mintID,
		productID,
		message,
	)

	return updateErr
}

func (u *MintUsecase) markParentFailedRetryable(
	ctx context.Context,
	mintEnt *mintdom.Mint,
) error {
	if u == nil || u.mintRepo == nil {
		return errors.New("mint repo is nil")
	}

	if mintEnt == nil {
		return errors.New("mint entity is nil")
	}

	if err := mintEnt.MarkFailedRetryable(); err != nil {
		return err
	}

	_, err := u.mintRepo.Update(ctx, *mintEnt)

	return err
}

func isRetryableMintError(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())

	retryablePatterns := []string{
		"429",
		"too many requests",
		"rate limit",
		"rate limits",
		"connection rate limits exceeded",
		"timeout",
		"deadline exceeded",
		"temporarily unavailable",
		"temporary failure",
		"connection reset",
		"connection refused",
		"i/o timeout",
		"internal error",
		"status=500",
		"status=502",
		"status=503",
		"status=504",
		"fee payer balance is below minimum",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(msg, pattern) {
			return true
		}
	}

	return false
}
