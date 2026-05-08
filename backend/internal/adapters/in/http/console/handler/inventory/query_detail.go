// backend/internal/adapters/in/http/console/handler/inventory/query_detail.go
package inventory

import (
	"errors"
	"net/http"

	invdom "narratives/internal/domain/inventory"
)

// ============================================================
// ✅ Detail endpoint（確定）
// - Query が必須（fallback は削除）
// ============================================================

func (h *InventoryHandler) GetDetailByIDQuery(w http.ResponseWriter, r *http.Request, inventoryID string) {
	if h == nil || h.Q == nil {
		writeError(w, http.StatusNotImplemented, "inventory query is not configured")
		return
	}

	ctx := r.Context()
	if inventoryID == "" {
		writeError(w, http.StatusBadRequest, "missing id")
		return
	}

	dto, err := h.Q.GetDetailByID(ctx, inventoryID)
	if err != nil {
		if errors.Is(err, invdom.ErrNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, dto)
}
