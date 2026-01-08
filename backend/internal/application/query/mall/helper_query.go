// backend\internal\application\query\mall\helper_query.go
package mall

import (
	"fmt"
	"strings"
)

// ============================================================
// shared helpers (query)
// ============================================================

func parseInventoryID(inventoryID string) (productBlueprintID string, tokenBlueprintID string, ok bool) {
	s := strings.TrimSpace(inventoryID)
	if s == "" {
		return "", "", false
	}

	parts := strings.Split(s, "__")
	if len(parts) != 2 {
		return "", "", false
	}

	pb := strings.TrimSpace(parts[0])
	tb := strings.TrimSpace(parts[1])
	if pb == "" || tb == "" {
		return "", "", false
	}
	return pb, tb, true
}

func pickString(m map[string]any, keys ...string) string {
	if m == nil {
		return ""
	}
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		if v, ok := m[k]; ok {
			s := strings.TrimSpace(fmt.Sprint(v))
			if s != "" {
				return s
			}
		}
	}
	return ""
}

func pickAny(m map[string]any, keys ...string) any {
	if m == nil {
		return nil
	}
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		if v, ok := m[k]; ok {
			return v
		}
	}
	return nil
}

func asIntAny(v any) (int, bool) {
	if v == nil {
		return 0, false
	}
	switch x := v.(type) {
	case int:
		return x, true
	case int8:
		return int(x), true
	case int16:
		return int(x), true
	case int32:
		return int(x), true
	case int64:
		return int(x), true
	case uint:
		return int(x), true
	case uint8:
		return int(x), true
	case uint16:
		return int(x), true
	case uint32:
		return int(x), true
	case uint64:
		return int(x), true
	case float32:
		return int(x), true
	case float64:
		return int(x), true
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return 0, false
		}
		var n int
		_, err := fmt.Sscanf(s, "%d", &n)
		return n, err == nil
	default:
		return 0, false
	}
}
