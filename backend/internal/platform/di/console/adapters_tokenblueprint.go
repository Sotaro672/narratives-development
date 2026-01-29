// backend/internal/platform/di/console/adapters_tokenblueprint.go
package console

import (
	"context"
	"fmt"
	"strings"

	tbdom "narratives/internal/domain/tokenBlueprint"
)

// ★ Adapter: MintUsecase port (EnsureMetadataURIByTokenBlueprintID)
type tbMetadataEnsurerByIDAdapter struct {
	tbRepo   tbdom.RepositoryPort
	metadata interface {
		EnsureMetadataURI(ctx context.Context, tb *tbdom.TokenBlueprint, actorID string) (*tbdom.TokenBlueprint, error)
	}
}

func (a *tbMetadataEnsurerByIDAdapter) EnsureMetadataURIByTokenBlueprintID(ctx context.Context, tokenBlueprintID string, actorID string) (string, error) {
	if a == nil || a.tbRepo == nil || a.metadata == nil {
		return "", fmt.Errorf("tbMetadataEnsurerByIDAdapter: deps are nil")
	}

	id := strings.TrimSpace(tokenBlueprintID)
	if id == "" {
		return "", fmt.Errorf("tokenBlueprintID is empty")
	}

	tb, err := a.tbRepo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}
	if tb == nil {
		return "", fmt.Errorf("tokenBlueprint not found id=%q", id)
	}

	updated, err := a.metadata.EnsureMetadataURI(ctx, tb, actorID)
	if err != nil {
		return "", err
	}
	if updated == nil {
		updated = tb
	}

	uri := strings.TrimSpace(updated.MetadataURI)
	if uri == "" {
		return "", fmt.Errorf("metadataUri is empty after ensure id=%q", id)
	}
	return uri, nil
}
