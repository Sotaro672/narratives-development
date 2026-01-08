// backend/internal/application/query/mall/order_query.go
package mall

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	dto "narratives/internal/application/query/mall/dto"
	appresolver "narratives/internal/application/resolver"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// OrderQuery resolves (mall buyer flow):
// - uid -> avatarId (avatars where userId == uid)
// - uid -> shippingAddress / billingAddress (best-effort; multiple shapes supported)
// - avatarId -> cartItems (via SNSCartQuery; best-effort)
// - userId -> fullName (via NameResolver.ResolveMemberName; best-effort)
type OrderQuery struct {
	FS *firestore.Client

	// optional: cart read-model
	// - if nil, ResolveByUID will create SNSCartQuery(fs) and fetch cart items best-effort
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

// ✅ Backward-compat: keep old exported name used by handlers/DI.
// NOTE: type alias so methods on OrderQuery are available.

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
	uid = strings.TrimSpace(uid)
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

	uid = strings.TrimSpace(uid)
	if uid == "" {
		return OrderContextDTO{}, errors.New("uid is required")
	}

	avatarID, avatarUserID, err := q.resolveAvatarIDByUID(ctx, uid)
	if err != nil {
		return OrderContextDTO{}, err
	}

	// userId は基本 uid と一致させる（avatars の userId も尊重）
	userID := strings.TrimSpace(avatarUserID)
	if userID == "" {
		userID = uid
	}

	ship := q.fetchAddressBestEffort(ctx, q.ShippingAddressCol, userID)
	bill := q.fetchAddressBestEffort(ctx, q.BillingAddressCol, userID)

	// cartItems（best-effort）
	cartItems := q.fetchCartItemsBestEffort(ctx, avatarID)

	// fullName（best-effort）
	fullName := ""
	if q.NameResolver != nil {
		// NameResolver は memberId を想定。userId と一致していればここで解決できる。
		// 一致しない運用の場合は空のまま（決済フローを止めない）。
		fullName = strings.TrimSpace(q.NameResolver.ResolveMemberName(ctx, userID))
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
	col := strings.TrimSpace(q.AvatarsCol)
	if col == "" {
		col = "avatars"
	}
	userField := strings.TrimSpace(q.AvatarUserIDField)
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
		u = strings.TrimSpace(fmt.Sprint(v))
	}
	aid := strings.TrimSpace(doc.Ref.ID)
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
	avatarID = strings.TrimSpace(avatarID)
	if avatarID == "" {
		return nil
	}

	cq := q.CartQ
	if cq == nil {
		// ListRepo / Resolver は nil のまま（必要なら DI 側で CartQ を注入）
		cq = NewCartQuery(q.FS)
	}

	cartDTO, err := cq.GetByAvatarID(ctx, avatarID)
	if err != nil {
		// carts/{avatarId} が無いケースは “空” 扱い（決済フローを止めない）
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
// uid(userId) -> address (best-effort)
// ------------------------------------------------------------

// fetchAddressBestEffort tries common patterns:
//
// 1) GET document by ID = userId (collection/{userId})
// 2) Query where userId == {userId} LIMIT 1
//
// If not found -> nil
func (q *OrderQuery) fetchAddressBestEffort(ctx context.Context, colName string, userID string) map[string]any {
	colName = strings.TrimSpace(colName)
	if colName == "" {
		return nil
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil
	}

	// (1) doc style: collection/{userId}
	docRef := q.FS.Collection(colName).Doc(userID)
	snap, err := docRef.Get(ctx)
	if err == nil && snap != nil && snap.Exists() {
		return normalizeMapAny(snap.Data())
	}
	if err != nil && !isFirestoreNotFound(err) {
		log.Printf("[mall_order_query] address doc get error col=%q userId=%q err=%v", colName, maskUID(userID), err)
	}

	// (2) query style: where userId == ...
	field := strings.TrimSpace(q.AddressUserIDField)
	if field == "" {
		field = "userId"
	}

	iter := q.FS.Collection(colName).
		Where(field, "==", userID).
		Limit(1).
		Documents(ctx)
	defer iter.Stop()

	doc, qerr := iter.Next()
	if qerr != nil {
		if qerr == iterator.Done {
			return nil
		}
		log.Printf("[mall_order_query] address query error col=%q userId=%q err=%v", colName, maskUID(userID), qerr)
		return nil
	}
	if doc == nil {
		return nil
	}
	return normalizeMapAny(doc.Data())
}

// ------------------------------------------------------------
// Firestore helpers
// ------------------------------------------------------------

// isFirestoreNotFound checks Firestore NotFound safely.
func isFirestoreNotFound(err error) bool {
	if err == nil {
		return false
	}
	if status.Code(err) == codes.NotFound {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") || strings.Contains(msg, "not_found")
}

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

// avoid logging raw uid
func maskUID(uid string) string {
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return ""
	}
	if len(uid) <= 6 {
		return "***"
	}
	return uid[:3] + "***" + uid[len(uid)-3:]
}
