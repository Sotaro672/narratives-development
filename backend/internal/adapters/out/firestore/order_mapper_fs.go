// backend/internal/adapters/out/firestore/order_mapper_fs.go
package firestore

import (
	"fmt"
	"time"

	"cloud.google.com/go/firestore"

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
			return v
		}
		return ""
	}

	getTime := func(key string) time.Time {
		if v, ok := data[key].(time.Time); ok && !v.IsZero() {
			return v.UTC()
		}
		return time.Time{}
	}

	getBool := func(key string) bool {
		if v, ok := data[key].(bool); ok {
			return v
		}
		return false
	}

	var ship orderdom.ShippingSnapshot
	if v, ok := data["shippingSnapshot"]; ok {
		if s, ok2 := decodeShippingSnapshot(v); ok2 {
			ship = s
		}
	}

	var paymentMethod orderdom.PaymentMethodSnapshot
	if v, ok := data["paymentMethodSnapshot"]; ok {
		if p, ok2 := decodePaymentMethodSnapshot(v); ok2 {
			paymentMethod = p
		}
	}

	items, ok := decodeItems(data["items"])
	if !ok {
		items = nil
	}

	createdAt := getTime("createdAt")
	paid := getBool("paid")
	avatarID := getStr("avatarId")

	if avatarID == "" {
		return orderdom.Order{}, fmt.Errorf("order %s: missing avatarId", doc.Ref.ID)
	}
	if ship.State == "" ||
		ship.City == "" ||
		ship.Street == "" ||
		ship.Country == "" {
		return orderdom.Order{}, fmt.Errorf("order %s: missing shippingSnapshot", doc.Ref.ID)
	}
	if paymentMethod.CustomerID == "" ||
		paymentMethod.Brand == "" ||
		paymentMethod.Last4 == "" ||
		paymentMethod.ExpMonth < 1 ||
		paymentMethod.ExpMonth > 12 ||
		paymentMethod.ExpYear < 2000 ||
		paymentMethod.ExpYear > 9999 ||
		paymentMethod.CardholderName == "" {
		return orderdom.Order{}, fmt.Errorf("order %s: missing paymentMethodSnapshot", doc.Ref.ID)
	}
	if len(items) == 0 {
		return orderdom.Order{}, fmt.Errorf("order %s: missing items", doc.Ref.ID)
	}

	return orderdom.Order{
		ID:       doc.Ref.ID,
		UserID:   getStr("userId"),
		AvatarID: avatarID,
		CartID:   getStr("cartId"),

		ShippingSnapshot:      ship,
		PaymentMethodSnapshot: paymentMethod,

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
		"zipCode": o.ShippingSnapshot.ZipCode,
		"state":   o.ShippingSnapshot.State,
		"city":    o.ShippingSnapshot.City,
		"street":  o.ShippingSnapshot.Street,
		"street2": o.ShippingSnapshot.Street2,
		"country": o.ShippingSnapshot.Country,
	}
	paymentMethod := map[string]any{
		"customerId":     o.PaymentMethodSnapshot.CustomerID,
		"brand":          o.PaymentMethodSnapshot.Brand,
		"last4":          o.PaymentMethodSnapshot.Last4,
		"expMonth":       o.PaymentMethodSnapshot.ExpMonth,
		"expYear":        o.PaymentMethodSnapshot.ExpYear,
		"cardholderName": o.PaymentMethodSnapshot.CardholderName,
		"isDefault":      o.PaymentMethodSnapshot.IsDefault,
	}

	items := make([]map[string]any, 0, len(o.Items))
	for _, it := range o.Items {
		im := map[string]any{
			"modelId":      it.ModelID,
			"inventoryId":  it.InventoryID,
			"listId":       it.ListID,
			"qty":          it.Qty,
			"price":        it.Price,
			"isCanceled":   it.IsCanceled,
			"isDispatched": it.IsDispatched,
			"transferred":  it.Transferred,
		}

		if it.Transferred && it.TransferredAt != nil && !it.TransferredAt.IsZero() {
			im["transferredAt"] = it.TransferredAt.UTC()
		}

		items = append(items, im)
	}

	m := map[string]any{
		"userId":   o.UserID,
		"avatarId": o.AvatarID,
		"cartId":   o.CartID,

		"shippingSnapshot":      ship,
		"paymentMethodSnapshot": paymentMethod,

		"paid": o.Paid,

		"items": items,
	}

	if !o.CreatedAt.IsZero() {
		m["createdAt"] = o.CreatedAt.UTC()
	}

	return m
}
