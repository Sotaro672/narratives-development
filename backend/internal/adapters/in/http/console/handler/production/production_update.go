// backend/internal/adapters/in/http/console/handler/production/production_update.go
package productionHandler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	productiondom "narratives/internal/domain/production"
)

func (h *ProductionHandler) update(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	defer r.Body.Close()

	if h.uc == nil {
		log.Printf("[productions] update: usecase is nil")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "usecase is nil"})
		return
	}

	id = strings.TrimSpace(id)
	if id == "" {
		log.Printf("[productions] update: invalid id (empty)")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	var req productiondom.Production
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[productions] update: invalid json id=%q err=%v", id, err)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	// パスの ID を優先
	req.ID = id

	log.Printf("[productions] update: start id=%q", id)

	p, err := h.uc.Update(ctx, id, req)
	if err != nil {
		log.Printf("[productions] update: error id=%q err=%v", id, err)
		writeProductionErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(p)
}
