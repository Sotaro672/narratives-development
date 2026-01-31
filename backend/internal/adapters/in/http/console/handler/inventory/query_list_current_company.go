// backend/internal/adapters/in/http/console/handler/inventory/query_list_current_company.go
package inventory

import (
	"net/http"

	querydto "narratives/internal/application/query/console/dto"
)

func (h *InventoryHandler) ListByCurrentCompanyQuery(w http.ResponseWriter, r *http.Request) {
	if h.Q == nil {
		writeError(w, http.StatusNotImplemented, "inventory query is not configured")
		return
	}

	ctx := r.Context()

	rows, err := h.Q.ListByCurrentCompany(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 参照だけして import を維持（返却型が interface の場合などに備える）
	_ = querydto.InventoryManagementRowDTO{}
	writeJSON(w, http.StatusOK, rows)
}
