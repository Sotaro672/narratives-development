// backend/internal/application/query/console/mint_token_blueprint_query.go
package query

import (
	"context"
	"errors"

	querydto "narratives/internal/application/query/console/dto"
	domcommon "narratives/internal/domain/common"
)

// ListTokenBlueprintsForMint returns tokenBlueprint options required by console mint screens.
func (s *MintRequestQueryService) ListTokenBlueprintsForMint(
	ctx context.Context,
	input querydto.ListTokenBlueprintsForMintInput,
) ([]querydto.TokenBlueprintForMintDTO, error) {
	if s == nil || s.mintUC == nil {
		return nil, ErrMintRequestQueryServiceNotConfigured
	}

	if input.BrandID == "" {
		return nil, errors.New("brandID is empty")
	}

	pageNumber := input.Page
	if pageNumber <= 0 {
		pageNumber = 1
	}

	perPage := input.PerPage
	if perPage <= 0 {
		perPage = 100
	}

	result, err := s.mintUC.ListTokenBlueprintsByBrand(ctx, input.BrandID, domcommon.Page{
		Number:  pageNumber,
		PerPage: perPage,
	})
	if err != nil {
		return nil, err
	}

	items := make([]querydto.TokenBlueprintForMintDTO, 0, len(result.Items))
	for _, tb := range result.Items {
		items = append(items, querydto.TokenBlueprintForMintDTO{
			ID:     tb.ID,
			Name:   tb.Name,
			Symbol: tb.Symbol,
		})
	}

	return items, nil
}
