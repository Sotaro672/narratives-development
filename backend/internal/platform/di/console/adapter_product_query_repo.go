// backend/internal/platform/di/console/adapter_product_query_repo.go
package console

import (
	"context"
	"errors"

	fs "narratives/internal/adapters/out/firestore"
	pbfs "narratives/internal/adapters/out/firestore/productBlueprint"
	modeldom "narratives/internal/domain/model"
	productdom "narratives/internal/domain/product"
	productbpdom "narratives/internal/domain/productBlueprint"
)

// ========================================
// ProductUsecase 用 ProductQueryRepo アダプタ
// ========================================

type productQueryRepoAdapter struct {
	productRepo          *fs.ProductRepositoryFS
	modelRepo            *fs.ModelRepositoryFS
	productionRepo       *fs.ProductionRepositoryFS
	productBlueprintRepo *pbfs.ProductBlueprintRepositoryFS
}

func (a *productQueryRepoAdapter) GetProductByID(
	ctx context.Context,
	productID string,
) (productdom.Product, error) {
	if a == nil || a.productRepo == nil {
		return productdom.Product{}, errors.New("productQueryRepoAdapter: productRepo is nil")
	}
	return a.productRepo.GetByID(ctx, productID)
}

func (a *productQueryRepoAdapter) GetModelByID(
	ctx context.Context,
	modelID string,
) (modeldom.ModelVariation, error) {
	if a == nil || a.modelRepo == nil {
		return modeldom.ModelVariation{}, errors.New("productQueryRepoAdapter: modelRepo is nil")
	}
	mv, err := a.modelRepo.GetModelVariationByID(ctx, modelID)
	if err != nil {
		return modeldom.ModelVariation{}, err
	}
	if mv == nil {
		return modeldom.ModelVariation{}, errors.New("productQueryRepoAdapter: modelRepo returned nil model variation")
	}
	return *mv, nil
}

func (a *productQueryRepoAdapter) GetProductionByID(
	ctx context.Context,
	productionID string,
) (interface{}, error) {
	if a == nil || a.productionRepo == nil {
		return nil, errors.New("productQueryRepoAdapter: productionRepo is nil")
	}
	return a.productionRepo.GetByID(ctx, productionID)
}

func (a *productQueryRepoAdapter) GetProductBlueprintByID(
	ctx context.Context,
	bpID string,
) (productbpdom.ProductBlueprint, error) {
	if a == nil || a.productBlueprintRepo == nil {
		return productbpdom.ProductBlueprint{}, errors.New("productQueryRepoAdapter: productBlueprintRepo is nil")
	}
	return a.productBlueprintRepo.GetByID(ctx, bpID)
}
