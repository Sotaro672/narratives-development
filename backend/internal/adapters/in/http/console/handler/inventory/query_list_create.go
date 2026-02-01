// backend/internal/adapters/in/http/console/handler/inventory/query_list_create.go
package inventory

import (
	"net/http"
	"strings"
)

// ============================================================
// ✅ ListCreate DTO endpoint
// - GET /inventory/list-create/{inventoryId}
// ============================================================

func (h *InventoryHandler) GetListCreateByPathQuery(w http.ResponseWriter, r *http.Request, path string) {
	if h.LQ == nil {
		writeError(w, http.StatusNotImplemented, "list create query is not configured")
		return
	}

	ctx := r.Context()

	rest := strings.TrimSpace(strings.TrimPrefix(path, "/inventory/list-create/"))
	rest = strings.Trim(rest, "/")
	if rest == "" {
		writeError(w, http.StatusBadRequest, "missing params")
		return
	}

	// ✅ inventoryId は docId をそのまま受け取る（pb/tb を path で受けない）
	inventoryID := strings.TrimSpace(rest)
	if inventoryID == "" {
		writeError(w, http.StatusBadRequest, "inventoryId is required")
		return
	}

	dto, err := h.LQ.GetByInventoryID(ctx, inventoryID)
	if err != nil {
		// validation系は 400、それ以外は 500 に寄せる
		if isProbablyBadRequest(err) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, dto)
}

func isProbablyBadRequest(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(msg, "required") ||
		strings.Contains(msg, "missing") ||
		strings.Contains(msg, "invalid")
}
