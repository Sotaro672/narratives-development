// backend/internal/adapters/in/http/handlers/inspector_handler.go
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"narratives/internal/application/usecase"
	productdom "narratives/internal/domain/product"
)

type InspectorHandler struct {
	productUC    *usecase.PrintUsecase
	inspectionUC *usecase.InspectionUsecase
}

func NewInspectorHandler(
	productUC *usecase.PrintUsecase,
	inspectionUC *usecase.InspectionUsecase,
) http.Handler {
	return &InspectorHandler{
		productUC:    productUC,
		inspectionUC: inspectionUC,
	}
}

func (h *InspectorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {

	// ------------------------------------------------------------
	// GET /inspector/products/{productId}
	//   → 検品アプリ（Flutter）用の詳細情報取得
	// ------------------------------------------------------------
	case r.Method == http.MethodGet &&
		strings.HasPrefix(r.URL.Path, "/inspector/products/"):

		productID := strings.TrimPrefix(r.URL.Path, "/inspector/products/")
		productID = strings.TrimSpace(productID)
		if productID == "" {
			http.Error(w, `{"error":"missing productId"}`, http.StatusBadRequest)
			return
		}

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

	// ------------------------------------------------------------
	// PATCH /products/inspections
	//   → inspections テーブル（1 productId 分）更新 API
	//   ※ products テーブル側も同時に更新される（InspectionUsecase 内で統合済）
	// ------------------------------------------------------------
	case r.Method == http.MethodPatch && r.URL.Path == "/products/inspections":
		h.updateInspection(w, r)
		return

	default:
		http.NotFound(w, r)
	}
}

// ------------------------------------------------------------
// PATCH /products/inspections
//
//	検品結果を更新する（1 productId 単位）
//	→ InspectionUsecase.UpdateInspectionForProduct に移譲
//
// ------------------------------------------------------------
func (h *InspectorHandler) updateInspection(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.inspectionUC == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "inspection usecase is not configured",
		})
		return
	}

	var req struct {
		ProductionID     string                       `json:"productionId"`
		ProductID        string                       `json:"productId"`
		InspectionResult *productdom.InspectionResult `json:"inspectionResult"`
		InspectedBy      *string                      `json:"inspectedBy"`
		InspectedAt      *time.Time                   `json:"inspectedAt"`
		Status           *productdom.InspectionStatus `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid json",
		})
		return
	}

	req.ProductionID = strings.TrimSpace(req.ProductionID)
	req.ProductID = strings.TrimSpace(req.ProductID)

	if req.ProductionID == "" || req.ProductID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "productionId and productId are required",
		})
		return
	}

	batch, err := h.inspectionUC.UpdateInspectionForProduct(
		ctx,
		req.ProductionID,
		req.ProductID,
		req.InspectionResult,
		req.InspectedBy,
		req.InspectedAt,
		req.Status,
	)
	if err != nil {

		code := http.StatusInternalServerError
		switch err {
		case productdom.ErrInvalidInspectionProductionID,
			productdom.ErrInvalidInspectionProductIDs,
			productdom.ErrInvalidInspectionResult,
			productdom.ErrInvalidInspectedBy,
			productdom.ErrInvalidInspectedAt,
			productdom.ErrInvalidInspectionStatus:
			code = http.StatusBadRequest

		case productdom.ErrNotFound:
			code = http.StatusNotFound
		}

		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(batch)
}
