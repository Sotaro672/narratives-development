// backend/internal/application/query/dto/list_create_dto.go
package dto

// ListCreateDTO is a minimal DTO for listCreate screen.
// Current requirements:
// - From ProductBlueprint: brandName, productName
// - From TokenBlueprint: tokenName, brandName
// - PriceCard: priceRows
type ListCreateDTO struct {
	// From ProductBlueprint
	ProductBrandName string `json:"productBrandName"`
	ProductName      string `json:"productName"`

	// From TokenBlueprint
	TokenBrandName string `json:"tokenBrandName"`
	TokenName      string `json:"tokenName"`

	// PriceCard 用
	PriceRows []ListCreatePriceRowDTO `json:"priceRows,omitempty"`
}

// ListCreateModelRefDTO is a lightweight ModelRef for query internal use.
// - displayOrder は「取得するのみ」
// - 0/未設定は null 扱いに寄せる
type ListCreateModelRefDTO struct {
	ModelID      string `json:"modelId"`
	DisplayOrder *int   `json:"displayOrder,omitempty"`
}

// ListCreatePriceRowDTO is a row DTO for PriceCard.
// - 更新や作成 payload で識別できるよう ModelID は保持する。
// - productBlueprintCategory / model kind に応じた model 情報を含める。
type ListCreatePriceRowDTO struct {
	ModelID string `json:"modelId"`

	// model kind
	// - apparel
	// - alcohol
	Kind string `json:"kind,omitempty"`

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

	// 価格
	// - 未入力: null
	Price *int `json:"price,omitempty"`
}
