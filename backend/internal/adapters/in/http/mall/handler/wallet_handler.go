// backend/internal/adapters/in/http/mall/handler/wallet_handler.go
package mallHandler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	walletdom "narratives/internal/domain/wallet"

	"narratives/internal/adapters/in/http/middleware"
)

// MallWalletHandler handles mall buyer-facing wallet endpoints.
//
// ✅ Routes (mall) - NEW ONLY (legacy removed):
// - GET  /mall/me/wallets
// - POST /mall/me/wallets/sync
//
// Contract assumptions (new only):
//   - uid is provided by UserAuthMiddleware in request context.
//   - avatarId + walletAddress are provided by AvatarContextMiddleware in request context.
//   - walletAddress is NOT accepted from client (not in path/query/body).
type MallWalletHandler struct {
	walletUC *usecase.WalletUsecase
}

func NewMallWalletHandler(walletUC *usecase.WalletUsecase) http.Handler {
	return &MallWalletHandler{walletUC: walletUC}
}

func (h *MallWalletHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// normalize path (drop trailing slash)
	path0 := strings.TrimSuffix(r.URL.Path, "/")

	switch {
	// ✅ read-only view (should NOT sync implicitly)
	case r.Method == http.MethodGet && path0 == "/mall/me/wallets":
		h.getMeWallets(w, r)
		return

	// ✅ explicit sync endpoint
	case r.Method == http.MethodPost && path0 == "/mall/me/wallets/sync":
		h.syncMeWallets(w, r)
		return

	default:
		notFound(w)
		return
	}
}

// GET /mall/me/wallets
// - returns persisted wallet snapshot (no RPC call here)
func (h *MallWalletHandler) getMeWallets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h == nil || h.walletUC == nil || h.walletUC.WalletRepo == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "wallet usecase not configured"})
		return
	}

	// ✅ typed-key context getter (fixes mismatch with AvatarContextMiddleware)
	avatarID, ok := middleware.CurrentAvatarID(r)
	avatarID = strings.TrimSpace(avatarID)
	if !ok || avatarID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	uid, _ := middleware.CurrentUserUID(r)
	log.Printf("[mall_wallet_handler] GET /mall/me/wallets uid=%q avatarId=%q", maskID(uid), maskID(avatarID))

	// read persisted wallet only (no legacy address resolution)
	wallet, err := h.walletUC.WalletRepo.GetByAvatarID(ctx, avatarID)
	if err != nil {
		writeMallWalletErr(w, err)
		return
	}

	// wallets を正として返却（現状は 1 avatar = 1 wallet 前提）
	_ = json.NewEncoder(w).Encode(map[string]any{"wallets": []walletdom.Wallet{wallet}})
}

// POST /mall/me/wallets/sync
// - runs on-chain RPC via WalletUsecase.SyncWalletTokens and returns updated wallet
func (h *MallWalletHandler) syncMeWallets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h == nil || h.walletUC == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "wallet usecase not configured"})
		return
	}

	// ✅ typed-key context getter (fixes mismatch with AvatarContextMiddleware)
	avatarID, ok := middleware.CurrentAvatarID(r)
	avatarID = strings.TrimSpace(avatarID)
	if !ok || avatarID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	uid, _ := middleware.CurrentUserUID(r)
	log.Printf("[mall_wallet_handler] POST /mall/me/wallets/sync uid=%q avatarId=%q", maskID(uid), maskID(avatarID))

	wallet, err := h.walletUC.SyncWalletTokens(ctx, avatarID)
	if err != nil {
		writeMallWalletErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{"wallets": []walletdom.Wallet{wallet}})
}

func writeMallWalletErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	switch {
	case errors.Is(err, walletdom.ErrNotFound):
		code = http.StatusNotFound
	case errors.Is(err, usecase.ErrWalletSyncAvatarIDEmpty),
		errors.Is(err, usecase.ErrWalletSyncWalletAddressEmpty):
		code = http.StatusBadRequest
	case errors.Is(err, usecase.ErrWalletSyncOnchainNotConfigured),
		errors.Is(err, usecase.ErrWalletUsecaseNotConfigured):
		code = http.StatusServiceUnavailable
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func maskID(s string) string {
	t := strings.TrimSpace(s)
	if len(t) <= 8 {
		return t
	}
	return t[:4] + "***" + t[len(t)-4:]
}
