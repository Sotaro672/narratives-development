package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	mintreqdom "narratives/internal/domain/mintRequest"
)

// MintRequestHandler は /mint-requests 関連のエンドポイントを担当します（単一取得のみ）。
type MintRequestHandler struct {
	uc *usecase.MintRequestUsecase
}

// NewMintRequestHandler はHTTPハンドラを初期化します。
func NewMintRequestHandler(uc *usecase.MintRequestUsecase) http.Handler {
	return &MintRequestHandler{uc: uc}
}

// ServeHTTP はHTTPルーティングの入口です。
func (h *MintRequestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/mint-requests/"):
		id := strings.TrimPrefix(r.URL.Path, "/mint-requests/")
		h.get(w, r, id)
	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// GET /mint-requests/{id}
func (h *MintRequestHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	mr, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeMintRequestErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(mr)
}

// エラーハンドリング
func writeMintRequestErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	if err == mintreqdom.ErrInvalidID {
		code = http.StatusBadRequest
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
