// backend/internal/adapters/in/http/console/handler/inspection/patch_update_inspection.go
package inspection

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	inspectiondom "narratives/internal/domain/inspection"
)

func (h *InspectorHandler) updateInspection(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.inspectionUC == nil {
		writeError(w, http.StatusInternalServerError, "inspection usecase is not configured")
		return
	}

	var req struct {
		ProductionID     string                          `json:"productionId"`
		ProductID        string                          `json:"productId"`
		InspectionResult *inspectiondom.InspectionResult `json:"inspectionResult"`
		InspectedBy      *string                         `json:"inspectedBy"` // ← 互換のため定義だけ残す（実際には無視）
		InspectedAt      *time.Time                      `json:"inspectedAt"`
		Status           *inspectiondom.InspectionStatus `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	req.ProductionID = strings.TrimSpace(req.ProductionID)
	req.ProductID = strings.TrimSpace(req.ProductID)

	if req.ProductionID == "" || req.ProductID == "" {
		writeError(w, http.StatusBadRequest, "productionId and productId are required")
		return
	}

	// ★ inspectedBy は「表示名」ではなく memberId（認証UID）を保存する方針
	uid := currentMemberUID(r)
	if uid == "" {
		writeError(w, http.StatusBadRequest, "inspectedBy (current member uid) could not be resolved")
		return
	}
	inspectedByMemberID := &uid

	batch, err := h.inspectionUC.UpdateInspectionForProduct(
		ctx,
		req.ProductionID,
		req.ProductID,
		req.InspectionResult,
		inspectedByMemberID, // ★ memberId (uid) を渡す
		req.InspectedAt,
		req.Status,
	)
	if err != nil {
		code := http.StatusInternalServerError
		switch err {
		case inspectiondom.ErrInvalidInspectionProductionID,
			inspectiondom.ErrInvalidInspectionProductIDs,
			inspectiondom.ErrInvalidInspectionResult,
			inspectiondom.ErrInvalidInspectedBy,
			inspectiondom.ErrInvalidInspectedAt,
			inspectiondom.ErrInvalidInspectionStatus:
			code = http.StatusBadRequest
		case inspectiondom.ErrNotFound:
			code = http.StatusNotFound
		}
		writeError(w, code, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, batch)
}
