// backend/internal/application/query/mall/order_query.go
package mall

import (
	"context"
	"errors"
	"fmt"
	"log"

	dto "narratives/internal/application/query/mall/dto"
	appresolver "narratives/internal/application/resolver"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

// OrderQuery resolves mall buyer order context.
//
// Responsibilities:
// - uid -> avatarId
// - userId -> shippingAddress / paymentMethod
// - avatarId -> cartItems
// - userId -> fullName
type OrderQuery struct {
	FS *firestore.Client

	// optional: cart read-model
	// - if nil, GetOrderContextByUID skips cart item resolution
	CartQ *CartQuery

	// optional: name resolver
	// - if nil, FullName will be empty
	NameResolver *appresolver.NameResolver

	// collection names
	AvatarsCol         string
	ShippingAddressCol string
	PaymentMethodCol   string

	// field name used in avatars collection
	AvatarUserIDField string

	// field name used in shipping/paymentMethod collections
	UserIDField string
}

func NewOrderQuery(fs *firestore.Client) *OrderQuery {
	return &OrderQuery{
		FS:                 fs,
		CartQ:              nil,
		NameResolver:       nil,
		AvatarsCol:         "avatars",
		ShippingAddressCol: "shippingAddresses",
		PaymentMethodCol:   "paymentMethods",
		AvatarUserIDField:  "userId",
		UserIDField:        "userId",
	}
}

func NewOrderQueryWithCartQuery(fs *firestore.Client, cartQ *CartQuery) *OrderQuery {
	q := NewOrderQuery(fs)
	q.CartQ = cartQ
	return q
}

// GetAvatarIDByUID resolves uid -> avatarId only.
//
// Intended for middleware use.
// If not found, returns ErrNotFound.
func (q *OrderQuery) GetAvatarIDByUID(ctx context.Context, uid string) (string, error) {
	if q == nil || q.FS == nil {
		return "", errors.New("mall order query: firestore client is nil")
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
// - shippingAddress
// - paymentMethod
// - cartItems
// - fullName
//
// If avatar is not found, returns ErrNotFound.
func (q *OrderQuery) GetOrderContextByUID(ctx context.Context, uid string) (dto.OrderContextDTO, error) {
	if q == nil || q.FS == nil {
		return dto.OrderContextDTO{}, errors.New("mall order query: firestore client is nil")
	}
	if uid == "" {
		return dto.OrderContextDTO{}, errors.New("uid is required")
	}

	avatarID, avatarUserID, err := q.findAvatarIdentityByUID(ctx, uid)
	if err != nil {
		return dto.OrderContextDTO{}, err
	}

	// userId は基本 uid と一致させる。
	// avatars の userId が取得できた場合はそちらを尊重する。
	userID := avatarUserID
	if userID == "" {
		userID = uid
	}

	// Firestore 実データ前提:
	// - shippingAddresses / paymentMethods の docID は userId ではない
	// - userId フィールドで検索する
	ship := q.fetchDocByUserID(ctx, q.ShippingAddressCol, userID, "shippingAddress")
	paymentMethod := q.fetchDocByUserID(ctx, q.PaymentMethodCol, userID, "paymentMethod")

	// cartItems（best-effort）
	cartItems := q.fetchCartItemsBestEffort(ctx, avatarID)

	// fullName（best-effort）
	fullName := ""
	if q.NameResolver != nil {
		fullName = q.NameResolver.ResolveMemberName(ctx, userID)
	}

	return dto.OrderContextDTO{
		UID:             uid,
		AvatarID:        avatarID,
		UserID:          userID,
		FullName:        fullName,
		ShippingAddress: ship,
		PaymentMethod:   paymentMethod,
		CartItems:       cartItems,
	}, nil
}

// ------------------------------------------------------------
// uid -> avatar identity
// ------------------------------------------------------------

// findAvatarIdentityByUID finds an avatar document by uid.
//
// It returns:
// - avatarID: avatar document ID
// - userID: avatars.{avatarID}.userId
func (q *OrderQuery) findAvatarIdentityByUID(ctx context.Context, uid string) (avatarID string, userID string, err error) {
	col := q.AvatarsCol
	if col == "" {
		col = "avatars"
	}

	userField := q.AvatarUserIDField
	if userField == "" {
		userField = "userId"
	}

	iter := q.FS.Collection(col).
		Where(userField, "==", uid).
		Limit(1).
		Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err != nil {
		if err == iterator.Done {
			return "", "", ErrNotFound
		}
		return "", "", err
	}
	if doc == nil || doc.Ref == nil {
		return "", "", ErrNotFound
	}

	m := doc.Data()

	resolvedUserID := ""
	if v, ok := m[userField]; ok {
		resolvedUserID = fmt.Sprint(v)
	}

	resolvedAvatarID := doc.Ref.ID
	if resolvedAvatarID == "" {
		return "", "", ErrNotFound
	}

	log.Printf(
		"[mall_order_query] findAvatarIdentityByUID ok uid=%q avatarId=%q userId=%q",
		uid,
		resolvedAvatarID,
		resolvedUserID,
	)

	return resolvedAvatarID, resolvedUserID, nil
}

// ------------------------------------------------------------
// avatarId -> cartItems
// ------------------------------------------------------------

func (q *OrderQuery) fetchCartItemsBestEffort(ctx context.Context, avatarID string) map[string]dto.CartItemDTO {
	if q == nil {
		return nil
	}
	if avatarID == "" {
		return nil
	}

	cq := q.CartQ
	if cq == nil {
		return nil
	}

	cartDTO, err := cq.GetByAvatarID(ctx, avatarID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return map[string]dto.CartItemDTO{}
		}
		return nil
	}

	if cartDTO.Items == nil {
		return map[string]dto.CartItemDTO{}
	}

	return cartDTO.Items
}

// ------------------------------------------------------------
// userId -> document
// ------------------------------------------------------------

// fetchDocByUserID returns the first matched document as map.
//
// kind:
// - "shippingAddress": injects id/addressId
// - "paymentMethod": injects id/paymentMethodId
func (q *OrderQuery) fetchDocByUserID(ctx context.Context, colName string, userID string, kind string) map[string]any {
	if q == nil || q.FS == nil {
		return nil
	}
	if colName == "" || userID == "" {
		return nil
	}

	field := q.UserIDField
	if field == "" {
		field = "userId"
	}

	iter := q.FS.Collection(colName).
		Where(field, "==", userID).
		Limit(1).
		Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err != nil {
		if err == iterator.Done {
			return nil
		}
		log.Printf("[mall_order_query] document query error col=%q userId=%q err=%v", colName, userID, err)
		return nil
	}
	if doc == nil {
		return nil
	}

	out := normalizeMapAny(doc.Data())
	if doc.Ref != nil {
		return attachDocIDByKind(out, doc.Ref.ID, kind)
	}

	return out
}

// ------------------------------------------------------------
// helpers
// ------------------------------------------------------------

func normalizeMapAny(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}

	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}

	return out
}

// attachDocIDByKind injects docID into map if not present.
// It intentionally does not overwrite existing keys.
func attachDocIDByKind(m map[string]any, docID string, kind string) map[string]any {
	if m == nil {
		return nil
	}
	if docID == "" {
		return m
	}

	if _, ok := m["id"]; !ok {
		m["id"] = docID
	}

	switch kind {
	case "shippingAddress":
		if _, ok := m["addressId"]; !ok {
			m["addressId"] = docID
		}
	case "paymentMethod":
		if _, ok := m["paymentMethodId"]; !ok {
			m["paymentMethodId"] = docID
		}
	}

	return m
}
