package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	productiondom "narratives/internal/domain/production"
)

// ProductionHandler は /productions 関連のエンドポイントを担当します（単一取得のみ）。
type ProductionHandler struct {
	uc *usecase.ProductionUsecase
}

func NewProductionHandler(uc *usecase.ProductionUsecase) http.Handler {
	return &ProductionHandler{uc: uc}
}

func (h *ProductionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/productions/"):
		id := strings.TrimPrefix(r.URL.Path, "/productions/")
		h.get(w, r, id)
	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// GET /productions/{id}
func (h *ProductionHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	p, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeProductionErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(p)
}

func writeProductionErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	switch err {
	case productiondom.ErrInvalidID:
		code = http.StatusBadRequest
	case productiondom.ErrNotFound:
		code = http.StatusNotFound
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
