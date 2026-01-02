// backend/internal/application/query/sns/order_query.go
package sns

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SNSOrderQuery resolves:
// - uid -> avatarId (avatars where userId == uid)
// - uid -> shippingAddress / billingAddress (best-effort; multiple shapes supported)
type SNSOrderQuery struct {
	FS *firestore.Client

	// collection names (override if your firestore schema differs)
	AvatarsCol         string
	ShippingAddressCol string
	BillingAddressCol  string

	// field name used in avatars collection
	AvatarUserIDField string

	// field name used in address collections
	AddressUserIDField string
}

func NewSNSOrderQuery(fs *firestore.Client) *SNSOrderQuery {
	return &SNSOrderQuery{
		FS:                 fs,
		AvatarsCol:         "avatars",
		ShippingAddressCol: "shippingAddresses",
		BillingAddressCol:  "billingAddresses",
		AvatarUserIDField:  "userId",
		AddressUserIDField: "userId",
	}
}

// OrderContextDTO is a minimal buyer-facing payload for payment/order flow.
type OrderContextDTO struct {
	UID             string            `json:"uid"`
	AvatarID        string            `json:"avatarId"`
	UserID          string            `json:"userId"`
	ShippingAddress map[string]any    `json:"shippingAddress,omitempty"`
	BillingAddress  map[string]any    `json:"billingAddress,omitempty"`
	Debug           map[string]string `json:"debug,omitempty"` // optional
}

var ErrNotFound = errors.New("not_found")

// ResolveAvatarIDByUID resolves uid -> avatarId only.
// - Intended for middleware use.
// - If not found, returns ErrNotFound.
func (q *SNSOrderQuery) ResolveAvatarIDByUID(ctx context.Context, uid string) (string, error) {
	if q == nil || q.FS == nil {
		return "", errors.New("sns order query: firestore client is nil")
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

// ResolveByUID resolves uid -> avatarId and addresses.
// - If avatar is not found, returns ErrNotFound.
func (q *SNSOrderQuery) ResolveByUID(ctx context.Context, uid string) (OrderContextDTO, error) {
	if q == nil || q.FS == nil {
		return OrderContextDTO{}, errors.New("sns order query: firestore client is nil")
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

	out := OrderContextDTO{
		UID:             uid,
		AvatarID:        avatarID,
		UserID:          userID,
		ShippingAddress: ship,
		BillingAddress:  bill,
	}
	return out, nil
}

// ------------------------------------------------------------
// uid -> avatarId
// ------------------------------------------------------------

func (q *SNSOrderQuery) resolveAvatarIDByUID(ctx context.Context, uid string) (avatarID string, userID string, err error) {
	col := strings.TrimSpace(q.AvatarsCol)
	if col == "" {
		col = "avatars"
	}
	userField := strings.TrimSpace(q.AvatarUserIDField)
	if userField == "" {
		userField = "userId"
	}

	// avatars where userId == uid LIMIT 1
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

	log.Printf("[sns_order_query] resolveAvatarIDByUID ok uid=%q avatarId=%q userId=%q", maskUID(uid), aid, maskUID(u))
	return aid, u, nil
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
func (q *SNSOrderQuery) fetchAddressBestEffort(ctx context.Context, colName string, userID string) map[string]any {
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
		// Firestore error other than NotFound: keep best-effort (log and fallback to query)
		log.Printf("[sns_order_query] address doc get error col=%q userId=%q err=%v", colName, maskUID(userID), err)
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
		// other errors: best-effort nil
		log.Printf("[sns_order_query] address query error col=%q userId=%q err=%v", colName, maskUID(userID), qerr)
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
// ✅ replaces nonexistent firestore.IsNotFound
func isFirestoreNotFound(err error) bool {
	if err == nil {
		return false
	}
	// Firestore returns gRPC status codes
	if status.Code(err) == codes.NotFound {
		return true
	}
	// some layers wrap; be tolerant
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") || strings.Contains(msg, "not_found")
}

func normalizeMapAny(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	// shallow copy to avoid accidental mutation across callers
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
