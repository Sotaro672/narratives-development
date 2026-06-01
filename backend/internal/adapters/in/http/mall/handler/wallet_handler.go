// backend/internal/adapters/in/http/mall/handler/wallet_handler.go
package mallHandler

import (
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

	usecase "narratives/internal/application/usecase"
	tokendom "narratives/internal/domain/token"
	walletdom "narratives/internal/domain/wallet"
)

type WalletHandler struct {
	uc *usecase.WalletUsecase

	// resolved token cache (Firestore wallets/{avatarId}/resolvedTokens/{mint})
	resolvedTokenRepo ResolvedTokenRepository

	// optional: allowlist for proxy host validation
	// if empty, defaults are used
	allowedProxyHosts map[string]struct{}
}

func NewWalletHandler(
	walletUC *usecase.WalletUsecase,
	resolvedTokenRepo ResolvedTokenRepository,
) http.Handler {
	return &WalletHandler{
		uc:                walletUC,
		resolvedTokenRepo: resolvedTokenRepo,
		allowedProxyHosts: map[string]struct{}{
			"gateway.irys.xyz":    {},
			"uploader.irys.xyz":   {},
			"arweave.net":         {},
			"www.arweave.net":     {},
			"ipfs.io":             {},
			"cloudflare-ipfs.com": {},
		},
	}
}

func (h *WalletHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	path0 := strings.TrimSuffix(r.URL.Path, "/")

	switch {
	case r.Method == http.MethodOptions && path0 == "/mall/wallets/metadata/proxy":
		h.preflightWalletMetadataProxy(w)
		return

	case r.Method == http.MethodGet && path0 == "/mall/wallets":
		h.get(w, r)
		return

	case r.Method == http.MethodGet && path0 == "/mall/wallets/tokens/resolve":
		h.resolveTokenByMintAddress(w, r)
		return

	case r.Method == http.MethodGet && path0 == "/mall/wallets/metadata/proxy":
		h.walletMetadataProxy(w, r)
		return

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}
}

// GET /mall/wallets
// query:
// - avatarId=...
func (h *WalletHandler) get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h == nil || h.uc == nil || h.uc.WalletRepo == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "wallet usecase not configured",
		})
		return
	}

	avatarID := r.URL.Query().Get("avatarId")
	if avatarID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "avatarId is required",
		})
		return
	}

	wallet, err := h.uc.WalletRepo.GetByAvatarID(ctx, avatarID)
	if err != nil {
		writeWalletErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"wallets": []walletdom.Wallet{wallet},
	})
}

// GET /mall/wallets/tokens/resolve?mintAddress=...
// GET /mall/wallets/tokens/resolve?avatarId=...&mintAddress=...
//
// avatarId がある場合:
// - 所有確認あり
//
// avatarId がない場合:
// - 公開情報のみ返す
func (h *WalletHandler) resolveTokenByMintAddress(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "wallet usecase not configured"})
		return
	}

	mintAddress := r.URL.Query().Get("mintAddress")
	if mintAddress == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "mintAddress is required"})
		return
	}

	avatarID := r.URL.Query().Get("avatarId")

	if avatarID == "" {
		res, err := h.resolvePublicTokenSummary(ctx, mintAddress)
		if err != nil {
			writeWalletErr(w, err)
			return
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
		return
	}

	if h.uc.WalletRepo == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "wallet repository not configured"})
		return
	}

	snap, err := h.uc.WalletRepo.GetByAvatarID(ctx, avatarID)
	if err != nil {
		writeWalletErr(w, err)
		return
	}

	owned := walletSnapshotHasMintPreferTokens(snap, mintAddress)

	if !owned && h.uc.OnchainReader != nil {
		addr := snap.WalletAddress
		if addr != "" {
			mints, e := h.uc.OnchainReader.ListOwnedTokenMints(ctx, addr)
			if e == nil {
				owned = stringSliceContainsExact(mints, mintAddress)
			}
		}
	}

	if !owned {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "mintAddress is not owned by avatarId"})
		return
	}

	var res usecase.ResolveTokenByMintAddressWithBrandNameResult
	var fromCache bool

	if !fromCache {
		rr, e := h.uc.ResolveTokenByMintAddressWithBrandName(ctx, mintAddress)
		if e != nil {
			writeWalletErr(w, e)
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

func (h *WalletHandler) resolvePublicTokenSummary(
	ctx context.Context,
	mintAddress string,
) (usecase.ResolveTokenByMintAddressWithBrandNameResult, error) {
	if h == nil || h.uc == nil {
		return usecase.ResolveTokenByMintAddressWithBrandNameResult{}, usecase.ErrWalletUsecaseNotConfigured
	}

	base, err := h.uc.ResolveTokenByMintAddress(ctx, mintAddress)
	if err != nil {
		return usecase.ResolveTokenByMintAddressWithBrandNameResult{}, err
	}

	productID := base.ProductID
	brandID := base.BrandID

	brandName := ""
	if brandID != "" {
		if h.uc.BrandResolver == nil {
			return usecase.ResolveTokenByMintAddressWithBrandNameResult{}, usecase.ErrWalletBrandResolverNotConfigured
		}

		n, err := h.uc.ResolveBrandNameByID(ctx, brandID)
		if err != nil {
			return usecase.ResolveTokenByMintAddressWithBrandNameResult{}, err
		}
		brandName = n
	}

	if h.uc.ProductReader == nil {
		return usecase.ResolveTokenByMintAddressWithBrandNameResult{}, usecase.ErrWalletProductReaderNotConfigured
	}
	p, err := h.uc.ProductReader.GetByID(ctx, productID)
	if err != nil {
		return usecase.ResolveTokenByMintAddressWithBrandNameResult{}, err
	}

	modelID := p.ModelID
	if modelID == "" {
		return usecase.ResolveTokenByMintAddressWithBrandNameResult{}, usecase.ErrWalletResolvedModelIDEmpty
	}

	if h.uc.ModelProductBlueprintID == nil {
		return usecase.ResolveTokenByMintAddressWithBrandNameResult{}, usecase.ErrWalletModelProductBlueprintNotConfigured
	}
	pbID, _, err := h.uc.ModelProductBlueprintID.GetIDByModelID(ctx, modelID)
	if err != nil {
		return usecase.ResolveTokenByMintAddressWithBrandNameResult{}, err
	}
	if pbID == "" {
		return usecase.ResolveTokenByMintAddressWithBrandNameResult{}, usecase.ErrWalletResolvedProductBlueprintIDEmpty
	}

	if h.uc.ProductBlueprintReader == nil {
		return usecase.ResolveTokenByMintAddressWithBrandNameResult{}, usecase.ErrWalletProductBlueprintReaderNotConfigured
	}
	pb, err := h.uc.ProductBlueprintReader.GetByID(ctx, pbID)
	if err != nil {
		return usecase.ResolveTokenByMintAddressWithBrandNameResult{}, err
	}

	return usecase.ResolveTokenByMintAddressWithBrandNameResult{
		ProductID:          productID,
		BrandID:            brandID,
		BrandName:          brandName,
		MetadataURI:        base.MetadataURI,
		MintAddress:        base.MintAddress,
		ProductBlueprintID: pbID,
		ProductName:        pb.ProductName,
	}, nil
}

// OPTIONS /mall/wallets/metadata/proxy
func (h *WalletHandler) preflightWalletMetadataProxy(w http.ResponseWriter) {
	h.setCORSHeaders(w)
	w.WriteHeader(http.StatusNoContent)
}

// GET /mall/wallets/metadata/proxy?url=https://...
//
// metadata 自体は公開情報取得用途でも使うため avatarId 必須にしない。
func (h *WalletHandler) walletMetadataProxy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "wallet usecase not configured"})
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
			"gateway.irys.xyz":    {},
			"uploader.irys.xyz":   {},
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
			if _, ok := allow[hh]; !ok {
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

func (h *WalletHandler) setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Vary", "Origin")
	w.Header().Set("Access-Control-Allow-Methods", "GET,OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Authorization,Content-Type,Accept")
	w.Header().Set("Access-Control-Max-Age", "600")
}

func writeWalletErr(w http.ResponseWriter, err error) {
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
		errors.Is(err, usecase.ErrWalletBrandResolverNotConfigured),
		errors.Is(err, usecase.ErrWalletProductReaderNotConfigured),
		errors.Is(err, usecase.ErrWalletModelProductBlueprintNotConfigured),
		errors.Is(err, usecase.ErrWalletProductBlueprintReaderNotConfigured):
		code = http.StatusServiceUnavailable
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
