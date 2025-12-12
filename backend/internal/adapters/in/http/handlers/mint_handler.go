// backend/internal/adapters/in/http/handlers/mint_handler.go
package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	usecase "narratives/internal/application/usecase"
	branddom "narratives/internal/domain/brand"
	mintdom "narratives/internal/domain/mint"
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
	// GET /mint/mints?inspectionIds=a,b,c
	//  → inspectionIds（= productionId）に対応する mints をまとめて返す
	//  → 戻り値: map[inspectionId]Mint
	// ------------------------------------------------------------
	case r.Method == http.MethodGet && r.URL.Path == "/mint/mints":
		h.listMintsByInspectionIDs(w, r)
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
// GET /mint/mints?inspectionIds=a,b,c
// ============================================================
//
// return:
//
//	{
//	  "inspectionIdA": { ...mint... },
//	  "inspectionIdB": { ...mint... }
//	}
func (h *MintHandler) listMintsByInspectionIDs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	log.Printf("[mint_handler] listMintsByInspectionIDs called method=%s path=%s rawQuery=%s",
		r.Method, r.URL.Path, r.URL.RawQuery)

	if h.mintUC == nil {
		log.Printf("[mint_handler] listMintsByInspectionIDs FAILED: mintUC is nil")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "mint usecase is not configured"})
		return
	}

	raw := strings.TrimSpace(r.URL.Query().Get("inspectionIds"))
	if raw == "" {
		// 空なら空マップを返す（フロントが扱いやすい）
		_ = json.NewEncoder(w).Encode(map[string]any{})
		return
	}

	parts := strings.Split(raw, ",")
	ids := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, p := range parts {
		s := strings.TrimSpace(p)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		ids = append(ids, s)
	}

	if len(ids) == 0 {
		_ = json.NewEncoder(w).Encode(map[string]any{})
		return
	}

	// ★ ここは MintUsecase 側に実装されている想定（repo.ListByInspectionIDs を内部で呼ぶ）
	mintsByInspectionID, err := h.mintUC.ListMintsByInspectionIDs(ctx, ids)
	if err != nil {
		log.Printf("[mint_handler] listMintsByInspectionIDs FAILED err=%v", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// フロントがそのまま使えるように JSON shape を整える（lower camel）
	out := make(map[string]any, len(mintsByInspectionID))
	for inspectionID, m := range mintsByInspectionID {
		out[inspectionID] = map[string]any{
			"id":               m.ID,
			"brandId":          m.BrandID,
			"tokenBlueprintId": m.TokenBlueprintID,
			"createdBy":        m.CreatedBy,
			"createdAt":        m.CreatedAt,
			"minted":           m.Minted,
			"mintedAt":         m.MintedAt,
			"scheduledBurnDate": func() any {
				if m.ScheduledBurnDate == nil || m.ScheduledBurnDate.IsZero() {
					return nil
				}
				return *m.ScheduledBurnDate
			}(),
			// products はドメインが map の場合でも、フロントは mint existence 判定しかしていないのでそのまま返す
			"products": m.Products,
		}
	}

	log.Printf("[mint_handler] listMintsByInspectionIDs OK count=%d", len(out))
	_ = json.NewEncoder(w).Encode(out)
}

// ============================================================
// POST /mint/requests/{mintRequestId}/mint
// ============================================================
//
// Body はなし。Path から mintRequestId を取り出し、TokenUsecase に委譲して
// チェーンミントを行う。
func (h *MintHandler) mintFromMintRequest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	log.Printf("[mint_handler] mintFromMintRequest called method=%s path=%s", r.Method, r.URL.Path)

	if h.tokenUC == nil {
		log.Printf("[mint_handler] mintFromMintRequest FAILED: tokenUC is nil")
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

	log.Printf("[mint_handler] mintFromMintRequest parsed mintRequestID=%s rawPath=%s", mintRequestID, r.URL.Path)

	if mintRequestID == "" {
		log.Printf("[mint_handler] mintFromMintRequest FAILED: mintRequestId is empty")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "mintRequestId is empty",
		})
		return
	}

	result, err := h.tokenUC.MintFromMintRequest(ctx, mintRequestID)
	if err != nil {
		log.Printf("[mint_handler] mintFromMintRequest FAILED: mintRequestID=%s err=%v", mintRequestID, err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	log.Printf("[mint_handler] mintFromMintRequest OK mintRequestID=%s result=%+v", mintRequestID, result)

	// tokendom.MintResult をそのまま JSON で返す
	_ = json.NewEncoder(w).Encode(result)
}

// ============================================================
// POST /mint/inspections/{productionId}/request
// ============================================================
func (h *MintHandler) updateRequestInfo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	log.Printf("[mint_handler] updateRequestInfo called method=%s path=%s", r.Method, r.URL.Path)

	if h.mintUC == nil {
		log.Printf("[mint_handler] updateRequestInfo FAILED: mintUC is nil")
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

	log.Printf("[mint_handler] updateRequestInfo parsed productionId=%s rawPath=%s", productionID, r.URL.Path)

	if productionID == "" {
		log.Printf("[mint_handler] updateRequestInfo FAILED: productionId is empty")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "productionId is empty",
		})
		return
	}

	// Body parse
	var body struct {
		TokenBlueprintID  string  `json:"tokenBlueprintId"`
		ScheduledBurnDate *string `json:"scheduledBurnDate,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		log.Printf("[mint_handler] updateRequestInfo FAILED: invalid body err=%v", err)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid body",
		})
		return
	}

	log.Printf("[mint_handler] updateRequestInfo body parsed productionId=%s tokenBlueprintId=%s scheduledBurnDate=%v",
		productionID, strings.TrimSpace(body.TokenBlueprintID), body.ScheduledBurnDate)

	tokenBlueprintID := strings.TrimSpace(body.TokenBlueprintID)
	if tokenBlueprintID == "" {
		log.Printf("[mint_handler] updateRequestInfo FAILED: tokenBlueprintId is required productionId=%s", productionID)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "tokenBlueprintId is required",
		})
		return
	}

	log.Printf("[mint_handler] calling MintUsecase.UpdateRequestInfo productionId=%s tokenBlueprintId=%s scheduledBurnDate=%v",
		productionID, tokenBlueprintID, body.ScheduledBurnDate)

	updated, err := h.mintUC.UpdateRequestInfo(
		ctx,
		productionID,
		tokenBlueprintID,
		body.ScheduledBurnDate,
	)
	if err != nil {
		log.Printf("[mint_handler] UpdateRequestInfo FAILED productionId=%s tokenBlueprintId=%s err=%v",
			productionID, tokenBlueprintID, err)

		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	log.Printf("[mint_handler] UpdateRequestInfo OK productionId=%s tokenBlueprintId=%s result=%+v",
		productionID, tokenBlueprintID, updated)

	_ = json.NewEncoder(w).Encode(updated)
}

// ============================================================
// GET /mint/inspections
// ============================================================
//
// ★ 追加:
//   - 取得した inspections に対して /mint/mints と同じロジックで mints を引いて
//     batch.mint を埋め込んで返す（detail 画面が minted モード判定できるように）
func (h *MintHandler) listInspectionsForCurrentCompany(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	log.Printf("[mint_handler] listInspectionsForCurrentCompany called method=%s path=%s", r.Method, r.URL.Path)

	if h.mintUC == nil {
		log.Printf("[mint_handler] listInspectionsForCurrentCompany FAILED: mintUC is nil")
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

		log.Printf("[mint_handler] listInspectionsForCurrentCompany FAILED err=%v status=%d", err, status)

		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	// ============================
	// ★ mints をまとめて取得して batch に埋め込む
	// ============================
	inspectionIDs := make([]string, 0, len(batches))
	for _, b := range batches {
		// productionId を inspectionId として扱う（フロントの requestId も productionId）
		inspectionIDs = append(inspectionIDs, b.ProductionID)
	}

	var mintsByInspectionID map[string]mintdom.Mint
	if len(inspectionIDs) > 0 {
		m, e := h.mintUC.ListMintsByInspectionIDs(ctx, inspectionIDs)
		if e != nil {
			// mints が取れなくても inspections 自体は返す（画面崩壊を避ける）
			log.Printf("[mint_handler] listInspectionsForCurrentCompany WARN: failed to attach mints err=%v", e)
		} else {
			mintsByInspectionID = m
		}
	}

	// batches は struct slice なので、mint を付与するために map 化して返す
	out := make([]any, 0, len(batches))
	for _, b := range batches {
		// struct → map
		var asMap map[string]any
		raw, _ := json.Marshal(b)
		_ = json.Unmarshal(raw, &asMap)

		// mint を付与（存在する場合のみ）
		if mintsByInspectionID != nil {
			if m, ok := mintsByInspectionID[b.ProductionID]; ok {
				asMap["mint"] = map[string]any{
					"id":               m.ID,
					"brandId":          m.BrandID,
					"tokenBlueprintId": m.TokenBlueprintID,
					"createdBy":        m.CreatedBy,
					"createdAt":        m.CreatedAt,
					"minted":           m.Minted,
					"mintedAt":         m.MintedAt,
					"scheduledBurnDate": func() any {
						if m.ScheduledBurnDate == nil || m.ScheduledBurnDate.IsZero() {
							return nil
						}
						return *m.ScheduledBurnDate
					}(),
					"products": m.Products,
				}
			}
		}

		out = append(out, asMap)
	}

	log.Printf("[mint_handler] listInspectionsForCurrentCompany OK count=%d", len(out))
	_ = json.NewEncoder(w).Encode(out)
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

	log.Printf("[mint_handler] getProductBlueprintPatchByID called method=%s path=%s", r.Method, r.URL.Path)

	if h.mintUC == nil {
		log.Printf("[mint_handler] getProductBlueprintPatchByID FAILED: mintUC is nil")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "mint usecase is not configured",
		})
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/mint/product_blueprints/")
	path = strings.TrimSuffix(path, "/patch")
	id := strings.Trim(path, "/")

	log.Printf("[mint_handler] getProductBlueprintPatchByID parsed productBlueprintID=%s", id)

	if id == "" {
		log.Printf("[mint_handler] getProductBlueprintPatchByID FAILED: productBlueprintID is empty")
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

		log.Printf("[mint_handler] getProductBlueprintPatchByID FAILED id=%s err=%v status=%d", id, err, status)

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
				log.Printf("[mint_handler] getProductBlueprintPatchByID ResolveBrandNameByID FAILED brandID=%s err=%v", bid, err)

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

	log.Printf("[mint_handler] getProductBlueprintPatchByID OK id=%s brandName=%s", id, brandName)

	_ = json.NewEncoder(w).Encode(resp)
}

// ============================================================
// GET /mint/brands
// ============================================================
func (h *MintHandler) listBrandsForCurrentCompany(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	log.Printf("[mint_handler] listBrandsForCurrentCompany called method=%s path=%s", r.Method, r.URL.Path)

	if h.mintUC == nil {
		log.Printf("[mint_handler] listBrandsForCurrentCompany FAILED: mintUC is nil")
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

		log.Printf("[mint_handler] listBrandsForCurrentCompany FAILED err=%v status=%d", err, status)

		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	log.Printf("[mint_handler] listBrandsForCurrentCompany OK count=%d", len(result.Items))

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

	log.Printf("[mint_handler] listTokenBlueprintsByBrand called method=%s path=%s rawQuery=%s",
		r.Method, r.URL.Path, r.URL.RawQuery)

	if h.mintUC == nil {
		log.Printf("[mint_handler] listTokenBlueprintsByBrand FAILED: mintUC is nil")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "mint usecase is not configured",
		})
		return
	}

	brandID := strings.TrimSpace(r.URL.Query().Get("brandId"))
	if brandID == "" {
		log.Printf("[mint_handler] listTokenBlueprintsByBrand FAILED: brandId is required")
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

	log.Printf("[mint_handler] listTokenBlueprintsByBrand params brandId=%s page=%d perPage=%d",
		brandID, pageNumber, perPage)

	page := tbdom.Page{
		Number:  pageNumber,
		PerPage: perPage,
	}

	result, err := h.mintUC.ListTokenBlueprintsByBrand(ctx, brandID, page)
	if err != nil {
		log.Printf("[mint_handler] listTokenBlueprintsByBrand FAILED brandId=%s err=%v", brandID, err)

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

	log.Printf("[mint_handler] listTokenBlueprintsByBrand OK brandId=%s count=%d", brandID, len(items))

	_ = json.NewEncoder(w).Encode(items)
}
