// backend/internal/adapters/out/firestore/mall/product_query_repo.go
package mall

import (
	"context"
	"errors"
	"strings"

	"cloud.google.com/go/firestore"

	fs "narratives/internal/adapters/out/firestore"
	modeldom "narratives/internal/domain/model"
	productdom "narratives/internal/domain/product"
	productbpdom "narratives/internal/domain/productBlueprint"
)

// ProductQueryRepo is a Firestore-backed outbound adapter used by ProductUsecase (query side).
//
// This file was split out from DI (di/mall/adapter.go) because it depends on Firestore repositories.
type ProductQueryRepo struct {
	productRepo          *fs.ProductRepositoryFS
	modelRepo            *fs.ModelRepositoryFS
	productionRepo       *fs.ProductionRepositoryFS
	productBlueprintRepo *fs.ProductBlueprintRepositoryFS
}

// NewProductQueryRepo builds the adapter with concrete Firestore repositories.
func NewProductQueryRepo(client *firestore.Client) *ProductQueryRepo {
	return &ProductQueryRepo{
		productRepo:          fs.NewProductRepositoryFS(client),
		modelRepo:            fs.NewModelRepositoryFS(client),
		productionRepo:       fs.NewProductionRepositoryFS(client),
		productBlueprintRepo: fs.NewProductBlueprintRepositoryFS(client),
	}
}

// NewProductQueryRepoWithRepos allows DI to pass already-constructed repos (optional).
func NewProductQueryRepoWithRepos(
	productRepo *fs.ProductRepositoryFS,
	modelRepo *fs.ModelRepositoryFS,
	productionRepo *fs.ProductionRepositoryFS,
	productBlueprintRepo *fs.ProductBlueprintRepositoryFS,
) *ProductQueryRepo {
	return &ProductQueryRepo{
		productRepo:          productRepo,
		modelRepo:            modelRepo,
		productionRepo:       productionRepo,
		productBlueprintRepo: productBlueprintRepo,
	}
}

func (r *ProductQueryRepo) GetProductByID(ctx context.Context, productID string) (productdom.Product, error) {
	if r == nil || r.productRepo == nil {
		return productdom.Product{}, errors.New("firestore.mall.ProductQueryRepo: productRepo is nil")
	}
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return productdom.Product{}, errors.New("firestore.mall.ProductQueryRepo: productID is empty")
	}
	return r.productRepo.GetByID(ctx, productID)
}

func (r *ProductQueryRepo) GetModelByID(ctx context.Context, modelID string) (modeldom.ModelVariation, error) {
	if r == nil || r.modelRepo == nil {
		return modeldom.ModelVariation{}, errors.New("firestore.mall.ProductQueryRepo: modelRepo is nil")
	}
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return modeldom.ModelVariation{}, errors.New("firestore.mall.ProductQueryRepo: modelID is empty")
	}

	mv, err := r.modelRepo.GetModelVariationByID(ctx, modelID)
	if err != nil {
		return modeldom.ModelVariation{}, err
	}
	if mv == nil {
		return modeldom.ModelVariation{}, errors.New("firestore.mall.ProductQueryRepo: model variation not found")
	}
	return *mv, nil
}

// GetProductionByID returns the raw production record.
// NOTE: In your codebase ProductionRepositoryFS.GetByID currently returns interface{}.
func (r *ProductQueryRepo) GetProductionByID(ctx context.Context, productionID string) (interface{}, error) {
	if r == nil || r.productionRepo == nil {
		return nil, errors.New("firestore.mall.ProductQueryRepo: productionRepo is nil")
	}
	productionID = strings.TrimSpace(productionID)
	if productionID == "" {
		return nil, errors.New("firestore.mall.ProductQueryRepo: productionID is empty")
	}
	return r.productionRepo.GetByID(ctx, productionID)
}

func (r *ProductQueryRepo) GetProductBlueprintByID(ctx context.Context, bpID string) (productbpdom.ProductBlueprint, error) {
	if r == nil || r.productBlueprintRepo == nil {
		return productbpdom.ProductBlueprint{}, errors.New("firestore.mall.ProductQueryRepo: productBlueprintRepo is nil")
	}
	bpID = strings.TrimSpace(bpID)
	if bpID == "" {
		return productbpdom.ProductBlueprint{}, errors.New("firestore.mall.ProductQueryRepo: bpID is empty")
	}
	return r.productBlueprintRepo.GetByID(ctx, bpID)
}
