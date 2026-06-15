// backend/internal/application/query/console/mint_product_blueprint_query.go
package query

import (
	"context"
	"errors"

	querydto "narratives/internal/application/query/console/dto"
	pbpdom "narratives/internal/domain/productBlueprint"
)

func (s *MintRequestQueryService) GetProductBlueprintForMint(
	ctx context.Context,
	productBlueprintID string,
) (*querydto.MintProductBlueprintDTO, error) {
	if s == nil || s.pbRepo == nil {
		return nil, ErrMintRequestQueryServiceNotConfigured
	}

	id := productBlueprintID
	if id == "" {
		return nil, errors.New("productBlueprintID is empty")
	}

	productBlueprint, err := s.pbRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	brandName := s.resolveBrandNameByID(ctx, productBlueprint.BrandID)

	return buildMintProductBlueprintDTO(productBlueprint, brandName), nil
}

func buildMintProductBlueprintDTO(
	productBlueprint pbpdom.ProductBlueprint,
	brandName string,
) *querydto.MintProductBlueprintDTO {
	return &querydto.MintProductBlueprintDTO{
		ProductBlueprint: productBlueprint,
		BrandName:        brandName,
	}
}
