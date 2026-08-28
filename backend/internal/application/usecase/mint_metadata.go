// backend/internal/application/usecase/mint_metadata.go
package usecase

import (
	"context"
	"fmt"
)

func (u *MintUsecase) resolveProductBlueprintIDFromProduction(
	ctx context.Context,
	productionID string,
) string {
	if u == nil || u.prodRepo == nil {
		return ""
	}

	if productionID == "" {
		return ""
	}

	productBlueprintID, err := u.prodRepo.
		GetProductBlueprintIDByProductionID(ctx, productionID)
	if err != nil {
		return ""
	}

	return productBlueprintID
}

func (u *MintUsecase) ensureMetadataURI(
	ctx context.Context,
	tokenBlueprintID string,
	actorID string,
	currentMetadataURI string,
) (string, error) {
	metadataURI := currentMetadataURI

	tbID := tokenBlueprintID
	if tbID == "" {
		return metadataURI, nil
	}

	if u.tbMetadataEnsurer == nil {
		return metadataURI, nil
	}

	if u.tbRepo == nil {
		return "", fmt.Errorf("tokenBlueprint repo is nil")
	}

	tb, err := u.tbRepo.GetByID(ctx, tbID)
	if err != nil {
		return "", fmt.Errorf(
			"get tokenBlueprint for metadata ensure: %w",
			err,
		)
	}

	if tb == nil {
		return "", fmt.Errorf(
			"tokenBlueprint not found (id=%s)",
			tbID,
		)
	}

	updated, err := u.tbMetadataEnsurer.EnsureMetadataURI(
		ctx,
		tb,
		actorID,
	)
	if err != nil {
		return "", fmt.Errorf("ensure metadata uri: %w", err)
	}

	if updated == nil {
		updated = tb
	}

	return updated.MetadataURI, nil
}
