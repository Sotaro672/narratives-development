// backend/internal/application/query/inspector/inspector_query.go
package inspector

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	branddom "narratives/internal/domain/brand"
	companydom "narratives/internal/domain/company"
	inspectiondom "narratives/internal/domain/inspection"
	modeldom "narratives/internal/domain/model"
	productdom "narratives/internal/domain/product"
	bpdom "narratives/internal/domain/productBlueprint"
)

// ------------------------------------------------------------
// Ports
// ------------------------------------------------------------

// InspectionRepository は inspector 画面表示 query が必要とする
// inspection 永続化ポートです。
type InspectionRepository interface {
	GetByProductionID(
		ctx context.Context,
		productionID string,
	) (inspectiondom.InspectionBatch, error)
}

// ProductRepository は inspector 商品詳細 query が必要とする
// product 永続化ポートです。
type ProductRepository interface {
	GetByID(ctx context.Context, id string) (productdom.Product, error)
}

// ModelVariationGetter は model variation を取得する最小ポートです。
//
// 戻り値は modeldom.ModelVariation に統一します。
// *modeldom.ModelVariation のような pointer-to-interface は扱いません。
type ModelVariationGetter interface {
	GetModelVariationByID(ctx context.Context, modelID string) (modeldom.ModelVariation, error)
}

// ProductBlueprintGetter は product blueprint を取得する最小ポートです。
type ProductBlueprintGetter interface {
	GetByID(ctx context.Context, bpID string) (bpdom.ProductBlueprint, error)
}

// ------------------------------------------------------------
// Query Service
// ------------------------------------------------------------

type QueryService struct {
	inspectionRepo InspectionRepository

	productRepo          ProductRepository
	modelRepo            ModelVariationGetter
	productBlueprintRepo ProductBlueprintGetter

	brandRepo   branddom.Repository
	companyRepo companydom.Repository
}

type NewQueryServiceParams struct {
	InspectionRepo InspectionRepository

	ProductRepo          ProductRepository
	ModelRepo            ModelVariationGetter
	ProductBlueprintRepo ProductBlueprintGetter

	BrandRepo   branddom.Repository
	CompanyRepo companydom.Repository
}

func NewQueryService(params NewQueryServiceParams) *QueryService {
	return &QueryService{
		inspectionRepo: params.InspectionRepo,

		productRepo:          params.ProductRepo,
		modelRepo:            params.ModelRepo,
		productBlueprintRepo: params.ProductBlueprintRepo,

		brandRepo:   params.BrandRepo,
		companyRepo: params.CompanyRepo,
	}
}

// ------------------------------------------------------------
// Inspection Query
// ------------------------------------------------------------

// GetByProductionID は productionId から InspectionBatch を取得し、
// inspector 画面用 DTO として返します。
//
// production / inspection の docId は同一値として扱うため、
// inspection 取得は productionID を起点にします。
func (q *QueryService) GetByProductionID(
	ctx context.Context,
	productionID string,
) (InspectionBatchForScreenDTO, error) {
	if q == nil || q.inspectionRepo == nil {
		return InspectionBatchForScreenDTO{}, fmt.Errorf("inspection query: inspectionRepo is nil")
	}

	pid := productionID
	if pid == "" {
		return InspectionBatchForScreenDTO{}, inspectiondom.ErrInvalidInspectionProductionID
	}

	batch, err := q.inspectionRepo.GetByProductionID(ctx, pid)
	if err != nil {
		return InspectionBatchForScreenDTO{}, err
	}

	return NewInspectionBatchForScreenDTO(batch), nil
}

// ------------------------------------------------------------
// Product Detail Query
// ------------------------------------------------------------

// GetInspectorProductDetail は productId を起点に各ドメインから情報を取得し、
// inspector 商品詳細 DTO に詰め替えて返します。
func (q *QueryService) GetInspectorProductDetail(
	ctx context.Context,
	productID string,
) (ProductDetail, error) {
	if productID == "" {
		return ProductDetail{}, productdom.ErrInvalidID
	}

	if q == nil || q.productRepo == nil {
		return ProductDetail{}, errors.New("inspector query: product repository is nil")
	}
	if q.modelRepo == nil {
		return ProductDetail{}, errors.New("inspector query: model repository is nil")
	}
	if q.productBlueprintRepo == nil {
		return ProductDetail{}, errors.New("inspector query: productBlueprint repository is nil")
	}

	// 1) Product を取得
	product, err := q.productRepo.GetByID(ctx, productID)
	if err != nil {
		return ProductDetail{}, err
	}

	// 2) ModelVariation を取得（Product.ModelID 起点）
	mv, err := q.modelRepo.GetModelVariationByID(ctx, product.ModelID)
	if err != nil {
		return ProductDetail{}, err
	}
	if mv == nil {
		return ProductDetail{}, errors.New("inspector query: model variation not found")
	}

	var (
		productBlueprintID string
		kind               string
		modelNumber        string
		modelLabel         string

		size         string
		colorDTO     ProductColorDTO
		measurements modeldom.Measurements

		volumeValue int
		volumeUnit  string
	)

	switch model := mv.(type) {
	case modeldom.ApparelModelVariation:
		productBlueprintID = model.ProductBlueprintID
		kind = "apparel"
		modelNumber = model.ModelNumber
		modelLabel = model.ModelNumber

		size = model.Size
		colorDTO = ProductColorDTO{
			RGB:  model.Color.RGB,
			Name: model.Color.Name,
		}
		measurements = model.Measurements

	case modeldom.AlcoholModelVariation:
		productBlueprintID = model.ProductBlueprintID
		kind = "alcohol"
		modelNumber = model.ModelNumber
		modelLabel = model.ModelNumber

		volumeValue = model.Volume.Value
		volumeUnit = model.Volume.Unit

	default:
		return ProductDetail{}, errors.New("inspector query: unsupported model variation type")
	}

	// 3) ProductBlueprint を取得（ModelVariation.ProductBlueprintID 起点）
	bp, err := q.productBlueprintRepo.GetByID(ctx, productBlueprintID)
	if err != nil {
		return ProductDetail{}, err
	}

	// 4) brandId → brandName 解決
	var brandName string
	if q.brandRepo != nil && bp.BrandID != "" {
		if brand, err := q.brandRepo.GetByID(ctx, bp.BrandID); err == nil {
			brandName = brand.Name
		}
	}

	// 5) companyId → companyName 解決
	var companyName string
	if q.companyRepo != nil && bp.CompanyID != "" {
		if companyEntity, err := q.companyRepo.GetByID(ctx, bp.CompanyID); err == nil {
			companyName = companyEntity.Name
		}
	}

	// 6) modelRefs を DTO 化する。
	// ModelRefs の空ID除外・重複除外・displayOrder 採番は productBlueprint domain 側の責務。
	// query では補正せず、表示用 DTO へ詰め替えるだけにする。
	modelRefsDTO := make([]ModelRefDTO, 0, len(bp.ModelRefs))
	for _, r := range bp.ModelRefs {
		modelRefsDTO = append(modelRefsDTO, ModelRefDTO{
			ModelID:      r.ModelID,
			DisplayOrder: r.DisplayOrder,
		})
	}

	// 念のため response の表示順は displayOrder 昇順に揃える。
	// displayOrder <= 0 の補正は行わない。domain validation 側で invalid として扱う前提。
	sort.SliceStable(modelRefsDTO, func(i, j int) bool {
		if modelRefsDTO[i].DisplayOrder != modelRefsDTO[j].DisplayOrder {
			return modelRefsDTO[i].DisplayOrder < modelRefsDTO[j].DisplayOrder
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
			ID:     string(category.ID),
			Code:   string(category.Code),
			NameJa: category.NameJa,
			NameEn: category.NameEn,
			Kind:   string(category.Kind),
			Path:   append([]string(nil), category.Path...),
		},

		// apparel / alcohol どちらも categoryFields を正として返す。
		// apparel: fit / material / weight / qualityAssurance など
		// alcohol: vintage / region / material / alcoholContent など
		CategoryFields: bp.CategoryFields,

		ProductIdTagType: string(bp.ProductIdTag.Type),

		ModelRefs: modelRefsDTO,
	}

	// 8) InspectionResult は domain の型を string にして詰める
	inspectionResult := string(product.InspectionResult)

	// 9) 最終的な DTO を組み立てて返す
	detail := ProductDetail{
		ProductID:        product.ID,
		ModelID:          product.ModelID,
		ProductionID:     product.ProductionID,
		InspectionResult: inspectionResult,

		// connectedToken をそのままフロントに返す
		// NOTE:
		// 現在の productdom.Product には ConnectedToken が存在しないため、
		// ここでは値を詰めません。
		// 将来 token 接続情報を返す場合は、別 repo / query から取得して設定します。
		ConnectedToken: nil,

		// common
		Kind:        kind,
		ModelNumber: modelNumber,
		ModelLabel:  modelLabel,

		// apparel
		Size:         size,
		Color:        colorDTO,
		Measurements: measurements,

		// alcohol
		VolumeValue: volumeValue,
		VolumeUnit:  volumeUnit,

		ProductBlueprintID:  bp.ID,
		ProductBlueprintDTO: pbDTO,
	}

	return detail, nil
}

// ------------------------------------------------------------
// Inspection DTO
// ------------------------------------------------------------

type InspectionItemDTO struct {
	ProductID        string  `json:"productId"`
	ModelID          string  `json:"modelId"`
	InspectionResult *string `json:"inspectionResult,omitempty"`
	InspectedBy      *string `json:"inspectedBy,omitempty"`
	InspectedAt      *string `json:"inspectedAt,omitempty"` // RFC3339
}

type InspectionBatchForScreenDTO struct {
	ProductionID string              `json:"productionId"`
	MintID       *string             `json:"mintId,omitempty"`
	Quantity     int                 `json:"quantity"`
	Status       string              `json:"status"`
	TotalPassed  int                 `json:"totalPassed"`
	Inspections  []InspectionItemDTO `json:"inspections"`
}

func NewInspectionBatchForScreenDTO(
	b inspectiondom.InspectionBatch,
) InspectionBatchForScreenDTO {
	items := make([]InspectionItemDTO, 0, len(b.Inspections))

	for _, it := range b.Inspections {
		var res *string
		if it.InspectionResult != nil {
			s := string(*it.InspectionResult)
			res = &s
		}

		var inspectedAt *string
		if it.InspectedAt != nil && !it.InspectedAt.IsZero() {
			s := it.InspectedAt.UTC().Format(time.RFC3339)
			inspectedAt = &s
		}

		items = append(items, InspectionItemDTO{
			ProductID:        it.ProductID,
			ModelID:          it.ModelID,
			InspectionResult: res,
			InspectedBy:      it.InspectedBy,
			InspectedAt:      inspectedAt,
		})
	}

	return InspectionBatchForScreenDTO{
		ProductionID: b.ProductionID,
		MintID:       b.MintID,
		Quantity:     b.Quantity,
		Status:       string(b.Status),
		TotalPassed:  b.TotalPassed,
		Inspections:  items,
	}
}

// ------------------------------------------------------------
// Product Detail DTO
// ------------------------------------------------------------

type ProductColorDTO struct {
	RGB  int    `json:"rgb"`
	Name string `json:"name,omitempty"`
}

// modelRefs（displayOrder含む）
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
	BrandName string `json:"brandName"`

	CompanyID   string `json:"companyId"`
	CompanyName string `json:"companyName"`

	ProductBlueprintCategory ProductBlueprintCategoryDTO `json:"productBlueprintCategory"`

	// categoryFields を正として返す。
	// apparel / alcohol の category 固有値はここに集約する。
	CategoryFields bpdom.CategoryFields `json:"categoryFields,omitempty"`

	ProductIdTagType string `json:"productIdTagType"`

	ModelRefs []ModelRefDTO `json:"modelRefs"`
}

type ProductDetail struct {
	ProductID        string `json:"productId"`
	ModelID          string `json:"modelId"`
	ProductionID     string `json:"productionId"`
	InspectionResult string `json:"inspectionResult"`

	// connectedToken をそのままフロントに返す
	// NOTE:
	// 現在の productdom.Product には ConnectedToken が存在しないため、
	// ここでは値を詰めません。
	// 将来 token 接続情報を返す場合は、別 repo / query から取得して設定します。
	ConnectedToken *string `json:"connectedToken,omitempty"`

	// common
	Kind        string `json:"kind,omitempty"` // "apparel" / "alcohol"
	ModelNumber string `json:"modelNumber"`
	ModelLabel  string `json:"modelLabel,omitempty"` // 表示用共通ラベル

	// apparel
	Size         string                `json:"size,omitempty"`
	Color        ProductColorDTO       `json:"color,omitempty"`
	Measurements modeldom.Measurements `json:"measurements,omitempty"`

	// alcohol
	VolumeValue int    `json:"volumeValue,omitempty"`
	VolumeUnit  string `json:"volumeUnit,omitempty"`

	ProductBlueprintID  string              `json:"productBlueprintId"`
	ProductBlueprintDTO ProductBlueprintDTO `json:"productBlueprint"` // Flutter 側の JSON キーに合わせる
}
