// backend/internal/adapters/in/http/console/handler/inspection/patch_complete_inspection.go
package inspection

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	inspectiondom "narratives/internal/domain/inspection"
)

func (h *InspectorHandler) completeInspection(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.inspectionUC == nil {
		writeError(w, http.StatusInternalServerError, "inspection usecase is not configured")
		return
	}

	var req struct {
		ProductionID string     `json:"productionId"`
		InspectedAt  *time.Time `json:"inspectedAt"` // 任意。nil の場合はサーバ時刻を使う
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	req.ProductionID = strings.TrimSpace(req.ProductionID)
	if req.ProductionID == "" {
		writeError(w, http.StatusBadRequest, "productionId is required")
		return
	}

	// ★ by は「表示名」ではなく memberId（認証UID）を保存する方針
	by := currentMemberUID(r)
	if by == "" {
		writeError(w, http.StatusBadRequest, "inspectedBy (current member uid) could not be resolved")
		return
	}

	// inspectedAt: 指定があればそれを UTC に、無ければ now.UTC()
	var at time.Time
	if req.InspectedAt != nil && !req.InspectedAt.IsZero() {
		at = req.InspectedAt.UTC()
	} else {
		at = time.Now().UTC()
	}

	batch, err := h.inspectionUC.CompleteInspectionForProduction(
		ctx,
		req.ProductionID,
		by, // ★ memberId (uid)
		at,
	)
	if err != nil {
		code := http.StatusInternalServerError
		switch err {
		case inspectiondom.ErrInvalidInspectionProductionID,
			inspectiondom.ErrInvalidInspectionResult:
			code = http.StatusBadRequest
		case inspectiondom.ErrNotFound:
			code = http.StatusNotFound
		}
		writeError(w, code, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, batch)
}
