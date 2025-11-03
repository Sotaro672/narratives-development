package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	invdom "narratives/internal/domain/inventory"
)

// InventoryHandler は /inventories 関連のエンドポイントを担当します（単一取得のみ）。
type InventoryHandler struct {
	uc *usecase.InventoryUsecase
}

// NewInventoryHandler はHTTPハンドラを初期化します。
func NewInventoryHandler(uc *usecase.InventoryUsecase) http.Handler {
	return &InventoryHandler{uc: uc}
}

// ServeHTTP はHTTPルーティングの入口です。
func (h *InventoryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/inventories/"):
		id := strings.TrimPrefix(r.URL.Path, "/inventories/")
		h.get(w, r, id)
	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// GET /inventories/{id}
func (h *InventoryHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	item, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeInventoryErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(item)
}

// エラーハンドリング
func writeInventoryErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	switch err {
	case invdom.ErrInvalidID:
		code = http.StatusBadRequest
	case invdom.ErrNotFound:
		code = http.StatusNotFound
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
