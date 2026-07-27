// backend/internal/application/query/console/mint_token_blueprint_query.go
package query

import (
	"context"
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

	result, err := tbdom.ListByBrandID(
		ctx,
		s.tbRepo,
		brandID,
		pageFromMintInput(input),
	)
	if err != nil {
		return nil, err
	}

	out := make(
		[]querydto.TokenBlueprintForMintDTO,
		0,
		len(result.Items),
	)

	for _, tokenBlueprint := range result.Items {
		out = append(
			out,
			querydto.TokenBlueprintForMintDTO{
				ID:          tokenBlueprint.ID,
				TokenName:   tokenBlueprint.Name,
				Symbol:      tokenBlueprint.Symbol,
				BrandID:     tokenBlueprint.BrandID,
				CompanyID:   tokenBlueprint.CompanyID,
				Description: tokenBlueprint.Description,
				Minted:      tokenBlueprint.Minted,
				MetadataURI: tokenBlueprint.MetadataURI,
				IconURL:     tokenBlueprint.IconURL,
			},
		)
	}

	return out, nil
}
