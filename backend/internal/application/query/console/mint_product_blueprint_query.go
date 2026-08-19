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

	if productBlueprintID == "" {
		return nil, errors.New("productBlueprintID is empty")
	}

	productBlueprint, err := s.pbRepo.GetByID(
		ctx,
		productBlueprintID,
	)
	if err != nil {
		return nil, err
	}

	brandName := s.resolveBrandNameByID(
		ctx,
		productBlueprint.BrandID,
	)

	return buildMintProductBlueprintDTO(
		productBlueprint,
		brandName,
	), nil
}

func buildMintProductBlueprintDTO(
	productBlueprint pbpdom.ProductBlueprint,
	brandName string,
) *querydto.MintProductBlueprintDTO {
	modelRefs := make(
		[]querydto.MintProductBlueprintModelRefDTO,
		0,
		len(productBlueprint.ModelRefs),
	)

	for _, modelRef := range productBlueprint.ModelRefs {
		modelRefs = append(
			modelRefs,
			querydto.MintProductBlueprintModelRefDTO{
				ModelID:      modelRef.ModelID,
				DisplayOrder: modelRef.DisplayOrder,
			},
		)
	}

	return &querydto.MintProductBlueprintDTO{
		ProductName: productBlueprint.ProductName,
		Description: productBlueprint.Description,

		BrandID:   productBlueprint.BrandID,
		BrandName: brandName,
		CompanyID: productBlueprint.CompanyID,

		ProductBlueprintCategoryPath: append(
			[]string(nil),
			productBlueprint.ProductBlueprintCategoryPath...,
		),

		CategoryFields: map[string]any(
			productBlueprint.CategoryFields,
		),

		ProductIDTag: querydto.MintProductIDTagDTO{
			Type: string(
				productBlueprint.ProductIdTag.Type,
			),
		},

		AssigneeID: productBlueprint.AssigneeID,
		ModelRefs:  modelRefs,
	}
}
