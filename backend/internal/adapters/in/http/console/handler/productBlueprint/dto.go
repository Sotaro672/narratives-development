// backend\internal\adapters\in\http\console\handler\productBlueprint\dto.go
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
// PATCH/PUT /product-blueprints/{id}
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
// POST /product-blueprints/{id}/model-refs
// - productBlueprint 起票後に modelRefs（modelId + displayOrder）を追記する
// - updatedAt / updatedBy は更新しない（repo 側で touch しない更新を行う）
//
// ★ 採用方針（案1）
//   - 入力: modelIds（順序は「色登録順→サイズ登録順」に並んだもの）
//   - 出力: detail（既存の toDetailOutput）
// ---------------------------------------------------

type AppendModelRefsInput struct {
	// model テーブルの docId の配列（順序は displayOrder の採番元）
	ModelIds []string `json:"modelIds"`
}

// ---------------------------------------------------
// GET /product-blueprints (list)
// - backend 側で name 解決済みを返す
// ---------------------------------------------------

type ProductBlueprintListOutput struct {
	ID           string `json:"id"`
	ProductName  string `json:"productName"`
	BrandName    string `json:"brandName"`
	AssigneeName string `json:"assigneeName"`
	Printed      bool   `json:"printed"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

// ---------------------------------------------------
// GET /product-blueprints/{id} (detail)
// - backend 側で name 解決済みを返す
// ---------------------------------------------------

type ModelRefOutput struct {
	ModelId      string `json:"modelId"`
	DisplayOrder int    `json:"displayOrder"`
}

type ProductBlueprintDetailOutput struct {
	ID               string   `json:"id"`
	ProductName      string   `json:"productName"`
	CompanyId        string   `json:"companyId"`
	BrandId          string   `json:"brandId"`
	BrandName        string   `json:"brandName"`
	ItemType         string   `json:"itemType"`
	Fit              string   `json:"fit"`
	Material         string   `json:"material"`
	Weight           float64  `json:"weight"`
	QualityAssurance []string `json:"qualityAssurance"`

	ProductIdTag *struct {
		Type string `json:"type"`
	} `json:"productIdTag,omitempty"`

	AssigneeId   string `json:"assigneeId"`
	AssigneeName string `json:"assigneeName"`

	Printed bool `json:"printed"`

	CreatedBy     string `json:"createdBy"`
	CreatedByName string `json:"createdByName"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`

	DeletedAt string `json:"deletedAt,omitempty"`

	// ★ 追加: modelRefs（model docId + displayOrder）
	ModelRefs []ModelRefOutput `json:"modelRefs,omitempty"`
}

// ---------------------------------------------------
// GET /product-blueprints/deleted`
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
