// backend/internal/application/query/console/production_query.go
package query

import (
	"context"
	"time"

	resolver "narratives/internal/application/resolver"
	usecase "narratives/internal/application/usecase"

	productbpdom "narratives/internal/domain/productBlueprint"
	productiondom "narratives/internal/domain/production"
)

type ProductBlueprintQueryRepo interface {
	ListIDsByCompany(ctx context.Context, companyID string) ([]string, error)

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
	now          func() time.Time
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
		now:          time.Now,
	}
}

func (s *CompanyProductionQueryService) ListProductionsByCurrentCompany(
	ctx context.Context,
) ([]productiondom.Production, error) {
	cid := usecase.CompanyIDFromContext(ctx)
	if cid == "" {
		return nil, productbpdom.ErrInvalidCompanyID
	}
	if s.pbRepo == nil || s.prodRepo == nil {
		return nil, productbpdom.ErrInternal
	}

	pbIDs, err := s.pbRepo.ListIDsByCompany(ctx, cid)
	if err != nil {
		return nil, err
	}
	if len(pbIDs) == 0 {
		return []productiondom.Production{}, nil
	}

	rows, err := s.prodRepo.ListByProductBlueprintID(ctx, pbIDs)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []productiondom.Production{}, nil
	}

	set := make(map[string]struct{}, len(pbIDs))
	for _, id0 := range pbIDs {
		if id0 != "" {
			set[id0] = struct{}{}
		}
	}

	out := make([]productiondom.Production, 0, len(rows))
	for _, p := range rows {
		if _, ok := set[p.ProductBlueprintID]; !ok {
			continue
		}
		out = append(out, p)
	}

	return out, nil
}

func (s *CompanyProductionQueryService) ListProductionsWithAssigneeName(
	ctx context.Context,
) ([]ProductionListItemDTO, error) {
	list, err := s.ListProductionsByCurrentCompany(ctx)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return []ProductionListItemDTO{}, nil
	}

	pbBrandCache := map[string]string{}
	brandNameCache := map[string]string{}

	out := make([]ProductionListItemDTO, 0, len(list))

	for _, p := range list {
		item := s.toProductionListItemDTO(ctx, p, pbBrandCache, brandNameCache)
		out = append(out, item)
	}

	return out, nil
}

func (s *CompanyProductionQueryService) GetProductionByIDForCurrentCompany(
	ctx context.Context,
	id string,
) (productiondom.Production, error) {
	cid := usecase.CompanyIDFromContext(ctx)
	if cid == "" {
		return productiondom.Production{}, productbpdom.ErrInvalidCompanyID
	}
	if s.pbRepo == nil || s.prodRepo == nil {
		return productiondom.Production{}, productbpdom.ErrInternal
	}
	if id == "" {
		return productiondom.Production{}, productiondom.ErrInvalidID
	}

	p, err := s.prodRepo.GetByID(ctx, id)
	if err != nil {
		return productiondom.Production{}, err
	}
	if p == nil {
		return productiondom.Production{}, productiondom.ErrNotFound
	}

	if p.ProductBlueprintID == "" {
		return productiondom.Production{}, productiondom.ErrInvalidProductBlueprintID
	}

	pb, err := s.pbRepo.GetByID(ctx, p.ProductBlueprintID)
	if err != nil {
		return productiondom.Production{}, err
	}

	if pb.CompanyID != cid {
		return productiondom.Production{}, productiondom.ErrNotFound
	}

	return *p, nil
}

func (s *CompanyProductionQueryService) GetProductionWithAssigneeNameByID(
	ctx context.Context,
	id string,
) (ProductionListItemDTO, error) {
	p, err := s.GetProductionByIDForCurrentCompany(ctx, id)
	if err != nil {
		return ProductionListItemDTO{}, err
	}

	pbBrandCache := map[string]string{}
	brandNameCache := map[string]string{}

	return s.toProductionListItemDTO(ctx, p, pbBrandCache, brandNameCache), nil
}

func (s *CompanyProductionQueryService) toProductionListItemDTO(
	ctx context.Context,
	p productiondom.Production,
	pbBrandCache map[string]string,
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
		productName = s.nameResolver.ResolveProductName(ctx, p.ProductBlueprintID)
		assigneeName = s.nameResolver.ResolveMemberName(ctx, p.AssigneeID)
		createdByName = s.nameResolver.ResolveCreatedByName(ctx, p.CreatedBy)
		updatedByName = s.nameResolver.ResolveUpdatedByName(ctx, p.UpdatedBy)
		printedByName = s.nameResolver.ResolvePrintedByName(ctx, p.PrintedBy)
	}

	pbID := p.ProductBlueprintID
	if pbID != "" && s.pbRepo != nil {
		if cached, ok := pbBrandCache[pbID]; ok {
			brandID = cached
		} else {
			pb, err := s.pbRepo.GetByID(ctx, pbID)
			if err == nil {
				brandID = extractBrandIDFromProductBlueprint(pb)
				pbBrandCache[pbID] = brandID
			}
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

func extractBrandIDFromProductBlueprint(pb productbpdom.ProductBlueprint) string {
	return pb.BrandID
}
