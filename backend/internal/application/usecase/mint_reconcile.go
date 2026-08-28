// backend/internal/application/usecase/mint_reconcile.go
package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	mintdom "narratives/internal/domain/mint"
)

// ReconcileMintCompletion は、親 Mint が MINTING のまま残っている場合に、
// product task の状態から親 Mint の完了状態を復元します.
//
// 親 Mint の全 productID に対応する task が存在し、
// それらが全て MINTED の場合のみ親 Mint を MINTED へ更新します。
func (u *MintUsecase) ReconcileMintCompletion(
	ctx context.Context,
	mintID string,
) error {
	if u == nil {
		return errors.New("mint usecase is nil")
	}

	if mintID == "" {
		return errors.New("mintID is empty")
	}

	if u.mintRepo == nil {
		return errors.New("mint repo is nil")
	}

	if u.mintTaskRepo == nil {
		return errors.New("mint task repo is nil")
	}

	mintEnt, err := u.mintRepo.GetByID(ctx, mintID)
	if err != nil {
		return fmt.Errorf(
			"get parent mint for reconciliation: %w",
			err,
		)
	}

	if mintEnt.Status != mintdom.MintStatusMinting {
		return nil
	}

	tasks, err := u.mintTaskRepo.ListByMintID(ctx, mintID)
	if err != nil {
		return fmt.Errorf(
			"list mint product tasks for reconciliation: %w",
			err,
		)
	}

	if len(tasks) == 0 {
		return nil
	}

	expectedProductIDs := make(
		map[string]struct{},
		len(mintEnt.Products),
	)

	for _, productID := range mintEnt.Products {
		if productID == "" {
			return mintdom.ErrInvalidProducts
		}

		expectedProductIDs[productID] = struct{}{}
	}

	if len(tasks) != len(expectedProductIDs) {
		return nil
	}

	seenProductIDs := make(map[string]struct{}, len(tasks))
	completedAt := time.Time{}
	representativeSignature := ""

	for _, task := range tasks {
		if _, exists := expectedProductIDs[task.ProductID]; !exists {
			return nil
		}

		if _, duplicated := seenProductIDs[task.ProductID]; duplicated {
			return nil
		}

		seenProductIDs[task.ProductID] = struct{}{}

		if task.Status != mintdom.MintProductTaskStatusMinted {
			return nil
		}

		if task.MintedAt == nil || task.MintedAt.IsZero() {
			return nil
		}

		if completedAt.IsZero() || task.MintedAt.After(completedAt) {
			completedAt = task.MintedAt.UTC()
			representativeSignature = task.Signature
		}
	}

	if len(seenProductIDs) != len(expectedProductIDs) {
		return nil
	}

	if completedAt.IsZero() {
		return nil
	}

	if err := mintEnt.MarkMinted(
		completedAt,
		representativeSignature,
	); err != nil {
		return fmt.Errorf(
			"mark parent mint completed during reconciliation: %w",
			err,
		)
	}

	if _, err := u.mintRepo.Update(ctx, mintEnt); err != nil {
		return fmt.Errorf(
			"update reconciled parent mint: %w",
			err,
		)
	}

	return nil
}
