// backend/internal/adapters/in/http/console/handler/tokenBlueprint_handler.go
package consoleHandler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	consolequery "narratives/internal/application/query/console"
	tbapp "narratives/internal/application/usecase"
	branddom "narratives/internal/domain/brand"
	domcommon "narratives/internal/domain/common"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

type TokenBlueprintHandler struct {
	uc              *tbapp.TokenBlueprintUsecase
	detailQuery     *consolequery.TokenBlueprintDetailQuery
	managementQuery *consolequery.TokenBlueprintManagementQuery
	brandSvc        *branddom.Service
}

func NewTokenBlueprintHandler(
	ucase *tbapp.TokenBlueprintUsecase,
	detailQuery *consolequery.TokenBlueprintDetailQuery,
	managementQuery *consolequery.TokenBlueprintManagementQuery,
	brandSvc *branddom.Service,
) http.Handler {
	return &TokenBlueprintHandler{
		uc:              ucase,
		detailQuery:     detailQuery,
		managementQuery: managementQuery,
		brandSvc:        brandSvc,
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
	names *consolequery.TokenBlueprintMemberNames,
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

	tb, err := h.uc.Create(ctx, tbapp.CreateBlueprintRequest{
		Name:         req.Name,
		Symbol:       req.Symbol,
		BrandID:      req.BrandID,
		CompanyID:    companyID,
		Description:  req.Description,
		AssigneeID:   req.AssigneeID,
		CreatedBy:    actorID,
		IconURL:      req.IconURL,
		ContentFiles: req.ContentFiles,
	})
	if err != nil {
		writeTokenBlueprintErr(w, err)
		return
	}

	tb, names, err := h.detailQuery.GetByID(ctx, tb.ID)
	if err != nil {
		writeTokenBlueprintErr(w, err)
		return
	}

	resp := h.toResponse(ctx, tb, true, &names)

	_ = json.NewEncoder(w).Encode(resp)
}

func (h *TokenBlueprintHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	companyID := strings.Trim(tbapp.CompanyIDFromContext(ctx), " \t\r\n")
	if companyID == "" {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "companyId not found in context"})
		return
	}

	tb, names, err := h.detailQuery.GetByID(ctx, strings.Trim(id, " \t\r\n"))
	if err != nil {
		writeTokenBlueprintErr(w, err)
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

	if brandID != "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "brandId filter is no longer supported",
		})
		return
	}

	if mintedFilter != "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "minted filter is no longer supported",
		})
		return
	}

	page := domcommon.Page{Number: pageNum, PerPage: perPage}

	result, err := h.managementQuery.ListByCompanyID(ctx, companyID, page)
	if err != nil {
		writeTokenBlueprintErr(w, err)
		return
	}

	items := make([]tokenBlueprintResponse, 0, len(result.Items))
	for i := range result.Items {
		item := result.Items[i]
		tb := item.TokenBlueprint
		names := item.MemberNames

		items = append(items, h.toResponse(ctx, &tb, false, &names))
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

	updated, err := h.uc.Update(ctx, tbapp.UpdateBlueprintRequest{
		ID:           id,
		Name:         req.Name,
		Symbol:       req.Symbol,
		BrandID:      req.BrandID,
		Description:  req.Description,
		AssigneeID:   req.AssigneeID,
		IconURL:      req.IconURL,
		ContentFiles: req.ContentFiles,
		MetadataURI:  req.MetadataURI,
		Minted:       req.Minted,
		UpdatedBy:    actorID,
	})
	if err != nil {
		writeTokenBlueprintErr(w, err)
		return
	}

	updated, names, err := h.detailQuery.GetByID(ctx, updated.ID)
	if err != nil {
		writeTokenBlueprintErr(w, err)
		return
	}

	resp := h.toResponse(ctx, updated, true, &names)

	_ = json.NewEncoder(w).Encode(resp)
}

func (h *TokenBlueprintHandler) delete(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	id = strings.Trim(id, " \t\r\n")

	companyID := strings.Trim(tbapp.CompanyIDFromContext(ctx), " \t\r\n")
	if companyID == "" {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "companyId not found in context"})
		return
	}

	if err := h.uc.Delete(ctx, id); err != nil {
		writeTokenBlueprintErr(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
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

	case errors.Is(err, tbdom.ErrInvalidID),
		errors.Is(err, tbdom.ErrInvalidName),
		errors.Is(err, tbdom.ErrInvalidSymbol),
		errors.Is(err, tbdom.ErrInvalidBrandID),
		errors.Is(err, tbdom.ErrInvalidCompanyID),
		errors.Is(err, tbdom.ErrInvalidCreatedBy),
		errors.Is(err, tbdom.ErrInvalidUpdatedBy),
		errors.Is(err, tbdom.ErrInvalidContentFile),
		errors.Is(err, tbdom.ErrInvalidContentType),
		errors.Is(err, tbdom.ErrInvalidContentVisibility):
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return

	default:
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
}
