package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	uc "narratives/internal/application/usecase"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

// TokenBlueprintHandler handles /token-blueprints endpoints (GET list / GET by id).
type TokenBlueprintHandler struct {
	uc *uc.TokenBlueprintUsecase
}

// NewTokenBlueprintHandler initializes the HTTP handler.
func NewTokenBlueprintHandler(ucase *uc.TokenBlueprintUsecase) http.Handler {
	return &TokenBlueprintHandler{uc: ucase}
}

// ServeHTTP routes requests.
func (h *TokenBlueprintHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	// 一覧: GET /token-blueprints
	case r.Method == http.MethodGet && r.URL.Path == "/token-blueprints":
		h.list(w, r)

	// 詳細: GET /token-blueprints/{id}
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/token-blueprints/"):
		id := strings.TrimPrefix(r.URL.Path, "/token-blueprints/")
		h.get(w, r, id)

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// GET /token-blueprints/{id}
func (h *TokenBlueprintHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	tb, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeTokenBlueprintErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(tb)
}

// GET /token-blueprints （currentMember.companyId で絞り込み）
func (h *TokenBlueprintHandler) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// CompanyIDFromContext は 1 値返却なのでそれに合わせる
	companyID := strings.TrimSpace(uc.CompanyIDFromContext(ctx))
	if companyID == "" {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "companyId not found in context"})
		return
	}

	// ページング (page, perPage クエリは任意)
	pageNum := 1
	perPage := 50

	if v := r.URL.Query().Get("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			pageNum = n
		}
	}
	if v := r.URL.Query().Get("perPage"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			perPage = n
		}
	}

	page := tbdom.Page{
		Number:  pageNum,
		PerPage: perPage,
	}

	result, err := h.uc.ListByCompanyID(ctx, companyID, page)
	if err != nil {
		writeTokenBlueprintErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(result)
}

// Error handling
func writeTokenBlueprintErr(w http.ResponseWriter, err error) {
	// Return 500 without depending on domain error types.
	w.WriteHeader(http.StatusInternalServerError)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
