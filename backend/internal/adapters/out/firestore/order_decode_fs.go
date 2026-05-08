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
// Decode helpers
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
		return s
	}
	return fmt.Sprint(v)
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
		s := strings.ToLower(t)
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

func decodePaymentMethodSnapshot(v any) (orderdom.PaymentMethodSnapshot, bool) {
	m := asMapAny(v)
	if m == nil {
		return orderdom.PaymentMethodSnapshot{}, false
	}

	return orderdom.PaymentMethodSnapshot{
		CustomerID:     mapGetStr(m, "customerId"),
		Brand:          mapGetStr(m, "brand"),
		Last4:          mapGetStr(m, "last4"),
		ExpMonth:       mapGetInt(m, "expMonth"),
		ExpYear:        mapGetInt(m, "expYear"),
		CardholderName: mapGetStr(m, "cardholderName"),
		IsDefault:      func() bool { b, _ := mapGetBool(m, "isDefault"); return b }(),
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

		transferred, _ := mapGetBool(m, "transferred")
		isCanceled, _ := mapGetBool(m, "isCanceled")
		isDispatched, _ := mapGetBool(m, "isDispatched")

		var transferredAt *time.Time
		if t, ok := mapGetTimeBestEffort(m, "transferredAt"); ok {
			tt := t.UTC()
			transferredAt = &tt
		}

		return orderdom.OrderItemSnapshot{
			ModelID:       mapGetStr(m, "modelId"),
			InventoryID:   mapGetStr(m, "inventoryId"),
			ListID:        mapGetStr(m, "listId"),
			Qty:           mapGetInt(m, "qty"),
			Price:         mapGetInt(m, "price"),
			IsCanceled:    isCanceled,
			IsDispatched:  isDispatched,
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
