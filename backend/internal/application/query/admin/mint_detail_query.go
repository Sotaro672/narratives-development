// backend/internal/application/query/admin/mint_detail_query.go
package query

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	appresolver "narratives/internal/application/resolver"
	branddom "narratives/internal/domain/brand"
	companydom "narratives/internal/domain/company"
	mintdom "narratives/internal/domain/mint"
	modeldom "narratives/internal/domain/model"
	productdom "narratives/internal/domain/product"
	productblueprintdom "narratives/internal/domain/productBlueprint"
	productiondom "narratives/internal/domain/production"
	tokenblueprintdom "narratives/internal/domain/tokenBlueprint"
)

var (
	ErrMintDetailQueryNotConfigured = errors.New("mint detail query is not configured")
	ErrMintDetailIDRequired         = errors.New("mint id is required")
)

type MintDetailMintReader interface {
	GetByID(ctx context.Context, id string) (mintdom.Mint, error)
}

type MintDetailProductionReader interface {
	GetByID(ctx context.Context, id string) (*productiondom.Production, error)
}

type MintDetailProductReader interface {
	ListByProductionID(ctx context.Context, productionID string) ([]productdom.Product, error)
}

type MintDetailProductBlueprintReader interface {
	GetByID(ctx context.Context, id string) (productblueprintdom.ProductBlueprint, error)
}

type MintDetailTokenBlueprintReader interface {
	GetByID(ctx context.Context, id string) (*tokenblueprintdom.TokenBlueprint, error)
}

type MintDetailCompanyReader interface {
	GetByID(ctx context.Context, id string) (companydom.Company, error)
}

type MintDetailBrandReader interface {
	GetByID(ctx context.Context, id string) (branddom.Brand, error)
}

type MintDetailModelResolver interface {
	ResolveModelResolved(ctx context.Context, variationID string) appresolver.ModelResolved
}

type MintDetailModel struct {
	ModelID      string         `json:"modelId"`
	Kind         string         `json:"kind,omitempty"`
	ModelNumber  string         `json:"modelNumber,omitempty"`
	Size         string         `json:"size,omitempty"`
	ColorName    string         `json:"colorName,omitempty"`
	RGB          *int           `json:"rgb,omitempty"`
	Measurements map[string]int `json:"measurements,omitempty"`
	Volume       *int           `json:"volume,omitempty"`
	VolumeUnit   string         `json:"volumeUnit,omitempty"`
	ProductCount int            `json:"productCount"`
}

type MintDetailResult struct {
	ID string `json:"id"`

	CompanyID   string `json:"companyId"`
	CompanyName string `json:"companyName"`

	TokenBlueprintID string `json:"tokenBlueprintId"`
	TokenBrandName   string `json:"tokenBrandName"`
	TokenName        string `json:"tokenName"`

	ProductBlueprintID string `json:"productBlueprintId"`
	ProductBrandName   string `json:"productBrandName"`
	ProductName        string `json:"productName"`

	Models []MintDetailModel `json:"models"`
}

type MintDetailQuery struct {
	mintReader             MintDetailMintReader
	productionReader       MintDetailProductionReader
	productReader          MintDetailProductReader
	productBlueprintReader MintDetailProductBlueprintReader
	tokenBlueprintReader   MintDetailTokenBlueprintReader
	companyReader          MintDetailCompanyReader
	brandReader            MintDetailBrandReader
	modelResolver          MintDetailModelResolver
}

func NewMintDetailQuery(
	mintReader MintDetailMintReader,
	productionReader MintDetailProductionReader,
	productReader MintDetailProductReader,
	productBlueprintReader MintDetailProductBlueprintReader,
	tokenBlueprintReader MintDetailTokenBlueprintReader,
	companyReader MintDetailCompanyReader,
	brandReader MintDetailBrandReader,
	modelResolver MintDetailModelResolver,
) *MintDetailQuery {
	return &MintDetailQuery{
		mintReader:             mintReader,
		productionReader:       productionReader,
		productReader:          productReader,
		productBlueprintReader: productBlueprintReader,
		tokenBlueprintReader:   tokenBlueprintReader,
		companyReader:          companyReader,
		brandReader:            brandReader,
		modelResolver:          modelResolver,
	}
}

func (q *MintDetailQuery) Get(
	ctx context.Context,
	mintID string,
) (MintDetailResult, error) {
	if q == nil ||
		q.mintReader == nil ||
		q.productionReader == nil ||
		q.productReader == nil ||
		q.productBlueprintReader == nil ||
		q.tokenBlueprintReader == nil ||
		q.companyReader == nil ||
		q.brandReader == nil ||
		q.modelResolver == nil {
		return MintDetailResult{}, ErrMintDetailQueryNotConfigured
	}

	mintID = strings.TrimSpace(mintID)
	if mintID == "" {
		return MintDetailResult{}, ErrMintDetailIDRequired
	}

	mint, err := q.mintReader.GetByID(ctx, mintID)
	if err != nil {
		return MintDetailResult{}, fmt.Errorf("get mint %q: %w", mintID, err)
	}

	production, err := q.productionReader.GetByID(ctx, mint.ID)
	if err != nil {
		return MintDetailResult{}, fmt.Errorf("get production %q: %w", mint.ID, err)
	}
	if production == nil {
		return MintDetailResult{}, fmt.Errorf("get production %q: nil production", mint.ID)
	}

	productBlueprint, err := q.productBlueprintReader.GetByID(
		ctx,
		production.ProductBlueprintID,
	)
	if err != nil {
		return MintDetailResult{}, fmt.Errorf(
			"get product blueprint %q: %w",
			production.ProductBlueprintID,
			err,
		)
	}

	tokenBlueprint, err := q.tokenBlueprintReader.GetByID(
		ctx,
		mint.TokenBlueprintID,
	)
	if err != nil {
		return MintDetailResult{}, fmt.Errorf(
			"get token blueprint %q: %w",
			mint.TokenBlueprintID,
			err,
		)
	}
	if tokenBlueprint == nil {
		return MintDetailResult{}, fmt.Errorf(
			"get token blueprint %q: nil token blueprint",
			mint.TokenBlueprintID,
		)
	}

	company, err := q.companyReader.GetByID(ctx, productBlueprint.CompanyID)
	if err != nil {
		return MintDetailResult{}, fmt.Errorf(
			"get company %q: %w",
			productBlueprint.CompanyID,
			err,
		)
	}

	productBrand, err := q.brandReader.GetByID(ctx, productBlueprint.BrandID)
	if err != nil {
		return MintDetailResult{}, fmt.Errorf(
			"get product brand %q: %w",
			productBlueprint.BrandID,
			err,
		)
	}

	tokenBrand, err := q.brandReader.GetByID(ctx, tokenBlueprint.BrandID)
	if err != nil {
		return MintDetailResult{}, fmt.Errorf(
			"get token brand %q: %w",
			tokenBlueprint.BrandID,
			err,
		)
	}

	products, err := q.productReader.ListByProductionID(ctx, mint.ID)
	if err != nil {
		return MintDetailResult{}, fmt.Errorf(
			"list products by production %q: %w",
			mint.ID,
			err,
		)
	}

	modelCounts := countMintProductsByModel(mint.Products, products)
	modelIDs := orderMintModelIDs(production.Models, modelCounts)
	models := make([]MintDetailModel, 0, len(modelIDs))

	for _, modelID := range modelIDs {
		resolved := q.modelResolver.ResolveModelResolved(ctx, modelID)

		models = append(models, MintDetailModel{
			ModelID:      modelID,
			Kind:         resolved.Kind,
			ModelNumber:  resolved.ModelNumber,
			Size:         resolved.Size,
			ColorName:    resolved.Color,
			RGB:          resolved.RGB,
			Measurements: cloneMintMeasurements(resolved.Measurements),
			Volume:       resolved.VolumeValue,
			VolumeUnit:   resolved.VolumeUnit,
			ProductCount: modelCounts[modelID],
		})
	}

	return MintDetailResult{
		ID:                 mint.ID,
		CompanyID:          productBlueprint.CompanyID,
		CompanyName:        company.Name,
		TokenBlueprintID:   tokenBlueprint.ID,
		TokenBrandName:     tokenBrand.Name,
		TokenName:          tokenBlueprint.Name,
		ProductBlueprintID: productBlueprint.ID,
		ProductBrandName:   productBrand.Name,
		ProductName:        productBlueprint.ProductName,
		Models:             models,
	}, nil
}

func countMintProductsByModel(
	mintProductIDs []string,
	products []productdom.Product,
) map[string]int {
	targetProductIDs := make(map[string]struct{}, len(mintProductIDs))
	for _, productID := range mintProductIDs {
		productID = strings.TrimSpace(productID)
		if productID != "" {
			targetProductIDs[productID] = struct{}{}
		}
	}

	counts := make(map[string]int)
	for _, product := range products {
		if _, ok := targetProductIDs[product.ID]; !ok {
			continue
		}
		if product.ModelID == "" {
			continue
		}

		counts[product.ModelID]++
	}

	return counts
}

func orderMintModelIDs(
	productionModels []productiondom.ModelQuantity,
	counts map[string]int,
) []string {
	modelIDs := make([]string, 0, len(counts))
	seen := make(map[string]struct{}, len(counts))

	for _, model := range productionModels {
		if counts[model.ModelID] <= 0 {
			continue
		}
		if _, ok := seen[model.ModelID]; ok {
			continue
		}

		seen[model.ModelID] = struct{}{}
		modelIDs = append(modelIDs, model.ModelID)
	}

	remaining := make([]string, 0)
	for modelID, count := range counts {
		if count <= 0 {
			continue
		}
		if _, ok := seen[modelID]; ok {
			continue
		}

		remaining = append(remaining, modelID)
	}

	sort.Strings(remaining)

	return append(modelIDs, remaining...)
}

func cloneMintMeasurements(
	measurements modeldom.Measurements,
) map[string]int {
	if len(measurements) == 0 {
		return nil
	}

	cloned := make(map[string]int, len(measurements))
	for key, value := range measurements {
		cloned[key] = value
	}

	return cloned
}
