// backend/internal/application/query/mall/dto/order_dto.go
package dto

type OrderContextDTO struct {
	UID             string                 `json:"uid"`
	AvatarID        string                 `json:"avatarId"`
	UserID          string                 `json:"userId"`
	FullName        string                 `json:"fullName,omitempty"`
	ShippingAddress map[string]any         `json:"shippingAddress,omitempty"`
	PaymentMethod   map[string]any         `json:"paymentMethod,omitempty"`
	CartItems       map[string]CartItemDTO `json:"cartItems,omitempty"`
}
