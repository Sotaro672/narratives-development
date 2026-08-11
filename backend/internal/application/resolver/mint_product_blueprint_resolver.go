// backend/internal/application/resolver/mint_product_blueprint_resolver.go
package resolver

import (
	"context"
	"errors"

	productdom "narratives/internal/domain/product"
	pbdom "narratives/internal/domain/productBlueprint"
	tokendom "narratives/internal/domain/token"
)

type MintProductBlueprint struct {
	ProductBlueprintID string `json:"productBlueprintId"`
	ProductName        string `json:"productName"`
}

type MintProductBlueprintResolveResult struct {
	ModelIDs          []string               `json:"modelIds"`
	ProductBlueprints []MintProductBlueprint `json:"productBlueprints"`
}

type AssetTokenResolver interface {
	ResolveTokenByAssetID(
		ctx context.Context,
		assetID string,
	) (tokendom.ResolveTokenByAssetIDResult, error)
}

type ProductReader interface {
	GetByID(ctx context.Context, id string) (productdom.Product, error)
}

type ProductBlueprintReader interface {
	GetIDByModelID(ctx context.Context, modelID string) (string, []pbdom.ModelRef, error)
	GetByID(ctx context.Context, id string) (pbdom.ProductBlueprint, error)
}

type MintProductBlueprintResolver struct {
	tokenQueryRepo       AssetTokenResolver
	productRepo          ProductReader
	productBlueprintRepo ProductBlueprintReader
}

func NewMintProductBlueprintResolver(
	tokenQueryRepo AssetTokenResolver,
	productRepo ProductReader,
	productBlueprintRepo ProductBlueprintReader,
) *MintProductBlueprintResolver {
	return &MintProductBlueprintResolver{
		tokenQueryRepo:       tokenQueryRepo,
		productRepo:          productRepo,
		productBlueprintRepo: productBlueprintRepo,
	}
}

func (r *MintProductBlueprintResolver) ResolveByAssetIDs(
	ctx context.Context,
	assetIDs []string,
) (MintProductBlueprintResolveResult, error) {
	if r == nil {
		return MintProductBlueprintResolveResult{}, errors.New("mint product blueprint resolver is nil")
	}
	if r.tokenQueryRepo == nil {
		return MintProductBlueprintResolveResult{}, errors.New("tokenQueryRepo is nil")
	}
	if r.productRepo == nil {
		return MintProductBlueprintResolveResult{}, errors.New("productRepo is nil")
	}
	if r.productBlueprintRepo == nil {
		return MintProductBlueprintResolveResult{}, errors.New("productBlueprintRepo is nil")
	}

	assetIDs = uniqueNonEmptyStrings(assetIDs)
	if len(assetIDs) == 0 {
		return MintProductBlueprintResolveResult{
			ModelIDs:          []string{},
			ProductBlueprints: []MintProductBlueprint{},
		}, nil
	}

	modelIDs := make([]string, 0, len(assetIDs))
	seenModelIDs := make(map[string]struct{}, len(assetIDs))

	productBlueprints := make([]MintProductBlueprint, 0, len(assetIDs))
	seenProductBlueprintIDs := make(map[string]struct{}, len(assetIDs))

	for _, assetID := range assetIDs {
		tokenResult, err := r.tokenQueryRepo.ResolveTokenByAssetID(ctx, assetID)
		if err != nil {
			if errors.Is(err, tokendom.ErrNotFound) {
				continue
			}
			return MintProductBlueprintResolveResult{}, err
		}

		productID := tokenResult.ProductID
		if productID == "" {
			continue
		}

		product, err := r.productRepo.GetByID(ctx, productID)
		if err != nil {
			if errors.Is(err, productdom.ErrNotFound) {
				continue
			}
			return MintProductBlueprintResolveResult{}, err
		}

		modelID := product.ModelID
		if modelID == "" {
			continue
		}

		if _, ok := seenModelIDs[modelID]; !ok {
			seenModelIDs[modelID] = struct{}{}
			modelIDs = append(modelIDs, modelID)
		}

		productBlueprintID, _, err := r.productBlueprintRepo.GetIDByModelID(ctx, modelID)
		if err != nil {
			continue
		}

		if productBlueprintID == "" {
			continue
		}

		if _, ok := seenProductBlueprintIDs[productBlueprintID]; ok {
			continue
		}

		productBlueprint, err := r.productBlueprintRepo.GetByID(ctx, productBlueprintID)
		if err != nil {
			continue
		}

		seenProductBlueprintIDs[productBlueprintID] = struct{}{}
		productBlueprints = append(productBlueprints, MintProductBlueprint{
			ProductBlueprintID: productBlueprintID,
			ProductName:        productBlueprint.ProductName,
		})
	}

	return MintProductBlueprintResolveResult{
		ModelIDs:          modelIDs,
		ProductBlueprints: productBlueprints,
	}, nil
}

func uniqueNonEmptyStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}

	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))

	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}

		seen[value] = struct{}{}
		result = append(result, value)
	}

	return result
}
