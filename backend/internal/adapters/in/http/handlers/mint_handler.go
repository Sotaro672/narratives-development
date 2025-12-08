// backend/internal/adapters/in/http/handlers/mint_handler.go
package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	usecase "narratives/internal/application/usecase"
	branddom "narratives/internal/domain/brand"
	pbpdom "narratives/internal/domain/productBlueprint"
	tbdom "narratives/internal/domain/tokenBlueprint"
	solana "narratives/internal/infra/solana"
)

type MintHandler struct {
	// ミント候補一覧やパッチ取得など「事前準備」系
	mintUC *usecase.MintUsecase
	// 実際のチェーンミントを行う Usecase（Solana + MintRequestPort）
	tokenUC *usecase.TokenUsecase
}

func NewMintHandler(mintUC *usecase.MintUsecase, tokenUC *usecase.TokenUsecase) http.Handler {
	return &MintHandler{
		mintUC:  mintUC,
		tokenUC: tokenUC,
	}
}

// デバッグ用エンドポイント /mint/debug で使用
func (h *MintHandler) HandleDebug(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"ok": true, "msg": "Mint API alive"}`)
}

func (h *MintHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {

	// ------------------------------------------------------------
	// POST /mint/debug/nft
	//  → Solana Devnet で Metaplex NFT ミントをテストするためのデバッグ用エンドポイント
	// Body:
	// {
	//   "walletAddress": "受取ウォレット(base58)",
	//   "name": "任意 (省略可)",
	//   "symbol": "任意 (省略可)",
	//   "uri": "任意 (省略可)"
	// }
	// ------------------------------------------------------------
	case r.Method == http.MethodPost && r.URL.Path == "/mint/debug/nft":
		h.debugMintNFT(w, r)
		return

	// ------------------------------------------------------------
	// 任意: GET /mint/debug で生存確認
	// ------------------------------------------------------------
	case r.Method == http.MethodGet && r.URL.Path == "/mint/debug":
		h.HandleDebug(w, r)
		return

	// ------------------------------------------------------------
	// POST /mint/requests/{mintRequestId}/mint
	//  → TokenUsecase を使ってチェーン上でミント実行
	// ------------------------------------------------------------
	case r.Method == http.MethodPost &&
		strings.HasPrefix(r.URL.Path, "/mint/requests/") &&
		strings.HasSuffix(r.URL.Path, "/mint"):
		h.mintFromMintRequest(w, r)
		return

	// ------------------------------------------------------------
	// POST /mint/inspections/{productionId}/request
	//  → 検品結果から MintRequest 情報を更新（ミント候補を作る）
	// ------------------------------------------------------------
	case r.Method == http.MethodPost &&
		strings.HasPrefix(r.URL.Path, "/mint/inspections/") &&
		strings.HasSuffix(r.URL.Path, "/request"):
		h.updateRequestInfo(w, r)
		return

	// ------------------------------------------------------------
	// GET /mint/inspections
	// ------------------------------------------------------------
	case r.Method == http.MethodGet &&
		(r.URL.Path == "/mint/inspections" || strings.HasPrefix(r.URL.Path, "/mint/inspections/")):
		h.listInspectionsForCurrentCompany(w, r)
		return

	// ------------------------------------------------------------
	// GET /mint/product_blueprints/{id}/patch
	// ------------------------------------------------------------
	case r.Method == http.MethodGet &&
		strings.HasPrefix(r.URL.Path, "/mint/product_blueprints/") &&
		strings.HasSuffix(r.URL.Path, "/patch"):
		h.getProductBlueprintPatchByID(w, r)
		return

	// ------------------------------------------------------------
	// GET /mint/brands
	// ------------------------------------------------------------
	case r.Method == http.MethodGet && r.URL.Path == "/mint/brands":
		h.listBrandsForCurrentCompany(w, r)
		return

	// ------------------------------------------------------------
	// GET /mint/token_blueprints?brandId=...
	// ------------------------------------------------------------
	case r.Method == http.MethodGet && r.URL.Path == "/mint/token_blueprints":
		h.listTokenBlueprintsByBrand(w, r)
		return

	default:
		http.NotFound(w, r)
	}
}

// ============================================================
// デバッグ用: POST /mint/debug/nft
// ============================================================
//
// Solana Devnet に対して直接 Metaplex NFT を 1 枚ミントする。
// - Usecase には依存せず、platform/solana の MintNFTToOwner を直呼び出し。
// - 開発環境での疎通確認専用。
func (h *MintHandler) debugMintNFT(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var body struct {
		WalletAddress string `json:"walletAddress"`
		Name          string `json:"name"`
		Symbol        string `json:"symbol"`
		URI           string `json:"uri"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid body",
		})
		return
	}

	wallet := strings.TrimSpace(body.WalletAddress)
	if wallet == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "walletAddress is required",
		})
		return
	}

	// デフォルト値（省略時）
	name := strings.TrimSpace(body.Name)
	if name == "" {
		name = "Narratives Debug NFT"
	}
	symbol := strings.TrimSpace(body.Symbol)
	if symbol == "" {
		symbol = "NDEBUG"
	}
	uri := strings.TrimSpace(body.URI)
	if uri == "" {
		// ひとまずダミー（将来的には GCS/Arweave の metadata.json に置き換える）
		uri = "https://example.com/narratives/debug-metadata.json"
	}

	meta := solana.NFTMetadataInput{
		Name:                 name,
		Symbol:               symbol,
		URI:                  uri,
		SellerFeeBasisPoints: 500, // 5%
	}

	mintAddr, sig, err := solana.MintNFTToOwner(ctx, wallet, meta)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]string{
		"mintAddress":  mintAddr,
		"txSignature":  sig,
		"cluster":      "devnet",
		"explorerLink": fmt.Sprintf("https://explorer.solana.com/tx/%s?cluster=devnet", sig),
	})
}

// ============================================================
// POST /mint/requests/{mintRequestId}/mint
// ============================================================
//
// Body はなし。Path から mintRequestId を取り出し、TokenUsecase に委譲して
// チェーンミントを行う。
func (h *MintHandler) mintFromMintRequest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.tokenUC == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "token usecase is not configured",
		})
		return
	}

	// /mint/requests/{id}/mint から {id} を抽出
	path := strings.TrimPrefix(r.URL.Path, "/mint/requests/")
	path = strings.TrimSuffix(path, "/mint")
	mintRequestID := strings.Trim(path, "/")

	if mintRequestID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "mintRequestId is empty",
		})
		return
	}

	result, err := h.tokenUC.MintFromMintRequest(ctx, mintRequestID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	// tokendom.MintResult をそのまま JSON で返す
	_ = json.NewEncoder(w).Encode(result)
}

// ============================================================
// POST /mint/inspections/{productionId}/request
// ============================================================
// Body: { "tokenBlueprintId": "xxxx" }
func (h *MintHandler) updateRequestInfo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.mintUC == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "mint usecase is not configured",
		})
		return
	}

	// productionId 抽出
	path := strings.TrimPrefix(r.URL.Path, "/mint/inspections/")
	path = strings.TrimSuffix(path, "/request")
	productionID := strings.TrimSpace(path)

	if productionID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "productionId is empty",
		})
		return
	}

	// Body parse
	var body struct {
		TokenBlueprintID string `json:"tokenBlueprintId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid body",
		})
		return
	}

	tokenBlueprintID := strings.TrimSpace(body.TokenBlueprintID)
	if tokenBlueprintID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "tokenBlueprintId is required",
		})
		return
	}

	// ★ Usecase 側で MemberIDFromContext(ctx) を参照するため requestedBy は渡さない
	updated, err := h.mintUC.UpdateRequestInfo(ctx, productionID, tokenBlueprintID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(updated)
}

// ============================================================
// GET /mint/inspections
// ============================================================
func (h *MintHandler) listInspectionsForCurrentCompany(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.mintUC == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "mint usecase is not configured",
		})
		return
	}

	batches, err := h.mintUC.ListInspectionsForCurrentCompany(ctx)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, usecase.ErrCompanyIDMissing) {
			status = http.StatusBadRequest
		}

		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(batches)
}

// ============================================================
// GET /mint/product_blueprints/{id}/patch
// ============================================================
type productBlueprintPatchResponse struct {
	pbpdom.Patch
	BrandName string `json:"brandName"`
}

func (h *MintHandler) getProductBlueprintPatchByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.mintUC == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "mint usecase is not configured",
		})
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/mint/product_blueprints/")
	path = strings.TrimSuffix(path, "/patch")
	id := strings.Trim(path, "/")

	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "productBlueprintID is empty",
		})
		return
	}

	patch, err := h.mintUC.GetProductBlueprintPatchByID(ctx, id)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, pbpdom.ErrNotFound) {
			status = http.StatusNotFound
		}

		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	brandName := ""
	if patch.BrandID != nil {
		bid := strings.TrimSpace(*patch.BrandID)
		if bid != "" {
			name, err := h.mintUC.ResolveBrandNameByID(ctx, bid)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": err.Error(),
				})
				return
			}
			brandName = name
		}
	}

	resp := productBlueprintPatchResponse{
		Patch:     patch,
		BrandName: brandName,
	}

	_ = json.NewEncoder(w).Encode(resp)
}

// ============================================================
// GET /mint/brands
// ============================================================
func (h *MintHandler) listBrandsForCurrentCompany(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.mintUC == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "mint usecase is not configured",
		})
		return
	}

	var page branddom.Page

	result, err := h.mintUC.ListBrandsForCurrentCompany(ctx, page)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, usecase.ErrCompanyIDMissing) {
			status = http.StatusBadRequest
		}

		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(result)
}

// ============================================================
// GET /mint/token_blueprints
// ============================================================
type tokenBlueprintForMintResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Symbol  string `json:"symbol"`
	IconURL string `json:"iconUrl"`
}

func (h *MintHandler) listTokenBlueprintsByBrand(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.mintUC == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "mint usecase is not configured",
		})
		return
	}

	brandID := strings.TrimSpace(r.URL.Query().Get("brandId"))
	if brandID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "brandId is required",
		})
		return
	}

	pageParam := strings.TrimSpace(r.URL.Query().Get("page"))
	perPageParam := strings.TrimSpace(r.URL.Query().Get("perPage"))

	pageNumber := 1
	perPage := 100

	if pageParam != "" {
		if n, err := strconv.Atoi(pageParam); err == nil && n > 0 {
			pageNumber = n
		}
	}
	if perPageParam != "" {
		if n, err := strconv.Atoi(perPageParam); err == nil && n > 0 {
			perPage = n
		}
	}

	page := tbdom.Page{
		Number:  pageNumber,
		PerPage: perPage,
	}

	result, err := h.mintUC.ListTokenBlueprintsByBrand(ctx, brandID, page)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	items := make([]tokenBlueprintForMintResponse, 0, len(result.Items))
	for _, tb := range result.Items {
		items = append(items, tokenBlueprintForMintResponse{
			ID:      tb.ID,
			Name:    tb.Name,
			Symbol:  tb.Symbol,
			IconURL: tb.IconURL,
		})
	}

	_ = json.NewEncoder(w).Encode(items)
}
