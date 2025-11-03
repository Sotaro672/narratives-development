package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
)

// TokenHandler は /tokens 関連のエンドポイントを担当します（単一取得のみ）。
type TokenHandler struct {
	uc *usecase.TokenUsecase
}

// NewTokenHandler はHTTPハンドラを初期化します。
func NewTokenHandler(uc *usecase.TokenUsecase) http.Handler {
	return &TokenHandler{uc: uc}
}

// ServeHTTP はHTTPルーティングの入口です。
func (h *TokenHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/tokens/"):
		mintAddress := strings.TrimPrefix(r.URL.Path, "/tokens/")
		h.get(w, r, mintAddress)
	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// GET /tokens/{mintAddress}
func (h *TokenHandler) get(w http.ResponseWriter, r *http.Request, mintAddress string) {
	ctx := r.Context()

	mintAddress = strings.TrimSpace(mintAddress)
	if mintAddress == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid mintAddress"})
		return
	}

	token, err := h.uc.GetByID(ctx, mintAddress)
	if err != nil {
		writeTokenErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(token)
}

// エラーハンドリング
func writeTokenErr(w http.ResponseWriter, err error) {
	// ドメインのエラー型に依存せず 500 を返す
	w.WriteHeader(http.StatusInternalServerError)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
