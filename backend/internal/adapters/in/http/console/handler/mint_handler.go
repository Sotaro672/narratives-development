// backend/internal/adapters/in/http/console/handler/mint_handler.go
package consoleHandler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"time"

	resolver "narratives/internal/application/resolver"

	mintapp "narratives/internal/application/mint"
	mintdto "narratives/internal/application/mint/dto"

	// productionIds 自動解決用
	productionapp "narratives/internal/application/production"

	// mintRequest 一覧の Query（productionId -> inspection + mint）
	querydto "narratives/internal/application/query/console/dto"

	branddom "narratives/internal/domain/brand"
	inspectiondom "narratives/internal/domain/inspection"
	mintdom "narratives/internal/domain/mint"
	pbpdom "narratives/internal/domain/productBlueprint"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

// handler が依存する最小 query IF
type MintRequestQueryService interface {
	// company 境界付きで、productionId と同 docId の inspection/mint を束ねた DTO を返す
	// NOTE: requestedBy は mint.CreatedBy に合わせる（DTO 側で担保）
	ListMintRequestManagementRows(ctx context.Context) ([]querydto.ProductionInspectionMintDTO, error)

	// detail 用（/mint/inspections/{productionId}）
	GetMintRequestDetail(ctx context.Context, productionID string) (*querydto.MintRequestDetailDTO, error)
}

type MintHandler struct {
	mintUC       *mintapp.MintUsecase
	nameResolver *resolver.NameResolver

	// /mint/inspections に productionIds が来ない場合に productions から自動生成する
	productionUC *productionapp.ProductionUsecase

	// /mint/requests 用 Query
	mintRequestQS MintRequestQueryService
}

func NewMintHandler(
	mintUC *mintapp.MintUsecase,
	nameResolver *resolver.NameResolver,
	productionUC *productionapp.ProductionUsecase,
	mintRequestQS MintRequestQueryService,
) http.Handler {
	// NameResolver は MintUsecase 側に保持
	if mintUC != nil {
		mintUC.SetNameResolver(nameResolver)
	}

	return &MintHandler{
		mintUC:        mintUC,
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

	// GET /mint/requests（mintRequest 管理一覧を 1shot で返す）
	case r.Method == http.MethodGet && r.URL.Path == "/mint/requests":
		h.listMintRequestsByCurrentCompany(w, r)
		return

	// GET /mint/inspections?productionIds=a,b,c
	case r.Method == http.MethodGet && r.URL.Path == "/mint/inspections":
		h.listInspectionsByProductionIDs(w, r)
		return

	// GET /mint/inspections/{productionId} (detail)
	case r.Method == http.MethodGet &&
		strings.HasPrefix(r.URL.Path, "/mint/inspections/") &&
		!strings.HasSuffix(r.URL.Path, "/request"):
		h.getMintRequestDetailByProductionID(w, r)
		return

	// GET /mint/mints?inspectionIds=a,b,c(&view=list|dto)
	case r.Method == http.MethodGet && r.URL.Path == "/mint/mints":
		h.listMintsByInspectionIDs(w, r)
		return

	// POST /mint/mints/{inspectionId}/execute
	case r.Method == http.MethodPost &&
		strings.HasPrefix(r.URL.Path, "/mint/mints/") &&
		strings.HasSuffix(r.URL.Path, "/execute"):
		h.executeMintByInspectionID(w, r)
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
// POST /mint/mints/{inspectionId}/execute
// ============================================================
func (h *MintHandler) executeMintByInspectionID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	start := time.Now()
	rawPath := strings.TrimSpace(r.URL.Path)
	rawQuery := strings.TrimSpace(r.URL.RawQuery)

	log.Printf(
		"[mint_handler] POST /mint/mints/{id}/execute start path=%q rawQuery=%q mintUC_nil=%t",
		rawPath, rawQuery, h.mintUC == nil,
	)

	if h.mintUC == nil {
		log.Printf("[mint_handler] POST /mint/mints/{id}/execute abort reason=mintUC_nil elapsed=%s", time.Since(start))
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "mint usecase is not configured"})
		return
	}

	// /mint/mints/{inspectionId}/execute
	path := strings.TrimPrefix(r.URL.Path, "/mint/mints/")
	path = strings.TrimSuffix(path, "/execute")
	inspectionID := strings.Trim(strings.TrimSpace(path), "/")

	if inspectionID == "" {
		log.Printf("[mint_handler] POST /mint/mints/{id}/execute bad_request reason=empty_inspectionId elapsed=%s", time.Since(start))
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "inspectionId is empty"})
		return
	}

	log.Printf("[mint_handler] POST /mint/mints/{id}/execute parsed inspectionId=%q", inspectionID)

	// 現状は mintRequestId と inspectionId が同一ID（docId）なので流用
	result, err := h.mintUC.MintFromMintRequest(ctx, inspectionID)
	if err != nil {
		log.Printf(
			"[mint_handler] POST /mint/mints/{id}/execute error inspectionId=%q err=%v elapsed=%s",
			inspectionID, err, time.Since(start),
		)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	log.Printf(
		"[mint_handler] POST /mint/mints/{id}/execute ok inspectionId=%q elapsed=%s result=%s",
		inspectionID, time.Since(start), toJSONForLog(result, 1500),
	)

	_ = json.NewEncoder(w).Encode(result)
}

// ============================================================
// GET /mint/inspections/{productionId}
// ============================================================
func (h *MintHandler) getMintRequestDetailByProductionID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.mintRequestQS == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "mintRequest query service is not configured"})
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
	detail, err := h.mintRequestQS.GetMintRequestDetail(ctx, productionID)
	elapsed := time.Since(start)

	if err != nil {
		if errors.Is(err, mintapp.ErrCompanyIDMissing) {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "companyId is missing"})
			return
		}

		if errors.Is(err, inspectiondom.ErrNotFound) || errors.Is(err, mintdom.ErrNotFound) ||
			strings.Contains(strings.ToLower(err.Error()), "not found") {
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
// GET /mint/requests
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

	// view=list: Usecase 側で tokenName/createdByName を解決した DTO を返す
	if view != "dto" {
		start := time.Now()
		out, err := h.mintUC.ListMintListRowsByInspectionIDs(ctx, ids)
		elapsed := time.Since(start)

		if err != nil {
			log.Printf("[mint_handler] /mint/mints ListMintListRowsByInspectionIDs error=%v elapsed=%s", err, elapsed)
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		log.Printf(
			"[mint_handler] /mint/mints(list) ok keys=%d elapsed=%s sampleKey=%q sampleVal=%s",
			len(out),
			elapsed,
			sampleFirstKey(out),
			toJSONForLog(sampleFirstValue(out), 1500),
		)

		_ = json.NewEncoder(w).Encode(out)
		return
	}

	// view=dto: 詳細フィールドを返しつつ、createdByName/tokenName も入れる
	start := time.Now()
	mintsByInspectionID, err := h.mintUC.ListMintsByInspectionIDs(ctx, ids)
	elapsed := time.Since(start)

	if err != nil {
		log.Printf("[mint_handler] /mint/mints ListMintsByInspectionIDs error=%v elapsed=%s", err, elapsed)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// createdByName/tokenName は Usecase の list-row DTO を流用して解決
	listRows, _ := h.mintUC.ListMintListRowsByInspectionIDs(ctx, ids)

	log.Printf(
		"[mint_handler] /mint/mints(dto) ok keys=%d elapsed=%s sampleKey=%q sampleVal=%s",
		len(mintsByInspectionID),
		elapsed,
		sampleFirstKey(mintsByInspectionID),
		toJSONForLog(sampleFirstValue(mintsByInspectionID), 1500),
	)

	out := make(map[string]any, len(mintsByInspectionID))
	for inspectionID, m := range mintsByInspectionID {
		iid := strings.TrimSpace(inspectionID)

		// handler 側は products を []string のみ返す方針（常に空）
		products := []string{}

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

		createdBy := strings.TrimSpace(m.CreatedBy)
		createdByName := createdBy
		tokenName := strings.TrimSpace(m.TokenBlueprintID)

		if row, ok := listRows[iid]; ok {
			if s := strings.TrimSpace(row.CreatedByName); s != "" {
				createdByName = s
			}
			if s := strings.TrimSpace(row.TokenName); s != "" {
				tokenName = s
			}
		}

		out[iid] = map[string]any{
			"id":                strings.TrimSpace(m.ID),
			"inspectionId":      iid,
			"brandId":           strings.TrimSpace(m.BrandID),
			"tokenBlueprintId":  strings.TrimSpace(m.TokenBlueprintID),
			"tokenName":         tokenName,
			"products":          products,
			"createdBy":         createdBy,
			"createdByName":     createdByName,
			"createdAt":         createdAt,
			"minted":            m.Minted,
			"mintedAt":          mintedAt,
			"scheduledBurnDate": scheduledBurnDate,
		}
	}

	_ = json.NewEncoder(w).Encode(out)
}

// ============================================================
// GET /mint/mints/{id}
// - handler は products を []string のみ返す（常に空）
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

	createdBy := strings.TrimSpace(mintEntity.CreatedBy)
	createdByName := createdBy
	tokenName := strings.TrimSpace(mintEntity.TokenBlueprintID)

	if rows, err := h.mintUC.ListMintListRowsByInspectionIDs(ctx, []string{id}); err == nil {
		if row, ok := rows[id]; ok {
			if s := strings.TrimSpace(row.CreatedByName); s != "" {
				createdByName = s
			}
			if s := strings.TrimSpace(row.TokenName); s != "" {
				tokenName = s
			}
		}
	}

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
		"tokenName":         tokenName,
		"products":          []string{},
		"createdBy":         createdBy,
		"createdByName":     createdByName,
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

	start := time.Now()
	rawPath := strings.TrimSpace(r.URL.Path)
	rawQuery := strings.TrimSpace(r.URL.RawQuery)

	log.Printf(
		"[mint_handler] POST /mint/requests/{id}/mint start path=%q rawQuery=%q mintUC_nil=%t",
		rawPath, rawQuery, h.mintUC == nil,
	)

	if h.mintUC == nil {
		log.Printf("[mint_handler] POST /mint/requests/{id}/mint abort reason=mintUC_nil elapsed=%s", time.Since(start))
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "mint usecase is not configured"})
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/mint/requests/")
	path = strings.TrimSuffix(path, "/mint")
	mintRequestID := strings.Trim(path, "/")

	if mintRequestID == "" {
		log.Printf("[mint_handler] POST /mint/requests/{id}/mint bad_request reason=empty_mintRequestId elapsed=%s", time.Since(start))
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "mintRequestId is empty"})
		return
	}

	log.Printf("[mint_handler] POST /mint/requests/{id}/mint parsed mintRequestId=%q", mintRequestID)

	result, err := h.mintUC.MintFromMintRequest(ctx, mintRequestID)
	if err != nil {
		log.Printf(
			"[mint_handler] POST /mint/requests/{id}/mint error mintRequestId=%q err=%v elapsed=%s",
			mintRequestID, err, time.Since(start),
		)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	log.Printf(
		"[mint_handler] POST /mint/requests/{id}/mint ok mintRequestId=%q elapsed=%s result=%s",
		mintRequestID, time.Since(start), toJSONForLog(result, 1500),
	)

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
	productionID := strings.TrimSpace(strings.Trim(path, "/"))

	if productionID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "productionId is empty"})
		return
	}

	// panic を握ってスタックを必ず出す（原因確定用）
	defer func() {
		if rec := recover(); rec != nil {
			log.Printf(
				"[mint_handler] updateRequestInfo PANIC productionId=%q rec=%v stack=%s",
				productionID, rec, string(debug.Stack()),
			)
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal server error"})
		}
	}()

	// body を生で読む（Decode 後だと読めない）
	raw, _ := io.ReadAll(r.Body)
	log.Printf(
		"[mint_handler] /mint/inspections/{productionId}/request rawBody productionId=%q body=%s",
		productionID, string(raw),
	)

	var body struct {
		TokenBlueprintID  string  `json:"tokenBlueprintId"`
		ScheduledBurnDate *string `json:"scheduledBurnDate,omitempty"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		log.Printf("[mint_handler] updateRequestInfo bad_request json_unmarshal_err=%v raw=%s", err, string(raw))
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

	// scheduledBurnDate の “値” を出す（ポインタアドレスじゃなく）
	sbd := "<nil>"
	if body.ScheduledBurnDate != nil {
		sbd = strings.TrimSpace(*body.ScheduledBurnDate)
	}

	log.Printf(
		"[mint_handler] /mint/inspections/{productionId}/request parsed productionId=%q tokenBlueprintId=%q scheduledBurnDate=%q",
		productionID, tokenBlueprintID, sbd,
	)

	updated, err := h.mintUC.UpdateRequestInfo(ctx, productionID, tokenBlueprintID, body.ScheduledBurnDate)
	if err != nil {
		log.Printf(
			"[mint_handler] updateRequestInfo ERROR productionId=%q tokenBlueprintId=%q scheduledBurnDate=%q err=%T %v",
			productionID, tokenBlueprintID, sbd, err, err,
		)
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
// - tokenBlueprint は iconUrl を保持しない（entity.go 正）
// - よって iconUrl は返さない（後方互換削除）
// ============================================================
type tokenBlueprintForMintResponse struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Symbol string `json:"symbol"`
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
			ID:     strings.TrimSpace(tb.ID),
			Name:   strings.TrimSpace(tb.Name),
			Symbol: strings.TrimSpace(tb.Symbol),
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

// keep imports referenced in some builds
var _ = mintdto.MintListRowDTO{}
var _ = context.Canceled
