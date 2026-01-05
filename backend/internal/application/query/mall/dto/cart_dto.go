// backend\internal\application\query\mall\dto\cart_dto.go
package dto

// CartDTO is the response shape for SNS cart list UI.
// NOTE: cart_query returns ONLY minimal fields required for cart screen.
type CartDTO struct {
	AvatarID string                 `json:"avatarId"`
	Items    map[string]CartItemDTO `json:"items"`

	CreatedAt *string `json:"createdAt,omitempty"`
	UpdatedAt *string `json:"updatedAt,omitempty"`
	ExpiresAt *string `json:"expiresAt,omitempty"`
}

type CartItemDTO struct {
	// ✅ identifiers (UI がそのまま表示/操作に使える)
	InventoryID string `json:"inventoryId,omitempty"`
	ListID      string `json:"listId,omitempty"`
	ModelID     string `json:"modelId,omitempty"`

	// ✅ resolved fields for cart view
	Title     string `json:"title,omitempty"`
	ListImage string `json:"listImage,omitempty"` // List.ImageID(URL or imageId)
	Price     *int   `json:"price,omitempty"`     // JPY

	ProductName string `json:"productName,omitempty"`
	Size        string `json:"size,omitempty"`
	Color       string `json:"color,omitempty"`

	Qty int `json:"qty"`
}
