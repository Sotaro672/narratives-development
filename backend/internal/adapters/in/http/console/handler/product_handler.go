// backend\internal\adapters\in\http\console\handler\product_handler.go
package consoleHandler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
)

type ProductHandler struct {
	uc *usecase.ProductUsecase
}

func NewProductHandler(uc *usecase.ProductUsecase) http.Handler {
	return &ProductHandler{uc: uc}
}

func (h *ProductHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	log.Printf("[ProductHandler] method=%s path=%s", r.Method, r.URL.Path)

	switch {
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/inspector/products/"):
		id := strings.TrimPrefix(r.URL.Path, "/inspector/products/")
		h.getInspectorDetail(w, r, id)
	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

func (h *ProductHandler) getInspectorDetail(w http.ResponseWriter, r *http.Request, productID string) {
	ctx := r.Context()
	productID = strings.TrimSpace(productID)
	if productID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid productId"})
		return
	}

	detail, err := h.uc.GetInspectorProductDetail(ctx, productID)
	if err != nil {
		// domain 側で ErrNotFound などを 404 にマッピング
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(detail)
}
