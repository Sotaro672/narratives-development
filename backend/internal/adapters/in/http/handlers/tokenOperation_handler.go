// backend\internal\adapters\in\http\handlers\tokenOperation_handler.go
package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	tokenopdom "narratives/internal/domain/tokenOperation"
)

// TokenOperationHandler は /token-operations 関連のエンドポイントを担当します（単一取得のみ）。
type TokenOperationHandler struct {
	uc *usecase.TokenOperationUsecase
}

// NewTokenOperationHandler はHTTPハンドラを初期化します。
func NewTokenOperationHandler(uc *usecase.TokenOperationUsecase) http.Handler {
	return &TokenOperationHandler{uc: uc}
}

// ServeHTTP はHTTPルーティングの入口です。
func (h *TokenOperationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/token-operations/"):
		id := strings.TrimPrefix(r.URL.Path, "/token-operations/")
		h.get(w, r, id)
	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// GET /token-operations/{id}
func (h *TokenOperationHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	op, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeTokenOperationErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(op)
}

// エラーハンドリング
func writeTokenOperationErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	if err == tokenopdom.ErrInvalidID {
		code = http.StatusBadRequest
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
