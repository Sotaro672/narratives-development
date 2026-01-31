// backend\internal\adapters\in\http\console\handler\inspection\inspection_handler.go
package inspection

import (
	"net/http"
	"strings"

	"narratives/internal/application/resolver"

	inspectionapp "narratives/internal/application/inspection"
	usecase "narratives/internal/application/usecase"
)

type InspectorHandler struct {
	productUC    *usecase.ProductUsecase
	inspectionUC *inspectionapp.InspectionUsecase
	nameResolver *resolver.NameResolver
}

func NewInspectorHandler(
	productUC *usecase.ProductUsecase,
	inspectionUC *inspectionapp.InspectionUsecase,
	nameResolver *resolver.NameResolver,
) http.Handler {
	return &InspectorHandler{
		productUC:    productUC,
		inspectionUC: inspectionUC,
		nameResolver: nameResolver,
	}
}

func (h *InspectorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {

	// ✅ 分割したメソッドをここから呼ぶ（unused 警告が消える）
	case r.Method == http.MethodGet &&
		strings.HasPrefix(r.URL.Path, "/inspector/products/"):
		h.getInspectorProductDetail(w, r)
		return

	case r.Method == http.MethodGet && r.URL.Path == "/products/inspections/mints":
		h.getMintByInspectionID(w, r)
		return

	case r.Method == http.MethodGet && r.URL.Path == "/products/inspections":
		h.getInspectionsByProductionID(w, r)
		return

	case r.Method == http.MethodPatch && r.URL.Path == "/products/inspections":
		h.updateInspection(w, r)
		return

	case r.Method == http.MethodPatch && r.URL.Path == "/products/inspections/complete":
		h.completeInspection(w, r)
		return

	default:
		http.NotFound(w, r)
	}
}
