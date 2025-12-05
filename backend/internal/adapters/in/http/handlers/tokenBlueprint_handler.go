// backend/internal/adapters/in/http/handlers/tokenBlueprint_handler.go
package handlers

import (
	"context"
	"encoding/json"
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

// DTO --------------------------------------------------------------------

type createTokenBlueprintRequest struct {
	Name         string   `json:"name"`
	Symbol       string   `json:"symbol"`
	BrandID      string   `json:"brandId"`
	Description  string   `json:"description"`
	AssigneeID   string   `json:"assigneeId"`
	CreatedBy    string   `json:"createdBy"`
	ContentFiles []string `json:"contentFiles,omitempty"`
	IconID       *string  `json:"iconId,omitempty"`
}

type updateTokenBlueprintRequest struct {
	Name         *string   `json:"name,omitempty"`
	Symbol       *string   `json:"symbol,omitempty"`
	BrandID      *string   `json:"brandId,omitempty"`
	Description  *string   `json:"description,omitempty"`
	AssigneeID   *string   `json:"assigneeId,omitempty"`
	IconID       *string   `json:"iconId,omitempty"`
	ContentFiles *[]string `json:"contentFiles,omitempty"`
}

type tokenBlueprintResponse struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Symbol       string     `json:"symbol"`
	BrandID      string     `json:"brandId"`
	BrandName    string     `json:"brandName"`
	CompanyID    string     `json:"companyId"`
	Description  string     `json:"description"`
	IconID       *string    `json:"iconId,omitempty"`
	ContentFiles []string   `json:"contentFiles"`
	AssigneeID   string     `json:"assigneeId"`
	AssigneeName string     `json:"assigneeName"`
	CreatedAt    time.Time  `json:"createdAt"`
	CreatedBy    string     `json:"createdBy"`
	UpdatedAt    *time.Time `json:"updatedAt,omitempty"`
	UpdatedBy    string     `json:"updatedBy"`
}

type tokenBlueprintPageResponse struct {
	Items      []tokenBlueprintResponse `json:"items"`
	TotalCount int                      `json:"totalCount"`
	TotalPages int                      `json:"totalPages"`
	Page       int                      `json:"page"`
	PerPage    int                      `json:"perPage"`
}

// name resolver helpers ----------------------------------------------------

func (h *TokenBlueprintHandler) resolveAssigneeName(ctx context.Context, id string) string {
	name, _ := h.memSvc.GetNameLastFirstByID(ctx, strings.TrimSpace(id))
	return name
}

func (h *TokenBlueprintHandler) resolveCreatorName(ctx context.Context, id string) string {
	name, _ := h.memSvc.GetNameLastFirstByID(ctx, strings.TrimSpace(id))
	return name
}

func (h *TokenBlueprintHandler) resolveBrandName(ctx context.Context, id string) string {
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

	return tokenBlueprintResponse{
		ID:           tb.ID,
		Name:         tb.Name,
		Symbol:       tb.Symbol,
		BrandID:      tb.BrandID,
		BrandName:    h.resolveBrandName(ctx, tb.BrandID),
		CompanyID:    tb.CompanyID,
		Description:  tb.Description,
		IconID:       tb.IconID,
		ContentFiles: tb.ContentFiles,
		AssigneeID:   tb.AssigneeID,
		AssigneeName: h.resolveAssigneeName(ctx, tb.AssigneeID),
		CreatedAt:    tb.CreatedAt,
		CreatedBy:    h.resolveCreatorName(ctx, tb.CreatedBy),
		UpdatedAt:    updPtr,
		UpdatedBy:    tb.UpdatedBy,
	}
}

// ServeHTTP ---------------------------------------------------------------

func (h *TokenBlueprintHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/token-blueprints":
		h.create(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/token-blueprints":
		h.list(w, r)
	case (r.Method == http.MethodPatch || r.Method == http.MethodPut) &&
		strings.HasPrefix(r.URL.Path, "/token-blueprints/"):
		id := strings.TrimPrefix(r.URL.Path, "/token-blueprints/")
		h.update(w, r, id)
	case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/token-blueprints/"):
		id := strings.TrimPrefix(r.URL.Path, "/token-blueprints/")
		h.delete(w, r, id)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/token-blueprints/"):
		id := strings.TrimPrefix(r.URL.Path, "/token-blueprints/")
		h.get(w, r, id)
	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// create ------------------------------------------------------------------

func (h *TokenBlueprintHandler) create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	companyID := strings.TrimSpace(uc.CompanyIDFromContext(ctx))
	if companyID == "" {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "companyId not found in context"})
		return
	}

	var req createTokenBlueprintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	if strings.TrimSpace(req.Name) == "" ||
		strings.TrimSpace(req.Symbol) == "" ||
		strings.TrimSpace(req.BrandID) == "" ||
		strings.TrimSpace(req.AssigneeID) == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "missing required fields"})
		return
	}

	actorID := strings.TrimSpace(r.Header.Get("X-Actor-Id"))
	createdBy := strings.TrimSpace(req.CreatedBy)
	if createdBy == "" {
		createdBy = actorID
	}

	tb, err := h.uc.CreateWithUploads(ctx, uc.CreateBlueprintRequest{
		Name:        strings.TrimSpace(req.Name),
		Symbol:      strings.TrimSpace(req.Symbol),
		BrandID:     strings.TrimSpace(req.BrandID),
		CompanyID:   companyID,
		Description: strings.TrimSpace(req.Description),
		AssigneeID:  strings.TrimSpace(req.AssigneeID),
		CreatedBy:   createdBy,
		ActorID:     actorID,
	})
	if err != nil {
		writeTokenBlueprintErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(h.toResponse(ctx, tb))
}

// get ---------------------------------------------------------------------

func (h *TokenBlueprintHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	tb, err := h.uc.GetByID(ctx, strings.TrimSpace(id))
	if err != nil {
		writeTokenBlueprintErr(w, err)
		return
	}

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

	result, err := h.uc.ListByCompanyID(ctx, companyID, tbdom.Page{
		Number:  pageNum,
		PerPage: perPage,
	})
	if err != nil {
		writeTokenBlueprintErr(w, err)
		return
	}

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

	var req updateTokenBlueprintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	actorID := strings.TrimSpace(r.Header.Get("X-Actor-Id"))

	// ★ update リクエスト内容をログ（デバッグ用）
	{
		b, _ := json.MarshalIndent(req, "", "  ")
		println("[TokenBlueprintHandler.update] raw update req for id=", id)
		println(string(b))
		println("actorId=", actorID)
	}

	tb, err := h.uc.Update(ctx, uc.UpdateBlueprintRequest{
		ID:           id,
		Name:         req.Name,
		Symbol:       req.Symbol,
		BrandID:      req.BrandID,
		Description:  req.Description,
		AssigneeID:   req.AssigneeID,
		IconID:       req.IconID,
		ContentFiles: req.ContentFiles,
		ActorID:      actorID,
	})
	if err != nil {
		writeTokenBlueprintErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(h.toResponse(ctx, tb))
}

// delete ------------------------------------------------------------------

func (h *TokenBlueprintHandler) delete(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	if err := h.uc.Delete(ctx, strings.TrimSpace(id)); err != nil {
		writeTokenBlueprintErr(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// error utility ------------------------------------------------------------

func writeTokenBlueprintErr(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
