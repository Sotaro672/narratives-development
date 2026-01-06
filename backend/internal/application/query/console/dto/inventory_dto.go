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
	Stock              int    `json:"stock"`
}

// ============================================================
// ✅ /inventory/ids response
// ============================================================

type InventoryIDsByProductAndTokenDTO struct {
	ProductBlueprintID string   `json:"productBlueprintId"`
	TokenBlueprintID   string   `json:"tokenBlueprintId"`
	InventoryIDs       []string `json:"inventoryIds"`
}
