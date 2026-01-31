// backend/internal/adapters/in/http/console/handler/inspection/get_inspector_product_detail.go
package inspection

import (
	"errors"
	"net/http"
	"strings"

	productdom "narratives/internal/domain/product"
)

func (h *InspectorHandler) getInspectorProductDetail(w http.ResponseWriter, r *http.Request) {
	productID := strings.TrimPrefix(r.URL.Path, "/inspector/products/")
	productID = strings.TrimSpace(productID)
	if productID == "" {
		writeError(w, http.StatusBadRequest, "missing productId")
		return
	}

	p, err := h.productUC.GetInspectorProductDetail(r.Context(), productID)
	if errors.Is(err, productdom.ErrNotFound) {
		writeError(w, http.StatusNotFound, "product not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, p)
}
