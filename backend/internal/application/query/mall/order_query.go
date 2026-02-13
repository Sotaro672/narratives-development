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

// OrderQuery resolves (mall buyer flow):
// - uid -> avatarId (avatars where userId == uid)
// - userId -> shippingAddress / billingAddress (query style only; docID is NOT userId)
// - avatarId -> cartItems (via SNSCartQuery; best-effort)
// - userId -> fullName (via NameResolver.ResolveMemberName; best-effort)
type OrderQuery struct {
	FS *firestore.Client

	// optional: cart read-model
	// - if nil, ResolveByUID will create CartQuery(fs) and fetch cart items best-effort
	CartQ *CartQuery

	// optional: name resolver (memberId -> "Last First")
	// - if nil, FullName will be empty
	NameResolver *appresolver.NameResolver

	// collection names (override if your firestore schema differs)
	AvatarsCol         string
	ShippingAddressCol string
	BillingAddressCol  string

	// field name used in avatars collection
	AvatarUserIDField string

	// field name used in address collections
	AddressUserIDField string
}

func NewOrderQuery(fs *firestore.Client) *OrderQuery {
	return &OrderQuery{
		FS:                 fs,
		CartQ:              nil,
		NameResolver:       nil,
		AvatarsCol:         "avatars",
		ShippingAddressCol: "shippingAddresses",
		BillingAddressCol:  "billingAddresses",
		AvatarUserIDField:  "userId",
		AddressUserIDField: "userId",
	}
}

func NewOrderQueryWithCartQuery(fs *firestore.Client, cartQ *CartQuery) *OrderQuery {
	q := NewOrderQuery(fs)
	q.CartQ = cartQ
	return q
}

// ✅ Backward-compat constructors
func NewMallOrderQuery(fs *firestore.Client) *OrderQuery {
	return NewOrderQuery(fs)
}
func NewMallOrderQueryWithCartQuery(fs *firestore.Client, cartQ *CartQuery) *OrderQuery {
	return NewOrderQueryWithCartQuery(fs, cartQ)
}

// OrderContextDTO is a minimal buyer-facing payload for payment/order flow.
type OrderContextDTO struct {
	UID             string                     `json:"uid"`
	AvatarID        string                     `json:"avatarId"`
	UserID          string                     `json:"userId"`
	FullName        string                     `json:"fullName,omitempty"`
	ShippingAddress map[string]any             `json:"shippingAddress,omitempty"`
	BillingAddress  map[string]any             `json:"billingAddress,omitempty"`
	CartItems       map[string]dto.CartItemDTO `json:"cartItems,omitempty"`
	Debug           map[string]string          `json:"debug,omitempty"`
}

// ResolveAvatarIDByUID resolves uid -> avatarId only.
// - Intended for middleware use.
// - If not found, returns ErrNotFound.
func (q *OrderQuery) ResolveAvatarIDByUID(ctx context.Context, uid string) (string, error) {
	if q == nil || q.FS == nil {
		return "", errors.New("mall order query: firestore client is nil")
	}
	if uid == "" {
		return "", errors.New("uid is required")
	}

	avatarID, _, err := q.resolveAvatarIDByUID(ctx, uid)
	if err != nil {
		return "", err
	}
	return avatarID, nil
}

// ResolveByUID resolves uid -> avatarId and addresses (+ cart items).
// - If avatar is not found, returns ErrNotFound.
func (q *OrderQuery) ResolveByUID(ctx context.Context, uid string) (OrderContextDTO, error) {
	if q == nil || q.FS == nil {
		return OrderContextDTO{}, errors.New("mall order query: firestore client is nil")
	}
	if uid == "" {
		return OrderContextDTO{}, errors.New("uid is required")
	}

	avatarID, avatarUserID, err := q.resolveAvatarIDByUID(ctx, uid)
	if err != nil {
		return OrderContextDTO{}, err
	}

	// userId は基本 uid と一致させる（avatars の userId も尊重）
	userID := avatarUserID
	if userID == "" {
		userID = uid
	}

	// ✅ Firestore 実データ前提:
	// - shippingAddresses / billingAddresses の docID は userId ではない
	// - userId フィールドで検索する (query style only)
	ship := q.fetchAddressByUserID(ctx, q.ShippingAddressCol, userID)
	bill := q.fetchAddressByUserID(ctx, q.BillingAddressCol, userID)

	// cartItems（best-effort）
	cartItems := q.fetchCartItemsBestEffort(ctx, avatarID)

	// fullName（best-effort）
	fullName := ""
	if q.NameResolver != nil {
		fullName = q.NameResolver.ResolveMemberName(ctx, userID)
	}

	out := OrderContextDTO{
		UID:             uid,
		AvatarID:        avatarID,
		UserID:          userID,
		FullName:        fullName,
		ShippingAddress: ship,
		BillingAddress:  bill,
		CartItems:       cartItems,
	}
	return out, nil
}

// ------------------------------------------------------------
// uid -> avatarId
// ------------------------------------------------------------

func (q *OrderQuery) resolveAvatarIDByUID(ctx context.Context, uid string) (avatarID string, userID string, err error) {
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
	u := ""
	if v, ok := m[userField]; ok {
		u = fmt.Sprint(v)
	}
	aid := doc.Ref.ID
	if aid == "" {
		return "", "", ErrNotFound
	}

	log.Printf("[mall_order_query] resolveAvatarIDByUID ok uid=%q avatarId=%q userId=%q", maskUID(uid), aid, maskUID(u))
	return aid, u, nil
}

// ------------------------------------------------------------
// avatarId -> cartItems (best-effort)
// ------------------------------------------------------------

func (q *OrderQuery) fetchCartItemsBestEffort(ctx context.Context, avatarID string) map[string]dto.CartItemDTO {
	if q == nil || q.FS == nil {
		return nil
	}
	if avatarID == "" {
		return nil
	}

	cq := q.CartQ
	if cq == nil {
		cq = NewCartQuery(q.FS)
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
// userId -> address (query style only)
// ------------------------------------------------------------

// ✅ Patch: return Firestore docID as well (without changing existing schema):
//   - Put doc ID into returned map as "id" and "addressId" if they don't already exist.
func (q *OrderQuery) fetchAddressByUserID(ctx context.Context, colName string, userID string) map[string]any {
	if q == nil || q.FS == nil {
		return nil
	}
	if colName == "" || userID == "" {
		return nil
	}

	field := q.AddressUserIDField
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
		log.Printf("[mall_order_query] address query error col=%q userId=%q err=%v", colName, maskUID(userID), err)
		return nil
	}
	if doc == nil {
		return nil
	}

	out := normalizeMapAny(doc.Data())
	if doc.Ref != nil {
		return attachDocID(out, doc.Ref.ID)
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

// attachDocID injects docID into map if not present.
// - We intentionally do NOT overwrite if the document already has those keys.
// - We set both "id" and "addressId" to maximize compatibility with existing frontends.
func attachDocID(m map[string]any, docID string) map[string]any {
	if m == nil {
		return nil
	}
	if docID == "" {
		return m
	}
	if _, ok := m["id"]; !ok {
		m["id"] = docID
	}
	if _, ok := m["addressId"]; !ok {
		m["addressId"] = docID
	}
	return m
}

// avoid logging raw uid
func maskUID(uid string) string {
	if uid == "" {
		return ""
	}
	if len(uid) <= 6 {
		return "***"
	}
	return uid[:3] + "***" + uid[len(uid)-3:]
}
