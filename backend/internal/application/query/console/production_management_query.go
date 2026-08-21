// backend/internal/application/query/console/production_management_query.go
package query

import (
	"context"
	"time"

	productbpdom "narratives/internal/domain/productBlueprint"
	productiondom "narratives/internal/domain/production"
)

// ============================================================
// Production Management DTO
// ============================================================

type ProductionListModelDTO struct {
	ModelID  string `json:"modelId"`
	Quantity int    `json:"quantity"`
}

type ProductionListItemDTO struct {
	ID                 string                   `json:"id"`
	ProductBlueprintID string                   `json:"productBlueprintId"`
	ProductName        string                   `json:"productName"`
	BrandName          string                   `json:"brandName"`
	AssigneeID         string                   `json:"assigneeId"`
	AssigneeName       string                   `json:"assigneeName"`
	Models             []ProductionListModelDTO `json:"models"`
	Printed            bool                     `json:"printed"`
	PrintedAt          *time.Time               `json:"printedAt"`
	PrintedBy          *string                  `json:"printedBy"`
	PrintedByName      string                   `json:"printedByName"`
	CreatedBy          *string                  `json:"createdBy"`
	CreatedByName      string                   `json:"createdByName"`
	CreatedAt          *time.Time               `json:"createdAt"`
	UpdatedBy          *string                  `json:"updatedBy"`
	UpdatedByName      string                   `json:"updatedByName"`
	UpdatedAt          *time.Time               `json:"updatedAt"`
	TotalQuantity      int                      `json:"totalQuantity"`
}

// ============================================================
// Production Management Query
// ============================================================

func (s *CompanyProductionQueryService) listProductionsByCurrentCompany(ctx context.Context) ([]productiondom.Production, map[string]productbpdom.ProductBlueprint, error) {
	if s == nil || s.companyIDFromContext == nil {
		return nil, nil, productbpdom.ErrInvalidCompanyID
	}

	cid := s.companyIDFromContext(ctx)
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

		if _, exists := pbByID[pb.ID]; exists {
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
	for _, production := range rows {
		if _, exists := pbByID[production.ProductBlueprintID]; !exists {
			continue
		}

		out = append(out, production)
	}

	return out, pbByID, nil
}

func (s *CompanyProductionQueryService) ListProductionsWithAssigneeName(ctx context.Context) ([]ProductionListItemDTO, error) {
	list, pbByID, err := s.listProductionsByCurrentCompany(ctx)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return []ProductionListItemDTO{}, nil
	}

	brandNameCache := make(map[string]string)
	out := make([]ProductionListItemDTO, 0, len(list))

	for _, production := range list {
		out = append(out, s.toProductionListItemDTO(ctx, production, pbByID, brandNameCache))
	}

	return out, nil
}

// ============================================================
// Production Management DTO Builder
// ============================================================

func (s *CompanyProductionQueryService) toProductionListItemDTO(ctx context.Context, production productiondom.Production, pbByID map[string]productbpdom.ProductBlueprint, brandNameCache map[string]string) ProductionListItemDTO {
	assigneeName := s.resolveProductionMemberNameByID(ctx, production.AssigneeID)
	productName := ""
	brandID := ""
	brandName := ""
	createdByName := ""
	updatedByName := ""
	printedByName := ""

	if s.nameResolver != nil {
		createdByName = s.nameResolver.ResolveCreatedByName(ctx, production.CreatedBy)
		updatedByName = s.nameResolver.ResolveUpdatedByName(ctx, production.UpdatedBy)
		printedByName = s.nameResolver.ResolvePrintedByName(ctx, production.PrintedBy)
	}

	if production.ProductBlueprintID != "" {
		if pb, exists := pbByID[production.ProductBlueprintID]; exists {
			productName = pb.ProductName
			brandID = pb.BrandID
		}
	}

	if s.nameResolver != nil && brandID != "" {
		if cached, exists := brandNameCache[brandID]; exists {
			brandName = cached
		} else {
			brandName = s.nameResolver.ResolveBrandName(ctx, brandID)
			brandNameCache[brandID] = brandName
		}
	}

	models := make([]ProductionListModelDTO, 0, len(production.Models))
	totalQuantity := 0

	for _, model := range production.Models {
		models = append(models, ProductionListModelDTO{
			ModelID:  model.ModelID,
			Quantity: model.Quantity,
		})

		if model.Quantity > 0 {
			totalQuantity += model.Quantity
		}
	}

	return ProductionListItemDTO{
		ID:                 production.ID,
		ProductBlueprintID: production.ProductBlueprintID,
		ProductName:        productName,
		BrandName:          brandName,
		AssigneeID:         production.AssigneeID,
		AssigneeName:       assigneeName,
		Models:             models,
		Printed:            production.Printed,
		PrintedAt:          production.PrintedAt,
		PrintedBy:          production.PrintedBy,
		PrintedByName:      printedByName,
		CreatedBy:          production.CreatedBy,
		CreatedByName:      createdByName,
		CreatedAt:          productionTimePointer(production.CreatedAt),
		UpdatedBy:          production.UpdatedBy,
		UpdatedByName:      updatedByName,
		UpdatedAt:          productionTimePointer(production.UpdatedAt),
		TotalQuantity:      totalQuantity,
	}
}
