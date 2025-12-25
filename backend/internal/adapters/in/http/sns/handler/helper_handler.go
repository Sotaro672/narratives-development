package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func methodNotAllowed(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
}

func notFound(w http.ResponseWriter) {
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
}

func badRequest(w http.ResponseWriter, msg string) {
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": strings.TrimSpace(msg)})
}

func internalError(w http.ResponseWriter, msg string) {
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": strings.TrimSpace(msg)})
}

func parseIntDefault(s string, def int) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

// getString picks the first non-empty string value from map by keys.
// - value types supported: string / []byte / fmt.Stringer / numbers/bools (stringified)
func getString(m map[string]any, keys ...string) (string, bool) {
	if m == nil || len(keys) == 0 {
		return "", false
	}

	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}

		v, ok := m[k]
		if !ok || v == nil {
			continue
		}

		switch x := v.(type) {
		case string:
			s := strings.TrimSpace(x)
			if s != "" {
				return s, true
			}
		case []byte:
			s := strings.TrimSpace(string(x))
			if s != "" {
				return s, true
			}
		default:
			// fallback: stringify common primitives without importing fmt
			// (keep helper tiny; enough for ids)
			switch y := v.(type) {
			case int:
				s := strconv.Itoa(y)
				if strings.TrimSpace(s) != "" {
					return s, true
				}
			case int64:
				s := strconv.FormatInt(y, 10)
				if strings.TrimSpace(s) != "" {
					return s, true
				}
			case float64:
				s := strconv.FormatFloat(y, 'f', -1, 64)
				if strings.TrimSpace(s) != "" {
					return s, true
				}
			case bool:
				if y {
					return "true", true
				}
				return "false", true
			}
		}
	}

	return "", false
}
