// backend/internal/adapters/in/http/mall/handler/wallet_handler.go
package mallHandler

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"narratives/internal/adapters/in/http/middleware"
	usecase "narratives/internal/application/usecase"
	tokendom "narratives/internal/domain/token"
	walletdom "narratives/internal/domain/wallet"
)

// MallWalletHandler handles mall buyer-facing wallet endpoints.
//
// ✅ Routes (mall) - NEW ONLY (legacy removed):
// - GET     /mall/me/wallets
// - POST    /mall/me/wallets/sync
// - GET     /mall/me/wallets/tokens/resolve?mintAddress=...
// - OPTIONS /mall/me/wallets/metadata/proxy?url=...
// - GET     /mall/me/wallets/metadata/proxy?url=...   (CORS avoidance; fetch metadata JSON)
//
// Contract assumptions (new only):
//   - uid is provided by UserAuthMiddleware in request context.
//   - avatarId + walletAddress are provided by AvatarContextMiddleware in request context.
//   - walletAddress is NOT accepted from client (not in path/query/body).
type MallWalletHandler struct {
	walletUC *usecase.WalletUsecase

	// optional: allowlist for proxy host validation
	// if empty, defaults are used
	allowedProxyHosts map[string]struct{}
}

func NewMallWalletHandler(walletUC *usecase.WalletUsecase) http.Handler {
	return &MallWalletHandler{
		walletUC: walletUC,
		allowedProxyHosts: map[string]struct{}{
			// ✅ add hosts you actually use
			"gateway.irys.xyz":    {},
			"arweave.net":         {},
			"www.arweave.net":     {},
			"ipfs.io":             {},
			"cloudflare-ipfs.com": {},
		},
	}
}

func (h *MallWalletHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// NOTE:
	// proxy may return JSON; we still set JSON by default.
	// if upstream has a content-type, we will override in proxy handler.
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

	// ✅ metadata proxy (CORS avoidance)
	case r.Method == http.MethodOptions && path0 == "/mall/me/wallets/metadata/proxy":
		h.preflightMetadataProxy(w, r)
		return
	case r.Method == http.MethodGet && path0 == "/mall/me/wallets/metadata/proxy":
		h.metadataProxy(w, r)
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

// OPTIONS /mall/me/wallets/metadata/proxy
func (h *MallWalletHandler) preflightMetadataProxy(w http.ResponseWriter, _ *http.Request) {
	h.setCORSHeaders(w)
	w.WriteHeader(http.StatusNoContent)
}

// GET /mall/me/wallets/metadata/proxy?url=https://...
// - Fetches metadata JSON from allowed hosts and returns it as-is (to avoid browser CORS).
func (h *MallWalletHandler) metadataProxy(w http.ResponseWriter, r *http.Request) {
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

	rawURL := strings.TrimSpace(r.URL.Query().Get("url"))
	if rawURL == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "url is required"})
		return
	}

	u, err := url.Parse(rawURL)
	if err != nil || u == nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid url"})
		return
	}
	if strings.ToLower(strings.TrimSpace(u.Scheme)) != "https" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "only https is allowed"})
		return
	}

	host := strings.ToLower(strings.TrimSpace(u.Hostname()))
	if host == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid url host"})
		return
	}

	// ✅ Host allowlist to prevent SSRF
	allow := h.allowedProxyHosts
	if len(allow) == 0 {
		allow = map[string]struct{}{
			"gateway.irys.xyz":    {},
			"arweave.net":         {},
			"www.arweave.net":     {},
			"ipfs.io":             {},
			"cloudflare-ipfs.com": {},
		}
	}
	if _, ok := allow[host]; !ok {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "host is not allowed"})
		return
	}

	uid, _ := middleware.CurrentUserUID(r)
	log.Printf(
		"[mall_wallet_handler] GET /mall/me/wallets/metadata/proxy uid=%q avatarId=%q url=%q",
		maskID(uid),
		maskID(avatarID),
		rawURL,
	)

	// ✅ CORS headers for browser clients
	h.setCORSHeaders(w)

	// ✅ HTTP client with timeout + sane transport
	client := &http.Client{
		Timeout: 8 * time.Second,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          50,
			IdleConnTimeout:       30 * time.Second,
			TLSHandshakeTimeout:   5 * time.Second,
			ResponseHeaderTimeout: 5 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to create upstream request"})
		return
	}
	req.Header.Set("Accept", "application/json")

	res, err := client.Do(req)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "upstream fetch failed"})
		return
	}
	defer res.Body.Close()

	// limit size (prevent abuse)
	const maxBytes = 1 << 20 // 1MB
	body, err := io.ReadAll(io.LimitReader(res.Body, maxBytes))
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to read upstream"})
		return
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":        "upstream returned non-2xx",
			"status":       res.Status,
			"statusText":   http.StatusText(res.StatusCode),
			"upstreamCode": strconv.Itoa(res.StatusCode),
		})
		return
	}

	// prefer upstream content-type if present
	ct := strings.TrimSpace(res.Header.Get("Content-Type"))
	if ct != "" {
		w.Header().Set("Content-Type", ct)
	} else {
		w.Header().Set("Content-Type", "application/json")
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func (h *MallWalletHandler) setCORSHeaders(w http.ResponseWriter) {
	// If you want tighter control, replace "*" with your frontend origin.
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Vary", "Origin")
	w.Header().Set("Access-Control-Allow-Methods", "GET,OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Authorization,Content-Type,Accept")
	w.Header().Set("Access-Control-Max-Age", "600")
}

func writeMallWalletErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	switch {
	case errors.Is(err, walletdom.ErrNotFound):
		code = http.StatusNotFound
	case errors.Is(err, tokendom.ErrNotFound):
		code = http.StatusNotFound
	case errors.Is(err, usecase.ErrWalletSyncAvatarIDEmpty),
		errors.Is(err, usecase.ErrWalletSyncWalletAddressEmpty):
		code = http.StatusBadRequest
	case errors.Is(err, usecase.ErrMintAddressEmpty),
		errors.Is(err, tokendom.ErrInvalidMintAddress):
		code = http.StatusBadRequest
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
