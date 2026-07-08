// backend/internal/application/query/mall/shared/display_resolver.go
package shared

import (
	"context"
	"errors"
	"fmt"

	branddom "narratives/internal/domain/brand"
	modeldom "narratives/internal/domain/model"
	productdom "narratives/internal/domain/product"
	productblueprintdom "narratives/internal/domain/productBlueprint"
	tokenblueprintdom "narratives/internal/domain/tokenBlueprint"
)

var (
	ErrMallDisplayResolverNotConfigured = errors.New("mall display resolver: not configured")
	ErrMallDisplayProductIDRequired     = errors.New("mall display resolver: productID is required")
	ErrMallDisplayModelIDRequired       = errors.New("mall display resolver: modelID is required")
)

// ProductReader resolves productId -> product.
//
// productdom.Repository satisfies this interface.
type ProductReader interface {
	GetByID(ctx context.Context, productID string) (productdom.Product, error)
}

// ProductBlueprintReader resolves productBlueprintId -> ProductBlueprint.
//
// productblueprintdom.Repository satisfies this interface.
type ProductBlueprintReader interface {
	GetByID(ctx context.Context, id string) (productblueprintdom.ProductBlueprint, error)
}

// TokenBlueprintReader resolves tokenBlueprintId -> TokenBlueprint.
//
// tokenblueprintdom.RepositoryPort satisfies this interface.
type TokenBlueprintReader interface {
	GetByID(ctx context.Context, id string) (*tokenblueprintdom.TokenBlueprint, error)
}

// BrandReader resolves brandId -> Brand.
//
// branddom.Repository satisfies this interface.
type BrandReader interface {
	GetByID(ctx context.Context, id string) (branddom.Brand, error)
}

// ModelReader resolves modelId -> ModelVariation.
//
// modeldom.RepositoryPort satisfies this interface.
type ModelReader interface {
	GetByID(ctx context.Context, id string) (modeldom.ModelVariation, error)
}

// MallDisplayResolver is the shared display query port for mall read models.
//
// It intentionally returns small display structs.
// Page-specific DTO composition must stay in each query service.
type MallDisplayResolver interface {
	ResolveProductBlueprintInfo(
		ctx context.Context,
		productBlueprintID string,
	) (ProductBlueprintDisplay, error)

	ResolveTokenBlueprintInfo(
		ctx context.Context,
		tokenBlueprintID string,
	) (TokenBlueprintDisplay, error)

	ResolveBrandInfo(
		ctx context.Context,
		brandID string,
	) (BrandDisplay, error)

	ResolveModelByModelID(
		ctx context.Context,
		modelID string,
	) (ModelDisplay, error)

	ResolveModelByProductID(
		ctx context.Context,
		productID string,
	) (ModelDisplay, error)
}

type DisplayResolver struct {
	productRepo          ProductReader
	modelRepo            ModelReader
	productBlueprintRepo ProductBlueprintReader
	tokenBlueprintRepo   TokenBlueprintReader
	brandRepo            BrandReader
}

func NewDisplayResolver(
	productRepo ProductReader,
	modelRepo ModelReader,
	productBlueprintRepo ProductBlueprintReader,
	tokenBlueprintRepo TokenBlueprintReader,
	brandRepo BrandReader,
) *DisplayResolver {
	return &DisplayResolver{
		productRepo:          productRepo,
		modelRepo:            modelRepo,
		productBlueprintRepo: productBlueprintRepo,
		tokenBlueprintRepo:   tokenBlueprintRepo,
		brandRepo:            brandRepo,
	}
}

type ProductBlueprintDisplay struct {
	ProductBlueprintID string
	ProductName        string
	BrandID            string
	CompanyID          string
}

type TokenBlueprintDisplay struct {
	TokenBlueprintID string
	TokenName        string
	TokenIcon        string
	BrandID          string
	CompanyID        string
	Symbol           string
	Description      string
}

type BrandDisplay struct {
	BrandID              string
	BrandName            string
	BrandIcon            string
	BrandBackgroundImage string
	CompanyID            string
	WebsiteURL           string
	Description          string
}

type ModelDisplay struct {
	ModelID            string
	ProductBlueprintID string

	Kind        string
	ModelNumber string
	ModelLabel  string

	// apparel
	Size         string
	ColorName    string
	ColorRGB     int
	Measurements map[string]int

	// alcohol
	VolumeValue *int
	VolumeUnit  string
}

func (r *DisplayResolver) ResolveProductBlueprintInfo(
	ctx context.Context,
	productBlueprintID string,
) (ProductBlueprintDisplay, error) {
	if productBlueprintID == "" {
		return ProductBlueprintDisplay{}, nil
	}

	if r == nil || r.productBlueprintRepo == nil {
		return ProductBlueprintDisplay{}, ErrMallDisplayResolverNotConfigured
	}

	pb, err := r.productBlueprintRepo.GetByID(ctx, productBlueprintID)
	if err != nil {
		return ProductBlueprintDisplay{}, err
	}

	return ProductBlueprintDisplay{
		ProductBlueprintID: pb.ID,
		ProductName:        pb.ProductName,
		BrandID:            pb.BrandID,
		CompanyID:          pb.CompanyID,
	}, nil
}

func (r *DisplayResolver) ResolveTokenBlueprintInfo(
	ctx context.Context,
	tokenBlueprintID string,
) (TokenBlueprintDisplay, error) {
	if tokenBlueprintID == "" {
		return TokenBlueprintDisplay{}, nil
	}

	if r == nil || r.tokenBlueprintRepo == nil {
		return TokenBlueprintDisplay{}, ErrMallDisplayResolverNotConfigured
	}

	tb, err := r.tokenBlueprintRepo.GetByID(ctx, tokenBlueprintID)
	if err != nil {
		return TokenBlueprintDisplay{}, err
	}
	if tb == nil {
		return TokenBlueprintDisplay{}, nil
	}

	return TokenBlueprintDisplay{
		TokenBlueprintID: tb.ID,
		TokenName:        tb.Name,
		TokenIcon:        tb.IconURL,
		BrandID:          tb.BrandID,
		CompanyID:        tb.CompanyID,
		Symbol:           tb.Symbol,
		Description:      tb.Description,
	}, nil
}

func (r *DisplayResolver) ResolveBrandInfo(
	ctx context.Context,
	brandID string,
) (BrandDisplay, error) {
	if brandID == "" {
		return BrandDisplay{}, nil
	}

	if r == nil || r.brandRepo == nil {
		return BrandDisplay{}, ErrMallDisplayResolverNotConfigured
	}

	b, err := r.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return BrandDisplay{}, err
	}

	return BrandDisplay{
		BrandID:              b.ID,
		BrandName:            b.Name,
		BrandIcon:            b.BrandIcon,
		BrandBackgroundImage: b.BrandBackgroundImage,
		CompanyID:            b.CompanyID,
		WebsiteURL:           b.URL,
		Description:          b.Description,
	}, nil
}

func (r *DisplayResolver) ResolveModelByProductID(
	ctx context.Context,
	productID string,
) (ModelDisplay, error) {
	if productID == "" {
		return ModelDisplay{}, ErrMallDisplayProductIDRequired
	}

	if r == nil || r.productRepo == nil {
		return ModelDisplay{}, ErrMallDisplayResolverNotConfigured
	}

	product, err := r.productRepo.GetByID(ctx, productID)
	if err != nil {
		return ModelDisplay{}, err
	}

	if product.ModelID == "" {
		return ModelDisplay{}, ErrMallDisplayModelIDRequired
	}

	return r.ResolveModelByModelID(ctx, product.ModelID)
}

func (r *DisplayResolver) ResolveModelByModelID(
	ctx context.Context,
	modelID string,
) (ModelDisplay, error) {
	if modelID == "" {
		return ModelDisplay{}, ErrMallDisplayModelIDRequired
	}

	if r == nil || r.modelRepo == nil {
		return ModelDisplay{}, ErrMallDisplayResolverNotConfigured
	}

	modelVariation, err := r.modelRepo.GetByID(ctx, modelID)
	if err != nil {
		return ModelDisplay{}, err
	}

	return ModelDisplayFromVariation(modelVariation), nil
}

func ModelDisplayFromVariation(
	modelVariation modeldom.ModelVariation,
) ModelDisplay {
	if modelVariation == nil {
		return ModelDisplay{}
	}

	out := ModelDisplay{
		ModelID:            modelVariation.GetID(),
		ProductBlueprintID: modelVariation.GetProductBlueprintID(),
		ModelNumber:        modelVariation.GetModelNumber(),
	}

	switch mv := modelVariation.(type) {
	case modeldom.ApparelModelVariation:
		out = applyApparelModelVariationToDisplay(out, mv)

	case *modeldom.ApparelModelVariation:
		if mv != nil {
			out = applyApparelModelVariationToDisplay(out, *mv)
		}

	case modeldom.AlcoholModelVariation:
		out = applyAlcoholModelVariationToDisplay(out, mv)

	case *modeldom.AlcoholModelVariation:
		if mv != nil {
			out = applyAlcoholModelVariationToDisplay(out, *mv)
		}
	}

	out.ModelLabel = BuildModelLabel(out)

	return out
}

func applyApparelModelVariationToDisplay(
	out ModelDisplay,
	modelVariation modeldom.ApparelModelVariation,
) ModelDisplay {
	out.Kind = string(modeldom.ModelVariationKindApparel)
	out.ModelID = firstNonEmpty(out.ModelID, modelVariation.ID)
	out.ProductBlueprintID = firstNonEmpty(out.ProductBlueprintID, modelVariation.ProductBlueprintID)
	out.ModelNumber = firstNonEmpty(out.ModelNumber, modelVariation.ModelNumber)
	out.Size = modelVariation.Size
	out.ColorName = modelVariation.Color.Name
	out.ColorRGB = modelVariation.Color.RGB
	out.Measurements = modelVariation.Measurements

	return out
}

func applyAlcoholModelVariationToDisplay(
	out ModelDisplay,
	modelVariation modeldom.AlcoholModelVariation,
) ModelDisplay {
	out.Kind = string(modeldom.ModelVariationKindAlcohol)
	out.ModelID = firstNonEmpty(out.ModelID, modelVariation.ID)
	out.ProductBlueprintID = firstNonEmpty(out.ProductBlueprintID, modelVariation.ProductBlueprintID)
	out.ModelNumber = firstNonEmpty(out.ModelNumber, modelVariation.ModelNumber)

	value := modelVariation.Volume.Value
	if value > 0 {
		out.VolumeValue = &value
	}
	out.VolumeUnit = modelVariation.Volume.Unit

	return out
}

func BuildModelLabel(model ModelDisplay) string {
	switch model.Kind {
	case string(modeldom.ModelVariationKindAlcohol):
		if model.ModelNumber != "" && model.VolumeValue != nil && model.VolumeUnit != "" {
			return fmt.Sprintf("%s / %d%s", model.ModelNumber, *model.VolumeValue, model.VolumeUnit)
		}

		if model.VolumeValue != nil && model.VolumeUnit != "" {
			return fmt.Sprintf("%d%s", *model.VolumeValue, model.VolumeUnit)
		}

		if model.ModelNumber != "" {
			return model.ModelNumber
		}

		return ""

	default:
		if model.ModelNumber != "" && model.Size != "" && model.ColorName != "" {
			return fmt.Sprintf("%s / %s / %s", model.ModelNumber, model.Size, model.ColorName)
		}

		if model.ModelNumber != "" && model.Size != "" {
			return fmt.Sprintf("%s / %s", model.ModelNumber, model.Size)
		}

		if model.ModelNumber != "" && model.ColorName != "" {
			return fmt.Sprintf("%s / %s", model.ModelNumber, model.ColorName)
		}

		if model.Size != "" && model.ColorName != "" {
			return fmt.Sprintf("%s / %s", model.Size, model.ColorName)
		}

		if model.ModelNumber != "" {
			return model.ModelNumber
		}

		if model.Size != "" {
			return model.Size
		}

		return model.ColorName
	}
}

func firstNonEmpty(primary string, fallback string) string {
	if primary != "" {
		return primary
	}

	return fallback
}
