// backend/internal/adapters/in/http/mall/handler/wallet_me_handler.go
package mallHandler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
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

// MallMeWalletHandler handles mall buyer-facing wallet endpoints.
//
// Routes:
//   - GET     /mall/me/wallets
//   - POST    /mall/me/wallets/sync
//   - GET     /mall/me/wallets/tokens/resolve?mintAddress=...
//   - OPTIONS /mall/me/wallets/metadata/proxy?url=...
//   - GET     /mall/me/wallets/metadata/proxy?url=...
type MallMeWalletHandler struct {
	walletUC *usecase.WalletUsecase

	// resolved token cache (Firestore wallets/{avatarId}/resolvedTokens/{mint})
	resolvedTokenRepo ResolvedTokenRepository

	// optional: allowlist for proxy host validation
	// if empty, defaults are used
	allowedProxyHosts map[string]struct{}
}

// ResolvedTokenRepository is a minimal port for resolvedTokens cache.
// (Firestore implementation lives in adapters/out)
type ResolvedTokenRepository interface {
	GetByAvatarIDAndMint(ctx context.Context, avatarID string, mintAddress string) (usecase.ResolveTokenByMintAddressWithBrandNameResult, error)
	Upsert(ctx context.Context, avatarID string, mintAddress string, res usecase.ResolveTokenByMintAddressWithBrandNameResult, now time.Time) error
}

// NewMallMeWalletHandler wires mall /me wallet endpoints.
// - resolvedTokenRepo can be nil (handler falls back to full resolve every time).
func NewMallMeWalletHandler(walletUC *usecase.WalletUsecase, resolvedTokenRepo ResolvedTokenRepository) http.Handler {
	return &MallMeWalletHandler{
		walletUC:          walletUC,
		resolvedTokenRepo: resolvedTokenRepo,
		allowedProxyHosts: map[string]struct{}{
			"gateway.irys.xyz":             {},
			"uploader.irys.xyz":            {},
			"mainnet-1.datasprite-cdn.com": {},
			"arweave.net":                  {},
			"www.arweave.net":              {},
			"ipfs.io":                      {},
			"cloudflare-ipfs.com":          {},
			"storage.googleapis.com":       {},
		},
	}
}

func (h *MallMeWalletHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	path0 := strings.TrimSuffix(r.URL.Path, "/")

	switch {
	case r.Method == http.MethodGet && path0 == "/mall/me/wallets":
		h.getMeWallets(w, r)
		return

	case r.Method == http.MethodPost && path0 == "/mall/me/wallets/sync":
		h.syncMeWallets(w, r)
		return

	case r.Method == http.MethodGet && path0 == "/mall/me/wallets/tokens/resolve":
		h.resolveMeTokenByMintAddress(w, r)
		return

	case r.Method == http.MethodOptions && path0 == "/mall/me/wallets/metadata/proxy":
		h.preflightMeWalletMetadataProxy(w)
		return

	case r.Method == http.MethodGet && path0 == "/mall/me/wallets/metadata/proxy":
		h.meWalletMetadataProxy(w, r)
		return

	default:
		notFound(w)
		return
	}
}

// GET /mall/me/wallets
// - returns current wallet snapshot
// - compares persisted wallet.tokens with Solana devnet owned mints
// - if different, syncs wallet.tokens from on-chain before returning
func (h *MallMeWalletHandler) getMeWallets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h == nil || h.walletUC == nil || h.walletUC.WalletRepo == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "wallet usecase not configured"})
		return
	}

	avatarID, ok := middleware.CurrentAvatarID(r)
	if !ok || avatarID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	wallet, err := h.walletUC.WalletRepo.GetByAvatarID(ctx, avatarID)
	if err != nil {
		writeMallMeWalletErr(w, err)
		return
	}

	if h.walletUC.OnchainReader == nil {
		_ = json.NewEncoder(w).Encode(map[string]any{"wallets": []walletdom.Wallet{wallet}})
		return
	}

	if wallet.WalletAddress == "" {
		_ = json.NewEncoder(w).Encode(map[string]any{"wallets": []walletdom.Wallet{wallet}})
		return
	}

	onchainMints, err := h.walletUC.OnchainReader.ListOwnedTokenMints(ctx, wallet.WalletAddress)
	if err != nil {
		_ = json.NewEncoder(w).Encode(map[string]any{"wallets": []walletdom.Wallet{wallet}})
		return
	}

	if !sameStringSet(wallet.Tokens, onchainMints) {
		synced, syncErr := h.walletUC.SyncWalletTokens(ctx, avatarID)
		if syncErr != nil {
			_ = json.NewEncoder(w).Encode(map[string]any{"wallets": []walletdom.Wallet{wallet}})
			return
		}

		wallet = synced
	}

	_ = json.NewEncoder(w).Encode(map[string]any{"wallets": []walletdom.Wallet{wallet}})
}

// POST /mall/me/wallets/sync
func (h *MallMeWalletHandler) syncMeWallets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h == nil || h.walletUC == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "wallet usecase not configured"})
		return
	}

	avatarID, ok := middleware.CurrentAvatarID(r)
	if !ok || avatarID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	wallet, err := h.walletUC.SyncWalletTokens(ctx, avatarID)
	if err != nil {
		writeMallMeWalletErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{"wallets": []walletdom.Wallet{wallet}})
}

// GET /mall/me/wallets/tokens/resolve?mintAddress=...
func (h *MallMeWalletHandler) resolveMeTokenByMintAddress(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h == nil || h.walletUC == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "wallet usecase not configured"})
		return
	}

	avatarID, ok := middleware.CurrentAvatarID(r)
	if !ok || avatarID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	mintAddress := r.URL.Query().Get("mintAddress")
	if mintAddress == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "mintAddress is required"})
		return
	}

	if h.walletUC.WalletRepo == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "wallet repository not configured"})
		return
	}

	snap, err := h.walletUC.WalletRepo.GetByAvatarID(ctx, avatarID)
	if err != nil {
		writeMallMeWalletErr(w, err)
		return
	}

	owned := walletSnapshotHasMintPreferTokens(snap, mintAddress)

	if !owned && h.walletUC.OnchainReader != nil {
		addr := snap.WalletAddress
		if addr != "" {
			mints, e := h.walletUC.OnchainReader.ListOwnedTokenMints(ctx, addr)
			if e == nil {
				owned = stringSliceContainsExact(mints, mintAddress)
			}
		}
	}

	if !owned {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "mintAddress is not owned by current avatar"})
		return
	}

	var res usecase.ResolveTokenByMintAddressWithBrandNameResult
	var fromCache bool

	if !fromCache {
		rr, e := h.walletUC.ResolveTokenByMintAddressWithBrandName(ctx, mintAddress)
		if e != nil {
			writeMallMeWalletErr(w, e)
			return
		}
		res = rr

		if h.resolvedTokenRepo != nil {
			_ = h.resolvedTokenRepo.Upsert(ctx, avatarID, mintAddress, res, time.Now().UTC())
		}
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"productId":          res.ProductID,
		"brandId":            res.BrandID,
		"brandName":          res.BrandName,
		"productBlueprintId": res.ProductBlueprintID,
		"productName":        res.ProductName,
		"metadataUri":        res.MetadataURI,
		"mintAddress":        res.MintAddress,
	})
}

// OPTIONS /mall/me/wallets/metadata/proxy
func (h *MallMeWalletHandler) preflightMeWalletMetadataProxy(w http.ResponseWriter) {
	h.setCORSHeaders(w)
	w.WriteHeader(http.StatusNoContent)
}

// GET /mall/me/wallets/metadata/proxy?url=https://...
func (h *MallMeWalletHandler) meWalletMetadataProxy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h == nil || h.walletUC == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "wallet usecase not configured"})
		return
	}

	avatarID, ok := middleware.CurrentAvatarID(r)
	if !ok || avatarID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	rawURL := r.URL.Query().Get("url")
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
	if strings.ToLower(u.Scheme) != "https" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "only https is allowed"})
		return
	}
	if u.Port() != "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "explicit port is not allowed"})
		return
	}

	host := strings.ToLower(u.Hostname())
	if host == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid url host"})
		return
	}

	allow := h.allowedProxyHosts
	if len(allow) == 0 {
		allow = map[string]struct{}{
			"gateway.irys.xyz":             {},
			"uploader.irys.xyz":            {},
			"mainnet-1.datasprite-cdn.com": {},
			"arweave.net":                  {},
			"www.arweave.net":              {},
			"ipfs.io":                      {},
			"cloudflare-ipfs.com":          {},
			"storage.googleapis.com":       {},
		}
	}
	if !isAllowedMetadataProxyHost(host, allow) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "host is not allowed"})
		return
	}

	h.setCORSHeaders(w)

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
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return errors.New("too many redirects")
			}
			if req == nil || req.URL == nil {
				return errors.New("invalid redirect url")
			}
			if strings.ToLower(req.URL.Scheme) != "https" {
				return errors.New("redirect to non-https is not allowed")
			}
			if req.URL.Port() != "" {
				return errors.New("redirect with explicit port is not allowed")
			}
			hh := strings.ToLower(req.URL.Hostname())
			if hh == "" {
				return errors.New("redirect host is empty")
			}
			if !isAllowedMetadataProxyHost(hh, allow) {
				return errors.New("redirect host is not allowed")
			}
			return nil
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

	const maxBytes = 1 << 20
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

	if filtered, ok, e := filterMetadataJSON(body); e == nil && ok {
		body = filtered
	}

	ct := res.Header.Get("Content-Type")
	if ct != "" {
		w.Header().Set("Content-Type", ct)
	} else {
		w.Header().Set("Content-Type", "application/json")
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func isAllowedMetadataProxyHost(host string, allow map[string]struct{}) bool {
	h := strings.ToLower(strings.TrimSpace(host))
	if h == "" {
		return false
	}

	if _, ok := allow[h]; ok {
		return true
	}

	// Irys gateway may redirect to generated subdomains under datasprite CDN.
	if h == "mainnet-1.datasprite-cdn.com" || strings.HasSuffix(h, ".mainnet-1.datasprite-cdn.com") {
		return true
	}

	return false
}

func (h *MallMeWalletHandler) setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Vary", "Origin")
	w.Header().Set("Access-Control-Allow-Methods", "GET,OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Authorization,Content-Type,Accept")
	w.Header().Set("Access-Control-Max-Age", "600")
}

func writeMallMeWalletErr(w http.ResponseWriter, err error) {
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
		errors.Is(err, usecase.ErrWalletTokenQueryNotConfigured),
		errors.Is(err, usecase.ErrWalletBrandNameNotConfigured),
		errors.Is(err, usecase.ErrWalletProductReaderNotConfigured),
		errors.Is(err, usecase.ErrWalletModelProductBlueprintNotConfigured),
		errors.Is(err, usecase.ErrWalletProductBlueprintReaderNotConfigured):
		code = http.StatusServiceUnavailable
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func stringSliceContainsExact(xs []string, target string) bool {
	if target == "" || len(xs) == 0 {
		return false
	}
	for _, x := range xs {
		if x == target {
			return true
		}
	}
	return false
}

func sameStringSet(a []string, b []string) bool {
	am := make(map[string]int, len(a))
	bm := make(map[string]int, len(b))

	for _, x := range a {
		if x == "" {
			continue
		}
		am[x]++
	}

	for _, x := range b {
		if x == "" {
			continue
		}
		bm[x]++
	}

	if len(am) != len(bm) {
		return false
	}

	for k, av := range am {
		if bm[k] != av {
			return false
		}
	}

	return true
}

// walletSnapshotHasMintPreferTokens uses only the canonical field "Tokens".
func walletSnapshotHasMintPreferTokens(w walletdom.Wallet, mintAddress string) bool {
	return stringSliceContainsExact(w.Tokens, mintAddress)
}

func isKeepObjectURI(raw string) bool {
	s := raw
	if s == "" {
		return false
	}

	u, err := url.Parse(s)
	if err != nil || u == nil {
		return strings.Contains(s, "/.keep") || strings.HasSuffix(s, ".keep")
	}

	p := u.Path
	if p == "" {
		return false
	}
	p = strings.TrimSuffix(p, "/")
	return strings.HasSuffix(p, "/.keep") || strings.HasSuffix(p, ".keep")
}

func filterMetadataJSON(body []byte) ([]byte, bool, error) {
	var root map[string]any
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	if err := dec.Decode(&root); err != nil {
		return nil, false, err
	}
	if len(root) == 0 {
		return body, false, nil
	}

	props, ok := root["properties"].(map[string]any)
	if !ok || props == nil {
		return body, false, nil
	}

	filesAny, ok := props["files"].([]any)
	if !ok || len(filesAny) == 0 {
		return body, false, nil
	}

	outFiles := make([]any, 0, len(filesAny))
	for _, item := range filesAny {
		m, ok := item.(map[string]any)
		if !ok || m == nil {
			continue
		}
		uri, _ := m["uri"].(string)
		if uri == "" {
			continue
		}

		if isKeepObjectURI(uri) {
			continue
		}

		outFiles = append(outFiles, m)
	}

	props["files"] = outFiles
	root["properties"] = props

	b, err := json.Marshal(root)
	if err != nil {
		return nil, false, err
	}
	return b, true, nil
}
