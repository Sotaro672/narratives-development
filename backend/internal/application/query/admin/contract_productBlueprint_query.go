// backend/internal/application/query/admin/contaract_productBlueprint_query.go
package query

import (
	"context"
	"errors"

	branddom "narratives/internal/domain/brand"
	companydom "narratives/internal/domain/company"
	memberdom "narratives/internal/domain/member"
	modeldom "narratives/internal/domain/model"
	productblueprintdom "narratives/internal/domain/productBlueprint"
)

var ErrContractProductBlueprintQueryNotConfigured = errors.New(
	"contract product blueprint query is not configured",
)

type contractProductBlueprintCompanyReader interface {
	GetByID(ctx context.Context, id string) (companydom.Company, error)
}

type contractProductBlueprintBrandReader interface {
	GetByID(ctx context.Context, id string) (branddom.Brand, error)
}

type contractProductBlueprintMemberReader interface {
	GetByID(ctx context.Context, id string) (memberdom.Record, error)
	GetByUID(ctx context.Context, uid string) (memberdom.Record, error)
}

type contractProductBlueprintReader interface {
	GetByID(ctx context.Context, id string) (productblueprintdom.ProductBlueprint, error)
}

type contractProductBlueprintModelReader interface {
	GetByID(ctx context.Context, variationID string) (modeldom.ModelVariation, error)
}

type ContractProductBlueprintQuery struct {
	companyRepo          contractProductBlueprintCompanyReader
	brandRepo            contractProductBlueprintBrandReader
	memberRepo           contractProductBlueprintMemberReader
	productBlueprintRepo contractProductBlueprintReader
	modelRepo            contractProductBlueprintModelReader
}

func NewContractProductBlueprintQuery(
	companyRepo contractProductBlueprintCompanyReader,
	brandRepo contractProductBlueprintBrandReader,
	memberRepo contractProductBlueprintMemberReader,
	productBlueprintRepo contractProductBlueprintReader,
	modelRepo contractProductBlueprintModelReader,
) *ContractProductBlueprintQuery {
	return &ContractProductBlueprintQuery{
		companyRepo:          companyRepo,
		brandRepo:            brandRepo,
		memberRepo:           memberRepo,
		productBlueprintRepo: productBlueprintRepo,
		modelRepo:            modelRepo,
	}
}

type ContractProductBlueprintDetailResult struct {
	Company          ContractCompanyRow                `json:"company"`
	ProductBlueprint ContractProductBlueprintDetailRow `json:"productBlueprint"`
}

type ContractProductBlueprintDetailRow struct {
	ID                           string                             `json:"id"`
	ProductName                  string                             `json:"productName"`
	Description                  string                             `json:"description"`
	BrandID                      string                             `json:"brandId"`
	BrandName                    string                             `json:"brandName"`
	CompanyID                    string                             `json:"companyId"`
	ProductBlueprintCategoryPath []string                           `json:"productBlueprintCategoryPath"`
	CategoryFields               map[string]any                     `json:"categoryFields"`
	ProductIDTagType             string                             `json:"productIdTagType"`
	AssigneeID                   string                             `json:"assigneeId"`
	AssigneeName                 string                             `json:"assigneeName"`
	ModelRefs                    []ContractProductBlueprintModelRef `json:"modelRefs"`
	Printed                      bool                               `json:"printed"`
	CreatedAt                    string                             `json:"createdAt"`
	UpdatedAt                    string                             `json:"updatedAt"`
}

type ContractProductBlueprintModelRef struct {
	ModelID      string `json:"modelId"`
	DisplayOrder int    `json:"displayOrder"`
	Kind         string `json:"kind"`
	ModelNumber  string `json:"modelNumber"`
	Size         string `json:"size,omitempty"`
	Color        string `json:"color,omitempty"`
	RGB          *int   `json:"rgb,omitempty"`
	VolumeValue  *int   `json:"volumeValue,omitempty"`
	VolumeUnit   string `json:"volumeUnit,omitempty"`
}

func (q *ContractProductBlueprintQuery) Get(
	ctx context.Context,
	companyID string,
	productBlueprintID string,
) (ContractProductBlueprintDetailResult, error) {
	if q == nil ||
		q.companyRepo == nil ||
		q.brandRepo == nil ||
		q.memberRepo == nil ||
		q.productBlueprintRepo == nil ||
		q.modelRepo == nil {
		return ContractProductBlueprintDetailResult{}, ErrContractProductBlueprintQueryNotConfigured
	}
	if companyID == "" {
		return ContractProductBlueprintDetailResult{}, companydom.ErrInvalidID
	}
	if productBlueprintID == "" {
		return ContractProductBlueprintDetailResult{}, productblueprintdom.ErrInvalidID
	}

	company, err := q.companyRepo.GetByID(ctx, companyID)
	if err != nil {
		return ContractProductBlueprintDetailResult{}, err
	}

	productBlueprint, err := q.productBlueprintRepo.GetByID(ctx, productBlueprintID)
	if err != nil {
		return ContractProductBlueprintDetailResult{}, err
	}
	if productBlueprint.ID != productBlueprintID || productBlueprint.CompanyID != companyID {
		return ContractProductBlueprintDetailResult{}, productblueprintdom.ErrNotFound
	}

	modelRefs := make([]ContractProductBlueprintModelRef, 0, len(productBlueprint.ModelRefs))
	for _, modelRef := range productBlueprint.ModelRefs {
		row := ContractProductBlueprintModelRef{
			ModelID:      modelRef.ModelID,
			DisplayOrder: modelRef.DisplayOrder,
		}

		if modelRef.ModelID != "" {
			variation, err := q.modelRepo.GetByID(ctx, modelRef.ModelID)
			if err == nil && variation != nil {
				switch model := variation.(type) {
				case modeldom.ApparelModelVariation:
					rgb := model.Color.RGB
					row.Kind = string(modeldom.ModelVariationKindApparel)
					row.ModelNumber = model.ModelNumber
					row.Size = model.Size
					row.Color = model.Color.Name
					row.RGB = &rgb

				case modeldom.AlcoholModelVariation:
					volumeValue := model.Volume.Value
					row.Kind = string(modeldom.ModelVariationKindAlcohol)
					row.ModelNumber = model.ModelNumber
					row.VolumeValue = &volumeValue
					row.VolumeUnit = model.Volume.Unit
				}
			}
		}

		modelRefs = append(modelRefs, row)
	}

	categoryFields := make(map[string]any, len(productBlueprint.CategoryFields))
	for key, value := range productBlueprint.CategoryFields {
		categoryFields[key] = value
	}

	representativeName := q.resolveMemberName(ctx, company.Admin)
	if representativeName == "" || representativeName == company.Admin {
		representativeName = "-"
	}

	return ContractProductBlueprintDetailResult{
		Company: ContractCompanyRow{
			ID:                 company.ID,
			Name:               company.Name,
			RepresentativeName: representativeName,
			IsActive:           company.IsActive,
			CreatedAt:          formatContractDetailTime(company.CreatedAt),
			UpdatedAt:          formatContractDetailTime(company.UpdatedAt),
		},
		ProductBlueprint: ContractProductBlueprintDetailRow{
			ID:          productBlueprint.ID,
			ProductName: productBlueprint.ProductName,
			Description: productBlueprint.Description,
			BrandID:     productBlueprint.BrandID,
			BrandName:   q.resolveBrandName(ctx, productBlueprint.BrandID),
			CompanyID:   productBlueprint.CompanyID,
			ProductBlueprintCategoryPath: append(
				[]string(nil),
				productBlueprint.ProductBlueprintCategoryPath...,
			),
			CategoryFields:   categoryFields,
			ProductIDTagType: string(productBlueprint.ProductIdTag.Type),
			AssigneeID:       productBlueprint.AssigneeID,
			AssigneeName:     q.resolveMemberName(ctx, productBlueprint.AssigneeID),
			ModelRefs:        modelRefs,
			Printed:          productBlueprint.Printed,
			CreatedAt:        formatContractDetailTime(productBlueprint.CreatedAt),
			UpdatedAt:        formatContractDetailTime(productBlueprint.UpdatedAt),
		},
	}, nil
}

func (q *ContractProductBlueprintQuery) resolveBrandName(ctx context.Context, brandID string) string {
	if brandID == "" {
		return "-"
	}

	brand, err := q.brandRepo.GetByID(ctx, brandID)
	if err != nil || brand.Name == "" {
		return brandID
	}

	return brand.Name
}

func (q *ContractProductBlueprintQuery) resolveMemberName(ctx context.Context, memberID string) string {
	if memberID == "" {
		return "-"
	}

	record, err := q.memberRepo.GetByID(ctx, memberID)
	if err == nil {
		name := memberdom.FormatLastFirst(record.Member.LastName, record.Member.FirstName)
		if name != "" {
			return name
		}
	}

	record, err = q.memberRepo.GetByUID(ctx, memberID)
	if err == nil {
		name := memberdom.FormatLastFirst(record.Member.LastName, record.Member.FirstName)
		if name != "" {
			return name
		}
	}

	return memberID
}
