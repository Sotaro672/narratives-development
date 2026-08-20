// backend/internal/application/query/console/dto/inventory_dto.go
package dto

// ============================================================
// DTOs (Inventory Management List)
// - /inventory (management list)
// ============================================================
type InventoryManagementRowDTO struct {
	ProductBlueprintID  string `json:"productBlueprintId"`
	ProductName         string `json:"productName"`
	TokenBlueprintID    string `json:"tokenBlueprintId"`
	TokenName           string `json:"tokenName"`
	ShippingAddressName string `json:"shippingAddressName"`
	AvailableStock      int    `json:"availableStock"`
	ReservedCount       int    `json:"reservedCount"`
}
