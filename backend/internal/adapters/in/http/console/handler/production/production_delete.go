// backend/internal/adapters/in/http/console/handler/production/production_delete.go
package productionHandler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

func (h *ProductionHandler) delete(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	if h.uc == nil {
		log.Printf("[productions] delete: usecase is nil")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "usecase is nil"})
		return
	}

	id = strings.TrimSpace(id)
	if id == "" {
		log.Printf("[productions] delete: invalid id (empty)")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	log.Printf("[productions] delete: start id=%q", id)

	if err := h.uc.Delete(ctx, id); err != nil {
		log.Printf("[productions] delete: error id=%q err=%v", id, err)
		writeProductionErr(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
