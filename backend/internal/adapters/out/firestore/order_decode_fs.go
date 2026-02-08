// backend/internal/adapters/out/firestore/order_decode_fs.go
package firestore

import (
	"fmt"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

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

// mapGetTimeBestEffort reads Firestore timestamp wobble.
// - time.Time
// - *timestamppb.Timestamp
// - firestore.Timestamp-like map is not handled here (keep simple)
func mapGetTimeBestEffort(m map[string]any, key string) (time.Time, bool) {
	if m == nil {
		return time.Time{}, false
	}
	v, ok := m[key]
	if !ok || v == nil {
		return time.Time{}, false
	}

	switch t := v.(type) {
	case time.Time:
		if t.IsZero() {
			return time.Time{}, false
		}
		return t.UTC(), true
	case *timestamppb.Timestamp:
		if t == nil {
			return time.Time{}, false
		}
		tt := t.AsTime()
		if tt.IsZero() {
			return time.Time{}, false
		}
		return tt.UTC(), true
	default:
		return time.Time{}, false
	}
}

// getListID tries both listId and listID (wobble absorb).
func getListID(m map[string]any) string {
	if m == nil {
		return ""
	}
	// primary
	if s := strings.TrimSpace(mapGetStr(m, "listId")); s != "" {
		return s
	}
	// wobble
	if s := strings.TrimSpace(mapGetStr(m, "listID")); s != "" {
		return s
	}
	return ""
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

	build := func(m map[string]any) orderdom.OrderItemSnapshot {
		if m == nil {
			return orderdom.OrderItemSnapshot{}
		}

		// ✅ listId を読む（ここが今まで欠けていた）
		lid := getListID(m)

		// ✅ item-level transfer flags も復元（paid更新でSaveしても壊れないように）
		transferred, _ := mapGetBool(m, "transferred")
		var transferredAt *time.Time
		if t, ok := mapGetTimeBestEffort(m, "transferredAt"); ok {
			tt := t.UTC()
			transferredAt = &tt
		}

		return orderdom.OrderItemSnapshot{
			ModelID:     strings.TrimSpace(mapGetStr(m, "modelId")),
			InventoryID: strings.TrimSpace(mapGetStr(m, "inventoryId")),
			ListID:      strings.TrimSpace(lid),

			Qty:   mapGetInt(m, "qty"),
			Price: mapGetInt(m, "price"),

			Transferred:   transferred,
			TransferredAt: transferredAt,
		}
	}

	switch raw := v.(type) {
	case []any:
		out := make([]orderdom.OrderItemSnapshot, 0, len(raw))
		for _, x := range raw {
			out = append(out, build(asMapAny(x)))
		}
		return out, true

	case []map[string]any:
		out := make([]orderdom.OrderItemSnapshot, 0, len(raw))
		for _, m := range raw {
			out = append(out, build(m))
		}
		return out, true

	default:
		return nil, false
	}
}
