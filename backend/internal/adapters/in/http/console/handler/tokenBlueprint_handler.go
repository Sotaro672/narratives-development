// backend\internal\adapters\in\http\console\handler\tokenBlueprint_handler.go
package consoleHandler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	tbapp "narratives/internal/application/usecase"
	branddom "narratives/internal/domain/brand"
	domcommon "narratives/internal/domain/common"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

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

func trimSlashSuffix(p string) string {
	p = strings.Trim(p, " \t\r\n")
	if p == "" {
		return ""
	}
	return strings.TrimSuffix(p, "/")
}

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
	return strings.Trim(parts[0], " \t\r\n")
}

type createTokenBlueprintRequest struct {
	Name         string              `json:"name"`
	Symbol       string              `json:"symbol"`
	BrandID      string              `json:"brandId"`
	CompanyID    string              `json:"companyId,omitempty"`
	Description  string              `json:"description,omitempty"`
	AssigneeID   string              `json:"assigneeId"`
	CreatedBy    string              `json:"createdBy,omitempty"`
	IconURL      string              `json:"iconUrl,omitempty"`
	ContentFiles []tbdom.ContentFile `json:"contentFiles,omitempty"`
}

type updateTokenBlueprintRequest struct {
	Name         *string              `json:"name,omitempty"`
	Symbol       *string              `json:"symbol,omitempty"`
	BrandID      *string              `json:"brandId,omitempty"`
	Description  *string              `json:"description,omitempty"`
	AssigneeID   *string              `json:"assigneeId,omitempty"`
	IconURL      *string              `json:"iconUrl,omitempty"`
	ContentFiles *[]tbdom.ContentFile `json:"contentFiles,omitempty"`
	MetadataURI  *string              `json:"metadataUri,omitempty"`
	Minted       *bool                `json:"minted,omitempty"`
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

func (h *TokenBlueprintHandler) resolveBrandName(ctx context.Context, id string) string {
	if h == nil || h.brandSvc == nil {
		return ""
	}

	name, _ := h.brandSvc.GetNameByID(ctx, strings.Trim(id, " \t\r\n"))
	return name
}

func resolveStoredIconURL(tb *tbdom.TokenBlueprint) string {
	if tb == nil {
		return ""
	}

	return strings.Trim(tb.IconURL, " \t\r\n")
}

func resolveStoredContentFileURL(f tbdom.ContentFile) string {
	return strings.Trim(f.URL, " \t\r\n")
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

	assigneeID := strings.Trim(tb.AssigneeID, " \t\r\n")
	createdBy := strings.Trim(tb.CreatedBy, " \t\r\n")
	updatedBy := strings.Trim(tb.UpdatedBy, " \t\r\n")

	assigneeName := ""
	createdByName := ""
	updatedByName := ""
	if names != nil {
		assigneeName = strings.Trim(names.AssigneeName, " \t\r\n")
		createdByName = strings.Trim(names.CreatedByName, " \t\r\n")
		updatedByName = strings.Trim(names.UpdatedByName, " \t\r\n")
	}

	return tokenBlueprintResponse{
		ID:           strings.Trim(tb.ID, " \t\r\n"),
		Name:         strings.Trim(tb.Name, " \t\r\n"),
		Symbol:       strings.Trim(tb.Symbol, " \t\r\n"),
		BrandID:      strings.Trim(tb.BrandID, " \t\r\n"),
		BrandName:    h.resolveBrandName(ctx, tb.BrandID),
		CompanyID:    strings.Trim(tb.CompanyID, " \t\r\n"),
		Description:  strings.Trim(tb.Description, " \t\r\n"),
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

		MetadataURI: strings.Trim(tb.MetadataURI, " \t\r\n"),

		IconURL:     resolveStoredIconURL(tb),
		ContentsURL: "",
	}
}

func (h *TokenBlueprintHandler) toPatchResponse(ctx context.Context, tb *tbdom.TokenBlueprint) tokenBlueprintPatchResponse {
	if tb == nil {
		return tokenBlueprintPatchResponse{}
	}

	return tokenBlueprintPatchResponse{
		ID:          strings.Trim(tb.ID, " \t\r\n"),
		TokenName:   strings.Trim(tb.Name, " \t\r\n"),
		Symbol:      strings.Trim(tb.Symbol, " \t\r\n"),
		BrandID:     strings.Trim(tb.BrandID, " \t\r\n"),
		BrandName:   h.resolveBrandName(ctx, tb.BrandID),
		Description: strings.Trim(tb.Description, " \t\r\n"),
		Minted:      tb.Minted,
		MetadataURI: strings.Trim(tb.MetadataURI, " \t\r\n"),
		IconURL:     resolveStoredIconURL(tb),
		ContentsURL: "",
	}
}

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

func (h *TokenBlueprintHandler) create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	companyID := strings.Trim(tbapp.CompanyIDFromContext(ctx), " \t\r\n")
	actorID := strings.Trim(r.Header.Get("X-Actor-Id"), " \t\r\n")

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
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	if strings.Trim(req.Name, " \t\r\n") == "" ||
		strings.Trim(req.Symbol, " \t\r\n") == "" ||
		strings.Trim(req.BrandID, " \t\r\n") == "" ||
		strings.Trim(req.AssigneeID, " \t\r\n") == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "missing required fields"})
		return
	}

	iconURL := strings.Trim(req.IconURL, " \t\r\n")
	contentFiles := normalizeContentFilesFromRequest(req.ContentFiles, actorID)

	tb, err := h.uc.Create(ctx, tbapp.CreateBlueprintRequest{
		Name:         strings.Trim(req.Name, " \t\r\n"),
		Symbol:       strings.Trim(req.Symbol, " \t\r\n"),
		BrandID:      strings.Trim(req.BrandID, " \t\r\n"),
		CompanyID:    companyID,
		Description:  strings.Trim(req.Description, " \t\r\n"),
		AssigneeID:   strings.Trim(req.AssigneeID, " \t\r\n"),
		CreatedBy:    actorID,
		ActorID:      actorID,
		IconURL:      iconURL,
		ContentFiles: contentFiles,
	})
	if err != nil {
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

func (h *TokenBlueprintHandler) getPatch(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	companyID := strings.Trim(tbapp.CompanyIDFromContext(ctx), " \t\r\n")
	id = strings.Trim(id, " \t\r\n")

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

	if strings.Trim(tb.CompanyID, " \t\r\n") != companyID {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
		return
	}

	_ = json.NewEncoder(w).Encode(h.toPatchResponse(ctx, tb))
}

func (h *TokenBlueprintHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	companyID := strings.Trim(tbapp.CompanyIDFromContext(ctx), " \t\r\n")
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

	tb, names, err := h.queryUc.GetByIDWithMemberNames(ctx, strings.Trim(id, " \t\r\n"))
	if err != nil {
		writeTokenBlueprintErr(w, err)
		return
	}

	if strings.Trim(tb.CompanyID, " \t\r\n") != companyID {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
		return
	}

	_ = json.NewEncoder(w).Encode(h.toResponse(ctx, tb, true, &names))
}

func (h *TokenBlueprintHandler) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	companyID := strings.Trim(tbapp.CompanyIDFromContext(ctx), " \t\r\n")
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
	brandID := strings.Trim(q.Get("brandId"), " \t\r\n")
	mintedFilter := strings.Trim(q.Get("minted"), " \t\r\n")

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
				strings.Trim(result.Items[i].AssigneeID, " \t\r\n"),
				strings.Trim(result.Items[i].CreatedBy, " \t\r\n"),
				strings.Trim(result.Items[i].UpdatedBy, " \t\r\n"),
			)
		}

		m, _ := h.queryUc.ResolveMemberNames(ctx, ids)
		nameByMemberID = m
	}

	items := make([]tokenBlueprintResponse, 0, len(result.Items))
	for i := range result.Items {
		tb := &result.Items[i]

		assigneeID := strings.Trim(tb.AssigneeID, " \t\r\n")
		createdBy := strings.Trim(tb.CreatedBy, " \t\r\n")
		updatedBy := strings.Trim(tb.UpdatedBy, " \t\r\n")

		names := &tbapp.TokenBlueprintMemberNames{
			AssigneeName:  strings.Trim(nameByMemberID[assigneeID], " \t\r\n"),
			CreatedByName: strings.Trim(nameByMemberID[createdBy], " \t\r\n"),
			UpdatedByName: strings.Trim(nameByMemberID[updatedBy], " \t\r\n"),
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

func (h *TokenBlueprintHandler) update(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	id = strings.Trim(id, " \t\r\n")

	companyID := strings.Trim(tbapp.CompanyIDFromContext(ctx), " \t\r\n")
	actorID := strings.Trim(r.Header.Get("X-Actor-Id"), " \t\r\n")

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

	if strings.Trim(tb.CompanyID, " \t\r\n") != companyID {
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

func (h *TokenBlueprintHandler) delete(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	id = strings.Trim(id, " \t\r\n")

	companyID := strings.Trim(tbapp.CompanyIDFromContext(ctx), " \t\r\n")

	tb, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeTokenBlueprintErr(w, err)
		return
	}

	if strings.Trim(tb.CompanyID, " \t\r\n") != companyID {
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

func normalizeContentFilesFromRequest(files []tbdom.ContentFile, actorID string) []tbdom.ContentFile {
	if len(files) == 0 {
		return []tbdom.ContentFile{}
	}

	now := time.Now().UTC()
	out := make([]tbdom.ContentFile, 0, len(files))

	for _, f := range files {
		f.ID = strings.Trim(f.ID, " \t\r\n")
		f.Name = strings.Trim(f.Name, " \t\r\n")
		f.Type = tbdom.ContentFileType(strings.Trim(string(f.Type), " \t\r\n"))
		f.ContentType = strings.Trim(f.ContentType, " \t\r\n")
		f.ObjectPath = strings.Trim(f.ObjectPath, " \t\r\n")
		f.URL = strings.Trim(f.URL, " \t\r\n")
		f.Visibility = tbdom.ContentVisibility(strings.Trim(string(f.Visibility), " \t\r\n"))

		if f.ContentType == "" {
			f.ContentType = "application/octet-stream"
		}

		if f.Visibility == "" {
			f.Visibility = tbdom.VisibilityPrivate
		}

		if f.CreatedAt.IsZero() {
			f.CreatedAt = now
		}

		if strings.Trim(f.CreatedBy, " \t\r\n") == "" {
			f.CreatedBy = actorID
		}

		if f.UpdatedAt.IsZero() {
			f.UpdatedAt = now
		}

		if strings.Trim(f.UpdatedBy, " \t\r\n") == "" {
			f.UpdatedBy = actorID
		}

		if f.ID == "" {
			continue
		}

		out = append(out, f)
	}

	return out
}

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
