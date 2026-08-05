// backend/internal/adapters/in/http/mall/handler/wallet_me_handler.go
package mallHandler

import (
	"bytes"
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

	// optional: allowlist for proxy host validation
	// if empty, defaults are used
	allowedProxyHosts map[string]struct{}
}

// NewMallMeWalletHandler wires mall /me wallet endpoints.
func NewMallMeWalletHandler(walletUC *usecase.WalletUsecase) http.Handler {
	return &MallMeWalletHandler{
		walletUC:          walletUC,
		allowedProxyHosts: defaultWalletMetadataProxyHosts(),
	}
}

func defaultWalletMetadataProxyHosts() map[string]struct{} {
	return map[string]struct{}{
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

func (h *MallMeWalletHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	path := strings.TrimSuffix(r.URL.Path, "/")

	switch {
	case r.Method == http.MethodGet && path == "/mall/me/wallets":
		h.getMeWallets(w, r)
		return

	case r.Method == http.MethodPost && path == "/mall/me/wallets/sync":
		h.syncMeWallets(w, r)
		return

	case r.Method == http.MethodGet && path == "/mall/me/wallets/tokens/resolve":
		h.resolveMeTokenByMintAddress(w, r)
		return

	case r.Method == http.MethodOptions && path == "/mall/me/wallets/metadata/proxy":
		h.preflightMeWalletMetadataProxy(w)
		return

	case r.Method == http.MethodGet && path == "/mall/me/wallets/metadata/proxy":
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

	if h == nil || h.walletUC == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "wallet usecase not configured",
		})
		return
	}

	avatarID, ok := middleware.CurrentAvatarID(r)
	if !ok || avatarID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "unauthorized",
		})
		return
	}

	wallet, err := h.walletUC.GetWalletByAvatarIDWithReadThroughSync(ctx, avatarID)
	if err != nil {
		writeMallMeWalletErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"wallets": []walletdom.Wallet{wallet},
	})
}

// POST /mall/me/wallets/sync
func (h *MallMeWalletHandler) syncMeWallets(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h == nil || h.walletUC == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "wallet usecase not configured",
		})
		return
	}

	avatarID, ok := middleware.CurrentAvatarID(r)
	if !ok || avatarID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "unauthorized",
		})
		return
	}

	wallet, err := h.walletUC.SyncWalletTokens(ctx, avatarID)
	if err != nil {
		writeMallMeWalletErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"wallets": []walletdom.Wallet{wallet},
	})
}

// GET /mall/me/wallets/tokens/resolve?mintAddress=...
func (h *MallMeWalletHandler) resolveMeTokenByMintAddress(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h == nil || h.walletUC == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "wallet usecase not configured",
		})
		return
	}

	avatarID, ok := middleware.CurrentAvatarID(r)
	if !ok || avatarID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "unauthorized",
		})
		return
	}

	mintAddress := r.URL.Query().Get("mintAddress")
	if mintAddress == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "mintAddress is required",
		})
		return
	}

	result, err := h.walletUC.ResolveOwnedTokenByMintAddressWithBrandName(
		ctx,
		avatarID,
		mintAddress,
	)
	if err != nil {
		writeMallMeWalletErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"productId":          result.ProductID,
		"brandId":            result.BrandID,
		"brandName":          result.BrandName,
		"productBlueprintId": result.ProductBlueprintID,
		"productName":        result.ProductName,
		"metadataUri":        result.MetadataURI,
		"mintAddress":        result.MintAddress,
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
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "wallet usecase not configured",
		})
		return
	}

	avatarID, ok := middleware.CurrentAvatarID(r)
	if !ok || avatarID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "unauthorized",
		})
		return
	}

	rawURL := r.URL.Query().Get("url")
	if rawURL == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "url is required",
		})
		return
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil || parsedURL == nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid url",
		})
		return
	}

	if strings.ToLower(parsedURL.Scheme) != "https" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "only https is allowed",
		})
		return
	}

	if parsedURL.Port() != "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "explicit port is not allowed",
		})
		return
	}

	host := strings.ToLower(parsedURL.Hostname())
	if host == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid url host",
		})
		return
	}

	allow := h.allowedProxyHosts
	if len(allow) == 0 {
		allow = defaultWalletMetadataProxyHosts()
	}

	if !isAllowedMetadataProxyHost(host, allow) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "host is not allowed",
		})
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

			redirectHost := strings.ToLower(req.URL.Hostname())
			if redirectHost == "" {
				return errors.New("redirect host is empty")
			}

			if !isAllowedMetadataProxyHost(redirectHost, allow) {
				return errors.New("redirect host is not allowed")
			}

			return nil
		},
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		parsedURL.String(),
		nil,
	)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "failed to create upstream request",
		})
		return
	}

	req.Header.Set("Accept", "application/json")

	response, err := client.Do(req)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "upstream fetch failed",
		})
		return
	}
	defer response.Body.Close()

	const maxBytes = 1 << 20

	body, err := io.ReadAll(
		io.LimitReader(response.Body, maxBytes),
	)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "failed to read upstream",
		})
		return
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":        "upstream returned non-2xx",
			"status":       response.Status,
			"statusText":   http.StatusText(response.StatusCode),
			"upstreamCode": strconv.Itoa(response.StatusCode),
		})
		return
	}

	if filtered, ok, filterErr := filterMetadataJSON(body); filterErr == nil && ok {
		body = filtered
	}

	contentType := response.Header.Get("Content-Type")
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	} else {
		w.Header().Set("Content-Type", "application/json")
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func isAllowedMetadataProxyHost(
	host string,
	allow map[string]struct{},
) bool {
	normalized := strings.ToLower(strings.TrimSpace(host))
	if normalized == "" {
		return false
	}

	if _, ok := allow[normalized]; ok {
		return true
	}

	// Irys gateway may redirect to generated subdomains under datasprite CDN.
	if normalized == "mainnet-1.datasprite-cdn.com" ||
		strings.HasSuffix(normalized, ".mainnet-1.datasprite-cdn.com") {
		return true
	}

	return false
}

func (h *MallMeWalletHandler) setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Vary", "Origin")
	w.Header().Set("Access-Control-Allow-Methods", "GET,OPTIONS")
	w.Header().Set(
		"Access-Control-Allow-Headers",
		"Authorization,Content-Type,Accept",
	)
	w.Header().Set("Access-Control-Max-Age", "600")
}

func writeMallMeWalletErr(w http.ResponseWriter, err error) {
	message := "internal_error"
	if err != nil {
		message = err.Error()
	}

	writeJSON(w, mallMeWalletHTTPStatus(err), map[string]string{
		"error": message,
	})
}

func mallMeWalletHTTPStatus(err error) int {
	switch {
	case err == nil:
		return http.StatusInternalServerError

	case errors.Is(err, walletdom.ErrNotFound),
		errors.Is(err, tokendom.ErrNotFound):
		return http.StatusNotFound

	case errors.Is(err, usecase.ErrWalletMintAddressNotOwned):
		return http.StatusForbidden

	case errors.Is(err, usecase.ErrWalletSyncAvatarIDEmpty),
		errors.Is(err, usecase.ErrWalletSyncWalletAddressEmpty),
		errors.Is(err, usecase.ErrMintAddressEmpty),
		errors.Is(err, tokendom.ErrInvalidMintAddress):
		return http.StatusBadRequest

	case errors.Is(err, usecase.ErrWalletSyncOnchainNotConfigured),
		errors.Is(err, usecase.ErrWalletUsecaseNotConfigured),
		errors.Is(err, usecase.ErrWalletTokenQueryNotConfigured),
		errors.Is(err, usecase.ErrWalletProductReaderNotConfigured),
		errors.Is(err, usecase.ErrWalletModelProductBlueprintNotConfigured),
		errors.Is(err, usecase.ErrWalletProductBlueprintReaderNotConfigured):
		return http.StatusServiceUnavailable

	default:
		return http.StatusInternalServerError
	}
}

func isKeepObjectURI(raw string) bool {
	if raw == "" {
		return false
	}

	parsedURL, err := url.Parse(raw)
	if err != nil || parsedURL == nil {
		return strings.Contains(raw, "/.keep") ||
			strings.HasSuffix(raw, ".keep")
	}

	path := parsedURL.Path
	if path == "" {
		return false
	}

	path = strings.TrimSuffix(path, "/")

	return strings.HasSuffix(path, "/.keep") ||
		strings.HasSuffix(path, ".keep")
}

func filterMetadataJSON(body []byte) ([]byte, bool, error) {
	var root map[string]any

	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()

	if err := decoder.Decode(&root); err != nil {
		return nil, false, err
	}

	if len(root) == 0 {
		return body, false, nil
	}

	properties, ok := root["properties"].(map[string]any)
	if !ok || properties == nil {
		return body, false, nil
	}

	files, ok := properties["files"].([]any)
	if !ok || len(files) == 0 {
		return body, false, nil
	}

	filteredFiles := make([]any, 0, len(files))

	for _, item := range files {
		file, ok := item.(map[string]any)
		if !ok || file == nil {
			continue
		}

		uri, _ := file["uri"].(string)
		if uri == "" {
			continue
		}

		if isKeepObjectURI(uri) {
			continue
		}

		filteredFiles = append(filteredFiles, file)
	}

	properties["files"] = filteredFiles
	root["properties"] = properties

	filteredBody, err := json.Marshal(root)
	if err != nil {
		return nil, false, err
	}

	return filteredBody, true, nil
}
