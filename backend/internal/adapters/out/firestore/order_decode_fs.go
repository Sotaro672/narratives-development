// backend/internal/adapters/out/firestore/order_decode_fs.go
package firestore

import (
	"fmt"
	"strings"

	orderdom "narratives/internal/domain/order"
)

// ========================
// Decode helpers (Firestore type wobble absorption)
// ========================

func asMapAny(v any) map[string]any {
	if v == nil {
		return nil
	}
	if m, ok := v.(map[string]any); ok {
		return m
	}
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	return nil
}

func mapGetStr(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

func mapGetInt(m map[string]any, key string) int {
	if m == nil {
		return 0
	}
	v, ok := m[key]
	if !ok || v == nil {
		return 0
	}
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	default:
		// Firestore decode の揺れがあっても落とさず 0 に寄せる（domain が弾く）
		return 0
	}
}

func mapGetBool(m map[string]any, key string) (bool, bool) {
	if m == nil {
		return false, false
	}
	v, ok := m[key]
	if !ok || v == nil {
		return false, false
	}
	switch t := v.(type) {
	case bool:
		return t, true
	case string:
		// "true"/"false" も受ける
		s := strings.TrimSpace(strings.ToLower(t))
		if s == "true" {
			return true, true
		}
		if s == "false" {
			return false, true
		}
		return false, false
	default:
		return false, false
	}
}

func decodeShippingSnapshot(v any) (orderdom.ShippingSnapshot, bool) {
	m := asMapAny(v)
	if m == nil {
		return orderdom.ShippingSnapshot{}, false
	}
	return orderdom.ShippingSnapshot{
		ZipCode: mapGetStr(m, "zipCode"),
		State:   mapGetStr(m, "state"),
		City:    mapGetStr(m, "city"),
		Street:  mapGetStr(m, "street"),
		Street2: mapGetStr(m, "street2"),
		Country: mapGetStr(m, "country"),
	}, true
}

func decodeBillingSnapshot(v any) (orderdom.BillingSnapshot, bool) {
	m := asMapAny(v)
	if m == nil {
		return orderdom.BillingSnapshot{}, false
	}
	return orderdom.BillingSnapshot{
		Last4:          mapGetStr(m, "last4"),
		CardHolderName: mapGetStr(m, "cardHolderName"),
	}, true
}

func decodeItems(v any) ([]orderdom.OrderItemSnapshot, bool) {
	if v == nil {
		return nil, false
	}

	switch raw := v.(type) {
	case []any:
		out := make([]orderdom.OrderItemSnapshot, 0, len(raw))
		for _, x := range raw {
			m := asMapAny(x)
			if m == nil {
				out = append(out, orderdom.OrderItemSnapshot{})
				continue
			}
			out = append(out, orderdom.OrderItemSnapshot{
				ModelID:     strings.TrimSpace(mapGetStr(m, "modelId")),
				InventoryID: strings.TrimSpace(mapGetStr(m, "inventoryId")),
				Qty:         mapGetInt(m, "qty"),
				Price:       mapGetInt(m, "price"),
			})
		}
		return out, true
	case []map[string]any:
		out := make([]orderdom.OrderItemSnapshot, 0, len(raw))
		for _, m := range raw {
			out = append(out, orderdom.OrderItemSnapshot{
				ModelID:     strings.TrimSpace(mapGetStr(m, "modelId")),
				InventoryID: strings.TrimSpace(mapGetStr(m, "inventoryId")),
				Qty:         mapGetInt(m, "qty"),
				Price:       mapGetInt(m, "price"),
			})
		}
		return out, true
	default:
		return nil, false
	}
}
