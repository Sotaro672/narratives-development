// backend/internal/adapters/in/http/console/handler/productBlueprint/dto.go
package productBlueprint

// ---------------------------------------------------
// Common DTOs
// ---------------------------------------------------

type ProductBlueprintCategoryInput struct {
	ID     string   `json:"id"`
	Code   string   `json:"code"`
	NameJa string   `json:"nameJa"`
	NameEn string   `json:"nameEn"`
	Kind   string   `json:"kind"`
	Path   []string `json:"path"`
}

type ProductBlueprintCategoryOutput struct {
	ID     string   `json:"id"`
	Code   string   `json:"code"`
	NameJa string   `json:"nameJa"`
	NameEn string   `json:"nameEn"`
	Kind   string   `json:"kind"`
	Path   []string `json:"path"`
}

// ---------------------------------------------------
// POST /product-blueprints
// ---------------------------------------------------

type CreateProductBlueprintInput struct {
	ProductName string `json:"productName"`
	Description string `json:"description"`

	BrandId   string `json:"brandId"`
	CompanyId string `json:"companyId"`

	ProductBlueprintCategory ProductBlueprintCategoryInput `json:"productBlueprintCategory"`

	// CategoryFields はカテゴリ別の productBlueprint 入力値を受け取る。
	//
	// 例:
	// - alcohol.sake:
	//   vintage, region, material, alcoholContent, volume
	// - apparel.tops:
	//   weight, fit, material
	// - cosmetics.skincare:
	//   material, volume
	//
	// brandId / productName / description などの共通 field はここには入れない。
	CategoryFields map[string]any `json:"categoryFields,omitempty"`

	AssigneeId string `json:"assigneeId"`
	CreatedBy  string `json:"createdBy,omitempty"`
}

// ---------------------------------------------------
// PATCH/PUT /product-blueprints/{id}
// ---------------------------------------------------

type UpdateProductBlueprintInput struct {
	ProductName string `json:"productName"`
	Description string `json:"description"`

	BrandId   string `json:"brandId"`
	CompanyId string `json:"companyId"`

	ProductBlueprintCategory ProductBlueprintCategoryInput `json:"productBlueprintCategory"`

	// nil / empty の扱いは handler / usecase / repository 側の方針に従う。
	// 今回の endpoint 実装では nil または空 map は nil として domain へ渡す。
	CategoryFields map[string]any `json:"categoryFields,omitempty"`

	AssigneeId string `json:"assigneeId"`
	UpdatedBy  string `json:"updatedBy,omitempty"`
}

// ---------------------------------------------------
// POST /product-blueprints/{id}/model-refs
// - productBlueprint 起票後に modelRefs（modelId + displayOrder）を追記する
// - updatedAt / updatedBy は更新しない（repo 側で touch しない更新を行う）
//
// 採用方針
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
	ID          string `json:"id"`
	ProductName string `json:"productName"`
	Description string `json:"description"`

	CompanyId string `json:"companyId"`
	BrandId   string `json:"brandId"`
	BrandName string `json:"brandName"`

	ProductBlueprintCategoryId string                         `json:"productBlueprintCategoryId"`
	ProductBlueprintCategory   ProductBlueprintCategoryOutput `json:"productBlueprintCategory"`

	// CategoryFields はカテゴリ別の productBlueprint 入力値。
	//
	// 例:
	// - alcohol.sake:
	//   vintage, region, material, alcoholContent, volume
	// - apparel.tops:
	//   weight, fit, material
	// - cosmetics.skincare:
	//   material, volume
	CategoryFields map[string]any `json:"categoryFields,omitempty"`

	AssigneeId   string `json:"assigneeId"`
	AssigneeName string `json:"assigneeName"`

	Printed bool `json:"printed"`

	CreatedBy     string `json:"createdBy"`
	CreatedByName string `json:"createdByName"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`

	// modelRefs（model docId + displayOrder）
	ModelRefs []ModelRefOutput `json:"modelRefs,omitempty"`
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
}
