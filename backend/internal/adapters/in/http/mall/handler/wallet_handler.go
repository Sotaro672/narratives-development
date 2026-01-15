// backend\internal\adapters\in\http\mall\handler\wallet_handler.go
package mallHandler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	walletdom "narratives/internal/domain/wallet"
)

// MallWalletHandler handles mall buyer-facing wallet endpoints.
//
// Routes (mall):
// - GET /mall/me/wallets   (canonical)
//
// NOTE:
// - WalletUsecase.SyncWalletTokens(ctx, avatarID, addr) requires addr for create.
// - For /mall/me/wallets, we try to resolve addr from Avatar.walletAddress.
type MallWalletHandler struct {
	walletUC *usecase.WalletUsecase
	avatarUC *usecase.AvatarUsecase
}

func NewMallWalletHandler(walletUC *usecase.WalletUsecase, avatarUC *usecase.AvatarUsecase) http.Handler {
	return &MallWalletHandler{walletUC: walletUC, avatarUC: avatarUC}
}

func (h *MallWalletHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// normalize path (drop trailing slash)
	path0 := strings.TrimSuffix(r.URL.Path, "/")

	switch {
	// canonical: plural only
	case r.Method == http.MethodGet && path0 == "/mall/me/wallets":
		h.getMeWallets(w, r)
		return
	default:
		notFound(w)
		return
	}
}

// GET /mall/me/wallets
func (h *MallWalletHandler) getMeWallets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h == nil || h.walletUC == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "wallet usecase not configured"})
		return
	}

	// ✅ avatarId を auth context から取得（なければ uid を使う）
	avatarID := strings.TrimSpace(getCtxString(ctx, "avatarId"))
	if avatarID == "" {
		avatarID = strings.TrimSpace(getCtxString(ctx, "uid"))
	}
	if avatarID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	// ✅ addr を avatar から解決（未作成時の create に必要）
	addr := ""
	if h.avatarUC != nil {
		if a, err := h.avatarUC.GetByID(ctx, avatarID); err == nil {
			if a.WalletAddress != nil {
				addr = strings.TrimSpace(*a.WalletAddress)
			}
		} else {
			// avatar が無い/引けない場合でも、wallet 側が既存なら動く可能性があるので続行
			log.Printf("[mall_wallet_handler] /mall/me/wallets avatar lookup failed avatarId=%q err=%v\n", maskID(avatarID), err)
		}
	}

	log.Printf("[mall_wallet_handler] GET /mall/me/wallets avatarId=%q addr_set=%t\n", maskID(avatarID), addr != "")

	wallet, err := h.walletUC.SyncWalletTokens(ctx, avatarID, addr)
	if err != nil {
		writeMallWalletErr(w, err)
		return
	}

	// wallets を正として返却（現状は 1 avatar = 1 wallet 前提）
	_ = json.NewEncoder(w).Encode(map[string]any{"wallets": []walletdom.Wallet{wallet}})
}

func writeMallWalletErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	switch {
	case errors.Is(err, walletdom.ErrNotFound):
		code = http.StatusNotFound
	}

	// usecase 側の validation error は 400 に寄せる（文字列ベースの最低限）
	if err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "avatarid is empty") ||
			strings.Contains(msg, "walletaddress is required") ||
			strings.Contains(msg, "walletaddress is empty") {
			code = http.StatusBadRequest
		}
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

// ctx.Value は interface{} なので string のみ拾う（既存ミドルウェアに合わせる）
func getCtxString(ctx interface{ Value(any) any }, key string) string {
	v := ctx.Value(key)
	s, _ := v.(string)
	return s
}

func maskID(s string) string {
	t := strings.TrimSpace(s)
	if len(t) <= 8 {
		return t
	}
	return t[:4] + "***" + t[len(t)-4:]
}
