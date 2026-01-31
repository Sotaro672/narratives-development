// backend/internal/adapters/in/http/console/handler/inventory/query_list_create.go
package inventory

import (
	"net/http"
	"strings"
)

// ============================================================
// ✅ NEW: ListCreate DTO endpoint
// - GET /inventory/list-create/{pbId}/{tbId}
// - GET /inventory/list-create/{inventoryId}  (inventoryId="{pbId}__{tbId}")
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

	seg := strings.Split(rest, "/")

	var pbID, tbID string

	switch len(seg) {
	case 1:
		// inventoryId = "{pbId}__{tbId}"
		invID := strings.TrimSpace(seg[0])
		parts := strings.Split(invID, "__")
		if len(parts) < 2 {
			writeError(w, http.StatusBadRequest, "invalid inventoryId format (expected {pbId}__{tbId})")
			return
		}
		pbID = strings.TrimSpace(parts[0])
		tbID = strings.TrimSpace(parts[1])

	case 2:
		pbID = strings.TrimSpace(seg[0])
		tbID = strings.TrimSpace(seg[1])

	default:
		writeError(w, http.StatusBadRequest, "invalid path params")
		return
	}

	if pbID == "" || tbID == "" {
		writeError(w, http.StatusBadRequest, "productBlueprintId and tokenBlueprintId are required")
		return
	}

	dto, err := h.LQ.GetByIDs(ctx, pbID, tbID)
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
