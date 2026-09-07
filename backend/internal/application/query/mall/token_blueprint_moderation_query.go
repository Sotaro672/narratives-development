// backend/internal/application/query/mall/token_blueprint_moderation_query.go
package mall

import (
	"context"
	"errors"
	"strings"

	applicationport "narratives/internal/application/port"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

var (
	ErrMallTokenBlueprintModerationQueryNotConfigured = errors.New(
		"mall token_blueprint_moderation_query: service not configured",
	)
	ErrMallTokenBlueprintModerationAvatarIDRequired = errors.New(
		"mall token_blueprint_moderation_query: avatarID is required",
	)
	ErrMallTokenBlueprintModerationTokenBlueprintIDRequired = errors.New(
		"mall token_blueprint_moderation_query: tokenBlueprintID is required",
	)
	ErrMallTokenBlueprintModerationForbidden = errors.New(
		"mall token_blueprint_moderation_query: forbidden",
	)
)

// TokenBlueprintModerationQuery returns the AMOL-side moderation state of a
// TokenBlueprint for an avatar that currently owns an asset associated with it.
//
// Ownership is resolved through ReportTokenAccessResolver, whose current
// implementation verifies on-chain ownership rather than trusting the
// Firestore wallet assetIds read model.
//
// This query does not expose or modify:
// - Firebase Storage assets
// - metadataUri contents
// - on-chain token state
// - on-chain metadata
type TokenBlueprintModerationQuery struct {
	tokenBlueprintRepo tbdom.RepositoryPort
	accessResolver     applicationport.ReportTokenAccessResolver
}

func NewTokenBlueprintModerationQuery(
	tokenBlueprintRepo tbdom.RepositoryPort,
	accessResolver applicationport.ReportTokenAccessResolver,
) *TokenBlueprintModerationQuery {
	return &TokenBlueprintModerationQuery{
		tokenBlueprintRepo: tokenBlueprintRepo,
		accessResolver:     accessResolver,
	}
}

type GetTokenBlueprintModerationStatusInput struct {
	AvatarID         string
	TokenBlueprintID string
}

type TokenBlueprintModerationStatusReadModel struct {
	TokenBlueprintID string                 `json:"tokenBlueprintId"`
	Status           tbdom.ModerationStatus `json:"status"`
}

// GetModerationStatus returns the effective moderation status of the requested
// TokenBlueprint.
//
// Legacy TokenBlueprint documents without moderationStatus are normalized to
// ACTIVE by TokenBlueprint.EffectiveModerationStatus().
func (q *TokenBlueprintModerationQuery) GetModerationStatus(
	ctx context.Context,
	input GetTokenBlueprintModerationStatusInput,
) (TokenBlueprintModerationStatusReadModel, error) {
	if err := q.validateConfigured(); err != nil {
		return TokenBlueprintModerationStatusReadModel{}, err
	}

	avatarID := strings.TrimSpace(input.AvatarID)
	if avatarID == "" {
		return TokenBlueprintModerationStatusReadModel{},
			ErrMallTokenBlueprintModerationAvatarIDRequired
	}

	tokenBlueprintID := strings.TrimSpace(input.TokenBlueprintID)
	if tokenBlueprintID == "" {
		return TokenBlueprintModerationStatusReadModel{},
			ErrMallTokenBlueprintModerationTokenBlueprintIDRequired
	}

	tokenBlueprintEntity, err := q.tokenBlueprintRepo.GetByID(
		ctx,
		tokenBlueprintID,
	)
	if err != nil {
		return TokenBlueprintModerationStatusReadModel{}, err
	}
	if tokenBlueprintEntity == nil ||
		tokenBlueprintEntity.ID != tokenBlueprintID {
		return TokenBlueprintModerationStatusReadModel{}, tbdom.ErrNotFound
	}

	allowed, err := q.accessResolver.CanReportTokenBlueprint(
		ctx,
		avatarID,
		tokenBlueprintID,
	)
	if err != nil {
		return TokenBlueprintModerationStatusReadModel{}, err
	}
	if !allowed {
		return TokenBlueprintModerationStatusReadModel{},
			ErrMallTokenBlueprintModerationForbidden
	}

	return TokenBlueprintModerationStatusReadModel{
		TokenBlueprintID: tokenBlueprintEntity.ID,
		Status:           tokenBlueprintEntity.EffectiveModerationStatus(),
	}, nil
}

func (q *TokenBlueprintModerationQuery) validateConfigured() error {
	if q == nil ||
		q.tokenBlueprintRepo == nil ||
		q.accessResolver == nil {
		return ErrMallTokenBlueprintModerationQueryNotConfigured
	}

	return nil
}
