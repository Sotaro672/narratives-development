// backend\internal\adapters\in\http\mall\handler\brand_handler.go
package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	branddom "narratives/internal/domain/brand"
)

// SNSBrandHandler serves buyer-facing brand endpoint.
//
// Route:
// - GET /sns/brands/{id}
type SNSBrandHandler struct {
	uc *usecase.BrandUsecase
}

func NewSNSBrandHandler(uc *usecase.BrandUsecase) http.Handler {
	return &SNSBrandHandler{uc: uc}
}

func (h *SNSBrandHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if h == nil || h.uc == nil {
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

	// GET /sns/brands/{id}
	if strings.HasPrefix(path, "/sns/brands/") {
		id := strings.TrimSpace(strings.TrimPrefix(path, "/sns/brands/"))
		h.get(w, r, id)
		return
	}

	// collection endpoint is not defined
	w.WriteHeader(http.StatusNotFound)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
}

// ---- GET /sns/brands/{id} ----
func (h *SNSBrandHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	brand, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeSNSBrandErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(brand)
}

func writeSNSBrandErr(w http.ResponseWriter, err error) {
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
