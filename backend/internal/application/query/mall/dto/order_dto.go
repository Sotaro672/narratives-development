// backend/internal/application/query/mall/dto/order_dto.go
package dto

import orderdom "narratives/internal/domain/order"

type OrderContextDTO struct {
	UID      string `json:"uid"`
	AvatarID string `json:"avatarId"`
	UserID   string `json:"userId"`
	FullName string `json:"fullName,omitempty"`

	ShippingSnapshot      *orderdom.ShippingSnapshot      `json:"shippingSnapshot,omitempty"`
	PaymentMethodSnapshot *orderdom.PaymentMethodSnapshot `json:"paymentMethodSnapshot,omitempty"`

	CartItems map[string]CartItemDTO `json:"cartItems,omitempty"`
}
