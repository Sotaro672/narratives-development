package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
)

// TransactionHandler は /transactions 関連のエンドポイントを担当します（単一取得のみ）。
type TransactionHandler struct {
	uc *usecase.TransactionUsecase
}

// NewTransactionHandler はHTTPハンドラを初期化します。
func NewTransactionHandler(uc *usecase.TransactionUsecase) http.Handler {
	return &TransactionHandler{uc: uc}
}

// ServeHTTP はHTTPルーティングの入口です。
func (h *TransactionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/transactions/"):
		id := strings.TrimPrefix(r.URL.Path, "/transactions/")
		h.get(w, r, id)
	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// GET /transactions/{id}
func (h *TransactionHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	tx, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeTransactionErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(tx)
}

// エラーハンドリング
func writeTransactionErr(w http.ResponseWriter, err error) {
	// ドメインのエラー型に依存せず 500 を返す
	w.WriteHeader(http.StatusInternalServerError)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
