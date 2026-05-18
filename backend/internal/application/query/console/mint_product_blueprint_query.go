// backend/internal/application/query/console/mint_product_blueprint_query.go
package query

import (
	"context"
	"errors"

	mintapp "narratives/internal/application/mint"
	querydto "narratives/internal/application/query/console/dto"
	resolver "narratives/internal/application/resolver"
	pbpdom "narratives/internal/domain/productBlueprint"
)

// GetProductBlueprintPatchForMint returns productBlueprint patch with display fields
// required by console mint screens.
func (s *MintRequestQueryService) GetProductBlueprintPatchForMint(
	ctx context.Context,
	productBlueprintID string,
) (*querydto.MintProductBlueprintPatchDTO, error) {
	if s == nil || s.mintUC == nil {
		return nil, ErrMintRequestQueryServiceNotConfigured
	}

	id := productBlueprintID
	if id == "" {
		return nil, errors.New("productBlueprintID is empty")
	}

	patch, err := s.mintUC.GetProductBlueprintPatchByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return buildMintProductBlueprintPatchDTO(ctx, patch, s.nameResolver), nil
}

func buildMintProductBlueprintPatchDTO(
	ctx context.Context,
	patch pbpdom.Patch,
	nameResolver *resolver.NameResolver,
) *querydto.MintProductBlueprintPatchDTO {
	brandName := ""

	if patch.BrandID != nil && nameResolver != nil {
		brandID := *patch.BrandID
		if brandID != "" {
			brandName = nameResolver.ResolveBrandName(ctx, brandID)
		}
	}

	return &querydto.MintProductBlueprintPatchDTO{
		Patch:     patch,
		BrandName: brandName,
	}
}

// compile-time check: keep dependency direction explicit.
var _ = (*mintapp.MintUsecase)(nil)
