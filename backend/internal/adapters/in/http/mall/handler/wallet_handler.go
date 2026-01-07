// backend/internal/adapters/in/http/mall/handler/wallet_handler.go
package mallHandler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	walletdom "narratives/internal/domain/wallet"
)

// WalletHandler は /wallets 関連のエンドポイントを担当します。
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

	// ✅ 末尾スラッシュを吸収
	path := strings.TrimSuffix(r.URL.Path, "/")

	// ✅ /mall プレフィックスを吸収（/mall/wallets -> /wallets）
	if strings.HasPrefix(path, "/mall/") {
		path = strings.TrimPrefix(path, "/mall")
		if path == "" {
			path = "/"
		}
	}

	switch {
	// GET /wallets/{wallet_address}?avatarId={avatarId}
	case r.Method == http.MethodGet && strings.HasPrefix(path, "/wallets/"):
		walletAddress := strings.TrimPrefix(path, "/wallets/")
		h.get(w, r, walletAddress)
		return

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}
}

// GET /wallets/{wallet_address}?avatarId={avatarId}
func (h *WalletHandler) get(w http.ResponseWriter, r *http.Request, walletAddress string) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "wallet usecase not configured"})
		return
	}

	walletAddress = strings.TrimSpace(walletAddress)
	if walletAddress == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid wallet_address"})
		return
	}

	// ✅ 新シグネチャ対応:
	// SyncWalletTokens(ctx, avatarId, walletAddress)
	// - docId=avatarId の永続化設計のため、可能なら avatarId を渡す。
	// - /wallets/{address} だけでは avatarId が分からないので query で受ける（互換のため未指定は空文字でOK）。
	avatarID := strings.TrimSpace(r.URL.Query().Get("avatarId"))

	wallet, err := h.uc.SyncWalletTokens(ctx, avatarID, walletAddress)
	if err != nil {
		writeWalletErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(wallet)
}

// エラーハンドリング
func writeWalletErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	switch {
	case errors.Is(err, walletdom.ErrNotFound):
		code = http.StatusNotFound
	}
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
