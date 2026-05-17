// backend\internal\application\query\mall\helper_query.go
package mall

import (
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ============================================================
// shared helpers (query)
// ============================================================

func parseInventoryID(inventoryID string) (productBlueprintID string, tokenBlueprintID string, ok bool) {
	s := inventoryID
	if s == "" {
		return "", "", false
	}

	parts := strings.Split(s, "__")
	if len(parts) != 2 {
		return "", "", false
	}

	pb := parts[0]
	tb := parts[1]
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
		if k == "" {
			continue
		}
		if v, ok := m[k]; ok {
			s := fmt.Sprint(v)
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
		if k == "" {
			continue
		}
		if v, ok := m[k]; ok {
			return v
		}
	}
	return nil
}

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

func cloneMeasurements(in map[string]int) map[string]int {
	if len(in) == 0 {
		return nil
	}

	out := make(map[string]int, len(in))
	for key, value := range in {
		out[key] = value
	}

	return out
}
