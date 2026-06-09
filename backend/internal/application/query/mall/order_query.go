// backend/internal/application/query/mall/order_query.go
package mall

import (
	"context"
	"errors"
	"log"

	dto "narratives/internal/application/query/mall/dto"
	appresolver "narratives/internal/application/resolver"
	avatardom "narratives/internal/domain/avatar"
	cart "narratives/internal/domain/cart"
	orderdom "narratives/internal/domain/order"
	paymentmethod "narratives/internal/domain/paymentMethod"
	shippingaddress "narratives/internal/domain/shippingAddress"
)

// OrderQuery resolves mall buyer order context.
//
// Responsibilities:
// - uid -> avatarId
// - userId -> shippingSnapshot / paymentMethodSnapshot
// - avatarId -> cartItems
// - userId -> fullName
type OrderQuery struct {
	// required: avatar repository
	// - uid/userId -> avatarId resolution
	AvatarRepo avatardom.Repository

	// optional: cart repository
	// - if nil, GetOrderContextByUID skips cart item resolution
	CartRepo cart.Repository

	// optional: shipping address repository
	// - if nil, GetOrderContextByUID skips shipping snapshot resolution
	// - ListByUserID の先頭を注文文脈用 shipping snapshot として採用する
	ShippingAddressRepo shippingaddress.RepositoryPort

	// optional: payment method repository
	// - if nil, GetOrderContextByUID skips payment method snapshot resolution
	// - GetDefaultByUser を注文文脈用 payment method snapshot として採用する
	PaymentMethodRepo paymentmethod.RepositoryPort

	// optional: name resolver
	// - if nil, FullName will be empty
	NameResolver *appresolver.NameResolver
}

func NewOrderQuery(
	avatarRepo avatardom.Repository,
	cartRepo cart.Repository,
	shippingAddressRepo shippingaddress.RepositoryPort,
	paymentMethodRepo paymentmethod.RepositoryPort,
	nameResolver *appresolver.NameResolver,
) *OrderQuery {
	return &OrderQuery{
		AvatarRepo:          avatarRepo,
		CartRepo:            cartRepo,
		ShippingAddressRepo: shippingAddressRepo,
		PaymentMethodRepo:   paymentMethodRepo,
		NameResolver:        nameResolver,
	}
}

// GetAvatarIDByUID resolves uid -> avatarId only.
//
// Intended for middleware use.
// If not found, returns ErrNotFound.
func (q *OrderQuery) GetAvatarIDByUID(ctx context.Context, uid string) (string, error) {
	if q == nil || q.AvatarRepo == nil {
		return "", errors.New("mall order query: avatar repository is nil")
	}
	if uid == "" {
		return "", errors.New("uid is required")
	}

	avatarID, _, err := q.findAvatarIdentityByUID(ctx, uid)
	if err != nil {
		return "", err
	}

	return avatarID, nil
}

// GetOrderContextByUID resolves uid -> order context.
//
// It resolves:
// - avatarId
// - userId
// - shippingSnapshot
// - paymentMethodSnapshot
// - cartItems
// - fullName
//
// If avatar is not found, returns ErrNotFound.
func (q *OrderQuery) GetOrderContextByUID(ctx context.Context, uid string) (dto.OrderContextDTO, error) {
	if q == nil || q.AvatarRepo == nil {
		return dto.OrderContextDTO{}, errors.New("mall order query: avatar repository is nil")
	}
	if uid == "" {
		return dto.OrderContextDTO{}, errors.New("uid is required")
	}

	avatarID, avatarUserID, err := q.findAvatarIdentityByUID(ctx, uid)
	if err != nil {
		return dto.OrderContextDTO{}, err
	}

	// userId は基本 uid と一致させる。
	// avatar の userId が取得できた場合はそちらを尊重する。
	userID := avatarUserID
	if userID == "" {
		userID = uid
	}

	// shippingAddresses:
	// - docID は userId ではない
	// - UserID = owner uid
	// - 1 user can have many shipping addresses
	// - 注文文脈では ListByUserID の先頭を snapshot 化する
	shippingSnapshot := q.fetchShippingSnapshotBestEffort(ctx, userID)

	// paymentMethods:
	// - docID は userId ではない
	// - 注文文脈では default paymentMethod を snapshot 化する
	paymentMethodSnapshot := q.fetchPaymentMethodSnapshotBestEffort(ctx, userID)

	// cartItems:
	// - carts docID = avatarId
	// - items の itemKey は上位層で分解しない
	cartItems := q.fetchCartItemsBestEffort(ctx, avatarID)

	// fullName（best-effort）
	fullName := ""
	if q.NameResolver != nil {
		fullName = q.NameResolver.ResolveMemberName(ctx, userID)
	}

	return dto.OrderContextDTO{
		UID:                   uid,
		AvatarID:              avatarID,
		UserID:                userID,
		FullName:              fullName,
		ShippingSnapshot:      shippingSnapshot,
		PaymentMethodSnapshot: paymentMethodSnapshot,
		CartItems:             cartItems,
	}, nil
}

// ------------------------------------------------------------
// uid -> avatar identity
// ------------------------------------------------------------

// findAvatarIdentityByUID finds an avatar by uid/userId.
//
// It returns:
// - avatarID: avatar document ID
// - userID: avatar.userId
func (q *OrderQuery) findAvatarIdentityByUID(ctx context.Context, uid string) (avatarID string, userID string, err error) {
	if q == nil || q.AvatarRepo == nil {
		return "", "", errors.New("mall order query: avatar repository is nil")
	}
	if uid == "" {
		return "", "", errors.New("uid is required")
	}

	a, err := q.AvatarRepo.GetByUserID(ctx, uid)
	if err != nil {
		return "", "", err
	}
	if a.ID == "" {
		return "", "", ErrNotFound
	}

	resolvedUserID := a.UserID
	if resolvedUserID == "" {
		resolvedUserID = uid
	}

	log.Printf(
		"[mall_order_query] findAvatarIdentityByUID ok uid=%q avatarId=%q userId=%q",
		uid,
		a.ID,
		resolvedUserID,
	)

	return a.ID, resolvedUserID, nil
}

// ------------------------------------------------------------
// userId -> shippingSnapshot
// ------------------------------------------------------------

func (q *OrderQuery) fetchShippingSnapshotBestEffort(ctx context.Context, userID string) *orderdom.ShippingSnapshot {
	if q == nil || q.ShippingAddressRepo == nil || userID == "" {
		return nil
	}

	addresses, err := q.ShippingAddressRepo.ListByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, shippingaddress.ErrNotFound) {
			return nil
		}
		log.Printf("[mall_order_query] shipping address query error userId=%q err=%v", userID, err)
		return nil
	}
	if len(addresses) == 0 {
		return nil
	}

	return shippingAddressToSnapshot(addresses[0])
}

func shippingAddressToSnapshot(a shippingaddress.ShippingAddress) *orderdom.ShippingSnapshot {
	return &orderdom.ShippingSnapshot{
		ZipCode: a.ZipCode,
		State:   a.State,
		City:    a.City,
		Street:  a.Street,
		Street2: a.Street2,
		Country: a.Country,
	}
}

// ------------------------------------------------------------
// userId -> paymentMethodSnapshot
// ------------------------------------------------------------

func (q *OrderQuery) fetchPaymentMethodSnapshotBestEffort(ctx context.Context, userID string) *orderdom.PaymentMethodSnapshot {
	if q == nil || q.PaymentMethodRepo == nil || userID == "" {
		return nil
	}

	pm, err := q.PaymentMethodRepo.GetDefaultByUser(ctx, userID)
	if err != nil {
		if errors.Is(err, paymentmethod.ErrNotFound) {
			return nil
		}
		log.Printf("[mall_order_query] payment method query error userId=%q err=%v", userID, err)
		return nil
	}
	if pm == nil {
		return nil
	}

	return paymentMethodToSnapshot(*pm)
}

func paymentMethodToSnapshot(pm paymentmethod.PaymentMethod) *orderdom.PaymentMethodSnapshot {
	return &orderdom.PaymentMethodSnapshot{
		CustomerID:     pm.StripeCustomerID,
		Brand:          pm.Brand,
		Last4:          pm.Last4,
		ExpMonth:       pm.ExpMonth,
		ExpYear:        pm.ExpYear,
		CardholderName: pm.CardholderName,
		IsDefault:      pm.IsDefault,
	}
}

// ------------------------------------------------------------
// avatarId -> cartItems
// ------------------------------------------------------------

func (q *OrderQuery) fetchCartItemsBestEffort(ctx context.Context, avatarID string) map[string]dto.CartItemDTO {
	if q == nil || q.CartRepo == nil || avatarID == "" {
		return nil
	}

	c, err := q.CartRepo.GetByAvatarID(ctx, avatarID)
	if err != nil {
		log.Printf("[mall_order_query] cart query error avatarId=%q err=%v", avatarID, err)
		return nil
	}
	if c == nil || len(c.Items) == 0 {
		return map[string]dto.CartItemDTO{}
	}

	return cartItemsToDTOMap(c.Items)
}

func cartItemsToDTOMap(items map[string]cart.CartItem) map[string]dto.CartItemDTO {
	if len(items) == 0 {
		return map[string]dto.CartItemDTO{}
	}

	out := make(map[string]dto.CartItemDTO, len(items))
	for itemKey, item := range items {
		out[itemKey] = cartItemToDTO(item)
	}
	return out
}

func cartItemToDTO(item cart.CartItem) dto.CartItemDTO {
	return dto.CartItemDTO{
		InventoryID: item.InventoryID,
		ListID:      item.ListID,
		ModelID:     item.ModelID,
		Qty:         item.Qty,
	}
}
