// backend/internal/platform/di/console/adapter_token_blueprint.go
package console

import (
	"context"
	"errors"
	"strings"

	fs "narratives/internal/adapters/out/firestore"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

// ========================================
// NameResolver 用 TokenBlueprint アダプタ
// ========================================

type tokenBlueprintNameRepoAdapter struct {
	repo *fs.TokenBlueprintRepositoryFS
}

func (a *tokenBlueprintNameRepoAdapter) GetByID(
	ctx context.Context,
	id string,
) (tbdom.TokenBlueprint, error) {
	if a == nil || a.repo == nil {
		return tbdom.TokenBlueprint{}, errors.New("tokenBlueprintNameRepoAdapter: repo is nil")
	}
	tb, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return tbdom.TokenBlueprint{}, err
	}
	if tb == nil {
		return tbdom.TokenBlueprint{}, tbdom.ErrNotFound
	}
	return *tb, nil
}

// ========================================
// InventoryQuery 用 TokenBlueprint Patch アダプタ
// ========================================

type tbPatchByIDAdapter struct {
	repo interface {
		GetPatchByID(ctx context.Context, id string) (tbdom.Patch, error)
	}
}

func (a *tbPatchByIDAdapter) GetPatchByID(ctx context.Context, id string) (tbdom.Patch, error) {
	if a == nil || a.repo == nil {
		return tbdom.Patch{}, errors.New("tbPatchByIDAdapter: repo is nil")
	}
	return a.repo.GetPatchByID(ctx, strings.TrimSpace(id))
}
