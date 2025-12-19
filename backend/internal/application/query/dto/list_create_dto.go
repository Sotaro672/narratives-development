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

	// ------------------------------------------------------------
	// ✅ PriceCard 用（サイズ/カラー別に価格を入力するための行）
	// - inventory_detail_dto.go の InventoryDetailRowDTO を参照しつつ、
	//   PriceCard では「型番列を出さない」前提なので ModelNumber は持たない
	// - price は未入力を許容するため *int（null）にする
	// ------------------------------------------------------------
	PriceRows   []ListCreatePriceRowDTO `json:"priceRows,omitempty"`
	TotalStock  int                     `json:"totalStock,omitempty"`
	PriceNote   string                  `json:"priceNote,omitempty"`   // 任意: 画面メモ用途（未使用なら空）
	CurrencyJPY bool                    `json:"currencyJpy,omitempty"` // 任意: フロントで "¥" を固定する用途（未使用なら false）
}

// ListCreatePriceRowDTO is a row DTO for PriceCard.
// - 型番列は出さないが、更新や作成 payload で識別できるよう ModelID は保持する。
type ListCreatePriceRowDTO struct {
	ModelID string `json:"modelId"`

	// 在庫数（まずは stock のみ通す）
	Stock int `json:"stock"`

	// 表示用
	Size  string `json:"size"`
	Color string `json:"color"`
	RGB   *int   `json:"rgb,omitempty"`

	// ✅ 追加: 価格（JPY想定）
	// - 未入力: null
	Price *int `json:"price,omitempty"`
}
