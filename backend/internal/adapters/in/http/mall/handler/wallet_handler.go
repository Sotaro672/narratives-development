// backend/internal/adapters/in/http/mall/handler/wallet_handler.go
package mallHandler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"narratives/internal/adapters/in/http/middleware"
	usecase "narratives/internal/application/usecase"
	tokendom "narratives/internal/domain/token"
	walletdom "narratives/internal/domain/wallet"
)

// MallWalletHandler handles mall buyer-facing wallet endpoints.
//
// ✅ Routes (mall) - NEW ONLY (legacy removed):
// - GET  /mall/me/wallets
// - POST /mall/me/wallets/sync
// - GET  /mall/me/wallets/tokens/resolve?mintAddress=...
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

	// ✅ resolve token by mintAddress
	case r.Method == http.MethodGet && path0 == "/mall/me/wallets/tokens/resolve":
		h.resolveTokenByMintAddress(w, r)
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

	avatarID, ok := middleware.CurrentAvatarID(r)
	avatarID = strings.TrimSpace(avatarID)
	if !ok || avatarID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	uid, _ := middleware.CurrentUserUID(r)
	log.Printf("[mall_wallet_handler] GET /mall/me/wallets uid=%q avatarId=%q", maskID(uid), maskID(avatarID))

	wallet, err := h.walletUC.WalletRepo.GetByAvatarID(ctx, avatarID)
	if err != nil {
		writeMallWalletErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{"wallets": []walletdom.Wallet{wallet}})
}

// POST /mall/me/wallets/sync
func (h *MallWalletHandler) syncMeWallets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h == nil || h.walletUC == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "wallet usecase not configured"})
		return
	}

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

// GET /mall/me/wallets/tokens/resolve?mintAddress=...
// - returns: { productId, brandId, metadataUri, mintAddress }
func (h *MallWalletHandler) resolveTokenByMintAddress(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h == nil || h.walletUC == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "wallet usecase not configured"})
		return
	}

	// auth context check (same rule as other /mall/me/* endpoints)
	avatarID, ok := middleware.CurrentAvatarID(r)
	avatarID = strings.TrimSpace(avatarID)
	if !ok || avatarID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	mintAddress := strings.TrimSpace(r.URL.Query().Get("mintAddress"))
	if mintAddress == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "mintAddress is required"})
		return
	}

	uid, _ := middleware.CurrentUserUID(r)
	log.Printf(
		"[mall_wallet_handler] GET /mall/me/wallets/tokens/resolve uid=%q avatarId=%q mint=%q",
		maskID(uid),
		maskID(avatarID),
		maskID(mintAddress),
	)

	res, err := h.walletUC.ResolveTokenByMintAddress(ctx, mintAddress)
	if err != nil {
		writeMallWalletErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"productId":   strings.TrimSpace(res.ProductID),
		"brandId":     strings.TrimSpace(res.BrandID),
		"metadataUri": strings.TrimSpace(res.MetadataURI),
		"mintAddress": strings.TrimSpace(res.MintAddress),
	})
}

func writeMallWalletErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	switch {
	// ✅ wallet not found
	case errors.Is(err, walletdom.ErrNotFound):
		code = http.StatusNotFound

	// ✅ token not found (mintAddress -> token doc not found)
	case errors.Is(err, tokendom.ErrNotFound):
		code = http.StatusNotFound

	// Sync: bad request
	case errors.Is(err, usecase.ErrWalletSyncAvatarIDEmpty),
		errors.Is(err, usecase.ErrWalletSyncWalletAddressEmpty):
		code = http.StatusBadRequest

	// Resolve: bad request
	case errors.Is(err, usecase.ErrMintAddressEmpty),
		errors.Is(err, tokendom.ErrInvalidMintAddress):
		code = http.StatusBadRequest

	// Not configured
	case errors.Is(err, usecase.ErrWalletSyncOnchainNotConfigured),
		errors.Is(err, usecase.ErrWalletUsecaseNotConfigured),
		errors.Is(err, usecase.ErrWalletTokenQueryNotConfigured):
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
