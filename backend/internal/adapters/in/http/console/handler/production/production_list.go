// backend/internal/adapters/in/http/console/handler/production/production_list.go
package productionHandler

import (
	"encoding/json"
	"log"
	"net/http"

	"narratives/internal/adapters/in/http/middleware"
)

func (h *ProductionHandler) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.query == nil {
		log.Printf("[productions] list: query service is nil")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "query service is nil"})
		return
	}

	// ✅ debug: log companyId again at handler entry (to catch ctx mismatch early)
	if cid, ok := middleware.CompanyID(r); ok {
		log.Printf("[productions] list: start companyId=%q", cid)
	} else {
		log.Printf("[productions] list: start companyId=<missing>")
	}

	// ★ QueryService（company境界付き）を使用
	rows, err := h.query.ListProductionsWithAssigneeName(ctx)
	if err != nil {
		// ✅ debug: classify error
		log.Printf("[productions] list: query error=%v", err)
		writeProductionErr(w, err)
		return
	}

	log.Printf("[productions] list: success rows=%d", len(rows))
	_ = json.NewEncoder(w).Encode(rows)
}
