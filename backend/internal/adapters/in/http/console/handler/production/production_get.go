// backend/internal/adapters/in/http/console/handler/production/production_get.go
package productionHandler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

func (h *ProductionHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	if h.uc == nil {
		log.Printf("[productions] get: usecase is nil")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "usecase is nil"})
		return
	}

	id = strings.TrimSpace(id)
	if id == "" {
		log.Printf("[productions] get: invalid id (empty)")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	log.Printf("[productions] get: start id=%q", id)

	p, err := h.uc.GetByID(ctx, id)
	if err != nil {
		log.Printf("[productions] get: error id=%q err=%v", id, err)
		writeProductionErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(p)
}
