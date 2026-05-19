// backend/internal/adapters/in/http/console/handler/mint_handler.go
package consoleHandler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"

	resolver "narratives/internal/application/resolver"

	mintapp "narratives/internal/application/mint"

	// productionIds 自動解決用
	productionapp "narratives/internal/application/production"

	// mintRequest 一覧の Query（productionId -> inspection + mint）
	querydto "narratives/internal/application/query/console/dto"

	branddom "narratives/internal/domain/brand"
	inspectiondom "narratives/internal/domain/inspection"
	mintdom "narratives/internal/domain/mint"
)

// handler が依存する最小 query IF
type MintRequestQueryService interface {
	// company 境界付きで、productionId と同 docId の inspection/mint を束ねた DTO を返す
	// NOTE: requestedBy は mint.CreatedBy に合わせる（DTO 側で担保）
	ListMintRequestManagementRows(
		ctx context.Context,
		input querydto.ListMintRequestManagementRowsInput,
	) ([]querydto.ProductionInspectionMintDTO, error)

	// detail 用（/mint/inspections/{productionId}）
	GetMintRequestDetail(ctx context.Context, productionID string) (*querydto.MintRequestDetailDTO, error)

	// list 用（productionId = inspectionId = mintId）
	ListMintListRowsByProductionIDs(
		ctx context.Context,
		productionIDs []string,
	) (map[string]querydto.MintListRowDTO, error)

	// dto 用（productionId = inspectionId = mintId）
	ListMintDTOsByProductionIDs(
		ctx context.Context,
		productionIDs []string,
	) (map[string]querydto.MintDTO, error)

	// single dto 用（productionId = inspectionId = mintId）
	GetMintByProductionID(
		ctx context.Context,
		productionID string,
	) (*querydto.MintDTO, error)

	// tokenBlueprint options for mint screen
	ListTokenBlueprintsForMint(
		ctx context.Context,
		input querydto.ListTokenBlueprintsForMintInput,
	) ([]querydto.TokenBlueprintForMintDTO, error)
}

type MintHandler struct {
	mintUC *mintapp.MintUsecase

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

	if h.mintUC == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "mint usecase is not configured"})
		return
	}

	// /mint/mints/{inspectionId}/execute
	path := strings.TrimPrefix(r.URL.Path, "/mint/mints/")
	path = strings.TrimSuffix(path, "/execute")
	inspectionID := strings.Trim(path, "/")

	if inspectionID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "inspectionId is empty"})
		return
	}

	// 現状は mintRequestId と inspectionId が同一ID（docId）なので流用
	result, err := h.mintUC.MintFromMintRequest(ctx, inspectionID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

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

	productionID := path

	detail, err := h.mintRequestQS.GetMintRequestDetail(ctx, productionID)
	if err != nil {
		if errors.Is(err, mintapp.ErrCompanyIDMissing) {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "companyId is missing"})
			return
		}

		if errors.Is(err, inspectiondom.ErrNotFound) ||
			errors.Is(err, mintdom.ErrNotFound) ||
			strings.Contains(strings.ToLower(err.Error()), "not found") {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "mint request detail not found"})
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

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

	view := r.URL.Query().Get("view")
	if view == "" {
		view = "management"
	}

	productionIDs := parseCommaSeparatedIDs(r.URL.Query().Get("productionIds"))

	rows, err := h.mintRequestQS.ListMintRequestManagementRows(
		ctx,
		querydto.ListMintRequestManagementRowsInput{
			ProductionIDs: productionIDs,
			View:          view,
		},
	)
	if err != nil {
		if errors.Is(err, mintapp.ErrCompanyIDMissing) {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "companyId is missing"})
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

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

	rawProductionIDs := r.URL.Query().Get("productionIds")
	rawInspectionIDs := r.URL.Query().Get("inspectionIds")

	raw := rawProductionIDs
	if raw == "" {
		raw = rawInspectionIDs
	}

	var ids []string

	if raw == "" {
		if h.productionUC == nil {
			_ = json.NewEncoder(w).Encode([]any{})
			return
		}

		prods, err := h.productionUC.ListWithAssigneeName(ctx)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		seen := make(map[string]struct{}, len(prods))
		ids = make([]string, 0, len(prods))
		for _, p := range prods {
			pid := p.ID
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

		if len(ids) == 0 {
			_ = json.NewEncoder(w).Encode([]any{})
			return
		}
	} else {
		ids = parseCommaSeparatedIDs(raw)
		if len(ids) == 0 {
			_ = json.NewEncoder(w).Encode([]any{})
			return
		}
	}

	batches, err := h.mintUC.ListInspectionBatchesByProductionIDs(ctx, ids)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(batches)
}

// ============================================================
// GET /mint/mints?inspectionIds=a,b,c
// ============================================================

func (h *MintHandler) listMintsByInspectionIDs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.mintRequestQS == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "mintRequest query service is not configured"})
		return
	}

	raw := r.URL.Query().Get("inspectionIds")
	if raw == "" {
		_ = json.NewEncoder(w).Encode(map[string]any{})
		return
	}

	ids := parseCommaSeparatedIDs(raw)
	if len(ids) == 0 {
		_ = json.NewEncoder(w).Encode(map[string]any{})
		return
	}

	view := r.URL.Query().Get("view")
	if view == "" {
		view = "list"
	}

	if view != "dto" {
		out, err := h.mintRequestQS.ListMintListRowsByProductionIDs(ctx, ids)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		_ = json.NewEncoder(w).Encode(out)
		return
	}

	out, err := h.mintRequestQS.ListMintDTOsByProductionIDs(ctx, ids)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(out)
}

// ============================================================
// GET /mint/mints/{id}
// ============================================================

func (h *MintHandler) getMintByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.mintRequestQS == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "mintRequest query service is not configured"})
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/mint/mints/")
	id = strings.Trim(id, "/")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "id is empty"})
		return
	}

	out, err := h.mintRequestQS.GetMintByProductionID(ctx, id)
	if err != nil {
		if errors.Is(err, mintdom.ErrNotFound) ||
			strings.Contains(strings.ToLower(err.Error()), "not found") {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "mint not found"})
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(out)
}

// ============================================================
// POST /mint/requests/{mintRequestId}/mint
// ============================================================

func (h *MintHandler) mintFromMintRequest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.mintUC == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "mint usecase is not configured"})
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

	result, err := h.mintUC.MintFromMintRequest(ctx, mintRequestID)
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
	productionID := strings.Trim(path, "/")

	if productionID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "productionId is empty"})
		return
	}

	defer func() {
		if rec := recover(); rec != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal server error"})
		}
	}()

	raw, _ := io.ReadAll(r.Body)

	var body struct {
		TokenBlueprintID  string  `json:"tokenBlueprintId"`
		ScheduledBurnDate *string `json:"scheduledBurnDate,omitempty"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid body"})
		return
	}

	tokenBlueprintID := body.TokenBlueprintID
	if tokenBlueprintID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "tokenBlueprintId is required"})
		return
	}

	updated, err := h.mintUC.UpdateRequestInfo(ctx, productionID, tokenBlueprintID, body.ScheduledBurnDate)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(updated)
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

func (h *MintHandler) listTokenBlueprintsByBrand(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if h.mintRequestQS == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "mintRequest query service is not configured"})
		return
	}

	brandID := r.URL.Query().Get("brandId")
	if brandID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "brandId is required"})
		return
	}

	pageNumber := 1
	perPage := 100

	if pageParam := r.URL.Query().Get("page"); pageParam != "" {
		if n, err := strconv.Atoi(pageParam); err == nil && n > 0 {
			pageNumber = n
		}
	}
	if perPageParam := r.URL.Query().Get("perPage"); perPageParam != "" {
		if n, err := strconv.Atoi(perPageParam); err == nil && n > 0 {
			perPage = n
		}
	}

	items, err := h.mintRequestQS.ListTokenBlueprintsForMint(
		ctx,
		querydto.ListTokenBlueprintsForMintInput{
			BrandID: brandID,
			Page:    pageNumber,
			PerPage: perPage,
		},
	)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(items)
}

func parseCommaSeparatedIDs(raw string) []string {
	if raw == "" {
		return []string{}
	}

	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))

	for _, p := range parts {
		id := strings.TrimSpace(p)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}

		seen[id] = struct{}{}
		out = append(out, id)
	}

	sort.Strings(out)
	return out
}

// keep imports referenced in some builds
var _ = context.Canceled
