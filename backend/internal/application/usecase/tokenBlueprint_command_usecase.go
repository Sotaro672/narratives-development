// backend/internal/application/usecase/tokenBlueprint_command_usecase.go
package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	tbdom "narratives/internal/domain/tokenBlueprint"
)

// TokenBlueprintCommandUsecase handles explicit state-transition commands.
type TokenBlueprintCommandUsecase struct {
	tbRepo tbdom.RepositoryPort
}

func NewTokenBlueprintCommandUsecase(tbRepo tbdom.RepositoryPort) *TokenBlueprintCommandUsecase {
	return &TokenBlueprintCommandUsecase{tbRepo: tbRepo}
}

// MarkTokenBlueprintMinted sets minted=true idempotently.
func (u *TokenBlueprintCommandUsecase) MarkTokenBlueprintMinted(
	ctx context.Context,
	tokenBlueprintID string,
	actorID string,
) (*tbdom.TokenBlueprint, error) {

	if u == nil {
		return nil, fmt.Errorf("tokenBlueprint command usecase is nil")
	}
	if u.tbRepo == nil {
		return nil, fmt.Errorf("tokenBlueprint repo is nil")
	}

	id := strings.TrimSpace(tokenBlueprintID)
	if id == "" {
		return nil, fmt.Errorf("tokenBlueprintID is empty")
	}

	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return nil, fmt.Errorf("actorID is empty")
	}

	tb, err := u.tbRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if tb == nil {
		return nil, tbdom.ErrNotFound
	}

	if tb.Minted {
		return tb, nil
	}

	now := time.Now().UTC()
	minted := true
	updatedBy := actorID

	updated, err := u.tbRepo.Update(ctx, id, tbdom.UpdateTokenBlueprintInput{
		// entity.go 正: iconId は存在しない
		ContentFiles: nil,
		AssigneeID:   nil,
		Description:  nil,

		Minted: &minted,

		UpdatedAt: &now,
		UpdatedBy: &updatedBy,
		DeletedAt: nil,
		DeletedBy: nil,
	})
	if err != nil {
		return nil, err
	}

	return updated, nil
}
