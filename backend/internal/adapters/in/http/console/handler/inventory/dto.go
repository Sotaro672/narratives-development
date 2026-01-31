// backend/internal/adapters/in/http/console/handler/inventory/dto.go
package inventory

type createInventoryMintRequest struct {
	TokenBlueprintID   string   `json:"tokenBlueprintId"`
	ProductBlueprintID string   `json:"productBlueprintId"`
	ModelID            string   `json:"modelId"`
	ProductIDs         []string `json:"productIds"`
}

type updateInventoryMintRequest struct {
	ModelID    string   `json:"modelId"`
	ProductIDs []string `json:"productIds"`
}
