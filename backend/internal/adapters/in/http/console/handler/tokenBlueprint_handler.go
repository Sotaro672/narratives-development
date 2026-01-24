// backend/internal/adapters/in/http/console/handler/tokenBlueprint_handler.go
package consoleHandler

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	uc "narratives/internal/application/usecase"
	branddom "narratives/internal/domain/brand"
	memdom "narratives/internal/domain/member"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

// TokenBlueprintHandler handles /token-blueprints endpoints.
type TokenBlueprintHandler struct {
	uc       *uc.TokenBlueprintUsecase
	memSvc   *memdom.Service
	brandSvc *branddom.Service
}

func NewTokenBlueprintHandler(
	ucase *uc.TokenBlueprintUsecase,
	memSvc *memdom.Service,
	brandSvc *branddom.Service,
) http.Handler {
	return &TokenBlueprintHandler{
		uc:       ucase,
		memSvc:   memSvc,
		brandSvc: brandSvc,
	}
}

// path helpers -------------------------------------------------------------

func trimSlashSuffix(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	return strings.TrimSuffix(p, "/")
}

// extractFirstSegmentAfterPrefix returns "{id}" from "/token-blueprints/{id}/xxx".
func extractFirstSegmentAfterPrefix(path, prefix string) string {
	path = trimSlashSuffix(path)
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	rest := strings.TrimPrefix(path, prefix)
	rest = strings.TrimPrefix(rest, "/")
	if rest == "" {
		return ""
	}
	parts := strings.SplitN(rest, "/", 2)
	return strings.TrimSpace(parts[0])
}

// DTO --------------------------------------------------------------------

// ✅ backward compatible fields removed:
// - createdBy (request) is not accepted; backend uses X-Actor-Id only
// - contentFiles (request) is not accepted on create (contents are managed via separate flows)
type createTokenBlueprintRequest struct {
	Name        string `json:"name"`
	Symbol      string `json:"symbol"`
	BrandID     string `json:"brandId"`
	Description string `json:"description,omitempty"`
	AssigneeID  string `json:"assigneeId"`

	// ★追加: icon upload を行うか（フロントが File を持っている場合 true）
	HasIconFile bool `json:"hasIconFile"`
	// ★追加: 例 "image/png"
	IconContentType string `json:"iconContentType,omitempty"`
}

type updateTokenBlueprintRequest struct {
	Name        *string `json:"name,omitempty"`
	Symbol      *string `json:"symbol,omitempty"`
	BrandID     *string `json:"brandId,omitempty"`
	Description *string `json:"description,omitempty"`
	AssigneeID  *string `json:"assigneeId,omitempty"`

	// entity.go 正: embedded contents (replace all when provided)
	ContentFiles *[]tbdom.ContentFile `json:"contentFiles,omitempty"`
}

type tokenBlueprintResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Symbol      string `json:"symbol"`
	BrandID     string `json:"brandId"`
	BrandName   string `json:"brandName"`
	CompanyID   string `json:"companyId"`
	Description string `json:"description,omitempty"`

	// entity.go 正: embedded
	ContentFiles []tbdom.ContentFile `json:"contentFiles"`

	AssigneeID   string `json:"assigneeId"`
	AssigneeName string `json:"assigneeName"`

	CreatedAt time.Time `json:"createdAt"`

	// ✅ backward compat removed:
	CreatedByID   string `json:"createdById"`
	CreatedByName string `json:"createdByName"`

	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
	UpdatedBy string     `json:"updatedBy"`

	// metadataUri
	MetadataURI string `json:"metadataUri"`

	// ★追加: create時に icon の署名付きURLを返す（フロントはこれでPUTする）
	IconUpload *uc.TokenIconUploadURL `json:"iconUpload,omitempty"`
}

type tokenBlueprintPageResponse struct {
	Items      []tokenBlueprintResponse `json:"items"`
	TotalCount int                      `json:"totalCount"`
	TotalPages int                      `json:"totalPages"`
	Page       int                      `json:"page"`
	PerPage    int                      `json:"perPage"`
}

// TokenBlueprintCard 用 patch
// GET /token-blueprints/{id}/patch
type tokenBlueprintPatchResponse struct {
	ID          string `json:"id"`
	TokenName   string `json:"tokenName"`
	Symbol      string `json:"symbol"`
	BrandID     string `json:"brandId"`
	BrandName   string `json:"brandName"`
	Description string `json:"description,omitempty"`

	Minted      bool   `json:"minted"`
	MetadataURI string `json:"metadataUri"`
}

// name resolver helpers ----------------------------------------------------

func (h *TokenBlueprintHandler) resolveAssigneeName(ctx context.Context, id string) string {
	if h == nil || h.memSvc == nil {
		return ""
	}
	name, _ := h.memSvc.GetNameLastFirstByID(ctx, strings.TrimSpace(id))
	return name
}

func (h *TokenBlueprintHandler) resolveCreatorName(ctx context.Context, id string) string {
	if h == nil || h.memSvc == nil {
		return ""
	}
	name, _ := h.memSvc.GetNameLastFirstByID(ctx, strings.TrimSpace(id))
	return name
}

func (h *TokenBlueprintHandler) resolveBrandName(ctx context.Context, id string) string {
	if h == nil || h.brandSvc == nil {
		return ""
	}
	name, _ := h.brandSvc.GetNameByID(ctx, strings.TrimSpace(id))
	return name
}

func (h *TokenBlueprintHandler) toResponse(ctx context.Context, tb *tbdom.TokenBlueprint) tokenBlueprintResponse {
	if tb == nil {
		return tokenBlueprintResponse{}
	}

	var updPtr *time.Time
	if !tb.UpdatedAt.IsZero() {
		t := tb.UpdatedAt
		updPtr = &t
	}

	createdByID := strings.TrimSpace(tb.CreatedBy)

	return tokenBlueprintResponse{
		ID:           strings.TrimSpace(tb.ID),
		Name:         strings.TrimSpace(tb.Name),
		Symbol:       strings.TrimSpace(tb.Symbol),
		BrandID:      strings.TrimSpace(tb.BrandID),
		BrandName:    h.resolveBrandName(ctx, tb.BrandID),
		CompanyID:    strings.TrimSpace(tb.CompanyID),
		Description:  strings.TrimSpace(tb.Description),
		ContentFiles: tb.ContentFiles,
		AssigneeID:   strings.TrimSpace(tb.AssigneeID),
		AssigneeName: h.resolveAssigneeName(ctx, tb.AssigneeID),
		CreatedAt:    tb.CreatedAt,

		CreatedByID:   createdByID,
		CreatedByName: h.resolveCreatorName(ctx, createdByID),

		UpdatedAt:   updPtr,
		UpdatedBy:   strings.TrimSpace(tb.UpdatedBy),
		MetadataURI: strings.TrimSpace(tb.MetadataURI),

		IconUpload: nil, // GET/LIST は返さない（create時だけ上書き）
	}
}

func (h *TokenBlueprintHandler) toPatchResponse(ctx context.Context, tb *tbdom.TokenBlueprint) tokenBlueprintPatchResponse {
	if tb == nil {
		return tokenBlueprintPatchResponse{}
	}
	return tokenBlueprintPatchResponse{
		ID:          strings.TrimSpace(tb.ID),
		TokenName:   strings.TrimSpace(tb.Name),
		Symbol:      strings.TrimSpace(tb.Symbol),
		BrandID:     strings.TrimSpace(tb.BrandID),
		BrandName:   h.resolveBrandName(ctx, tb.BrandID),
		Description: strings.TrimSpace(tb.Description),
		Minted:      tb.Minted,
		MetadataURI: strings.TrimSpace(tb.MetadataURI),
	}
}

// ServeHTTP ---------------------------------------------------------------

func (h *TokenBlueprintHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	path := trimSlashSuffix(r.URL.Path)

	switch {
	case r.Method == http.MethodPost && path == "/token-blueprints":
		h.create(w, r)
		return

	case r.Method == http.MethodGet && path == "/token-blueprints":
		h.list(w, r)
		return

	// TokenBlueprintCard 用 patch
	// GET /token-blueprints/{id}/patch
	case r.Method == http.MethodGet &&
		strings.HasPrefix(path, "/token-blueprints/") &&
		strings.HasSuffix(path, "/patch"):
		id := strings.TrimSuffix(strings.TrimPrefix(path, "/token-blueprints/"), "/patch")
		id = strings.Trim(id, "/")
		h.getPatch(w, r, id)
		return

	// update/delete/get は「{id} の次に余計なセグメントが来ても巻き込まない」ように first segment を取る
	case (r.Method == http.MethodPatch || r.Method == http.MethodPut) &&
		strings.HasPrefix(path, "/token-blueprints/"):
		id := extractFirstSegmentAfterPrefix(path, "/token-blueprints/")
		h.update(w, r, id)
		return

	case r.Method == http.MethodDelete && strings.HasPrefix(path, "/token-blueprints/"):
		id := extractFirstSegmentAfterPrefix(path, "/token-blueprints/")
		h.delete(w, r, id)
		return

	case r.Method == http.MethodGet && strings.HasPrefix(path, "/token-blueprints/"):
		id := extractFirstSegmentAfterPrefix(path, "/token-blueprints/")
		h.get(w, r, id)
		return

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}
}

// create ------------------------------------------------------------------

func (h *TokenBlueprintHandler) create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	companyID := strings.TrimSpace(uc.CompanyIDFromContext(ctx))
	actorID := strings.TrimSpace(r.Header.Get("X-Actor-Id"))

	log.Printf("[tokenBlueprint_handler] create start companyId=%q actorId=%q", companyID, actorID)

	if companyID == "" {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "companyId not found in context"})
		return
	}
	if actorID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "X-Actor-Id is required"})
		return
	}

	var req createTokenBlueprintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[tokenBlueprint_handler] create decode failed err=%v", err)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	log.Printf("[tokenBlueprint_handler] create request name=%q symbol=%q brandId=%q assigneeId=%q hasIconFile=%v iconContentType=%q",
		strings.TrimSpace(req.Name),
		strings.TrimSpace(req.Symbol),
		strings.TrimSpace(req.BrandID),
		strings.TrimSpace(req.AssigneeID),
		req.HasIconFile,
		strings.TrimSpace(req.IconContentType),
	)

	if strings.TrimSpace(req.Name) == "" ||
		strings.TrimSpace(req.Symbol) == "" ||
		strings.TrimSpace(req.BrandID) == "" ||
		strings.TrimSpace(req.AssigneeID) == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "missing required fields"})
		return
	}

	// CreatedBy は request から受け取らない（X-Actor-Id）
	tb, err := h.uc.Create(ctx, uc.CreateBlueprintRequest{
		Name:        strings.TrimSpace(req.Name),
		Symbol:      strings.TrimSpace(req.Symbol),
		BrandID:     strings.TrimSpace(req.BrandID),
		CompanyID:   companyID,
		Description: strings.TrimSpace(req.Description),
		AssigneeID:  strings.TrimSpace(req.AssigneeID),
		CreatedBy:   actorID,
		ActorID:     actorID,
	})
	if err != nil {
		log.Printf("[tokenBlueprint_handler] create failed err=%v", err)
		writeTokenBlueprintErr(w, err)
		return
	}

	log.Printf("[tokenBlueprint_handler] create success id=%q companyId=%q brandId=%q assigneeId=%q minted=%v metadataUri=%q",
		tb.ID, tb.CompanyID, tb.BrandID, tb.AssigneeID, tb.Minted, tb.MetadataURI,
	)

	resp := h.toResponse(ctx, tb)

	// ★ここが統一の肝：hasIconFile=true の場合、署名URLを返す
	if req.HasIconFile {
		ct := strings.TrimSpace(req.IconContentType)
		if ct == "" {
			// フロントが file.type を渡していない場合のフォールバック
			ct = "application/octet-stream"
		}

		iconUpload, err := h.uc.IssueTokenIconUploadURL(ctx, tb.ID, "", ct)
		if err != nil {
			// ここは要件次第：
			// - 厳格にするなら create 自体を失敗させる
			// - 現状運用では「作成は成功、アイコンだけ失敗」を許容しログで追うのが扱いやすい
			log.Printf("[tokenBlueprint_handler] iconUpload issue FAILED id=%q err=%v", tb.ID, err)
		} else {
			resp.IconUpload = iconUpload
		}
	}

	_ = json.NewEncoder(w).Encode(resp)
}

// getPatch -----------------------------------------------------------------

func (h *TokenBlueprintHandler) getPatch(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	companyID := strings.TrimSpace(uc.CompanyIDFromContext(ctx))
	id = strings.TrimSpace(id)

	log.Printf("[tokenBlueprint_handler] getPatch start id=%q companyId(ctx)=%q", id, companyID)

	if companyID == "" {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "companyId not found in context"})
		return
	}
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "id is empty"})
		return
	}

	tb, err := h.uc.GetByID(ctx, id)
	if err != nil {
		log.Printf("[tokenBlueprint_handler] getPatch failed id=%q err=%v", id, err)
		writeTokenBlueprintErr(w, err)
		return
	}

	// tenant boundary
	if strings.TrimSpace(tb.CompanyID) != companyID {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
		return
	}

	resp := h.toPatchResponse(ctx, tb)

	log.Printf("[tokenBlueprint_handler] getPatch success id=%q brandId=%q minted=%v metadataUri=%q",
		resp.ID, resp.BrandID, resp.Minted, resp.MetadataURI,
	)

	_ = json.NewEncoder(w).Encode(resp)
}

// get ---------------------------------------------------------------------

func (h *TokenBlueprintHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	companyID := strings.TrimSpace(uc.CompanyIDFromContext(ctx))
	id = strings.TrimSpace(id)

	log.Printf("[tokenBlueprint_handler] get start id=%q companyId(ctx)=%q", id, companyID)

	tb, err := h.uc.GetByID(ctx, id)
	if err != nil {
		log.Printf("[tokenBlueprint_handler] get failed id=%q err=%v", id, err)
		writeTokenBlueprintErr(w, err)
		return
	}

	log.Printf("[tokenBlueprint_handler] get success id=%q companyId=%q brandId=%q assigneeId=%q minted=%v metadataUri=%q",
		tb.ID, tb.CompanyID, tb.BrandID, tb.AssigneeID, tb.Minted, tb.MetadataURI,
	)

	_ = json.NewEncoder(w).Encode(h.toResponse(ctx, tb))
}

// list --------------------------------------------------------------------

func (h *TokenBlueprintHandler) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	companyID := strings.TrimSpace(uc.CompanyIDFromContext(ctx))
	if companyID == "" {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "companyId not found in context"})
		return
	}

	pageNum := 1
	perPage := 50

	if v := r.URL.Query().Get("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			pageNum = n
		}
	}
	if v := r.URL.Query().Get("perPage"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			perPage = n
		}
	}

	q := r.URL.Query()
	brandID := strings.TrimSpace(q.Get("brandId"))
	mintedFilter := strings.TrimSpace(q.Get("minted"))

	page := tbdom.Page{
		Number:  pageNum,
		PerPage: perPage,
	}

	log.Printf("[tokenBlueprint_handler] list start companyId=%q page=%d perPage=%d brandId=%q minted=%q",
		companyID, pageNum, perPage, brandID, mintedFilter,
	)

	var (
		result tbdom.PageResult
		err    error
		mode   string
	)

	switch {
	case brandID != "" && mintedFilter == "":
		mode = "ListByBrandID"
		result, err = h.uc.ListByBrandID(ctx, brandID, page)
	case mintedFilter == "notYet":
		mode = "ListMintedNotYet"
		result, err = h.uc.ListMintedNotYet(ctx, page)
	case mintedFilter == "minted":
		mode = "ListMintedCompleted"
		result, err = h.uc.ListMintedCompleted(ctx, page)
	default:
		mode = "ListByCompanyID"
		result, err = h.uc.ListByCompanyID(ctx, companyID, page)
	}

	if err != nil {
		log.Printf("[tokenBlueprint_handler] list failed mode=%s companyId=%q err=%v", mode, companyID, err)
		writeTokenBlueprintErr(w, err)
		return
	}

	log.Printf("[tokenBlueprint_handler] list success mode=%s companyId=%q totalCount=%d page=%d perPage=%d items=%d",
		mode, companyID, result.TotalCount, result.Page, result.PerPage, len(result.Items),
	)

	items := make([]tokenBlueprintResponse, 0, len(result.Items))
	for i := range result.Items {
		items = append(items, h.toResponse(ctx, &result.Items[i]))
	}

	_ = json.NewEncoder(w).Encode(tokenBlueprintPageResponse{
		Items:      items,
		TotalCount: result.TotalCount,
		TotalPages: result.TotalPages,
		Page:       result.Page,
		PerPage:    result.PerPage,
	})
}

// update ------------------------------------------------------------------

func (h *TokenBlueprintHandler) update(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	id = strings.TrimSpace(id)

	companyID := strings.TrimSpace(uc.CompanyIDFromContext(ctx))
	actorID := strings.TrimSpace(r.Header.Get("X-Actor-Id"))

	log.Printf("[tokenBlueprint_handler] update start id=%q companyId(ctx)=%q actorId=%q", id, companyID, actorID)

	var req updateTokenBlueprintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[tokenBlueprint_handler] update decode failed id=%q err=%v", id, err)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	log.Printf("[tokenBlueprint_handler] update request id=%q hasName=%v hasSymbol=%v hasBrandId=%v hasDesc=%v hasAssignee=%v hasContentFiles=%v",
		id,
		req.Name != nil,
		req.Symbol != nil,
		req.BrandID != nil,
		req.Description != nil,
		req.AssigneeID != nil,
		req.ContentFiles != nil,
	)

	// tenant boundary check (companyId一致)
	tb, err := h.uc.GetByID(ctx, id)
	if err != nil {
		log.Printf("[tokenBlueprint_handler] update get failed id=%q err=%v", id, err)
		writeTokenBlueprintErr(w, err)
		return
	}
	if strings.TrimSpace(tb.CompanyID) != companyID {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
		return
	}

	updated, err := h.uc.Update(ctx, uc.UpdateBlueprintRequest{
		ID:           id,
		Name:         req.Name,
		Symbol:       req.Symbol,
		BrandID:      req.BrandID,
		Description:  req.Description,
		AssigneeID:   req.AssigneeID,
		ContentFiles: req.ContentFiles,
		ActorID:      actorID,
	})
	if err != nil {
		log.Printf("[tokenBlueprint_handler] update failed id=%q err=%v", id, err)
		writeTokenBlueprintErr(w, err)
		return
	}

	log.Printf("[tokenBlueprint_handler] update success id=%q companyId=%q brandId=%q assigneeId=%q minted=%v metadataUri=%q contents=%d",
		updated.ID, updated.CompanyID, updated.BrandID, updated.AssigneeID, updated.Minted, updated.MetadataURI, len(updated.ContentFiles),
	)

	resp := h.toResponse(ctx, updated)
	_ = json.NewEncoder(w).Encode(resp)
}

// delete ------------------------------------------------------------------

func (h *TokenBlueprintHandler) delete(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	id = strings.TrimSpace(id)

	companyID := strings.TrimSpace(uc.CompanyIDFromContext(ctx))
	actorID := strings.TrimSpace(r.Header.Get("X-Actor-Id"))

	log.Printf("[tokenBlueprint_handler] delete start id=%q companyId(ctx)=%q actorId=%q", id, companyID, actorID)

	// tenant boundary check
	tb, err := h.uc.GetByID(ctx, id)
	if err != nil {
		log.Printf("[tokenBlueprint_handler] delete get failed id=%q err=%v", id, err)
		writeTokenBlueprintErr(w, err)
		return
	}
	if strings.TrimSpace(tb.CompanyID) != companyID {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
		return
	}

	if err := h.uc.Delete(ctx, id); err != nil {
		log.Printf("[tokenBlueprint_handler] delete failed id=%q err=%v", id, err)
		writeTokenBlueprintErr(w, err)
		return
	}

	log.Printf("[tokenBlueprint_handler] delete success id=%q", id)
	w.WriteHeader(http.StatusNoContent)
}

// error utility ------------------------------------------------------------

func writeTokenBlueprintErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, tbdom.ErrNotFound):
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return

	case errors.Is(err, tbdom.ErrConflict):
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return

	default:
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
}
