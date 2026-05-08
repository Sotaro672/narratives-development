// backend/internal/adapters/in/http/console/handler/sales_handler.go
package consoleHandler

import (
	"encoding/json"
	"net/http"

	"narratives/internal/adapters/in/http/middleware"
	query "narratives/internal/application/query/console"
)

type SalesHandler struct {
	SalesQuery *query.SalesQuery
}

func (h *SalesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if h.SalesQuery == nil {
		http.Error(w, "SalesQuery is not initialized", http.StatusInternalServerError)
		return
	}

	companyID, ok := middleware.CompanyID(r)
	if !ok || companyID == "" {
		http.Error(w, "companyId not found in auth context", http.StatusForbidden)
		return
	}

	result, err := h.SalesQuery.ListByCompanyID(r.Context(), companyID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(result); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
