// backend/internal/application/query/console/production_detail_query.go
package query

import (
	"context"
	"sort"
	"time"

	applicationport "narratives/internal/application/port"
	resolver "narratives/internal/application/resolver"
	memberdom "narratives/internal/domain/member"
	productbpdom "narratives/internal/domain/productBlueprint"
	productiondom "narratives/internal/domain/production"
)

type ProductionQueryRepo interface {
	GetByID(ctx context.Context, id string) (*productiondom.Production, error)
	ListByProductBlueprintID(ctx context.Context, productBlueprintIDs []string) ([]productiondom.Production, error)
}

// ============================================================
// Common BFF DTO
// ============================================================

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
	ID                           string                     `json:"id"`
	ProductBlueprintID           string                     `json:"productBlueprintId"`
	ProductName                  string                     `json:"productName"`
	ProductBlueprintCategoryPath []string                   `json:"productBlueprintCategoryPath"`
	BrandID                      string                     `json:"brandId"`
	BrandName                    string                     `json:"brandName"`
	AssigneeID                   string                     `json:"assigneeId"`
	AssigneeName                 string                     `json:"assigneeName"`
	Models                       []ProductionDetailModelDTO `json:"models"`
	TotalQuantity                int                        `json:"totalQuantity"`

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

// ============================================================
// Query Service
// ============================================================

type CompanyProductionQueryService struct {
	pbRepo               applicationport.ProductBlueprintReader
	prodRepo             ProductionQueryRepo
	memberRepo           memberdom.Repository
	nameResolver         *resolver.NameResolver
	companyIDFromContext applicationport.CompanyIDResolver
}

func NewCompanyProductionQueryService(
	pbRepo applicationport.ProductBlueprintReader,
	prodRepo ProductionQueryRepo,
	memberRepo memberdom.Repository,
	nameResolver *resolver.NameResolver,
	companyIDFromContext applicationport.CompanyIDResolver,
) *CompanyProductionQueryService {
	return &CompanyProductionQueryService{
		pbRepo:               pbRepo,
		prodRepo:             prodRepo,
		memberRepo:           memberRepo,
		nameResolver:         nameResolver,
		companyIDFromContext: companyIDFromContext,
	}
}

// ============================================================
// Production Detail
// ============================================================

func (s *CompanyProductionQueryService) getProductionByIDForCurrentCompany(
	ctx context.Context,
	id string,
) (productiondom.Production, productbpdom.ProductBlueprint, error) {
	if s == nil || s.companyIDFromContext == nil {
		return productiondom.Production{}, productbpdom.ProductBlueprint{}, productbpdom.ErrInvalidCompanyID
	}

	cid := s.companyIDFromContext(ctx)
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

	assigneeName := s.resolveProductionMemberNameByID(ctx, p.AssigneeID)
	brandName := ""
	createdByName := ""
	updatedByName := ""
	printedByName := ""

	if s.nameResolver != nil {
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

		models = append(models, s.resolveProductionModelDTO(ctx, model.ModelID, displayOrder, model.Quantity))
		if model.Quantity > 0 {
			totalQuantity += model.Quantity
		}
	}

	sortProductionModelDTOs(models)

	return ProductionDetailDTO{
		ID:                           p.ID,
		ProductBlueprintID:           p.ProductBlueprintID,
		ProductName:                  pb.ProductName,
		ProductBlueprintCategoryPath: append([]string(nil), pb.ProductBlueprintCategoryPath...),
		BrandID:                      pb.BrandID,
		BrandName:                    brandName,
		AssigneeID:                   p.AssigneeID,
		AssigneeName:                 assigneeName,
		Models:                       models,
		TotalQuantity:                totalQuantity,
		Printed:                      p.Printed,
		PrintedAt:                    p.PrintedAt,
		PrintedBy:                    p.PrintedBy,
		PrintedByName:                printedByName,
		CreatedBy:                    p.CreatedBy,
		CreatedByName:                createdByName,
		CreatedAt:                    productionTimePointer(p.CreatedAt),
		UpdatedBy:                    p.UpdatedBy,
		UpdatedByName:                updatedByName,
		UpdatedAt:                    productionTimePointer(p.UpdatedAt),
	}, nil
}

// ============================================================
// Member Resolver
// ============================================================

func (s *CompanyProductionQueryService) resolveProductionMemberNameByID(
	ctx context.Context,
	memberID string,
) string {
	if memberID == "" {
		return ""
	}

	if s.memberRepo == nil {
		return memberID
	}

	rec, err := s.memberRepo.GetByID(ctx, memberID)
	if err != nil {
		return memberID
	}

	name := memberdom.FormatLastFirst(rec.Member.LastName, rec.Member.FirstName)
	if name == "" {
		return memberID
	}

	return name
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

func productionTimePointer(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}

	t := value.UTC()
	return &t
}
