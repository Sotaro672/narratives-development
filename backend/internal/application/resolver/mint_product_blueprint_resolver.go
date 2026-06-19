// backend/internal/application/resolver/mint_product_blueprint_resolver.go
package resolver

import (
	"context"
	"errors"
	"strings"

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

type MintTokenResolver interface {
	ResolveTokenByMintAddress(
		ctx context.Context,
		mintAddress string,
	) (tokendom.ResolveTokenByMintAddressResult, error)
}

type ProductReader interface {
	GetByID(ctx context.Context, id string) (productdom.Product, error)
}

type ProductBlueprintReader interface {
	GetIDByModelID(ctx context.Context, modelID string) (string, []pbdom.ModelRef, error)
	GetByID(ctx context.Context, id string) (pbdom.ProductBlueprint, error)
}

type MintProductBlueprintResolver struct {
	tokenQueryRepo       MintTokenResolver
	productRepo          ProductReader
	productBlueprintRepo ProductBlueprintReader
}

func NewMintProductBlueprintResolver(
	tokenQueryRepo MintTokenResolver,
	productRepo ProductReader,
	productBlueprintRepo ProductBlueprintReader,
) *MintProductBlueprintResolver {
	return &MintProductBlueprintResolver{
		tokenQueryRepo:       tokenQueryRepo,
		productRepo:          productRepo,
		productBlueprintRepo: productBlueprintRepo,
	}
}

func (r *MintProductBlueprintResolver) ResolveByMintAddresses(
	ctx context.Context,
	mintAddresses []string,
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

	mintAddresses = uniqueNonEmptyStrings(mintAddresses)
	if len(mintAddresses) == 0 {
		return MintProductBlueprintResolveResult{
			ModelIDs:          []string{},
			ProductBlueprints: []MintProductBlueprint{},
		}, nil
	}

	modelIDs := make([]string, 0, len(mintAddresses))
	seenModelIDs := make(map[string]struct{}, len(mintAddresses))

	productBlueprints := make([]MintProductBlueprint, 0, len(mintAddresses))
	seenProductBlueprintIDs := make(map[string]struct{}, len(mintAddresses))

	for _, mintAddress := range mintAddresses {
		tokenResult, err := r.tokenQueryRepo.ResolveTokenByMintAddress(ctx, mintAddress)
		if err != nil {
			if errors.Is(err, tokendom.ErrNotFound) {
				continue
			}
			return MintProductBlueprintResolveResult{}, err
		}

		productID := strings.TrimSpace(tokenResult.ProductID)
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

		modelID := strings.TrimSpace(product.ModelID)
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

		productBlueprintID = strings.TrimSpace(productBlueprintID)
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
		normalized := strings.TrimSpace(value)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}

		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}

	return result
}
