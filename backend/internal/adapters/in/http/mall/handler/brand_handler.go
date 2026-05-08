// backend\internal\adapters\in\http\mall\handler\brand_handler.go
package mallHandler

import (
	"encoding/json"
	"net/http"
	"strings"

	mallquery "narratives/internal/application/query/mall"
	branddom "narratives/internal/domain/brand"
)

// MallBrandHandler serves buyer-facing brand endpoint.
//
// Route:
// - GET /mall/brands/{id}
type MallBrandHandler struct {
	q *mallquery.BrandQuery
}

func NewMallBrandHandler(q *mallquery.BrandQuery) http.Handler {
	return &MallBrandHandler{q: q}
}

func (h *MallBrandHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if h == nil || h.q == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "brand handler is not ready"})
		return
	}

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "method_not_allowed"})
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/")

	// GET /mall/brands/{id}
	if strings.HasPrefix(path, "/mall/brands/") {
		id := strings.TrimPrefix(path, "/mall/brands/")
		h.get(w, r, id)
		return
	}

	w.WriteHeader(http.StatusNotFound)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
}

// ---- GET /mall/brands/{id} ----
func (h *MallBrandHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	brand, err := h.q.GetBrandDetailByID(ctx, id)
	if err != nil {
		writeMallBrandErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(brand)
}

func writeMallBrandErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	switch err {
	case branddom.ErrInvalidID:
		code = http.StatusBadRequest
	case branddom.ErrNotFound:
		code = http.StatusNotFound
	case branddom.ErrConflict:
		code = http.StatusConflict
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
