// backend/internal/application/query/console/mint_token_blueprint_query.go
package query

import (
	"context"
	"errors"

	querydto "narratives/internal/application/query/console/dto"
	domcommon "narratives/internal/domain/common"
	tokenblueprintdom "narratives/internal/domain/tokenBlueprint"
)

// ListTokenBlueprintsForMint returns tokenBlueprint patch-shaped options required by console mint screens.
func (s *MintRequestQueryService) ListTokenBlueprintsForMint(
	ctx context.Context,
	input querydto.ListTokenBlueprintsForMintInput,
) ([]querydto.TokenBlueprintPatchDTO, error) {
	if s == nil || s.mintUC == nil {
		return nil, ErrMintRequestQueryServiceNotConfigured
	}
	if s.tokenBlueprintRepo == nil {
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

	items := make([]querydto.TokenBlueprintPatchDTO, 0, len(result.Items))
	for _, tb := range result.Items {
		if tb.ID == "" {
			continue
		}

		patch, err := s.tokenBlueprintRepo.GetPatchByID(ctx, tb.ID)
		if err != nil {
			return nil, err
		}

		items = append(items, *buildTokenBlueprintPatchDTO(patch))
	}

	return items, nil
}

// GetTokenBlueprintPatchForMint returns tokenBlueprint patch required by console mint detail screens.
func (s *MintRequestQueryService) GetTokenBlueprintPatchForMint(
	ctx context.Context,
	tokenBlueprintID string,
) (*querydto.TokenBlueprintPatchDTO, error) {
	if s == nil || s.tokenBlueprintRepo == nil {
		return nil, ErrMintRequestQueryServiceNotConfigured
	}

	if tokenBlueprintID == "" {
		return nil, errors.New("tokenBlueprintID is empty")
	}

	patch, err := s.tokenBlueprintRepo.GetPatchByID(ctx, tokenBlueprintID)
	if err != nil {
		return nil, err
	}

	return buildTokenBlueprintPatchDTO(patch), nil
}

func buildTokenBlueprintPatchDTO(
	patch tokenblueprintdom.Patch,
) *querydto.TokenBlueprintPatchDTO {
	return &querydto.TokenBlueprintPatchDTO{
		ID:          patch.ID,
		TokenName:   patch.TokenName,
		Symbol:      patch.Symbol,
		BrandID:     patch.BrandID,
		BrandName:   patch.BrandName,
		CompanyID:   patch.CompanyID,
		Description: patch.Description,
		Minted:      patch.Minted,
		MetadataURI: patch.MetadataURI,
		IconURL:     patch.IconURL,
	}
}
