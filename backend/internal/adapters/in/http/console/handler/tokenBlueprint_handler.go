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

	tbapp "narratives/internal/application/tokenBlueprint"
	usecase "narratives/internal/application/usecase"
	branddom "narratives/internal/domain/brand"
	domcommon "narratives/internal/domain/common"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

// TokenBlueprintHandler handles /token-blueprints endpoints.
type TokenBlueprintHandler struct {
	uc       *tbapp.TokenBlueprintUsecase
	queryUc  *tbapp.TokenBlueprintQueryUsecase
	brandSvc *branddom.Service
}

func NewTokenBlueprintHandler(
	ucase *tbapp.TokenBlueprintUsecase,
	queryUcase *tbapp.TokenBlueprintQueryUsecase,
	brandSvc *branddom.Service,
) http.Handler {
	return &TokenBlueprintHandler{
		uc:       ucase,
		queryUc:  queryUcase,
		brandSvc: brandSvc,
	}
}

// path helpers -------------------------------------------------------------

func trimSlashSuffix(p string) string {
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
	return parts[0]
}

// DTO --------------------------------------------------------------------

type createTokenBlueprintRequest struct {
	Name        string `json:"name"`
	Symbol      string `json:"symbol"`
	BrandID     string `json:"brandId"`
	CompanyID   string `json:"companyId,omitempty"`
	Description string `json:"description,omitempty"`
	AssigneeID  string `json:"assigneeId"`
	CreatedBy   string `json:"createdBy,omitempty"`

	// Firebase Storage へ frontend から直接 upload した downloadURL を保存する。
	IconURL string `json:"iconUrl,omitempty"`

	ContentFiles []tbdom.ContentFile `json:"contentFiles,omitempty"`
}

type updateTokenBlueprintRequest struct {
	Name        *string `json:"name,omitempty"`
	Symbol      *string `json:"symbol,omitempty"`
	BrandID     *string `json:"brandId,omitempty"`
	Description *string `json:"description,omitempty"`
	AssigneeID  *string `json:"assigneeId,omitempty"`

	// Firebase Storage へ frontend から直接 upload した downloadURL を保存する。
	IconURL *string `json:"iconUrl,omitempty"`

	ContentFiles *[]tbdom.ContentFile `json:"contentFiles,omitempty"`

	MetadataURI *string `json:"metadataUri,omitempty"`
	Minted      *bool   `json:"minted,omitempty"`
}

type contentFileResponse struct {
	tbdom.ContentFile
	URL string `json:"url,omitempty"`
}

type tokenBlueprintResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Symbol      string `json:"symbol"`
	BrandID     string `json:"brandId"`
	BrandName   string `json:"brandName"`
	CompanyID   string `json:"companyId"`
	Description string `json:"description,omitempty"`
	Minted      bool   `json:"minted"`

	ContentFiles []contentFileResponse `json:"contentFiles"`

	AssigneeID   string `json:"assigneeId"`
	AssigneeName string `json:"assigneeName"`

	CreatedAt time.Time `json:"createdAt"`

	CreatedBy     string `json:"createdBy"`
	CreatedByName string `json:"createdByName"`

	UpdatedAt     *time.Time `json:"updatedAt,omitempty"`
	UpdatedBy     string     `json:"updatedBy"`
	UpdatedByName string     `json:"updatedByName"`

	MetadataURI string `json:"metadataUri"`

	IconURL     string `json:"iconUrl,omitempty"`
	ContentsURL string `json:"contentsUrl,omitempty"`
}

type tokenBlueprintPageResponse struct {
	Items      []tokenBlueprintResponse `json:"items"`
	TotalCount int                      `json:"totalCount"`
	TotalPages int                      `json:"totalPages"`
	Page       int                      `json:"page"`
	PerPage    int                      `json:"perPage"`
}

type tokenBlueprintPatchResponse struct {
	ID          string `json:"id"`
	TokenName   string `json:"tokenName"`
	Symbol      string `json:"symbol"`
	BrandID     string `json:"brandId"`
	BrandName   string `json:"brandName"`
	Description string `json:"description,omitempty"`

	Minted      bool   `json:"minted"`
	MetadataURI string `json:"metadataUri"`

	IconURL     string `json:"iconUrl,omitempty"`
	ContentsURL string `json:"contentsUrl,omitempty"`
}

// name resolver helpers ----------------------------------------------------

func (h *TokenBlueprintHandler) resolveBrandName(ctx context.Context, id string) string {
	if h == nil || h.brandSvc == nil {
		return ""
	}

	name, _ := h.brandSvc.GetNameByID(ctx, id)
	return name
}

func resolveStoredIconURL(tb *tbdom.TokenBlueprint) string {
	if tb == nil {
		return ""
	}

	return tb.IconURL
}

func resolveStoredContentFileURL(f tbdom.ContentFile) string {
	return f.URL
}

func (h *TokenBlueprintHandler) toContentFilesResponse(
	_ context.Context,
	tb *tbdom.TokenBlueprint,
	_ bool,
) []contentFileResponse {
	if tb == nil || len(tb.ContentFiles) == 0 {
		return []contentFileResponse{}
	}

	out := make([]contentFileResponse, 0, len(tb.ContentFiles))
	for _, f := range tb.ContentFiles {
		out = append(out, contentFileResponse{
			ContentFile: f,
			URL:         resolveStoredContentFileURL(f),
		})
	}

	return out
}

func (h *TokenBlueprintHandler) toResponse(
	ctx context.Context,
	tb *tbdom.TokenBlueprint,
	includeContentViewURL bool,
	names *tbapp.TokenBlueprintMemberNames,
) tokenBlueprintResponse {
	if tb == nil {
		return tokenBlueprintResponse{}
	}

	var updPtr *time.Time
	if !tb.UpdatedAt.IsZero() {
		t := tb.UpdatedAt
		updPtr = &t
	}

	assigneeID := tb.AssigneeID
	createdBy := tb.CreatedBy
	updatedBy := tb.UpdatedBy

	assigneeName := ""
	createdByName := ""
	updatedByName := ""
	if names != nil {
		assigneeName = names.AssigneeName
		createdByName = names.CreatedByName
		updatedByName = names.UpdatedByName
	}

	return tokenBlueprintResponse{
		ID:           tb.ID,
		Name:         tb.Name,
		Symbol:       tb.Symbol,
		BrandID:      tb.BrandID,
		BrandName:    h.resolveBrandName(ctx, tb.BrandID),
		CompanyID:    tb.CompanyID,
		Description:  tb.Description,
		Minted:       tb.Minted,
		ContentFiles: h.toContentFilesResponse(ctx, tb, includeContentViewURL),

		AssigneeID:   assigneeID,
		AssigneeName: assigneeName,

		CreatedAt: tb.CreatedAt,

		CreatedBy:     createdBy,
		CreatedByName: createdByName,

		UpdatedAt:     updPtr,
		UpdatedBy:     updatedBy,
		UpdatedByName: updatedByName,

		MetadataURI: tb.MetadataURI,

		IconURL:     resolveStoredIconURL(tb),
		ContentsURL: "",
	}
}

func (h *TokenBlueprintHandler) toPatchResponse(ctx context.Context, tb *tbdom.TokenBlueprint) tokenBlueprintPatchResponse {
	if tb == nil {
		return tokenBlueprintPatchResponse{}
	}

	return tokenBlueprintPatchResponse{
		ID:          tb.ID,
		TokenName:   tb.Name,
		Symbol:      tb.Symbol,
		BrandID:     tb.BrandID,
		BrandName:   h.resolveBrandName(ctx, tb.BrandID),
		Description: tb.Description,
		Minted:      tb.Minted,
		MetadataURI: tb.MetadataURI,

		IconURL:     resolveStoredIconURL(tb),
		ContentsURL: "",
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

	case r.Method == http.MethodGet &&
		strings.HasPrefix(path, "/token-blueprints/") &&
		strings.HasSuffix(path, "/patch"):
		id := strings.TrimSuffix(strings.TrimPrefix(path, "/token-blueprints/"), "/patch")
		id = strings.Trim(id, "/")
		h.getPatch(w, r, id)
		return

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

	companyID := usecase.CompanyIDFromContext(ctx)
	actorID := r.Header.Get("X-Actor-Id")

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

	if req.Name == "" ||
		req.Symbol == "" ||
		req.BrandID == "" ||
		req.AssigneeID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "missing required fields"})
		return
	}

	iconURL := req.IconURL
	contentFiles := normalizeContentFilesFromRequest(req.ContentFiles, actorID)

	tb, err := h.uc.Create(ctx, tbapp.CreateBlueprintRequest{
		Name:         req.Name,
		Symbol:       req.Symbol,
		BrandID:      req.BrandID,
		CompanyID:    companyID,
		Description:  req.Description,
		AssigneeID:   req.AssigneeID,
		CreatedBy:    actorID,
		ActorID:      actorID,
		IconURL:      iconURL,
		ContentFiles: contentFiles,
	})
	if err != nil {
		log.Printf("[tokenBlueprint_handler] create failed err=%v", err)
		writeTokenBlueprintErr(w, err)
		return
	}

	var names tbapp.TokenBlueprintMemberNames
	if h != nil && h.queryUc != nil {
		if tb2, n, e := h.queryUc.GetByIDWithMemberNames(ctx, tb.ID); e == nil && tb2 != nil {
			tb = tb2
			names = n
		}
	}

	resp := h.toResponse(ctx, tb, true, &names)

	_ = json.NewEncoder(w).Encode(resp)
}

// getPatch -----------------------------------------------------------------

func (h *TokenBlueprintHandler) getPatch(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	companyID := usecase.CompanyIDFromContext(ctx)

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
		writeTokenBlueprintErr(w, err)
		return
	}

	if tb.CompanyID != companyID {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
		return
	}

	_ = json.NewEncoder(w).Encode(h.toPatchResponse(ctx, tb))
}

// get ---------------------------------------------------------------------

func (h *TokenBlueprintHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	companyID := usecase.CompanyIDFromContext(ctx)
	if companyID == "" {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "companyId not found in context"})
		return
	}

	if h == nil || h.queryUc == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "query usecase is not configured"})
		return
	}

	tb, names, err := h.queryUc.GetByIDWithMemberNames(ctx, id)
	if err != nil {
		writeTokenBlueprintErr(w, err)
		return
	}

	if tb.CompanyID != companyID {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
		return
	}

	_ = json.NewEncoder(w).Encode(h.toResponse(ctx, tb, true, &names))
}

// list --------------------------------------------------------------------

func (h *TokenBlueprintHandler) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	companyID := usecase.CompanyIDFromContext(ctx)
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
	brandID := q.Get("brandId")
	mintedFilter := q.Get("minted")

	page := domcommon.Page{Number: pageNum, PerPage: perPage}

	var (
		result domcommon.PageResult[tbdom.TokenBlueprint]
		err    error
	)

	switch {
	case brandID != "" && mintedFilter == "":
		result, err = h.uc.ListByBrandID(ctx, brandID, page)
	case mintedFilter == "minted":
		result, err = h.uc.ListMintedCompleted(ctx, page)
	default:
		result, err = h.uc.ListByCompanyID(ctx, companyID, page)
	}

	if err != nil {
		writeTokenBlueprintErr(w, err)
		return
	}

	var nameByMemberID map[string]string
	if h != nil && h.queryUc != nil && len(result.Items) > 0 {
		ids := make([]string, 0, len(result.Items)*3)
		for i := range result.Items {
			ids = append(ids,
				result.Items[i].AssigneeID,
				result.Items[i].CreatedBy,
				result.Items[i].UpdatedBy,
			)
		}

		m, _ := h.queryUc.ResolveMemberNames(ctx, ids)
		nameByMemberID = m
	}

	items := make([]tokenBlueprintResponse, 0, len(result.Items))
	for i := range result.Items {
		tb := &result.Items[i]

		assigneeID := tb.AssigneeID
		createdBy := tb.CreatedBy
		updatedBy := tb.UpdatedBy

		names := &tbapp.TokenBlueprintMemberNames{
			AssigneeName:  nameByMemberID[assigneeID],
			CreatedByName: nameByMemberID[createdBy],
			UpdatedByName: nameByMemberID[updatedBy],
		}

		items = append(items, h.toResponse(ctx, tb, false, names))
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

	companyID := usecase.CompanyIDFromContext(ctx)
	actorID := r.Header.Get("X-Actor-Id")

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

	var req updateTokenBlueprintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	tb, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeTokenBlueprintErr(w, err)
		return
	}

	if tb.CompanyID != companyID {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
		return
	}

	var contentFiles *[]tbdom.ContentFile
	if req.ContentFiles != nil {
		normalized := normalizeContentFilesFromRequest(*req.ContentFiles, actorID)
		contentFiles = &normalized
	}

	updated, err := h.uc.Update(ctx, tbapp.UpdateBlueprintRequest{
		ID:           id,
		Name:         req.Name,
		Symbol:       req.Symbol,
		BrandID:      req.BrandID,
		Description:  req.Description,
		AssigneeID:   req.AssigneeID,
		IconURL:      req.IconURL,
		ContentFiles: contentFiles,
		MetadataURI:  req.MetadataURI,
		Minted:       req.Minted,
		ActorID:      actorID,
	})
	if err != nil {
		writeTokenBlueprintErr(w, err)
		return
	}

	var names tbapp.TokenBlueprintMemberNames
	if h != nil && h.queryUc != nil {
		if tb2, n, e := h.queryUc.GetByIDWithMemberNames(ctx, updated.ID); e == nil && tb2 != nil {
			updated = tb2
			names = n
		}
	}

	resp := h.toResponse(ctx, updated, true, &names)

	_ = json.NewEncoder(w).Encode(resp)
}

// delete ------------------------------------------------------------------

func (h *TokenBlueprintHandler) delete(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	companyID := usecase.CompanyIDFromContext(ctx)

	tb, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeTokenBlueprintErr(w, err)
		return
	}

	if tb.CompanyID != companyID {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
		return
	}

	if err := h.uc.Delete(ctx, id); err != nil {
		writeTokenBlueprintErr(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// normalization ------------------------------------------------------------

func normalizeContentFilesFromRequest(files []tbdom.ContentFile, actorID string) []tbdom.ContentFile {
	if len(files) == 0 {
		return []tbdom.ContentFile{}
	}

	now := time.Now().UTC()
	out := make([]tbdom.ContentFile, 0, len(files))

	for _, f := range files {
		if f.ContentType == "" {
			f.ContentType = "application/octet-stream"
		}

		if f.Visibility == "" {
			f.Visibility = tbdom.VisibilityPrivate
		}

		if f.CreatedAt.IsZero() {
			f.CreatedAt = now
		}

		if f.CreatedBy == "" {
			f.CreatedBy = actorID
		}

		if f.UpdatedAt.IsZero() {
			f.UpdatedAt = now
		}

		if f.UpdatedBy == "" {
			f.UpdatedBy = actorID
		}

		if f.ID == "" {
			continue
		}

		out = append(out, f)
	}

	return out
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
