// backend/internal/adapters/in/http/console/handler/inspection/get_mint_by_inspection_id.go
package inspection

import (
	"net/http"
	"strings"

	inspectiondom "narratives/internal/domain/inspection"
	mintdom "narratives/internal/domain/mint"
)

func (h *InspectorHandler) getMintByInspectionID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.inspectionUC == nil {
		writeError(w, http.StatusInternalServerError, "inspection usecase is not configured")
		return
	}

	inspectionID := strings.TrimSpace(r.URL.Query().Get("inspectionId"))
	if inspectionID == "" {
		writeError(w, http.StatusBadRequest, "inspectionId is required")
		return
	}

	m, err := h.inspectionUC.GetMintByInspectionID(ctx, inspectionID)
	if err != nil {
		code := http.StatusInternalServerError
		switch err {
		case inspectiondom.ErrInvalidInspectionProductionID:
			code = http.StatusBadRequest
		case mintdom.ErrNotFound:
			code = http.StatusNotFound
		}
		writeError(w, code, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, m)
}
