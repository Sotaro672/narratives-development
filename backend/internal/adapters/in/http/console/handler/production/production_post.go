// backend/internal/adapters/in/http/console/handler/production/production_post.go
package productionHandler

import (
	"encoding/json"
	"log"
	"net/http"

	productiondom "narratives/internal/domain/production"
)

func (h *ProductionHandler) post(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	defer r.Body.Close()

	if h.uc == nil {
		log.Printf("[productions] post: usecase is nil")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "usecase is nil"})
		return
	}

	var req productiondom.Production
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[productions] post: invalid json err=%v", err)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	log.Printf("[productions] post: start")

	p, err := h.uc.Create(ctx, req)
	if err != nil {
		log.Printf("[productions] post: error err=%v", err)
		writeProductionErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(p)
}
