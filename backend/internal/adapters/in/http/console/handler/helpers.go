// backend\internal\adapters\in\http\console\handler\helpers.go
package consoleHandler

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

func methodNotAllowed(w http.ResponseWriter) {
	w.WriteHeader(http.StatusMethodNotAllowed)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "method_not_allowed"})
}

// usecase.ErrNotSupported は型が見えないので message 判定
func isNotSupported(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not supported") ||
		strings.Contains(msg, "not_supported") ||
		strings.Contains(msg, "notsupported")
}

func parseIntDefault(s string, def int) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

// splitCSV parses "a,b,c" / "a, b, c" into []string (empty trimmed items are removed).
func splitCSV(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// ------------------------------------------------------------
// log helpers
// ------------------------------------------------------------

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func sampleFirst[T any](xs []T) any {
	if len(xs) == 0 {
		return nil
	}
	return xs[0]
}

func toJSONForLog(v any, max int) string {
	if v == nil {
		return "null"
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "<marshal_error>"
	}
	s := string(b)
	if max > 0 && len(s) > max {
		return s[:max] + "...(truncated)"
	}
	return s
}

func sampleFirstKey[V any](m map[string]V) string {
	if len(m) == 0 {
		return ""
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys[0]
}

func sampleFirstValue[V any](m map[string]V) any {
	if len(m) == 0 {
		return nil
	}
	k := sampleFirstKey(m)
	if k == "" {
		return nil
	}
	return m[k]
}
