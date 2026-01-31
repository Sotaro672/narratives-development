// backend/internal/adapters/in/http/console/handler/inventory/query_resolve_ids.go
package inventory

import (
	"net/http"
	"strings"

	querydto "narratives/internal/application/query/console/dto"
)

func (h *InventoryHandler) ResolveInventoryIDsByProductAndTokenQuery(w http.ResponseWriter, r *http.Request) {
	if h.Q == nil {
		writeError(w, http.StatusNotImplemented, "inventory query is not configured")
		return
	}

	ctx := r.Context()

	pbID := strings.TrimSpace(r.URL.Query().Get("productBlueprintId"))
	tbID := strings.TrimSpace(r.URL.Query().Get("tokenBlueprintId"))
	if pbID == "" || tbID == "" {
		writeError(w, http.StatusBadRequest, "productBlueprintId and tokenBlueprintId are required")
		return
	}

	ids, err := h.Q.ListInventoryIDsByProductAndToken(ctx, pbID, tbID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := querydto.InventoryIDsByProductAndTokenDTO{
		ProductBlueprintID: pbID,
		TokenBlueprintID:   tbID,
		InventoryIDs:       ids,
	}

	writeJSON(w, http.StatusOK, resp)
}
