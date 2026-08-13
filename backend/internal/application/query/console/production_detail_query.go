// backend/internal/application/query/console/production_detail_query.go
package query

import (
	"context"
	"sort"
	"time"

	resolver "narratives/internal/application/resolver"
	usecase "narratives/internal/application/usecase"

	productbpdom "narratives/internal/domain/productBlueprint"
	productiondom "narratives/internal/domain/production"
)

type ProductBlueprintQueryRepo interface {
	ListByCompanyID(ctx context.Context, companyID string) ([]productbpdom.ProductBlueprint, error)
	GetByID(ctx context.Context, id string) (productbpdom.ProductBlueprint, error)
}

type ProductionQueryRepo interface {
	GetByID(ctx context.Context, id string) (*productiondom.Production, error)
	ListByProductBlueprintID(ctx context.Context, productBlueprintIDs []string) ([]productiondom.Production, error)
}

// ============================================================
// Production List DTO
// ============================================================

type ProductionListItemDTO struct {
	productiondom.Production

	TotalQuantity int `json:"totalQuantity"`

	ProductName   string `json:"productName,omitempty"`
	BrandName     string `json:"brandName,omitempty"`
	AssigneeName  string `json:"assigneeName,omitempty"`
	CreatedByName string `json:"createdByName,omitempty"`
	UpdatedByName string `json:"updatedByName,omitempty"`
	PrintedByName string `json:"printedByName,omitempty"`
}

// ============================================================
// Common BFF DTO
// ============================================================

type ProductionProductBlueprintCategoryDTO struct {
	ID     string   `json:"id"`
	Code   string   `json:"code"`
	NameJa string   `json:"nameJa"`
	NameEn string   `json:"nameEn"`
	Kind   string   `json:"kind"`
	Path   []string `json:"path"`
}

type ProductionDetailModelDTO struct {
	ModelID      string `json:"modelId"`
	Kind         string `json:"kind,omitempty"`
	ModelNumber  string `json:"modelNumber"`
	Size         string `json:"size,omitempty"`
	Color        string `json:"color,omitempty"`
	RGB          *int   `json:"rgb,omitempty"`
	VolumeValue  *int   `json:"volumeValue,omitempty"`
	VolumeUnit   string `json:"volumeUnit,omitempty"`
	DisplayOrder *int   `json:"displayOrder,omitempty"`
	Quantity     int    `json:"quantity"`
}

// ============================================================
// Production Detail BFF DTO
// ============================================================

type ProductionDetailDTO struct {
	ID                       string                                `json:"id"`
	ProductBlueprintID       string                                `json:"productBlueprintId"`
	ProductName              string                                `json:"productName"`
	ProductBlueprintCategory ProductionProductBlueprintCategoryDTO `json:"productBlueprintCategory"`
	BrandID                  string                                `json:"brandId"`
	BrandName                string                                `json:"brandName"`
	AssigneeID               string                                `json:"assigneeId"`
	AssigneeName             string                                `json:"assigneeName"`
	Models                   []ProductionDetailModelDTO            `json:"models"`
	TotalQuantity            int                                   `json:"totalQuantity"`

	Printed       bool       `json:"printed"`
	PrintedAt     *time.Time `json:"printedAt,omitempty"`
	PrintedBy     *string    `json:"printedBy,omitempty"`
	PrintedByName string     `json:"printedByName,omitempty"`

	CreatedBy     *string    `json:"createdBy,omitempty"`
	CreatedByName string     `json:"createdByName,omitempty"`
	CreatedAt     *time.Time `json:"createdAt,omitempty"`

	UpdatedBy     *string    `json:"updatedBy,omitempty"`
	UpdatedByName string     `json:"updatedByName,omitempty"`
	UpdatedAt     *time.Time `json:"updatedAt,omitempty"`
}

type CompanyProductionQueryService struct {
	pbRepo       ProductBlueprintQueryRepo
	prodRepo     ProductionQueryRepo
	nameResolver *resolver.NameResolver
}

func NewCompanyProductionQueryService(
	pbRepo ProductBlueprintQueryRepo,
	prodRepo ProductionQueryRepo,
	nameResolver *resolver.NameResolver,
) *CompanyProductionQueryService {
	return &CompanyProductionQueryService{
		pbRepo:       pbRepo,
		prodRepo:     prodRepo,
		nameResolver: nameResolver,
	}
}

// ============================================================
// Production List
// ============================================================

func (s *CompanyProductionQueryService) listProductionsByCurrentCompany(
	ctx context.Context,
) ([]productiondom.Production, map[string]productbpdom.ProductBlueprint, error) {
	cid := usecase.CompanyIDFromContext(ctx)
	if cid == "" {
		return nil, nil, productbpdom.ErrInvalidCompanyID
	}
	if s.pbRepo == nil || s.prodRepo == nil {
		return nil, nil, productbpdom.ErrInternal
	}

	productBlueprints, err := s.pbRepo.ListByCompanyID(ctx, cid)
	if err != nil {
		return nil, nil, err
	}
	if len(productBlueprints) == 0 {
		return []productiondom.Production{}, map[string]productbpdom.ProductBlueprint{}, nil
	}

	pbIDs := make([]string, 0, len(productBlueprints))
	pbByID := make(map[string]productbpdom.ProductBlueprint, len(productBlueprints))

	for _, pb := range productBlueprints {
		if pb.ID == "" {
			continue
		}
		if _, ok := pbByID[pb.ID]; ok {
			continue
		}

		pbByID[pb.ID] = pb
		pbIDs = append(pbIDs, pb.ID)
	}

	if len(pbIDs) == 0 {
		return []productiondom.Production{}, map[string]productbpdom.ProductBlueprint{}, nil
	}

	rows, err := s.prodRepo.ListByProductBlueprintID(ctx, pbIDs)
	if err != nil {
		return nil, nil, err
	}
	if len(rows) == 0 {
		return []productiondom.Production{}, pbByID, nil
	}

	out := make([]productiondom.Production, 0, len(rows))
	for _, p := range rows {
		if _, ok := pbByID[p.ProductBlueprintID]; !ok {
			continue
		}
		out = append(out, p)
	}

	return out, pbByID, nil
}

func (s *CompanyProductionQueryService) ListProductionsWithAssigneeName(
	ctx context.Context,
) ([]ProductionListItemDTO, error) {
	list, pbByID, err := s.listProductionsByCurrentCompany(ctx)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return []ProductionListItemDTO{}, nil
	}

	brandNameCache := map[string]string{}
	out := make([]ProductionListItemDTO, 0, len(list))

	for _, p := range list {
		out = append(out, s.toProductionListItemDTO(ctx, p, pbByID, brandNameCache))
	}

	return out, nil
}

// ============================================================
// Production Detail
// ============================================================

func (s *CompanyProductionQueryService) getProductionByIDForCurrentCompany(
	ctx context.Context,
	id string,
) (productiondom.Production, productbpdom.ProductBlueprint, error) {
	cid := usecase.CompanyIDFromContext(ctx)
	if cid == "" {
		return productiondom.Production{}, productbpdom.ProductBlueprint{}, productbpdom.ErrInvalidCompanyID
	}
	if s.pbRepo == nil || s.prodRepo == nil {
		return productiondom.Production{}, productbpdom.ProductBlueprint{}, productbpdom.ErrInternal
	}
	if id == "" {
		return productiondom.Production{}, productbpdom.ProductBlueprint{}, productiondom.ErrInvalidID
	}

	p, err := s.prodRepo.GetByID(ctx, id)
	if err != nil {
		return productiondom.Production{}, productbpdom.ProductBlueprint{}, err
	}
	if p == nil {
		return productiondom.Production{}, productbpdom.ProductBlueprint{}, productiondom.ErrNotFound
	}
	if p.ProductBlueprintID == "" {
		return productiondom.Production{}, productbpdom.ProductBlueprint{}, productiondom.ErrInvalidProductBlueprintID
	}

	pb, err := s.pbRepo.GetByID(ctx, p.ProductBlueprintID)
	if err != nil {
		return productiondom.Production{}, productbpdom.ProductBlueprint{}, err
	}
	if pb.CompanyID != cid {
		return productiondom.Production{}, productbpdom.ProductBlueprint{}, productiondom.ErrNotFound
	}

	return *p, pb, nil
}

func (s *CompanyProductionQueryService) GetProductionDetailByID(
	ctx context.Context,
	id string,
) (ProductionDetailDTO, error) {
	p, pb, err := s.getProductionByIDForCurrentCompany(ctx, id)
	if err != nil {
		return ProductionDetailDTO{}, err
	}

	assigneeName := ""
	brandName := ""
	createdByName := ""
	updatedByName := ""
	printedByName := ""

	if s.nameResolver != nil {
		assigneeName = s.nameResolver.ResolveMemberName(ctx, p.AssigneeID)
		createdByName = s.nameResolver.ResolveCreatedByName(ctx, p.CreatedBy)
		updatedByName = s.nameResolver.ResolveUpdatedByName(ctx, p.UpdatedBy)
		printedByName = s.nameResolver.ResolvePrintedByName(ctx, p.PrintedBy)

		if pb.BrandID != "" {
			brandName = s.nameResolver.ResolveBrandName(ctx, pb.BrandID)
		}
	}

	displayOrderByModelID := make(map[string]int, len(pb.ModelRefs))
	for _, ref := range pb.ModelRefs {
		if ref.ModelID == "" {
			continue
		}
		displayOrderByModelID[ref.ModelID] = ref.DisplayOrder
	}

	models := make([]ProductionDetailModelDTO, 0, len(p.Models))
	totalQuantity := 0

	for _, model := range p.Models {
		var displayOrder *int
		if order, ok := displayOrderByModelID[model.ModelID]; ok {
			value := order
			displayOrder = &value
		}

		models = append(
			models,
			s.resolveProductionModelDTO(ctx, model.ModelID, displayOrder, model.Quantity),
		)

		if model.Quantity > 0 {
			totalQuantity += model.Quantity
		}
	}

	sortProductionModelDTOs(models)

	return ProductionDetailDTO{
		ID:                       p.ID,
		ProductBlueprintID:       p.ProductBlueprintID,
		ProductName:              pb.ProductName,
		ProductBlueprintCategory: toProductionProductBlueprintCategoryDTO(pb.ProductBlueprintCategory),
		BrandID:                  pb.BrandID,
		BrandName:                brandName,
		AssigneeID:               p.AssigneeID,
		AssigneeName:             assigneeName,
		Models:                   models,
		TotalQuantity:            totalQuantity,

		Printed:       p.Printed,
		PrintedAt:     p.PrintedAt,
		PrintedBy:     p.PrintedBy,
		PrintedByName: printedByName,

		CreatedBy:     p.CreatedBy,
		CreatedByName: createdByName,
		CreatedAt:     productionTimePointer(p.CreatedAt),

		UpdatedBy:     p.UpdatedBy,
		UpdatedByName: updatedByName,
		UpdatedAt:     productionTimePointer(p.UpdatedAt),
	}, nil
}

// ============================================================
// Production Model Resolver
// ============================================================

func (s *CompanyProductionQueryService) resolveProductionModelDTO(
	ctx context.Context,
	modelID string,
	displayOrder *int,
	quantity int,
) ProductionDetailModelDTO {
	attr := resolver.ModelResolved{}
	if s.nameResolver != nil {
		attr = s.nameResolver.ResolveModelResolved(ctx, modelID)
	}

	modelNumber := attr.ModelNumber
	if modelNumber == "" {
		modelNumber = modelID
	}

	row := ProductionDetailModelDTO{
		ModelID:      modelID,
		Kind:         attr.Kind,
		ModelNumber:  modelNumber,
		DisplayOrder: displayOrder,
		Quantity:     quantity,
	}

	if attr.Kind == "alcohol" {
		row.VolumeValue = attr.VolumeValue
		row.VolumeUnit = attr.VolumeUnit
	} else {
		row.Size = attr.Size
		row.Color = attr.Color
		row.RGB = attr.RGB
	}

	return row
}

func sortProductionModelDTOs(models []ProductionDetailModelDTO) {
	sort.SliceStable(models, func(i, j int) bool {
		left := models[i].DisplayOrder
		right := models[j].DisplayOrder

		if left == nil && right == nil {
			return false
		}
		if left == nil {
			return false
		}
		if right == nil {
			return true
		}
		return *left < *right
	})
}

func toProductionProductBlueprintCategoryDTO(
	category productbpdom.ProductBlueprintCategorySnapshot,
) ProductionProductBlueprintCategoryDTO {
	return ProductionProductBlueprintCategoryDTO{
		ID:     category.ID,
		Code:   category.Code,
		NameJa: category.NameJa,
		NameEn: category.NameEn,
		Kind:   string(category.Kind),
		Path:   append([]string(nil), category.Path...),
	}
}

// ============================================================
// Production List DTO Builder
// ============================================================

func (s *CompanyProductionQueryService) toProductionListItemDTO(
	ctx context.Context,
	p productiondom.Production,
	pbByID map[string]productbpdom.ProductBlueprint,
	brandNameCache map[string]string,
) ProductionListItemDTO {
	assigneeName := ""
	productName := ""
	brandID := ""
	brandName := ""
	createdByName := ""
	updatedByName := ""
	printedByName := ""

	if s.nameResolver != nil {
		assigneeName = s.nameResolver.ResolveMemberName(ctx, p.AssigneeID)
		createdByName = s.nameResolver.ResolveCreatedByName(ctx, p.CreatedBy)
		updatedByName = s.nameResolver.ResolveUpdatedByName(ctx, p.UpdatedBy)
		printedByName = s.nameResolver.ResolvePrintedByName(ctx, p.PrintedBy)
	}

	pbID := p.ProductBlueprintID
	if pbID != "" {
		if pb, ok := pbByID[pbID]; ok {
			productName = pb.ProductName
			brandID = pb.BrandID
		}
	}

	if s.nameResolver != nil && brandID != "" {
		if cached, ok := brandNameCache[brandID]; ok {
			brandName = cached
		} else {
			brandName = s.nameResolver.ResolveBrandName(ctx, brandID)
			brandNameCache[brandID] = brandName
		}
	}

	totalQuantity := 0
	for _, model := range p.Models {
		if model.Quantity > 0 {
			totalQuantity += model.Quantity
		}
	}

	return ProductionListItemDTO{
		Production:    p,
		TotalQuantity: totalQuantity,
		ProductName:   productName,
		BrandName:     brandName,
		AssigneeName:  assigneeName,
		CreatedByName: createdByName,
		UpdatedByName: updatedByName,
		PrintedByName: printedByName,
	}
}

func productionTimePointer(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}

	t := value.UTC()
	return &t
}
