// backend/internal/adapters/in/http/handlers/inspector_handler.go
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"narratives/internal/application/usecase"
	productdom "narratives/internal/domain/product"
)

type InspectorHandler struct {
	productUC *usecase.ProductUsecase
}

func NewInspectorHandler(productUC *usecase.ProductUsecase) http.Handler {
	return &InspectorHandler{productUC: productUC}
}

func (h *InspectorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	// GET /inspector/products/{productId}
	case r.Method == http.MethodGet &&
		strings.HasPrefix(r.URL.Path, "/inspector/products/"):

		productID := strings.TrimPrefix(r.URL.Path, "/inspector/products/")
		productID = strings.TrimSpace(productID)
		if productID == "" {
			http.Error(w, `{"error":"missing productId"}`, http.StatusBadRequest)
			return
		}

		// ProductUsecase 側に「検品用のビュー」を返すメソッドを追加してもOK
		p, err := h.productUC.GetByID(r.Context(), productID)
		if errors.Is(err, productdom.ErrNotFound) {
			http.Error(w, `{"error":"product not found"}`, http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
			return
		}

		if err := json.NewEncoder(w).Encode(p); err != nil {
			http.Error(w, `{"error":"encode error"}`, http.StatusInternalServerError)
			return
		}
		return

	default:
		http.NotFound(w, r)
	}
}
