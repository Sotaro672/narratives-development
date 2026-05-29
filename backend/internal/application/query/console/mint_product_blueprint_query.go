// backend/internal/application/query/console/mint_product_blueprint_query.go
package query

import (
	"context"
	"errors"

	querydto "narratives/internal/application/query/console/dto"
	resolver "narratives/internal/application/resolver"
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

	return buildMintProductBlueprintDTO(ctx, productBlueprint, s.nameResolver), nil
}

func buildMintProductBlueprintDTO(
	ctx context.Context,
	productBlueprint pbpdom.ProductBlueprint,
	nameResolver *resolver.NameResolver,
) *querydto.MintProductBlueprintDTO {
	brandName := ""

	brandID := productBlueprint.BrandID
	if brandID != "" && nameResolver != nil {
		brandName = nameResolver.ResolveBrandName(ctx, brandID)
	}

	return &querydto.MintProductBlueprintDTO{
		ProductBlueprint: productBlueprint,
		BrandName:        brandName,
	}
}
