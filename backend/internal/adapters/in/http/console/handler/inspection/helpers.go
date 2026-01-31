// backend/internal/adapters/in/http/console/handler/inspection/helpers.go
package inspection

import (
	"encoding/json"
	"net/http"
	"strings"

	"narratives/internal/adapters/in/http/middleware"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// 認証UIDを inspectedBy として使う（取れない場合は "" を返す）
func currentMemberUID(r *http.Request) string {
	uid, _, ok := middleware.CurrentUIDAndEmail(r)
	if !ok {
		return ""
	}
	return strings.TrimSpace(uid)
}
