// backend/internal/application/query/console/production_create_query.go
package query

import (
	"context"
	"sort"

	usecase "narratives/internal/application/usecase"

	productbpdom "narratives/internal/domain/productBlueprint"
	productiondom "narratives/internal/domain/production"
)

// ============================================================
// Production Create BFF DTO
// ============================================================

type ProductionCreateProductBlueprintDTO struct {
	ID                         string                                `json:"id"`
	ProductName                string                                `json:"productName"`
	BrandID                    string                                `json:"brandId"`
	BrandName                  string                                `json:"brandName"`
	ProductBlueprintCategoryID string                                `json:"productBlueprintCategoryId"`
	ProductBlueprintCategory   ProductionProductBlueprintCategoryDTO `json:"productBlueprintCategory"`
	CategoryFields             map[string]any                        `json:"categoryFields,omitempty"`
	AssigneeID                 string                                `json:"assigneeId,omitempty"`
}

type ProductionCreateContextDTO struct {
	ProductBlueprintPatch ProductionCreateProductBlueprintDTO `json:"productBlueprintPatch"`
	Rows                  []ProductionDetailModelDTO          `json:"rows"`
}

// ============================================================
// Production Create Query
// ============================================================

// GetProductionCreateContext は Production Create 画面用BFFを構築する。
//
// frontend側では ProductBlueprint detail / ModelVariation を個別取得せず、
// このresponseの productBlueprintPatch / rows を表示の正とする。
func (s *CompanyProductionQueryService) GetProductionCreateContext(
	ctx context.Context,
	productBlueprintID string,
) (ProductionCreateContextDTO, error) {
	if productBlueprintID == "" {
		return ProductionCreateContextDTO{}, productiondom.ErrInvalidProductBlueprintID
	}

	companyID := usecase.CompanyIDFromContext(ctx)
	if companyID == "" {
		return ProductionCreateContextDTO{}, productbpdom.ErrInvalidCompanyID
	}

	if s.pbRepo == nil {
		return ProductionCreateContextDTO{}, productbpdom.ErrInternal
	}

	pb, err := s.pbRepo.GetByID(ctx, productBlueprintID)
	if err != nil {
		if productbpdom.IsNotFound(err) {
			return ProductionCreateContextDTO{}, productiondom.ErrNotFound
		}
		return ProductionCreateContextDTO{}, err
	}

	if pb.CompanyID == "" || pb.CompanyID != companyID {
		return ProductionCreateContextDTO{}, productiondom.ErrNotFound
	}

	brandName := ""
	if s.nameResolver != nil && pb.BrandID != "" {
		brandName = s.nameResolver.ResolveBrandName(ctx, pb.BrandID)
	}

	modelRefs := append([]productbpdom.ModelRef(nil), pb.ModelRefs...)
	sort.SliceStable(modelRefs, func(i, j int) bool {
		return modelRefs[i].DisplayOrder < modelRefs[j].DisplayOrder
	})

	rows := make([]ProductionDetailModelDTO, 0, len(modelRefs))
	for _, ref := range modelRefs {
		if ref.ModelID == "" {
			continue
		}

		rows = append(rows, s.buildProductionCreateModelRow(ctx, ref))
	}

	return ProductionCreateContextDTO{
		ProductBlueprintPatch: ProductionCreateProductBlueprintDTO{
			ID:                         pb.ID,
			ProductName:                pb.ProductName,
			BrandID:                    pb.BrandID,
			BrandName:                  brandName,
			ProductBlueprintCategoryID: pb.ProductBlueprintCategory.ID,
			ProductBlueprintCategory: ProductionProductBlueprintCategoryDTO{
				ID:     pb.ProductBlueprintCategory.ID,
				Code:   pb.ProductBlueprintCategory.Code,
				NameJa: pb.ProductBlueprintCategory.NameJa,
				NameEn: pb.ProductBlueprintCategory.NameEn,
				Kind:   string(pb.ProductBlueprintCategory.Kind),
				Path:   append([]string(nil), pb.ProductBlueprintCategory.Path...),
			},
			CategoryFields: cloneProductionCreateCategoryFields(pb.CategoryFields),
			AssigneeID:     pb.AssigneeID,
		},
		Rows: rows,
	}, nil
}

// ============================================================
// Production Create Model Builder
// ============================================================

func (s *CompanyProductionQueryService) buildProductionCreateModelRow(
	ctx context.Context,
	ref productbpdom.ModelRef,
) ProductionDetailModelDTO {
	displayOrder := ref.DisplayOrder

	row := ProductionDetailModelDTO{
		ModelID:      ref.ModelID,
		ModelNumber:  ref.ModelID,
		DisplayOrder: &displayOrder,
		Quantity:     0,
	}

	if s.nameResolver == nil {
		return row
	}

	attr := s.nameResolver.ResolveModelResolved(ctx, ref.ModelID)

	if attr.ModelNumber != "" {
		row.ModelNumber = attr.ModelNumber
	}

	row.Kind = attr.Kind

	if attr.Kind == "alcohol" {
		row.VolumeValue = attr.VolumeValue
		row.VolumeUnit = attr.VolumeUnit
		return row
	}

	row.Size = attr.Size
	row.Color = attr.Color
	row.RGB = attr.RGB

	return row
}

func cloneProductionCreateCategoryFields(
	fields productbpdom.CategoryFields,
) map[string]any {
	if fields == nil {
		return nil
	}

	out := make(map[string]any, len(fields))
	for key, value := range fields {
		out[key] = value
	}

	return out
}
