// backend/internal/adapters/in/http/handlers/mint_handler.go
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	usecase "narratives/internal/application/usecase"
	branddom "narratives/internal/domain/brand"
	pbpdom "narratives/internal/domain/productBlueprint"
	tbdom "narratives/internal/domain/tokenBlueprint"
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
	_, _ = w.Write([]byte(`{"ok": true, "msg": "Mint API alive"}`))
}

func (h *MintHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {

	// ------------------------------------------------------------
	// 任意: GET /mint/debug で生存確認
	// ------------------------------------------------------------
	case r.Method == http.MethodGet && r.URL.Path == "/mint/debug":
		h.HandleDebug(w, r)
		return

	// ------------------------------------------------------------
	// POST /mint/requests/{mintRequestId}/mint
	//  → TokenUsecase を使ってチェーン上でミント実行
	//    （※ mints テーブル作成がゴールの場合、実際に呼ばなければよい）
	// ------------------------------------------------------------
	case r.Method == http.MethodPost &&
		strings.HasPrefix(r.URL.Path, "/mint/requests/") &&
		strings.HasSuffix(r.URL.Path, "/mint"):
		h.mintFromMintRequest(w, r)
		return

	// ------------------------------------------------------------
	// POST /mint/inspections/{productionId}/request
	//  → 検品結果から MintRequest 情報を更新
	//     ＋ MintUsecase 側で mints テーブルのレコードを作成
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
//
// Body:
//
//	{
//	  "tokenBlueprintId": "xxxx",
//	  "scheduledBurnDate": "2025-12-31" // 任意（HTML date input の "YYYY-MM-DD" を想定）
//	}
//
// MintUsecase.UpdateRequestInfo(ctx, productionID, tokenBlueprintID, scheduledBurnDate)
// が内部で inspections の RequestInfo 更新と mints テーブル作成を行う想定。
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
		TokenBlueprintID  string  `json:"tokenBlueprintId"`
		ScheduledBurnDate *string `json:"scheduledBurnDate,omitempty"` // ★ 任意の焼却予定日
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

	// ポインタ(*string) → string へ変換（未指定なら空文字）
	scheduledBurnDate := ""
	if body.ScheduledBurnDate != nil {
		scheduledBurnDate = strings.TrimSpace(*body.ScheduledBurnDate)
	}

	// ★ Usecase 側で MemberIDFromContext(ctx) を参照するため requestedBy は渡さない
	updated, err := h.mintUC.UpdateRequestInfo(
		ctx,
		productionID,
		tokenBlueprintID,
		scheduledBurnDate, // 焼却予定日（空文字なら未指定扱い）
	)
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
