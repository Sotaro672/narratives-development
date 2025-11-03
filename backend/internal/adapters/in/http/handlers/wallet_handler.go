package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	walletdom "narratives/internal/domain/wallet"
)

// WalletHandler は /wallets 関連のエンドポイントを担当します（単一取得のみ）。
type WalletHandler struct {
	uc *usecase.WalletUsecase
}

// NewWalletHandler はHTTPハンドラを初期化します。
func NewWalletHandler(uc *usecase.WalletUsecase) http.Handler {
	return &WalletHandler{uc: uc}
}

// ServeHTTP はHTTPルーティングの入口です。
func (h *WalletHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/wallets/"):
		walletAddress := strings.TrimPrefix(r.URL.Path, "/wallets/")
		h.get(w, r, walletAddress)
	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// GET /wallets/{wallet_address}
func (h *WalletHandler) get(w http.ResponseWriter, r *http.Request, walletAddress string) {
	ctx := r.Context()

	walletAddress = strings.TrimSpace(walletAddress)
	if walletAddress == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid wallet_address"})
		return
	}

	wallet, err := h.uc.GetByID(ctx, walletAddress)
	if err != nil {
		writeWalletErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(wallet)
}

// エラーハンドリング
func writeWalletErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	switch err {
	case walletdom.ErrNotFound:
		code = http.StatusNotFound
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
