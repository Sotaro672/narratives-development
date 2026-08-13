// backend/internal/application/query/console/dto/mint_product_blueprint.go
package dto

type MintProductBlueprintCategoryDTO struct {
	ID     string   `json:"id"`
	Code   string   `json:"code"`
	NameJa string   `json:"nameJa"`
	NameEn string   `json:"nameEn"`
	Kind   string   `json:"kind"`
	Path   []string `json:"path"`
}

type MintProductIDTagDTO struct {
	Type string `json:"type"`
}

type MintProductBlueprintModelRefDTO struct {
	ModelID      string `json:"modelId"`
	DisplayOrder int    `json:"displayOrder"`
}

type MintProductBlueprintDTO struct {
	ProductName string `json:"productName"`
	Description string `json:"description"`

	BrandID   string `json:"brandId"`
	BrandName string `json:"brandName"`
	CompanyID string `json:"companyId"`

	ProductBlueprintCategory MintProductBlueprintCategoryDTO `json:"productBlueprintCategory"`
	CategoryFields           map[string]any                  `json:"categoryFields"`

	ProductIDTag MintProductIDTagDTO `json:"productIdTag"`
	AssigneeID   string              `json:"assigneeId"`

	ModelRefs []MintProductBlueprintModelRefDTO `json:"modelRefs"`
}
