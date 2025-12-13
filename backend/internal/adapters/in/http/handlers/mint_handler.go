// backend/internal/adapters/in/http/handlers/mint_handler.go
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	resolver "narratives/internal/application/resolver"

	mintapp "narratives/internal/application/mint"
	mintdto "narratives/internal/application/mint/dto"

	usecase "narratives/internal/application/usecase"
	branddom "narratives/internal/domain/brand"
	mintdom "narratives/internal/domain/mint"
	pbpdom "narratives/internal/domain/productBlueprint"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

type MintHandler struct {
	mintUC       *mintapp.MintUsecase
	tokenUC      *usecase.TokenUsecase
	nameResolver *resolver.NameResolver
}

func NewMintHandler(
	mintUC *mintapp.MintUsecase,
	tokenUC *usecase.TokenUsecase,
	nameResolver *resolver.NameResolver,
) http.Handler {
	return &MintHandler{
		mintUC:       mintUC,
		tokenUC:      tokenUC,
		nameResolver: nameResolver,
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

	// ★ GET /mint/inspections は旧フローのため削除
	// （productionId 一覧は ProductionUsecase 側で作り、/mint/mints に渡す方式）

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
// GET /mint/mints?inspectionIds=a,b,c
// - docId 同一設計なので inspectionId (= productionId) をキーに map 返却
// - view=list: MintListRowDTO
// - view=dto : MintDTO 相当（画面で inspection と 1:1 結合しやすい形）
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

	mintsByInspectionID, err := h.mintUC.ListMintsByInspectionIDs(ctx, ids)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// view=dto: フロントが normalizeMintDTO しやすい shape で返す
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

	// view=list（デフォルト）: MintListRowDTO
	out := make(map[string]mintdto.MintListRowDTO, len(mintsByInspectionID))
	for inspectionID, m := range mintsByInspectionID {
		iid := strings.TrimSpace(inspectionID)

		// tokenName
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
// - docId 同一設計なので {id} は inspectionId (= productionId = mint docId)
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

	// dto 形で返す（フロント normalizeMintDTO に合わせる）
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

	updated, err := h.mintUC.UpdateRequestInfo(ctx, productionID, tokenBlueprintID, body.ScheduledBurnDate)
	if err != nil {
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

// compile-time guard（未使用でも import が消されないように）
var _ = mintdom.ErrNotFound
