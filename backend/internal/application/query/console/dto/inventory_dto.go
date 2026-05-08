// backend/internal/application/query/dto/inventory_dto.go
package dto

// ============================================================
// DTOs (Inventory Management List)
// - ✅ /inventory (management list)
// ============================================================

type InventoryManagementRowDTO struct {
	ProductBlueprintID string `json:"productBlueprintId"`
	ProductName        string `json:"productName"`
	TokenBlueprintID   string `json:"tokenBlueprintId"` // ✅ 必須
	TokenName          string `json:"tokenName"`
	ModelNumber        string `json:"modelNumber"`

	// 互換: 従来の stock は availableStock と同義にする
	Stock int `json:"stock"`

	// ✅ NEW: 画面で必要な内訳
	AvailableStock int `json:"availableStock"`
	ReservedCount  int `json:"reservedCount"`
}

// ============================================================
// ✅ /inventory/ids response
// ============================================================

type InventoryIDsByProductAndTokenDTO struct {
	ProductBlueprintID string   `json:"productBlueprintId"`
	TokenBlueprintID   string   `json:"tokenBlueprintId"`
	InventoryIDs       []string `json:"inventoryIds"`
}
