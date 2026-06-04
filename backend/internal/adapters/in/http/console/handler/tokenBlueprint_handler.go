// backend/internal/adapters/in/http/console/handler/tokenBlueprint_handler.go
package consoleHandler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	consolequery "narratives/internal/application/query/console"
	tbapp "narratives/internal/application/usecase"
	domcommon "narratives/internal/domain/common"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

type TokenBlueprintHandler struct {
	uc              *tbapp.TokenBlueprintUsecase
	detailQuery     *consolequery.TokenBlueprintDetailQuery
	managementQuery *consolequery.TokenBlueprintManagementQuery
}

func NewTokenBlueprintHandler(
	ucase *tbapp.TokenBlueprintUsecase,
	detailQuery *consolequery.TokenBlueprintDetailQuery,
	managementQuery *consolequery.TokenBlueprintManagementQuery,
) http.Handler {
	return &TokenBlueprintHandler{
		uc:              ucase,
		detailQuery:     detailQuery,
		managementQuery: managementQuery,
	}
}

func withoutTrailingSlash(p string) string {
	if p == "" {
		return ""
	}
	if len(p) > 1 && p[len(p)-1:] == "/" {
		return p[:len(p)-1]
	}
	return p
}

func extractFirstSegmentAfterPrefix(path, prefix string) string {
	path = withoutTrailingSlash(path)
	if !strings.HasPrefix(path, prefix) {
		return ""
	}

	rest := path[len(prefix):]
	if len(rest) > 0 && rest[0:1] == "/" {
		rest = rest[1:]
	}
	if rest == "" {
		return ""
	}

	parts := strings.SplitN(rest, "/", 2)
	return parts[0]
}

type createTokenBlueprintRequest struct {
	Name        string `json:"name"`
	Symbol      string `json:"symbol"`
	BrandID     string `json:"brandId"`
	CompanyID   string `json:"companyId,omitempty"`
	Description string `json:"description,omitempty"`

	AssigneeID string `json:"assigneeId"`
	CreatedBy  string `json:"createdBy,omitempty"`

	IconURL         string `json:"iconUrl,omitempty"`
	IconObjectPath  string `json:"iconObjectPath,omitempty"`
	IconFileName    string `json:"iconFileName,omitempty"`
	IconContentType string `json:"iconContentType,omitempty"`
	IconSize        int64  `json:"iconSize,omitempty"`

	ContentFiles []tbdom.ContentFile `json:"contentFiles,omitempty"`
}

type updateTokenBlueprintRequest struct {
	Name        *string `json:"name,omitempty"`
	Symbol      *string `json:"symbol,omitempty"`
	BrandID     *string `json:"brandId,omitempty"`
	Description *string `json:"description,omitempty"`
	AssigneeID  *string `json:"assigneeId,omitempty"`

	IconURL         *string `json:"iconUrl,omitempty"`
	IconObjectPath  *string `json:"iconObjectPath,omitempty"`
	IconFileName    *string `json:"iconFileName,omitempty"`
	IconContentType *string `json:"iconContentType,omitempty"`
	IconSize        *int64  `json:"iconSize,omitempty"`

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

	IconURL         string `json:"iconUrl,omitempty"`
	IconObjectPath  string `json:"iconObjectPath,omitempty"`
	IconFileName    string `json:"iconFileName,omitempty"`
	IconContentType string `json:"iconContentType,omitempty"`
	IconSize        int64  `json:"iconSize,omitempty"`

	// Deprecated: content files are returned via contentFiles[].url.
	ContentsURL string `json:"contentsUrl,omitempty"`
}

type tokenBlueprintPageResponse struct {
	Items      []tokenBlueprintResponse `json:"items"`
	TotalCount int                      `json:"totalCount"`
	TotalPages int                      `json:"totalPages"`
	Page       int                      `json:"page"`
	PerPage    int                      `json:"perPage"`
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
	tb *tbdom.TokenBlueprint,
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
	tb *tbdom.TokenBlueprint,
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

	brandName := ""
	assigneeName := ""
	createdByName := ""
	updatedByName := ""
	if names != nil {
		brandName = names.BrandName
		assigneeName = names.AssigneeName
		createdByName = names.CreatedByName
		updatedByName = names.UpdatedByName
	}

	return tokenBlueprintResponse{
		ID:          tb.ID,
		Name:        tb.Name,
		Symbol:      tb.Symbol,
		BrandID:     tb.BrandID,
		BrandName:   brandName,
		CompanyID:   tb.CompanyID,
		Description: tb.Description,
		Minted:      tb.Minted,

		ContentFiles: h.toContentFilesResponse(tb),

		AssigneeID:   tb.AssigneeID,
		AssigneeName: assigneeName,

		CreatedAt: tb.CreatedAt,

		CreatedBy:     tb.CreatedBy,
		CreatedByName: createdByName,

		UpdatedAt:     updPtr,
		UpdatedBy:     tb.UpdatedBy,
		UpdatedByName: updatedByName,

		MetadataURI: tb.MetadataURI,

		IconURL:         resolveStoredIconURL(tb),
		IconObjectPath:  tb.IconObjectPath,
		IconFileName:    tb.IconFileName,
		IconContentType: tb.IconContentType,
		IconSize:        tb.IconSize,

		ContentsURL: "",
	}
}

func (h *TokenBlueprintHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	path := withoutTrailingSlash(r.URL.Path)

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

func actorMemberIDFromContext(r *http.Request) (string, error) {
	if r == nil {
		return "", errors.New("request is nil")
	}

	memberID := tbapp.MemberIDFromContext(r.Context())
	if memberID == "" {
		return "", errors.New("memberId not found in context")
	}

	return memberID, nil
}

func (h *TokenBlueprintHandler) create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	companyID := tbapp.CompanyIDFromContext(ctx)
	actorMemberID, err := actorMemberIDFromContext(r)

	if companyID == "" {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "companyId not found in context"})
		return
	}

	if err != nil {
		writeActorResolveErr(w, err)
		return
	}

	var req createTokenBlueprintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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

	tb, err := h.uc.Create(ctx, tbapp.CreateBlueprintRequest{
		Name:        req.Name,
		Symbol:      req.Symbol,
		BrandID:     req.BrandID,
		CompanyID:   companyID,
		Description: req.Description,

		AssigneeID: req.AssigneeID,
		CreatedBy:  actorMemberID,

		IconURL:         req.IconURL,
		IconObjectPath:  req.IconObjectPath,
		IconFileName:    req.IconFileName,
		IconContentType: req.IconContentType,
		IconSize:        req.IconSize,

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

	resp := h.toResponse(tb, &names)

	_ = json.NewEncoder(w).Encode(resp)
}

func (h *TokenBlueprintHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	companyID := tbapp.CompanyIDFromContext(ctx)
	if companyID == "" {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "companyId not found in context"})
		return
	}

	tb, names, err := h.detailQuery.GetByID(ctx, id)
	if err != nil {
		writeTokenBlueprintErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(h.toResponse(tb, &names))
}

func (h *TokenBlueprintHandler) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	companyID := tbapp.CompanyIDFromContext(ctx)
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

		items = append(items, h.toResponse(&tb, &names))
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

	companyID := tbapp.CompanyIDFromContext(ctx)
	actorMemberID, err := actorMemberIDFromContext(r)

	if companyID == "" {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "companyId not found in context"})
		return
	}

	if err != nil {
		writeActorResolveErr(w, err)
		return
	}

	var req updateTokenBlueprintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	updated, err := h.uc.Update(ctx, tbapp.UpdateBlueprintRequest{
		ID:          id,
		Name:        req.Name,
		Symbol:      req.Symbol,
		BrandID:     req.BrandID,
		Description: req.Description,
		AssigneeID:  req.AssigneeID,

		IconURL:         req.IconURL,
		IconObjectPath:  req.IconObjectPath,
		IconFileName:    req.IconFileName,
		IconContentType: req.IconContentType,
		IconSize:        req.IconSize,

		ContentFiles: req.ContentFiles,
		MetadataURI:  req.MetadataURI,
		Minted:       req.Minted,
		UpdatedBy:    actorMemberID,
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

	resp := h.toResponse(updated, &names)

	_ = json.NewEncoder(w).Encode(resp)
}

func (h *TokenBlueprintHandler) delete(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	companyID := tbapp.CompanyIDFromContext(ctx)
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

func writeActorResolveErr(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	w.WriteHeader(http.StatusForbidden)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
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
		errors.Is(err, tbdom.ErrInvalidAssigneeID),
		errors.Is(err, tbdom.ErrInvalidCreatedBy),
		errors.Is(err, tbdom.ErrInvalidUpdatedBy),
		errors.Is(err, tbdom.ErrInvalidIconURL),
		errors.Is(err, tbdom.ErrInvalidIconObjectPath),
		errors.Is(err, tbdom.ErrInvalidIconFileName),
		errors.Is(err, tbdom.ErrInvalidIconContentType),
		errors.Is(err, tbdom.ErrInvalidIconSize),
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
