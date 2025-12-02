// backend\internal\application\usecase\inspector_usecase.go
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

type InspectorColorDTO struct {
	RGB  int    `json:"rgb"`
	Name string `json:"name,omitempty"`
}

type InspectorBlueprintDTO struct {
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

type InspectorProductDetail struct {
	ProductID        string `json:"productId"`
	ModelID          string `json:"modelId"`
	ProductionID     string `json:"productionId"`
	InspectionResult string `json:"inspectionResult"`

	ModelNumber  string             `json:"modelNumber"`
	Size         string             `json:"size"`
	Color        InspectorColorDTO  `json:"color"`
	Measurements map[string]float64 `json:"measurements"`

	ProductBlueprintID string                `json:"productBlueprintId"`
	Blueprint          InspectorBlueprintDTO `json:"blueprint"`

	Inspections []InspectionHistoryItemDTO `json:"inspections"`
}

// ------------------------------------------------------------
// Usecase / Repository インターフェース
// ------------------------------------------------------------

// InspectionQueryRepo は検品詳細画面用の ReadModel を構築するための
// 最小限の読み取り専用ポートです。
// 旧: InspectionQueryRepo
type InspectorQueryRepo interface {
	GetProductByID(ctx context.Context, productID string) (productdom.Product, error)
	GetModelByID(ctx context.Context, modelID string) (modeldom.ModelVariation, error)
	GetProductionByID(ctx context.Context, productionID string) ( /* productiondom.Production */ interface{}, error)
	GetProductBlueprintByID(ctx context.Context, bpID string) (bpdom.ProductBlueprint, error)
}

// InspectionUsecase は Inspector 用 DTO を組み立てるユースケースです。
type InspectorUsecase struct {
	repo InspectorQueryRepo
}

func NewInspectorUsecase(repo InspectorQueryRepo) *InspectorUsecase {
	return &InspectorUsecase{repo: repo}
}

// GetInspectorProductDetail は productId を起点に各ドメインから情報を取得し，
// InspectorProductDetail DTO に詰め替えて返します。
func (u *InspectionUsecase) GetInspectorProductDetail(
	ctx context.Context,
	productID string,
) (InspectorProductDetail, error) {
	// TODO: ここで各ドメインリポジトリから取得して DTO に詰め替える
	// - product: repo.GetProductByID
	// - model:   repo.GetModelByID
	// - production → productBlueprintId: repo.GetProductionByID
	// - blueprint: repo.GetProductBlueprintByID
	// - inspections: repo.ListInspectionsByProductID
	// を呼び出して、InspectorProductDetail を構築する。
	return InspectorProductDetail{}, nil
}
