package usecase

import (
	"context"
	"strings"

	branddom "narratives/internal/domain/brand"
	companydom "narratives/internal/domain/company"
	modeldom "narratives/internal/domain/model"
	productdom "narratives/internal/domain/product"
	bpdom "narratives/internal/domain/productBlueprint"
)

// ------------------------------------------------------------
// DTO 群（Inspector 用 ReadModel）
// ------------------------------------------------------------

type ProductColorDTO struct {
	RGB  int    `json:"rgb"`
	Name string `json:"name,omitempty"`
}

type ProductBlueprintDTO struct {
	ID string `json:"id"`

	ProductName string `json:"productName"`

	BrandID   string `json:"brandId"`
	BrandName string `json:"brandName"` // ★ brandId → brandName を解決して詰める

	CompanyID   string `json:"companyId"`
	CompanyName string `json:"companyName"` // ★ companyId → companyName を解決して詰める

	ItemType         string   `json:"itemType"` // bpdom.ItemType を string 化して詰める
	Fit              string   `json:"fit"`
	Material         string   `json:"material"`
	Weight           float64  `json:"weight"` // domain に合わせて float64
	QualityAssurance []string `json:"qualityAssurance"`
	ProductIdTagType string   `json:"productIdTagType"`
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

	ModelNumber string          `json:"modelNumber"`
	Size        string          `json:"size"`
	Color       ProductColorDTO `json:"color"`

	// modeldom.Measurements は map[string]int の type alias
	Measurements modeldom.Measurements `json:"measurements"`

	ProductBlueprintID  string              `json:"productBlueprintId"`
	ProductBlueprintDTO ProductBlueprintDTO `json:"productBlueprint"` // Flutter 側の JSON キーに合わせる
}

// ------------------------------------------------------------
// Usecase / Repository インターフェース
// ------------------------------------------------------------

// ProductQueryRepo は検品詳細画面用の ReadModel を構築するための
// 最小限の読み取り専用ポートです。
//
// 旧: InspectionQueryRepo / InspectorQueryRepo
type ProductQueryRepo interface {
	GetProductByID(ctx context.Context, productID string) (productdom.Product, error)
	GetModelByID(ctx context.Context, modelID string) (modeldom.ModelVariation, error)
	GetProductionByID(ctx context.Context, productionID string) ( /* productiondom.Production */ interface{}, error)
	GetProductBlueprintByID(ctx context.Context, bpID string) (bpdom.ProductBlueprint, error)
}

// ProductUsecase は Inspector 用 DTO を組み立てるユースケースです。
type ProductUsecase struct {
	repo           ProductQueryRepo
	brandService   *branddom.Service
	companyService *companydom.Service
}

func NewProductUsecase(
	repo ProductQueryRepo,
	brandSvc *branddom.Service,
	companySvc *companydom.Service,
) *ProductUsecase {
	return &ProductUsecase{
		repo:           repo,
		brandService:   brandSvc,
		companyService: companySvc,
	}
}

// GetInspectorProductDetail は productId を起点に各ドメインから情報を取得し，
// ProductDetail DTO に詰め替えて返します。
func (u *ProductUsecase) GetInspectorProductDetail(
	ctx context.Context,
	productID string,
) (ProductDetail, error) {
	productID = strings.TrimSpace(productID)
	if productID == "" {
		return ProductDetail{}, productdom.ErrInvalidID
	}

	// 1) Product を取得
	product, err := u.repo.GetProductByID(ctx, productID)
	if err != nil {
		return ProductDetail{}, err
	}

	// 2) ModelVariation を取得（Product.ModelID 起点）
	model, err := u.repo.GetModelByID(ctx, product.ModelID)
	if err != nil {
		return ProductDetail{}, err
	}

	// 3) ProductBlueprint を取得（ModelVariation.ProductBlueprintID 起点）
	bp, err := u.repo.GetProductBlueprintByID(ctx, model.ProductBlueprintID)
	if err != nil {
		return ProductDetail{}, err
	}

	// 4) brandId → brandName 解決
	var brandName string
	if u.brandService != nil && strings.TrimSpace(bp.BrandID) != "" {
		if name, err := u.brandService.GetNameByID(ctx, bp.BrandID); err == nil {
			brandName = name
		}
	}

	// 5) companyId → companyName 解決
	var companyName string
	if u.companyService != nil && strings.TrimSpace(bp.CompanyID) != "" {
		if name, err := u.companyService.GetCompanyNameByID(ctx, bp.CompanyID); err == nil {
			companyName = name
		}
	}

	// 6) Color DTO
	colorDTO := ProductColorDTO{
		RGB:  model.Color.RGB,
		Name: model.Color.Name,
	}

	// 7) ProductBlueprintDTO を構築
	pbDTO := ProductBlueprintDTO{
		ID:               bp.ID,
		ProductName:      bp.ProductName,
		BrandID:          bp.BrandID,
		BrandName:        brandName,
		CompanyID:        bp.CompanyID,
		CompanyName:      companyName,
		ItemType:         string(bp.ItemType),
		Fit:              bp.Fit,
		Material:         bp.Material,
		Weight:           bp.Weight,
		QualityAssurance: append([]string(nil), bp.QualityAssurance...),
		ProductIdTagType: bp.ProductIdTag.Type,
	}

	// 8) InspectionResult は domain の型 (productdom.InspectionResult) を string にして詰める
	inspectionResult := string(product.InspectionResult)

	// 9) 最終的な DTO を組み立てて返す
	detail := ProductDetail{
		ProductID:           product.ID,
		ModelID:             product.ModelID,
		ProductionID:        product.ProductionID,
		InspectionResult:    inspectionResult,
		ModelNumber:         model.ModelNumber,
		Size:                model.Size,
		Color:               colorDTO,
		Measurements:        model.Measurements,
		ProductBlueprintID:  bp.ID,
		ProductBlueprintDTO: pbDTO,
	}

	return detail, nil
}
