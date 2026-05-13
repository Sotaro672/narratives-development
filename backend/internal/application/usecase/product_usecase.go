// backend/internal/application/usecase/product_usecase.go
package usecase

import (
	"context"
	"errors"
	"sort"

	branddom "narratives/internal/domain/brand"
	companydom "narratives/internal/domain/company"
	modeldom "narratives/internal/domain/model"
	productdom "narratives/internal/domain/product"
	bpdom "narratives/internal/domain/productBlueprint"
	proddom "narratives/internal/domain/production"
)

// ------------------------------------------------------------
// DTO 群（Inspector 用 ReadModel）
// ------------------------------------------------------------

type ProductColorDTO struct {
	RGB  int    `json:"rgb"`
	Name string `json:"name,omitempty"`
}

// ✅ 追加: modelRefs（displayOrder含む）
type ModelRefDTO struct {
	ModelID      string `json:"modelId"`
	DisplayOrder int    `json:"displayOrder"`
}

// ProductBlueprintCategoryDTO は productBlueprint 側に denormalize 保存された
// productBlueprintCategory の表示用 snapshot を返す DTO。
type ProductBlueprintCategoryDTO struct {
	ID     string   `json:"id"`
	Code   string   `json:"code"`
	NameJa string   `json:"nameJa"`
	NameEn string   `json:"nameEn"`
	Kind   string   `json:"kind"`
	Path   []string `json:"path"`
}

type ProductBlueprintDTO struct {
	ID string `json:"id"`

	ProductName string `json:"productName"`

	BrandID   string `json:"brandId"`
	BrandName string `json:"brandName"` // ★ brandId → brandName を解決して詰める

	CompanyID   string `json:"companyId"`
	CompanyName string `json:"companyName"` // ★ companyId → companyName を解決して詰める

	ProductBlueprintCategory ProductBlueprintCategoryDTO `json:"productBlueprintCategory"`

	Fit              string   `json:"fit"`
	Material         string   `json:"material"`
	Weight           float64  `json:"weight"` // domain に合わせて float64
	QualityAssurance []string `json:"qualityAssurance"`
	ProductIdTagType string   `json:"productIdTagType"`

	// ✅ 追加: modelRefs（displayOrder含む）
	ModelRefs []ModelRefDTO `json:"modelRefs"`
}

type InspectionHistoryItemDTO struct {
	ProductID        string  `json:"productId"`
	InspectionResult *string `json:"inspectionResult,omitempty"`
	InspectedBy      *string `json:"inspectedBy,omitempty"`
	InspectedAt      *string `json:"inspectedAt,omitempty"`
}

type ProductDetail struct {
	ProductID        string `json:"productId"`
	ModelID          string `json:"modelId"`
	ProductionID     string `json:"productionId"`
	InspectionResult string `json:"inspectionResult"`

	// ★ 追加: connectedToken をそのままフロントに返す
	ConnectedToken *string `json:"connectedToken,omitempty"`

	ModelNumber string          `json:"modelNumber"`
	Size        string          `json:"size"`
	Color       ProductColorDTO `json:"color"`

	// modeldom.Measurements は map[string]int の type alias
	Measurements modeldom.Measurements `json:"measurements"`

	ProductBlueprintID  string              `json:"productBlueprintId"`
	ProductBlueprintDTO ProductBlueprintDTO `json:"productBlueprint"` // Flutter 側の JSON キーに合わせる
}

// ------------------------------------------------------------
// Usecase / Repository インターフェース（型ズレ吸収を廃止）
// ------------------------------------------------------------

// ProductGetter は product を取得する最小ポート
type ProductGetter interface {
	GetByID(ctx context.Context, productID string) (productdom.Product, error)
}

// ModelVariationGetter は model variation を取得する最小ポート
// Firestore 実装に合わせて *modeldom.ModelVariation を正とする（nil は not found 扱い）
type ModelVariationGetter interface {
	GetModelVariationByID(ctx context.Context, modelID string) (*modeldom.ModelVariation, error)
}

// ProductionGetter は production を取得する最小ポート
// Firestore 実装に合わせて *proddom.Production を正とする（nil は not found 扱い）
type ProductionGetter interface {
	GetByID(ctx context.Context, productionID string) (*proddom.Production, error)
}

// ProductBlueprintGetter は product blueprint を取得する最小ポート
type ProductBlueprintGetter interface {
	GetByID(ctx context.Context, bpID string) (bpdom.ProductBlueprint, error)
}

// ProductUsecase は Inspector 用 DTO を組み立てるユースケースです。
type ProductUsecase struct {
	productRepo          ProductGetter
	modelRepo            ModelVariationGetter
	productionRepo       ProductionGetter // 今は未使用だが、将来の参照のために保持
	productBlueprintRepo ProductBlueprintGetter

	brandService   *branddom.Service
	companyService *companydom.Service
}

func NewProductUsecase(
	productRepo ProductGetter,
	modelRepo ModelVariationGetter,
	productionRepo ProductionGetter,
	productBlueprintRepo ProductBlueprintGetter,
	brandSvc *branddom.Service,
	companySvc *companydom.Service,
) *ProductUsecase {
	return &ProductUsecase{
		productRepo:          productRepo,
		modelRepo:            modelRepo,
		productionRepo:       productionRepo,
		productBlueprintRepo: productBlueprintRepo,
		brandService:         brandSvc,
		companyService:       companySvc,
	}
}

// GetInspectorProductDetail は productId を起点に各ドメインから情報を取得し，
// ProductDetail DTO に詰め替えて返します。
func (u *ProductUsecase) GetInspectorProductDetail(
	ctx context.Context,
	productID string,
) (ProductDetail, error) {
	if productID == "" {
		return ProductDetail{}, productdom.ErrInvalidID
	}

	// 1) Product を取得
	product, err := u.productRepo.GetByID(ctx, productID)
	if err != nil {
		return ProductDetail{}, err
	}

	// 2) ModelVariation を取得（Product.ModelID 起点）
	mv, err := u.modelRepo.GetModelVariationByID(ctx, product.ModelID)
	if err != nil {
		return ProductDetail{}, err
	}
	if mv == nil || *mv == nil {
		return ProductDetail{}, errors.New("product: model variation not found")
	}

	model, ok := toProductUsecaseApparelModelVariation(*mv)
	if !ok {
		return ProductDetail{}, errors.New("product: unsupported model variation type")
	}

	// 3) ProductBlueprint を取得（ModelVariation.ProductBlueprintID 起点）
	bp, err := u.productBlueprintRepo.GetByID(ctx, model.ProductBlueprintID)
	if err != nil {
		return ProductDetail{}, err
	}

	// 4) brandId → brandName 解決
	var brandName string
	if u.brandService != nil && bp.BrandID != "" {
		if name, err := u.brandService.GetNameByID(ctx, bp.BrandID); err == nil {
			brandName = name
		}
	}

	// 5) companyId → companyName 解決
	var companyName string
	if u.companyService != nil && bp.CompanyID != "" {
		if name, err := u.companyService.GetCompanyNameByID(ctx, bp.CompanyID); err == nil {
			companyName = name
		}
	}

	// 6) Color DTO
	colorDTO := ProductColorDTO{
		RGB:  model.Color.RGB,
		Name: model.Color.Name,
	}

	// ✅ modelRefs を DTO 化（displayOrder 含む）
	modelRefsDTO := make([]ModelRefDTO, 0, len(bp.ModelRefs))
	for _, r := range bp.ModelRefs {
		if r.ModelID == "" {
			continue
		}
		modelRefsDTO = append(modelRefsDTO, ModelRefDTO{
			ModelID:      r.ModelID,
			DisplayOrder: r.DisplayOrder,
		})
	}

	// ✅ displayOrder 昇順（0は末尾）
	sort.SliceStable(modelRefsDTO, func(i, j int) bool {
		ai := modelRefsDTO[i].DisplayOrder
		aj := modelRefsDTO[j].DisplayOrder
		if ai == 0 {
			ai = 1 << 30
		}
		if aj == 0 {
			aj = 1 << 30
		}
		if ai != aj {
			return ai < aj
		}
		return modelRefsDTO[i].ModelID < modelRefsDTO[j].ModelID
	})

	category := bp.ProductBlueprintCategory

	// 7) ProductBlueprintDTO を構築
	pbDTO := ProductBlueprintDTO{
		ID:          bp.ID,
		ProductName: bp.ProductName,
		BrandID:     bp.BrandID,
		BrandName:   brandName,
		CompanyID:   bp.CompanyID,
		CompanyName: companyName,

		ProductBlueprintCategory: ProductBlueprintCategoryDTO{
			ID:     category.ID,
			Code:   category.Code,
			NameJa: category.NameJa,
			NameEn: category.NameEn,
			Kind:   string(category.Kind),
			Path:   append([]string(nil), category.Path...),
		},

		Fit:              bp.Fit,
		Material:         bp.Material,
		Weight:           bp.Weight,
		QualityAssurance: append([]string(nil), bp.QualityAssurance...),
		ProductIdTagType: string(bp.ProductIdTag.Type),

		// ✅ 追加
		ModelRefs: modelRefsDTO,
	}

	// 8) InspectionResult は domain の型 (productdom.InspectionResult) を string にして詰める
	inspectionResult := string(product.InspectionResult)

	// 9) 最終的な DTO を組み立てて返す
	detail := ProductDetail{
		ProductID:        product.ID,
		ModelID:          product.ModelID,
		ProductionID:     product.ProductionID,
		InspectionResult: inspectionResult,

		ModelNumber:         model.ModelNumber,
		Size:                model.Size,
		Color:               colorDTO,
		Measurements:        model.Measurements,
		ProductBlueprintID:  bp.ID,
		ProductBlueprintDTO: pbDTO,
	}

	return detail, nil
}

func toProductUsecaseApparelModelVariation(v modeldom.ModelVariation) (modeldom.ApparelModelVariation, bool) {
	if v == nil {
		return modeldom.ApparelModelVariation{}, false
	}

	switch x := v.(type) {
	case modeldom.ApparelModelVariation:
		return x, true
	case *modeldom.ApparelModelVariation:
		if x == nil {
			return modeldom.ApparelModelVariation{}, false
		}
		return *x, true
	default:
		return modeldom.ApparelModelVariation{}, false
	}
}
