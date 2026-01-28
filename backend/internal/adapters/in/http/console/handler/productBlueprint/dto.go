// backend/internal/adapters/in/http/console/handler/productBlueprint/dto.go
package productBlueprint

// ---------------------------------------------------
// POST /product-blueprints
// ---------------------------------------------------

type ProductIdTagInput struct {
	Type string `json:"type"`
}

type CreateProductBlueprintInput struct {
	ProductName      string            `json:"productName"`
	BrandId          string            `json:"brandId"`
	ItemType         string            `json:"itemType"`
	Fit              string            `json:"fit"`
	Material         string            `json:"material"`
	Weight           float64           `json:"weight"`
	QualityAssurance []string          `json:"qualityAssurance"`
	ProductIdTag     ProductIdTagInput `json:"productIdTag"`
	AssigneeId       string            `json:"assigneeId"`
	CompanyId        string            `json:"companyId"`
	CreatedBy        string            `json:"createdBy,omitempty"`
}

// ---------------------------------------------------
// PUT/PATCH /product-blueprints/{id}
// ---------------------------------------------------

type UpdateProductBlueprintInput struct {
	ProductName      string            `json:"productName"`
	BrandId          string            `json:"brandId"`
	ItemType         string            `json:"itemType"`
	Fit              string            `json:"fit"`
	Material         string            `json:"material"`
	Weight           float64           `json:"weight"`
	QualityAssurance []string          `json:"qualityAssurance"`
	ProductIdTag     ProductIdTagInput `json:"productIdTag"`
	AssigneeId       string            `json:"assigneeId"`
	CompanyId        string            `json:"companyId"`
	UpdatedBy        string            `json:"updatedBy,omitempty"`
}

// ---------------------------------------------------
// GET /product-blueprints
// - backend 側で brandName / assigneeName を解決済みで返す
// - Printed を追加
// ---------------------------------------------------

type ProductBlueprintListOutput struct {
	ID           string `json:"id"`
	ProductName  string `json:"productName"`
	BrandName    string `json:"brandName"`
	AssigneeName string `json:"assigneeName"`
	ProductIdTag string `json:"productIdTag"`
	Printed      bool   `json:"printed"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

// ---------------------------------------------------
// GET /product-blueprints/deleted
// ---------------------------------------------------

type ProductBlueprintDeletedListOutput struct {
	ID          string `json:"id"`
	ProductName string `json:"productName"`
	BrandId     string `json:"brandId"`
	AssigneeId  string `json:"assigneeId"`
	DeletedAt   string `json:"deletedAt"`
	ExpireAt    string `json:"expireAt"`
}

// ---------------------------------------------------
// GET /product-blueprints/{id}/history
// ---------------------------------------------------

type ProductBlueprintHistoryOutput struct {
	ID          string  `json:"id"`
	ProductName string  `json:"productName"`
	BrandId     string  `json:"brandId"`
	AssigneeId  string  `json:"assigneeId"`
	UpdatedAt   string  `json:"updatedAt"`
	UpdatedBy   *string `json:"updatedBy,omitempty"`
	DeletedAt   string  `json:"deletedAt,omitempty"`
	ExpireAt    string  `json:"expireAt,omitempty"`
}
