// backend/internal/adapters/in/http/console/handler/wallet_handler.go
package consoleHandler

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

	switch {
	case r.Method == http.MethodGet && strings.HasPrefix(strings.TrimSuffix(r.URL.Path, "/"), "/wallets/"):
		path := strings.TrimSuffix(r.URL.Path, "/")
		avatarID := strings.TrimSpace(strings.TrimPrefix(path, "/wallets/"))
		h.get(w, r, avatarID)
		return
	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}
}

// GET /wallets/{avatarId}
// - docId=avatarId を正とする Wallet を取得し、on-chain と同期して返す
func (h *WalletHandler) get(w http.ResponseWriter, r *http.Request, avatarID string) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "wallet usecase not configured"})
		return
	}

	aid := strings.TrimSpace(avatarID)
	if aid == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid avatarId"})
		return
	}

	// ✅ 新シグネチャ: SyncWalletTokens(ctx, avatarId)
	wallet, err := h.uc.SyncWalletTokens(ctx, aid)
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
	// usecase 側の validation error は 400 に寄せる（最低限）
	if err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "avatarid is empty") ||
			strings.Contains(msg, "walletaddress is empty") ||
			strings.Contains(msg, "onchain reader not configured") {
			code = http.StatusBadRequest
		}
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
