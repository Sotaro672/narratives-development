// backend/internal/adapters/in/http/console/handler/inspection/get_inspections_by_production_id.go
package inspection

import (
	"net/http"
	"strings"

	inspectiondom "narratives/internal/domain/inspection"
)

type inspectionRecordResponse struct {
	ProductID        string  `json:"productId"`
	ModelID          string  `json:"modelId,omitempty"` // ✅ domain が string のため *string ではなく string
	ModelNumber      string  `json:"modelNumber,omitempty"`
	InspectionResult any     `json:"inspectionResult,omitempty"`
	InspectedBy      string  `json:"inspectedBy,omitempty"`   // ✅ 表示名
	InspectedByID    *string `json:"inspectedById,omitempty"` // ✅ デバッグ用（UIは無視してOK）
	InspectedAt      any     `json:"inspectedAt,omitempty"`
}

type inspectionBatchResponse struct {
	ProductionID string                     `json:"productionId"`
	Status       any                        `json:"status"`
	Quantity     int                        `json:"quantity"`
	TotalPassed  int                        `json:"totalPassed"`
	Inspections  []inspectionRecordResponse `json:"inspections"`
}

func (h *InspectorHandler) getInspectionsByProductionID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.inspectionUC == nil {
		writeError(w, http.StatusInternalServerError, "inspection usecase is not configured")
		return
	}

	productionID := strings.TrimSpace(r.URL.Query().Get("productionId"))
	if productionID == "" {
		writeError(w, http.StatusBadRequest, "productionId is required")
		return
	}

	// ✅ 互換削除後：GetBatchByProductionID を使う
	batch, err := h.inspectionUC.GetBatchByProductionID(ctx, productionID)
	if err != nil {
		code := http.StatusInternalServerError
		switch err {
		case inspectiondom.ErrInvalidInspectionProductionID:
			code = http.StatusBadRequest
		case inspectiondom.ErrNotFound:
			code = http.StatusNotFound
		}
		writeError(w, code, err.Error())
		return
	}

	resp := inspectionBatchResponse{
		ProductionID: batch.ProductionID,
		Status:       batch.Status,
		Quantity:     batch.Quantity,
		TotalPassed:  batch.TotalPassed,
		Inspections:  make([]inspectionRecordResponse, 0, len(batch.Inspections)),
	}

	for _, item := range batch.Inspections {
		// inspectedBy: *string (memberId) -> 表示名
		inspectedByName := ""
		inspectedByID := item.InspectedBy
		if h.nameResolver != nil {
			inspectedByName = h.nameResolver.ResolveInspectedByName(ctx, inspectedByID)
		}

		// modelId -> modelNumber
		modelNumber := ""
		modelID := strings.TrimSpace(item.ModelID) // ✅ item.ModelID は string
		if h.nameResolver != nil && modelID != "" {
			modelNumber = h.nameResolver.ResolveModelNumber(ctx, modelID)
		}

		resp.Inspections = append(resp.Inspections, inspectionRecordResponse{
			ProductID:        item.ProductID,
			ModelID:          modelID,
			ModelNumber:      strings.TrimSpace(modelNumber),
			InspectionResult: item.InspectionResult,
			InspectedBy:      strings.TrimSpace(inspectedByName), // ✅ UI に表示される
			InspectedByID:    inspectedByID,                      // ✅ UI は無視してOK
			InspectedAt:      item.InspectedAt,
		})
	}

	writeJSON(w, http.StatusOK, resp)
}
