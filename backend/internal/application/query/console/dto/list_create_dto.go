// backend/internal/application/query/dto/list_create_dto.go
package dto

// ListCreateDTO is a minimal DTO for listCreate screen.
// Current requirements:
// - From ProductBlueprint: brandName, productName
// - From TokenBlueprint: tokenName, brandName
//
// Note:
// - IDs are optional but useful for navigation/debugging and future expansion.
type ListCreateDTO struct {
	InventoryID        string `json:"inventoryId,omitempty"`
	ProductBlueprintID string `json:"productBlueprintId,omitempty"`
	TokenBlueprintID   string `json:"tokenBlueprintId,omitempty"`

	// From ProductBlueprint
	ProductBrandName string `json:"productBrandName"`
	ProductName      string `json:"productName"`

	// From TokenBlueprint
	TokenBrandName string `json:"tokenBrandName"`
	TokenName      string `json:"tokenName"`

	// list image url (primary / representative)
	// - list 作成画面で代表画像をプレビューする用途
	// - 未設定の場合は空文字
	ListImageURL string `json:"listImageUrl,omitempty"`

	// ProductBlueprintPatch.ModelRefs をそのまま返す（順序もそのまま）
	// - displayOrder を取得するのみ（並べ替えはしない）
	ModelRefs []ListCreateModelRefDTO `json:"modelRefs,omitempty"`

	// ------------------------------------------------------------
	// PriceCard 用
	// - ModelRefs を母集団に行を作る
	// - displayOrder は取得してそのまま渡す（並べ替えはしない）
	// - productBlueprintCategory / model kind に応じた表示用 model 情報を含める
	//   - apparel: size / color / rgb
	//   - alcohol: volumeValue / volumeUnit
	// - price は未入力を許容するため *int（null）にする
	// ------------------------------------------------------------
	PriceRows   []ListCreatePriceRowDTO `json:"priceRows,omitempty"`
	TotalStock  int                     `json:"totalStock,omitempty"`
	PriceNote   string                  `json:"priceNote,omitempty"`
	CurrencyJPY bool                    `json:"currencyJpy,omitempty"`
}

// ListCreateModelRefDTO is a lightweight ModelRef for UI.
// - displayOrder は「取得するのみ」
// - 0/未設定は null 扱いに寄せる
type ListCreateModelRefDTO struct {
	ModelID      string `json:"modelId"`
	DisplayOrder *int   `json:"displayOrder,omitempty"`
}

// ListCreatePriceRowDTO is a row DTO for PriceCard.
// - 型番列は出さないが、更新や作成 payload で識別できるよう ModelID は保持する。
// - productBlueprintCategory / model kind に応じた model 情報を含める。
type ListCreatePriceRowDTO struct {
	ModelID string `json:"modelId"`

	// model kind
	// - apparel
	// - alcohol
	Kind string `json:"kind,omitempty"`

	// 型番
	ModelNumber string `json:"modelNumber,omitempty"`

	// displayOrder（ProductBlueprintPatch.ModelRefs.DisplayOrder）
	// - 取得するのみ（サーバ側で並べ替えしない）
	// - 0/未設定は null 扱いに寄せる
	DisplayOrder *int `json:"displayOrder,omitempty"`

	// 在庫数
	Stock int `json:"stock"`

	// apparel 系表示用
	Size  string `json:"size,omitempty"`
	Color string `json:"color,omitempty"`
	RGB   *int   `json:"rgb,omitempty"`

	// alcohol 系表示用
	VolumeValue *int   `json:"volumeValue,omitempty"`
	VolumeUnit  string `json:"volumeUnit,omitempty"`

	// 価格（JPY想定）
	// - 未入力: null
	Price *int `json:"price,omitempty"`
}
