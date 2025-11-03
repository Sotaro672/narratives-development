package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	txdom "narratives/internal/domain/transaction"
)

// TransferHandler は /transfers 関連のエンドポイントを担当します（単一取得のみ）。
type TransferHandler struct {
	uc *usecase.TransferUsecase
}

// NewTransferHandler はHTTPハンドラを初期化します。
func NewTransferHandler(uc *usecase.TransferUsecase) http.Handler {
	return &TransferHandler{uc: uc}
}

// ServeHTTP はHTTPルーティングの入口です。
func (h *TransferHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/transfers/"):
		id := strings.TrimPrefix(r.URL.Path, "/transfers/")
		h.get(w, r, id)
	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// GET /transfers/{id}
func (h *TransferHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	trf, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeTransferErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(trf)
}

// エラーハンドリング
func writeTransferErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	switch err {
	case txdom.ErrInvalidID:
		code = http.StatusBadRequest
	case txdom.ErrNotFound:
		code = http.StatusNotFound
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
