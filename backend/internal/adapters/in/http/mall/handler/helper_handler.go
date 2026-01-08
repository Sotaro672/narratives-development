// backend\internal\adapters\in\http\mall\handler\helper_handler.go
package mallHandler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	pbdom "narratives/internal/domain/productBlueprint"
)

// ============================================================
// Shared types (avoid DuplicateDecl in same package)
// ============================================================

// productBlueprintGetter: Mall handlers 用（read-only）
type productBlueprintGetter interface {
	GetByID(ctx context.Context, id string) (pbdom.ProductBlueprint, error)
}

// ============================================================
// Shared helpers (avoid UndeclaredName)
// ============================================================

// getString tries keys in order and returns the first string value found.
// - accepts string / []byte only (avoid fmt.Sprint surprises)
func getString(m map[string]any, keys ...string) (string, bool) {
	if m == nil {
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
		switch t := v.(type) {
		case string:
			s := strings.TrimSpace(t)
			if s != "" {
				return s, true
			}
		case []byte:
			s := strings.TrimSpace(string(t))
			if s != "" {
				return s, true
			}
		default:
			// ignore
		}
	}
	return "", false
}

// ============================================================
// HTTP helpers
// ============================================================

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

// helpers (PII漏洩を避けるため短く)
func headString(b []byte, max int) string {
	if len(b) == 0 {
		return ""
	}
	if len(b) > max {
		b = b[:max]
	}
	s := string(b)
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	return s
}

// trimPtr: shared
func trimPtr(p *string) *string {
	if p == nil {
		return nil
	}
	s := strings.TrimSpace(*p)
	if s == "" {
		return nil
	}
	return &s
}

// ptrStr: shared
func ptrStr(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

// ptrLen: shared (rune length)
func ptrLen(p *string) int {
	if p == nil {
		return 0
	}
	return len([]rune(strings.TrimSpace(*p)))
}

// maskUID: shared (Firebase UID をそのまま出さない)
func maskUID(uid string) string {
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return ""
	}
	if len(uid) <= 6 {
		return "***"
	}
	return "***" + uid[len(uid)-6:]
}

func toRFC3339(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func readJSON(r *http.Request, dst any) error {
	if dst == nil {
		return errors.New("dst is nil")
	}
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20)) // 1MB
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}
