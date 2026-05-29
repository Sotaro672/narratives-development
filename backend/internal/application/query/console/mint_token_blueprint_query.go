// backend/internal/application/query/console/mint_token_blueprint_query.go
package query

import (
	"context"
	"encoding/json"
	"errors"

	querydto "narratives/internal/application/query/console/dto"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

func (s *MintRequestQueryService) ListTokenBlueprintsForMint(
	ctx context.Context,
	input querydto.ListTokenBlueprintsForMintInput,
) ([]querydto.TokenBlueprintForMintDTO, error) {
	if s == nil || s.tbRepo == nil {
		return nil, ErrMintRequestQueryServiceNotConfigured
	}

	brandID := input.BrandID
	if brandID == "" {
		return nil, errors.New("brandID is empty")
	}

	result, err := tbdom.ListByBrandID(ctx, s.tbRepo, brandID, pageFromMintInput(input))
	if err != nil {
		return nil, err
	}

	out := make([]querydto.TokenBlueprintForMintDTO, 0, len(result.Items))
	if len(result.Items) == 0 {
		return out, nil
	}

	b, err := json.Marshal(result.Items)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}

	return out, nil
}
