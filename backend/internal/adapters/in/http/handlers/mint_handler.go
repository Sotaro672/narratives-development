// backend/internal/adapters/in/http/handlers/mint_handler.go
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	resolver "narratives/internal/application/resolver"

	mintapp "narratives/internal/application/mint"
	mintdto "narratives/internal/application/mint/dto"
	mintpresenter "narratives/internal/application/mint/presenter"

	// ★ productionIds 自動解決用
	productionapp "narratives/internal/application/production"

	// ★ NEW: mintRequest 一覧の Query（productionId -> inspection + mint）
	querydto "narratives/internal/application/query/dto"

	usecase "narratives/internal/application/usecase"
	branddom "narratives/internal/domain/brand"
	inspectiondom "narratives/internal/domain/inspection"
	mintdom "narratives/internal/domain/mint"
	pbpdom "narratives/internal/domain/productBlueprint"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

// ★ NEW: handler が依存する最小 query IF
// - 実装は backend/internal/application/query/* に置く想定
type MintRequestQueryService interface {
	// company 境界付きで、productionId と同 docId の inspection/mint を束ねた DTO を返す
	// NOTE: requestedBy は mint.CreatedBy に合わせる（DTO 側で担保）
	ListMintRequestManagementRows(ctx context.Context) ([]querydto.ProductionInspectionMintDTO, error)
}

type MintHandler struct {
	mintUC       *mintapp.MintUsecase
	tokenUC      *usecase.TokenUsecase
	nameResolver *resolver.NameResolver

	// ★ /mint/inspections に productionIds が来ない場合に productions から自動生成する
	productionUC *productionapp.ProductionUsecase

	// ★ NEW: /mint/requests 用 Query
	mintRequestQS MintRequestQueryService
}

func NewMintHandler(
	mintUC *mintapp.MintUsecase,
	tokenUC *usecase.TokenUsecase,
	nameResolver *resolver.NameResolver,
	productionUC *productionapp.ProductionUsecase,
	mintRequestQS MintRequestQueryService, // ★ 追加
) http.Handler {
	return &MintHandler{
		mintUC:        mintUC,
		tokenUC:       tokenUC,
		nameResolver:  nameResolver,
		productionUC:  productionUC,
		mintRequestQS: mintRequestQS,
	}
}

func (h *MintHandler) HandleDebug(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"ok": true, "msg": "Mint API alive"}`))
}

func (h *MintHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	log.Printf("[mint_handler] request method=%s path=%s rawQuery=%q", r.Method, r.URL.Path, r.URL.RawQuery)

	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/mint/debug":
		h.HandleDebug(w, r)
		return

	// ★ NEW: GET /mint/requests（mintRequest 管理一覧を 1shot で返す）
	case r.Method == http.MethodGet && r.URL.Path == "/mint/requests":
		h.listMintRequestsByCurrentCompany(w, r)
		return

	// GET /mint/inspections?productionIds=a,b,c
	case r.Method == http.MethodGet && r.URL.Path == "/mint/inspections":
		h.listInspectionsByProductionIDs(w, r)
		return

	// ★ NEW: GET /mint/inspections/{productionId}
	// - detail 用（1件返す）
	case r.Method == http.MethodGet &&
		strings.HasPrefix(r.URL.Path, "/mint/inspections/") &&
		!strings.HasSuffix(r.URL.Path, "/request"):
		h.getMintRequestDetailByProductionID(w, r)
		return

	// GET /mint/mints?inspectionIds=a,b,c(&view=list|dto)
	case r.Method == http.MethodGet && r.URL.Path == "/mint/mints":
		h.listMintsByInspectionIDs(w, r)
		return

	// GET /mint/mints/{id}
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/mint/mints/"):
		h.getMintByID(w, r)
		return

	// POST /mint/requests/{mintRequestId}/mint
	case r.Method == http.MethodPost &&
		strings.HasPrefix(r.URL.Path, "/mint/requests/") &&
		strings.HasSuffix(r.URL.Path, "/mint"):
		h.mintFromMintRequest(w, r)
		return

	// POST /mint/inspections/{productionId}/request
	case r.Method == http.MethodPost &&
		strings.HasPrefix(r.URL.Path, "/mint/inspections/") &&
		strings.HasSuffix(r.URL.Path, "/request"):
		h.updateRequestInfo(w, r)
		return

	// GET /mint/product_blueprints/{id}/patch
	case r.Method == http.MethodGet &&
		strings.HasPrefix(r.URL.Path, "/mint/product_blueprints/") &&
		strings.HasSuffix(r.URL.Path, "/patch"):
		h.getProductBlueprintPatchByID(w, r)
		return

	// GET /mint/brands
	case r.Method == http.MethodGet && r.URL.Path == "/mint/brands":
		h.listBrandsForCurrentCompany(w, r)
		return

	// GET /mint/token_blueprints?brandId=...
	case r.Method == http.MethodGet && r.URL.Path == "/mint/token_blueprints":
		h.listTokenBlueprintsByBrand(w, r)
		return

	default:
		http.NotFound(w, r)
	}
}

// ============================================================
// ★ NEW: GET /mint/inspections/{productionId}
// - detail 用: MintUsecase.GetMintRequestDetail を呼ぶ
// ============================================================
func (h *MintHandler) getMintRequestDetailByProductionID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.mintUC == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "mint usecase is not configured"})
		return
	}

	// /mint/inspections/{productionId}
	path := strings.TrimPrefix(r.URL.Path, "/mint/inspections/")
	path = strings.Trim(path, "/")
	if path == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "productionId is empty"})
		return
	}
	// 余計なセグメントを弾く（/mint/inspections/{id}/xxx など）
	if strings.Contains(path, "/") {
		http.NotFound(w, r)
		return
	}
	productionID := strings.TrimSpace(path)

	log.Printf("[mint_handler] /mint/inspections/{productionId} start productionId=%q", productionID)

	start := time.Now()
	detail, err := h.mintUC.GetMintRequestDetail(ctx, productionID)
	elapsed := time.Since(start)

	if err != nil {
		if errors.Is(err, mintapp.ErrCompanyIDMissing) {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "companyId is missing"})
			return
		}
		if errors.Is(err, inspectiondom.ErrNotFound) || errors.Is(err, mintdom.ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "mint request detail not found"})
			return
		}

		log.Printf("[mint_handler] /mint/inspections/{productionId} error=%v elapsed=%s", err, elapsed)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	log.Printf(
		"[mint_handler] /mint/inspections/{productionId} ok elapsed=%s detail=%s",
		elapsed,
		toJSONForLog(detail, 2000),
	)

	_ = json.NewEncoder(w).Encode(detail)
}

// ============================================================
// ★ NEW: GET /mint/requests
// - company 境界付き Query を呼び、mintRequest 管理画面用の rows を返す
// - optional: ?productionIds=a,b,c でサーバ側フィルタ
// - optional: ?view=management|list（現状どちらでも同じ rows を返す）
// ============================================================
func (h *MintHandler) listMintRequestsByCurrentCompany(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.mintRequestQS == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "mintRequest query service is not configured"})
		return
	}

	view := strings.TrimSpace(r.URL.Query().Get("view"))
	if view == "" {
		view = "management"
	}

	rawProductionIDs := strings.TrimSpace(r.URL.Query().Get("productionIds"))
	filterSet := map[string]struct{}{}
	if rawProductionIDs != "" {
		parts := strings.Split(rawProductionIDs, ",")
		for _, p := range parts {
			id := strings.TrimSpace(p)
			if id == "" {
				continue
			}
			filterSet[id] = struct{}{}
		}
	}

	log.Printf(
		"[mint_handler] /mint/requests query view=%q rawProductionIds=%q filterCount=%d",
		view, rawProductionIDs, len(filterSet),
	)

	start := time.Now()
	rows, err := h.mintRequestQS.ListMintRequestManagementRows(ctx)
	elapsed := time.Since(start)

	if err != nil {
		// companyId なしは 400 に寄せてフロントで判定しやすくする
		if errors.Is(err, mintapp.ErrCompanyIDMissing) {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "companyId is missing"})
			return
		}

		log.Printf("[mint_handler] /mint/requests query error=%v elapsed=%s", err, elapsed)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// optional server-side filter
	if len(filterSet) > 0 {
		filtered := make([]querydto.ProductionInspectionMintDTO, 0, len(rows))
		for _, row := range rows {
			pid := strings.TrimSpace(row.ProductionID)
			if pid == "" {
				pid = strings.TrimSpace(row.ID)
			}
			if pid == "" {
				continue
			}
			if _, ok := filterSet[pid]; ok {
				filtered = append(filtered, row)
			}
		}
		rows = filtered
	}

	log.Printf(
		"[mint_handler] /mint/requests result rows len=%d elapsed=%s sampleRow[0]=%s",
		len(rows),
		elapsed,
		toJSONForLog(sampleFirst(rows), 1500),
	)

	_ = json.NewEncoder(w).Encode(rows)
}

// ============================================================
// GET /mint/inspections?productionIds=a,b,c
// ============================================================
func (h *MintHandler) listInspectionsByProductionIDs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.mintUC == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "mint usecase is not configured"})
		return
	}

	rawProductionIDs := strings.TrimSpace(r.URL.Query().Get("productionIds"))
	rawInspectionIDs := strings.TrimSpace(r.URL.Query().Get("inspectionIds"))

	raw := rawProductionIDs
	if raw == "" {
		raw = rawInspectionIDs
	}

	log.Printf(
		"[mint_handler] /mint/inspections query rawProductionIds=%q rawInspectionIds=%q chosenRaw=%q",
		rawProductionIDs, rawInspectionIDs, raw,
	)

	var ids []string

	if raw == "" {
		if h.productionUC == nil {
			log.Printf("[mint_handler] /mint/inspections productionIds is empty AND productionUC is nil -> return []")
			_ = json.NewEncoder(w).Encode([]any{})
			return
		}

		prods, err := h.productionUC.ListWithAssigneeName(ctx)
		if err != nil {
			log.Printf("[mint_handler] /mint/inspections auto productionIds resolve error (ListWithAssigneeName) err=%v", err)
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		seen := make(map[string]struct{}, len(prods))
		ids = make([]string, 0, len(prods))
		for _, p := range prods {
			pid := strings.TrimSpace(p.ID)
			if pid == "" {
				continue
			}
			if _, ok := seen[pid]; ok {
				continue
			}
			seen[pid] = struct{}{}
			ids = append(ids, pid)
		}
		sort.Strings(ids)

		log.Printf(
			"[mint_handler] /mint/inspections auto productionIds from /productions len=%d sample[0..4]=%v sampleProd[0]=%s",
			len(ids),
			ids[:min(5, len(ids))],
			toJSONForLog(sampleFirst(prods), 1500),
		)

		if len(ids) == 0 {
			log.Printf("[mint_handler] /mint/inspections auto productionIds resolved but EMPTY -> return []")
			_ = json.NewEncoder(w).Encode([]any{})
			return
		}

	} else {
		parts := strings.Split(raw, ",")
		seen := make(map[string]struct{}, len(parts))
		ids = make([]string, 0, len(parts))
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
			log.Printf("[mint_handler] /mint/inspections productionIds parsed empty (raw=%q) -> return []", raw)
			_ = json.NewEncoder(w).Encode([]any{})
			return
		}

		sort.Strings(ids)

		log.Printf(
			"[mint_handler] /mint/inspections productionIds parsed len=%d sample[0..4]=%v",
			len(ids), ids[:min(5, len(ids))],
		)
	}

	start := time.Now()
	batches, err := h.mintUC.ListInspectionBatchesByProductionIDs(ctx, ids)
	elapsed := time.Since(start)

	if err != nil {
		log.Printf("[mint_handler] /mint/inspections ListInspectionBatchesByProductionIDs error=%v elapsed=%s", err, elapsed)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	log.Printf(
		"[mint_handler] /mint/inspections result batches len=%d elapsed=%s sampleBatch[0]=%s",
		len(batches),
		elapsed,
		toJSONForLog(sampleFirst(batches), 1500),
	)

	_ = json.NewEncoder(w).Encode(batches)
}

// ============================================================
// GET /mint/mints?inspectionIds=a,b,c
// ============================================================
func (h *MintHandler) listMintsByInspectionIDs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.mintUC == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "mint usecase is not configured"})
		return
	}

	raw := strings.TrimSpace(r.URL.Query().Get("inspectionIds"))
	if raw == "" {
		_ = json.NewEncoder(w).Encode(map[string]any{})
		return
	}

	view := strings.TrimSpace(r.URL.Query().Get("view"))
	if view == "" {
		view = "list"
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

	sort.Strings(ids)

	log.Printf("[mint_handler] /mint/mints inspectionIds len=%d sample[0..4]=%v view=%s", len(ids), ids[:min(5, len(ids))], view)

	start := time.Now()
	mintsByInspectionID, err := h.mintUC.ListMintsByInspectionIDs(ctx, ids)
	elapsed := time.Since(start)

	if err != nil {
		log.Printf("[mint_handler] /mint/mints ListMintsByInspectionIDs error=%v elapsed=%s", err, elapsed)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	log.Printf(
		"[mint_handler] /mint/mints result keys=%d elapsed=%s sampleKey=%q sampleVal=%s",
		len(mintsByInspectionID),
		elapsed,
		sampleFirstKey(mintsByInspectionID),
		toJSONForLog(sampleFirstValue(mintsByInspectionID), 1500),
	)

	if view == "dto" {
		out := make(map[string]any, len(mintsByInspectionID))
		for inspectionID, m := range mintsByInspectionID {
			iid := strings.TrimSpace(inspectionID)

			products := make([]string, 0, len(m.Products))
			for pid := range m.Products {
				p := strings.TrimSpace(pid)
				if p != "" {
					products = append(products, p)
				}
			}
			sort.Strings(products)

			var createdAt *string
			if !m.CreatedAt.IsZero() {
				s := m.CreatedAt.UTC().Format(time.RFC3339)
				createdAt = &s
			}

			var mintedAt *string
			if m.MintedAt != nil && !m.MintedAt.IsZero() {
				s := m.MintedAt.UTC().Format(time.RFC3339)
				mintedAt = &s
			}

			var scheduledBurnDate *string
			if m.ScheduledBurnDate != nil && !m.ScheduledBurnDate.IsZero() {
				s := m.ScheduledBurnDate.UTC().Format(time.RFC3339)
				scheduledBurnDate = &s
			}

			out[iid] = map[string]any{
				"id":                strings.TrimSpace(m.ID),
				"inspectionId":      iid,
				"brandId":           strings.TrimSpace(m.BrandID),
				"tokenBlueprintId":  strings.TrimSpace(m.TokenBlueprintID),
				"products":          products,
				"createdBy":         strings.TrimSpace(m.CreatedBy),
				"createdAt":         createdAt,
				"minted":            m.Minted,
				"mintedAt":          mintedAt,
				"scheduledBurnDate": scheduledBurnDate,
			}
		}
		_ = json.NewEncoder(w).Encode(out)
		return
	}

	out := make(map[string]mintdto.MintListRowDTO, len(mintsByInspectionID))
	for inspectionID, m := range mintsByInspectionID {
		iid := strings.TrimSpace(inspectionID)

		tbID := strings.TrimSpace(m.TokenBlueprintID)
		tokenName := ""
		if h.nameResolver != nil && tbID != "" {
			tokenName = strings.TrimSpace(h.nameResolver.ResolveTokenName(ctx, tbID))
		}
		if tokenName == "" {
			tokenName = tbID
		}

		createdBy := strings.TrimSpace(m.CreatedBy)
		createdByName := ""
		if h.nameResolver != nil && createdBy != "" {
			createdByName = strings.TrimSpace(h.nameResolver.ResolveMemberName(ctx, createdBy))
		}
		if createdByName == "" {
			createdByName = createdBy
		}

		var mintedAtPtr *string
		if m.MintedAt != nil && !m.MintedAt.IsZero() {
			s := m.MintedAt.UTC().Format(time.RFC3339)
			mintedAtPtr = &s
		}

		out[iid] = mintdto.MintListRowDTO{
			InspectionID:   iid,
			MintID:         strings.TrimSpace(m.ID),
			TokenBlueprint: tbID,
			TokenName:      tokenName,
			CreatedByName:  createdByName,
			MintedAt:       mintedAtPtr,
		}
	}

	_ = json.NewEncoder(w).Encode(out)
}

// ============================================================
// GET /mint/mints/{id}
// ============================================================
func (h *MintHandler) getMintByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.mintUC == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "mint usecase is not configured"})
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/mint/mints/")
	id = strings.Trim(id, "/")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "id is empty"})
		return
	}

	m, err := h.mintUC.ListMintsByInspectionIDs(ctx, []string{id})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	mintEntity, ok := m[id]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "mint not found"})
		return
	}

	products := make([]string, 0, len(mintEntity.Products))
	for pid := range mintEntity.Products {
		p := strings.TrimSpace(pid)
		if p != "" {
			products = append(products, p)
		}
	}
	sort.Strings(products)

	var createdAt *string
	if !mintEntity.CreatedAt.IsZero() {
		s := mintEntity.CreatedAt.UTC().Format(time.RFC3339)
		createdAt = &s
	}

	var mintedAt *string
	if mintEntity.MintedAt != nil && !mintEntity.MintedAt.IsZero() {
		s := mintEntity.MintedAt.UTC().Format(time.RFC3339)
		mintedAt = &s
	}

	var scheduledBurnDate *string
	if mintEntity.ScheduledBurnDate != nil && !mintEntity.ScheduledBurnDate.IsZero() {
		s := mintEntity.ScheduledBurnDate.UTC().Format(time.RFC3339)
		scheduledBurnDate = &s
	}

	out := map[string]any{
		"id":                strings.TrimSpace(mintEntity.ID),
		"inspectionId":      strings.TrimSpace(id),
		"brandId":           strings.TrimSpace(mintEntity.BrandID),
		"tokenBlueprintId":  strings.TrimSpace(mintEntity.TokenBlueprintID),
		"products":          products,
		"createdBy":         strings.TrimSpace(mintEntity.CreatedBy),
		"createdAt":         createdAt,
		"minted":            mintEntity.Minted,
		"mintedAt":          mintedAt,
		"scheduledBurnDate": scheduledBurnDate,
	}

	_ = json.NewEncoder(w).Encode(out)
}

// ============================================================
// POST /mint/requests/{mintRequestId}/mint
// ============================================================
func (h *MintHandler) mintFromMintRequest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.tokenUC == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "token usecase is not configured"})
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/mint/requests/")
	path = strings.TrimSuffix(path, "/mint")
	mintRequestID := strings.Trim(path, "/")

	if mintRequestID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "mintRequestId is empty"})
		return
	}

	result, err := h.tokenUC.MintFromMintRequest(ctx, mintRequestID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(result)
}

// ============================================================
// POST /mint/inspections/{productionId}/request
// ============================================================
func (h *MintHandler) updateRequestInfo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.mintUC == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "mint usecase is not configured"})
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/mint/inspections/")
	path = strings.TrimSuffix(path, "/request")
	productionID := strings.TrimSpace(path)

	if productionID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "productionId is empty"})
		return
	}

	var body struct {
		TokenBlueprintID  string  `json:"tokenBlueprintId"`
		ScheduledBurnDate *string `json:"scheduledBurnDate,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid body"})
		return
	}

	tokenBlueprintID := strings.TrimSpace(body.TokenBlueprintID)
	if tokenBlueprintID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "tokenBlueprintId is required"})
		return
	}

	log.Printf(
		"[mint_handler] /mint/inspections/{productionId}/request productionId=%q tokenBlueprintId=%q scheduledBurnDate=%v",
		productionID, tokenBlueprintID, body.ScheduledBurnDate,
	)

	updated, err := h.mintUC.UpdateRequestInfo(ctx, productionID, tokenBlueprintID, body.ScheduledBurnDate)
	if err != nil {
		log.Printf("[mint_handler] updateRequestInfo error=%v", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(updated)
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
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "mint usecase is not configured"})
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/mint/product_blueprints/")
	path = strings.TrimSuffix(path, "/patch")
	id := strings.Trim(path, "/")

	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "productBlueprintID is empty"})
		return
	}

	patch, err := h.mintUC.GetProductBlueprintPatchByID(ctx, id)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, pbpdom.ErrNotFound) {
			status = http.StatusNotFound
		}
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	brandName := ""
	if patch.BrandID != nil && h.nameResolver != nil {
		bid := strings.TrimSpace(*patch.BrandID)
		if bid != "" {
			brandName = strings.TrimSpace(h.nameResolver.ResolveBrandName(ctx, bid))
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
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "mint usecase is not configured"})
		return
	}

	var page branddom.Page

	result, err := h.mintUC.ListBrandsForCurrentCompany(ctx, page)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, mintapp.ErrCompanyIDMissing) {
			status = http.StatusBadRequest
		}
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(result)
}

// ============================================================
// GET /mint/token_blueprints?brandId=...
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
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "mint usecase is not configured"})
		return
	}

	brandID := strings.TrimSpace(r.URL.Query().Get("brandId"))
	if brandID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "brandId is required"})
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
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
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

// ============================================================
// Helpers (logging)
// ============================================================

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func sampleFirst[T any](xs []T) any {
	if len(xs) == 0 {
		return nil
	}
	return xs[0]
}

func toJSONForLog(v any, max int) string {
	if v == nil {
		return "null"
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "<marshal_error>"
	}
	s := string(b)
	if max > 0 && len(s) > max {
		return s[:max] + "...(truncated)"
	}
	return s
}

func sampleFirstKey[V any](m map[string]V) string {
	if len(m) == 0 {
		return ""
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys[0]
}

func sampleFirstValue[V any](m map[string]V) any {
	if len(m) == 0 {
		return nil
	}
	k := sampleFirstKey(m)
	if k == "" {
		return nil
	}
	return m[k]
}

// ============================================================
// compile-time guards（未使用でも import が消されないように）
// ============================================================

var _ = mintdom.ErrNotFound
var _ = mintpresenter.PresentInspectionViews

// keep unused in some builds
var _ = context.Canceled
