// backend/internal/application/usecase/product_usecase.go
package usecase

import (
	"context"

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
	ProductName      string   `json:"productName"`
	BrandID          string   `json:"brandId"`
	CompanyID        string   `json:"companyId"`
	ItemType         string   `json:"itemType"`
	Fit              string   `json:"fit"`
	Material         string   `json:"material"`
	Weight           int      `json:"weight"`
	QualityAssurance []string `json:"qualityAssurance"`
	ProductIdTagType string   `json:"productIdTagType"`
	AssigneeID       string   `json:"assigneeId"`
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

	ModelNumber  string             `json:"modelNumber"`
	Size         string             `json:"size"`
	Color        ProductColorDTO    `json:"color"`
	Measurements map[string]float64 `json:"measurements"`

	ProductBlueprintID  string              `json:"productBlueprintId"`
	ProductBlueprintDTO ProductBlueprintDTO `json:"productBlueprintDTO"`
}

// ------------------------------------------------------------
// Usecase / Repository インターフェース
// ------------------------------------------------------------

// ProductQueryRepo は検品詳細画面用の ReadModel を構築するための
// 最小限の読み取り専用ポートです。
// 旧: InspectionQueryRepo / InspectorQueryRepo
type ProductQueryRepo interface {
	GetProductByID(ctx context.Context, productID string) (productdom.Product, error)
	GetModelByID(ctx context.Context, modelID string) (modeldom.ModelVariation, error)
	GetProductionByID(ctx context.Context, productionID string) ( /* productiondom.Production */ interface{}, error)
	GetProductBlueprintByID(ctx context.Context, bpID string) (bpdom.ProductBlueprint, error)
}

// ProductUsecase は Inspector 用 DTO を組み立てるユースケースです。
type ProductUsecase struct {
	repo ProductQueryRepo
}

func NewProductUsecase(repo ProductQueryRepo) *ProductUsecase {
	return &ProductUsecase{repo: repo}
}

// GetInspectorProductDetail は productId を起点に各ドメインから情報を取得し，
// InspectorProductDetail DTO に詰め替えて返します。
func (u *ProductUsecase) GetInspectorProductDetail(
	ctx context.Context,
	productID string,
) (ProductDetail, error) {
	// TODO: ここで各ドメインリポジトリから取得して DTO に詰め替える
	// - product:   u.repo.GetProductByID
	// - model:     u.repo.GetModelByID
	// - production → productBlueprintId: u.repo.GetProductionByID
	// - blueprint: u.repo.GetProductBlueprintByID
	// - inspections: （別途 DTO 専用リポジトリを定義して取得する想定）
	// を呼び出して、ProductDetail を構築する。
	return ProductDetail{}, nil
}
