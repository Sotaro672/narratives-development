// backend/internal/adapters/out/firestore/order_mapper_fs.go
package firestore

import (
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/protobuf/types/known/timestamppb"

	orderdom "narratives/internal/domain/order"
)

// docToOrder converts a Firestore document snapshot to orderdom.Order (NEW schema only).
// NEW schema:
// - paid is on order root
// - transferred/transferredAt are on each item (items[].transferred / items[].transferredAt)
func docToOrder(doc *firestore.DocumentSnapshot) (orderdom.Order, error) {
	data := doc.Data()
	if data == nil {
		return orderdom.Order{}, fmt.Errorf("empty order document: %s", doc.Ref.ID)
	}

	getStr := func(key string) string {
		if v, ok := data[key].(string); ok {
			return strings.TrimSpace(v)
		}
		if v, ok := data[key]; ok && v != nil {
			return strings.TrimSpace(fmt.Sprint(v))
		}
		return ""
	}

	getTime := func(key string) time.Time {
		if v, ok := data[key].(time.Time); ok && !v.IsZero() {
			return v.UTC()
		}
		// 念のため protobuf Timestamp も受ける（環境差の吸収）
		if ts, ok := data[key].(*timestamppb.Timestamp); ok && ts != nil {
			t := ts.AsTime()
			if !t.IsZero() {
				return t.UTC()
			}
		}
		return time.Time{}
	}

	// snapshots
	var ship orderdom.ShippingSnapshot
	if v, ok := data["shippingSnapshot"]; ok {
		if s, ok2 := decodeShippingSnapshot(v); ok2 {
			ship = s
		}
	}
	var bill orderdom.BillingSnapshot
	if v, ok := data["billingSnapshot"]; ok {
		if b, ok2 := decodeBillingSnapshot(v); ok2 {
			bill = b
		}
	}

	// items (must exist)
	items, ok := decodeItems(data["items"])
	if !ok {
		items = nil
	}

	createdAt := getTime("createdAt")
	if createdAt.IsZero() && !doc.CreateTime.IsZero() {
		createdAt = doc.CreateTime.UTC()
	}

	// ✅ NEW field (order-level)
	paid, _ := mapGetBool(data, "paid")

	// Strict minimums (entity validate 前に、アダプタとして最低限守る)
	avatarID := getStr("avatarId")
	if avatarID == "" {
		// 旧データ救済（念のため）
		avatarID = getStr("avatarID")
	}

	if strings.TrimSpace(avatarID) == "" {
		return orderdom.Order{}, fmt.Errorf("order %s: missing avatarId", doc.Ref.ID)
	}
	if strings.TrimSpace(ship.State) == "" ||
		strings.TrimSpace(ship.City) == "" ||
		strings.TrimSpace(ship.Street) == "" ||
		strings.TrimSpace(ship.Country) == "" {
		return orderdom.Order{}, fmt.Errorf("order %s: missing shippingSnapshot", doc.Ref.ID)
	}
	if strings.TrimSpace(bill.Last4) == "" {
		return orderdom.Order{}, fmt.Errorf("order %s: missing billingSnapshot.last4", doc.Ref.ID)
	}
	if len(items) == 0 {
		return orderdom.Order{}, fmt.Errorf("order %s: missing items", doc.Ref.ID)
	}

	return orderdom.Order{
		ID:       doc.Ref.ID,
		UserID:   getStr("userId"),
		AvatarID: avatarID,
		CartID:   getStr("cartId"),

		ShippingSnapshot: ship,
		BillingSnapshot:  bill,

		// ✅ order-level
		Paid: paid,

		Items:     items,
		CreatedAt: createdAt,
	}, nil
}

// orderToDoc converts orderdom.Order into a Firestore-storable map (NEW schema only).
// NEW schema:
// - paid is on order root
// - transferred/transferredAt are on each item (items[].transferred / items[].transferredAt)
func orderToDoc(o orderdom.Order) map[string]any {
	ship := map[string]any{
		"zipCode": strings.TrimSpace(o.ShippingSnapshot.ZipCode),
		"state":   strings.TrimSpace(o.ShippingSnapshot.State),
		"city":    strings.TrimSpace(o.ShippingSnapshot.City),
		"street":  strings.TrimSpace(o.ShippingSnapshot.Street),
		"street2": strings.TrimSpace(o.ShippingSnapshot.Street2),
		"country": strings.TrimSpace(o.ShippingSnapshot.Country),
	}
	bill := map[string]any{
		"last4":          strings.TrimSpace(o.BillingSnapshot.Last4),
		"cardHolderName": strings.TrimSpace(o.BillingSnapshot.CardHolderName),
	}

	items := make([]map[string]any, 0, len(o.Items))
	for _, it := range o.Items {
		im := map[string]any{
			"modelId":     strings.TrimSpace(it.ModelID),
			"inventoryId": strings.TrimSpace(it.InventoryID),
			"qty":         it.Qty,
			"price":       it.Price,

			// ✅ item-level transfer flags
			"transferred": it.Transferred,
		}
		// transferred=true のときだけ timestamp を持たせる
		if it.Transferred && it.TransferredAt != nil && !it.TransferredAt.IsZero() {
			im["transferredAt"] = it.TransferredAt.UTC()
		}
		items = append(items, im)
	}

	m := map[string]any{
		"userId":   strings.TrimSpace(o.UserID),
		"avatarId": strings.TrimSpace(o.AvatarID),
		"cartId":   strings.TrimSpace(o.CartID),

		"shippingSnapshot": ship,
		"billingSnapshot":  bill,

		// ✅ order-level
		"paid": o.Paid,

		"items": items,
	}

	if !o.CreatedAt.IsZero() {
		m["createdAt"] = o.CreatedAt.UTC()
	}

	return m
}
