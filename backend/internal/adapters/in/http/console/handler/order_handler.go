// backend\internal\adapters\in\http\handlers\order_handler.go
package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	orderdom "narratives/internal/domain/order"
)

// OrderHandler は /orders 関連のエンドポイントを担当します（単一取得のみ）。
type OrderHandler struct {
	uc *usecase.OrderUsecase
}

// NewOrderHandler はHTTPハンドラを初期化します。
func NewOrderHandler(uc *usecase.OrderUsecase) http.Handler {
	return &OrderHandler{uc: uc}
}

// ServeHTTP はHTTPルーティングの入口です。
func (h *OrderHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/orders/"):
		id := strings.TrimPrefix(r.URL.Path, "/orders/")
		h.get(w, r, id)
	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// GET /orders/{id}
func (h *OrderHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	order, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeOrderErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(order)
}

// エラーハンドリング
func writeOrderErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	// NotFound 有無が不明なため InvalidID のみ特別扱い
	if err == orderdom.ErrInvalidID {
		code = http.StatusBadRequest
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
