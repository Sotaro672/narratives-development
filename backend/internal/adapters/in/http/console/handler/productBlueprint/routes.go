// backend\internal\adapters\in\http\console\handler\productBlueprint\routes.go
package productBlueprint

import (
	"encoding/json"
	"net/http"
	"strings"
)

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	path := strings.TrimRight(r.URL.Path, "/")

	switch {
	case r.Method == http.MethodGet && path == "/product-blueprints":
		h.list(w, r)

	case r.Method == http.MethodGet && path == "/product-blueprints/deleted":
		h.listDeleted(w, r)

	case r.Method == http.MethodPost && path == "/product-blueprints":
		h.post(w, r)

	// ★ 追加：POST /product-blueprints/{id}/model-refs
	case r.Method == http.MethodPost &&
		strings.HasPrefix(path, "/product-blueprints/") &&
		strings.HasSuffix(path, "/model-refs"):
		trimmed := strings.TrimPrefix(path, "/product-blueprints/")
		trimmed = strings.TrimSuffix(trimmed, "/model-refs")
		id := strings.Trim(trimmed, "/")
		h.appendModelRefs(w, r, id)

	case r.Method == http.MethodGet &&
		strings.HasPrefix(path, "/product-blueprints/") &&
		strings.HasSuffix(path, "/history"):
		trimmed := strings.TrimPrefix(path, "/product-blueprints/")
		trimmed = strings.TrimSuffix(trimmed, "/history")
		id := strings.Trim(trimmed, "/")
		h.listHistory(w, r, id)

	case r.Method == http.MethodPost &&
		strings.HasPrefix(path, "/product-blueprints/") &&
		strings.HasSuffix(path, "/restore"):
		trimmed := strings.TrimPrefix(path, "/product-blueprints/")
		trimmed = strings.TrimSuffix(trimmed, "/restore")
		id := strings.Trim(trimmed, "/")
		h.restore(w, r, id)

	// 重要：suffix 付きルート（/history /restore /model-refs）より後に置く
	case (r.Method == http.MethodPut || r.Method == http.MethodPatch) &&
		strings.HasPrefix(path, "/product-blueprints/"):
		id := strings.TrimPrefix(path, "/product-blueprints/")
		h.update(w, r, id)

	case r.Method == http.MethodDelete &&
		strings.HasPrefix(path, "/product-blueprints/"):
		id := strings.TrimPrefix(path, "/product-blueprints/")
		h.delete(w, r, id)

	case r.Method == http.MethodGet && strings.HasPrefix(path, "/product-blueprints/"):
		id := strings.TrimPrefix(path, "/product-blueprints/")
		h.get(w, r, id)

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}
