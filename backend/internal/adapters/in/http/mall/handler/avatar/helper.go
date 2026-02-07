// backend\internal\adapters\in\http\mall\handler\helper_handler.go
package avatarHandler

import (
	"encoding/json"
	"net/http"
	"strings"
)

// ============================================================
// HTTP helpers
// ============================================================

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

func notFound(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
}
