// backend/internal/adapters/in/http/mall/handler/helper_handler.go
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
		if k == "" {
			continue
		}
		v, ok := m[k]
		if !ok || v == nil {
			continue
		}
		switch t := v.(type) {
		case string:
			s := t
			if s != "" {
				return s, true
			}
		case []byte:
			s := string(t)
			if s != "" {
				return s, true
			}
		default:
			// ignore
		}
	}
	return "", false
}

// extractLastPathSegment extracts the last segment after a prefix.
// Example:
//
//	path="/mall/preview/abc", prefix="/mall/preview" -> "abc"
//	path="/mall/preview", prefix="/mall/preview" -> ""
func extractLastPathSegment(path string, prefix string) string {
	p := strings.TrimSuffix(path, "/")
	prefix = strings.TrimSuffix(prefix, "/")

	if p == prefix {
		return ""
	}
	if !strings.HasPrefix(p, prefix+"/") {
		return ""
	}
	rest := strings.TrimPrefix(p, prefix+"/")
	if rest == "" {
		return ""
	}
	if i := strings.Index(rest, "/"); i >= 0 {
		rest = rest[:i]
	}
	return rest
}

// isNotFound is a package-level helper used by multiple handlers.
// It delegates to isNotFoundLike for backwards compatibility.
func isNotFound(err error) bool {
	return isNotFoundLike(err)
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
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": msg})
}

func internalError(w http.ResponseWriter, msg string) {
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": msg})
}

func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

// ptrStr: shared
func ptrStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
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

func isNotFoundLike(err error) bool {
	if err == nil {
		return false
	}
	// errors.Is で拾えるケースもあるので一応入れておく
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not_found") ||
		strings.Contains(msg, "not found") ||
		strings.Contains(msg, "404") ||
		strings.Contains(msg, "avatar_not_found_for_uid")
}
