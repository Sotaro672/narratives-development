// backend/internal/application/query/console/production_query.go
package query

import (
	"context"

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

func (s *CompanyProductionQueryService) ListProductionsByCurrentCompany(
	ctx context.Context,
) ([]productiondom.Production, error) {
	rows, _, err := s.listProductionsByCurrentCompany(
		ctx,
	)
	if err != nil {
		return nil, err
	}

	return rows, nil
}

func (s *CompanyProductionQueryService) listProductionsByCurrentCompany(
	ctx context.Context,
) (
	[]productiondom.Production,
	map[string]productbpdom.ProductBlueprint,
	error,
) {
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
		return []productiondom.Production{},
			map[string]productbpdom.ProductBlueprint{},
			nil
	}

	pbIDs := make([]string, 0, len(productBlueprints))
	pbByID := make(
		map[string]productbpdom.ProductBlueprint,
		len(productBlueprints),
	)

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
		return []productiondom.Production{},
			map[string]productbpdom.ProductBlueprint{},
			nil
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
	list, pbByID, err := s.listProductionsByCurrentCompany(
		ctx,
	)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return []ProductionListItemDTO{}, nil
	}

	brandNameCache := map[string]string{}

	out := make([]ProductionListItemDTO, 0, len(list))

	for _, p := range list {
		item := s.toProductionListItemDTO(
			ctx,
			p,
			pbByID,
			brandNameCache,
		)
		out = append(out, item)
	}

	return out, nil
}

func (s *CompanyProductionQueryService) GetProductionByIDForCurrentCompany(
	ctx context.Context,
	id string,
) (productiondom.Production, error) {
	p, _, err := s.getProductionByIDForCurrentCompany(
		ctx,
		id,
	)
	if err != nil {
		return productiondom.Production{}, err
	}

	return p, nil
}

func (s *CompanyProductionQueryService) getProductionByIDForCurrentCompany(
	ctx context.Context,
	id string,
) (
	productiondom.Production,
	productbpdom.ProductBlueprint,
	error,
) {
	cid := usecase.CompanyIDFromContext(ctx)
	if cid == "" {
		return productiondom.Production{},
			productbpdom.ProductBlueprint{},
			productbpdom.ErrInvalidCompanyID
	}
	if s.pbRepo == nil || s.prodRepo == nil {
		return productiondom.Production{},
			productbpdom.ProductBlueprint{},
			productbpdom.ErrInternal
	}
	if id == "" {
		return productiondom.Production{},
			productbpdom.ProductBlueprint{},
			productiondom.ErrInvalidID
	}

	p, err := s.prodRepo.GetByID(ctx, id)
	if err != nil {
		return productiondom.Production{},
			productbpdom.ProductBlueprint{},
			err
	}
	if p == nil {
		return productiondom.Production{},
			productbpdom.ProductBlueprint{},
			productiondom.ErrNotFound
	}

	if p.ProductBlueprintID == "" {
		return productiondom.Production{},
			productbpdom.ProductBlueprint{},
			productiondom.ErrInvalidProductBlueprintID
	}

	pb, err := s.pbRepo.GetByID(ctx, p.ProductBlueprintID)
	if err != nil {
		return productiondom.Production{},
			productbpdom.ProductBlueprint{},
			err
	}

	if pb.CompanyID != cid {
		return productiondom.Production{},
			productbpdom.ProductBlueprint{},
			productiondom.ErrNotFound
	}

	return *p, pb, nil
}

func (s *CompanyProductionQueryService) GetProductionWithAssigneeNameByID(
	ctx context.Context,
	id string,
) (ProductionListItemDTO, error) {
	p, pb, err := s.getProductionByIDForCurrentCompany(
		ctx,
		id,
	)
	if err != nil {
		return ProductionListItemDTO{}, err
	}

	pbByID := map[string]productbpdom.ProductBlueprint{
		pb.ID: pb,
	}
	brandNameCache := map[string]string{}

	return s.toProductionListItemDTO(
		ctx,
		p,
		pbByID,
		brandNameCache,
	), nil
}

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

	totalQty := 0
	for _, mq := range p.Models {
		if mq.Quantity > 0 {
			totalQty += mq.Quantity
		}
	}

	return ProductionListItemDTO{
		Production: p,

		TotalQuantity: totalQty,

		ProductName:   productName,
		BrandName:     brandName,
		AssigneeName:  assigneeName,
		CreatedByName: createdByName,
		UpdatedByName: updatedByName,
		PrintedByName: printedByName,
	}
}
