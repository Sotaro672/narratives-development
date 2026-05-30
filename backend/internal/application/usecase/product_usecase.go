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
	BrandName string `json:"brandName"` // brandId → brandName を解決して詰める

	CompanyID   string `json:"companyId"`
	CompanyName string `json:"companyName"` // companyId → companyName を解決して詰める

	ProductBlueprintCategory ProductBlueprintCategoryDTO `json:"productBlueprintCategory"`

	// NOTE:
	// fit / material / weight / qualityAssurance は ProductBlueprint 直下ではなく
	// CategoryFields に集約済み。
	// Inspector 既存レスポンス互換のため DTO field は残し、CategoryFields から詰める。
	Fit              string   `json:"fit"`
	Material         string   `json:"material"`
	Weight           float64  `json:"weight"`
	QualityAssurance []string `json:"qualityAssurance"`

	ProductIdTagType string `json:"productIdTagType"`

	// modelRefs（displayOrder含む）
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

// ------------------------------------------------------------
// Usecase / Repository インターフェース
// ------------------------------------------------------------

// ModelVariationGetter は model variation を取得する最小ポート。
// 戻り値は modeldom.ModelVariation に統一する。
// *modeldom.ModelVariation のような pointer-to-interface は扱わない。
type ModelVariationGetter interface {
	GetModelVariationByID(ctx context.Context, modelID string) (modeldom.ModelVariation, error)
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
	productRepo          productdom.Repository
	modelRepo            ModelVariationGetter
	productionRepo       ProductionGetter // 今は未使用だが、将来の参照のために保持
	productBlueprintRepo ProductBlueprintGetter

	brandService   *branddom.Service
	companyService *companydom.Service
}

func NewProductUsecase(
	productRepo productdom.Repository,
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

	if u == nil || u.productRepo == nil {
		return ProductDetail{}, errors.New("product: repository is nil")
	}
	if u.modelRepo == nil {
		return ProductDetail{}, errors.New("product: model repository is nil")
	}
	if u.productBlueprintRepo == nil {
		return ProductDetail{}, errors.New("product: productBlueprint repository is nil")
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
	if mv == nil {
		return ProductDetail{}, errors.New("product: model variation not found")
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
		return ProductDetail{}, errors.New("product: unsupported model variation type")
	}

	// 3) ProductBlueprint を取得（ModelVariation.ProductBlueprintID 起点）
	bp, err := u.productBlueprintRepo.GetByID(ctx, productBlueprintID)
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

	// 6) modelRefs を DTO 化する。
	// ModelRefs の空ID除外・重複除外・displayOrder 採番は productBlueprint domain 側の責務。
	// usecase では補正せず、表示用 DTO へ詰め替えるだけにする。
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

		// fit / material / weight / qualityAssurance は CategoryFields から復元する。
		// CategoryFields 自体の空 key 除外などの正規化は productBlueprint domain 側の責務。
		Fit:              categoryFieldString(bp.CategoryFields, "fit"),
		Material:         categoryFieldString(bp.CategoryFields, "material"),
		Weight:           categoryFieldFloat64(bp.CategoryFields, "weight"),
		QualityAssurance: categoryFieldStringSlice(bp.CategoryFields, "qualityAssurance"),

		ProductIdTagType: string(bp.ProductIdTag.Type),

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

		Kind:        kind,
		ModelNumber: modelNumber,
		ModelLabel:  modelLabel,

		Size:         size,
		Color:        colorDTO,
		Measurements: measurements,

		VolumeValue: volumeValue,
		VolumeUnit:  volumeUnit,

		ProductBlueprintID:  bp.ID,
		ProductBlueprintDTO: pbDTO,
	}

	return detail, nil
}

func categoryFieldString(fields bpdom.CategoryFields, key string) string {
	if len(fields) == 0 || key == "" {
		return ""
	}

	v, ok := fields[key]
	if !ok || v == nil {
		return ""
	}

	switch x := v.(type) {
	case string:
		return x
	default:
		return ""
	}
}

func categoryFieldFloat64(fields bpdom.CategoryFields, key string) float64 {
	if len(fields) == 0 || key == "" {
		return 0
	}

	v, ok := fields[key]
	if !ok || v == nil {
		return 0
	}

	switch x := v.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case int:
		return float64(x)
	case int8:
		return float64(x)
	case int16:
		return float64(x)
	case int32:
		return float64(x)
	case int64:
		return float64(x)
	case uint:
		return float64(x)
	case uint8:
		return float64(x)
	case uint16:
		return float64(x)
	case uint32:
		return float64(x)
	case uint64:
		return float64(x)
	default:
		return 0
	}
}

func categoryFieldStringSlice(fields bpdom.CategoryFields, key string) []string {
	if len(fields) == 0 || key == "" {
		return nil
	}

	v, ok := fields[key]
	if !ok || v == nil {
		return nil
	}

	switch x := v.(type) {
	case []string:
		return append([]string(nil), x...)

	case []any:
		out := make([]string, 0, len(x))
		for _, item := range x {
			s, ok := item.(string)
			if !ok || s == "" {
				continue
			}
			out = append(out, s)
		}
		if len(out) == 0 {
			return nil
		}
		return out

	default:
		return nil
	}
}
