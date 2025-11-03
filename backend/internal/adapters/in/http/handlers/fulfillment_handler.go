package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	fulfillmentdom "narratives/internal/domain/fulfillment"
)

// FulfillmentHandler は /fulfillments 関連のエンドポイントを担当します（単一取得のみ）。
type FulfillmentHandler struct {
	uc *usecase.FulfillmentUsecase
}

// NewFulfillmentHandler はHTTPハンドラを初期化します。
func NewFulfillmentHandler(uc *usecase.FulfillmentUsecase) http.Handler {
	return &FulfillmentHandler{uc: uc}
}

// ServeHTTP はHTTPルーティングの入口です。
func (h *FulfillmentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/fulfillments/"):
		id := strings.TrimPrefix(r.URL.Path, "/fulfillments/")
		h.get(w, r, id)
	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// GET /fulfillments/{id}
func (h *FulfillmentHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	f, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeFulfillmentErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(f)
}

// エラーハンドリング
func writeFulfillmentErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	switch err {
	case fulfillmentdom.ErrInvalidID:
		code = http.StatusBadRequest
	case fulfillmentdom.ErrNotFound:
		code = http.StatusNotFound
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
