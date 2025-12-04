package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	uc "narratives/internal/application/usecase"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

// TokenBlueprintHandler handles /token-blueprints endpoints (list/get/create/update/delete).
type TokenBlueprintHandler struct {
	uc *uc.TokenBlueprintUsecase
}

// NewTokenBlueprintHandler initializes the HTTP handler.
func NewTokenBlueprintHandler(ucase *uc.TokenBlueprintUsecase) http.Handler {
	return &TokenBlueprintHandler{uc: ucase}
}

// リクエスト DTO

type createTokenBlueprintRequest struct {
	Name        string `json:"name"`
	Symbol      string `json:"symbol"`
	BrandID     string `json:"brandId"`
	Description string `json:"description"`
	AssigneeID  string `json:"assigneeId"`
	// 将来用: contentFiles / iconId は別エンドポイント等で扱う想定
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

// ServeHTTP routes requests.
func (h *TokenBlueprintHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	// 作成: POST /token-blueprints
	case r.Method == http.MethodPost && r.URL.Path == "/token-blueprints":
		h.create(w, r)

	// 一覧: GET /token-blueprints
	case r.Method == http.MethodGet && r.URL.Path == "/token-blueprints":
		h.list(w, r)

	// 更新: PATCH or PUT /token-blueprints/{id}
	case (r.Method == http.MethodPatch || r.Method == http.MethodPut) &&
		strings.HasPrefix(r.URL.Path, "/token-blueprints/"):
		id := strings.TrimPrefix(r.URL.Path, "/token-blueprints/")
		h.update(w, r, id)

	// 削除: DELETE /token-blueprints/{id}
	case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/token-blueprints/"):
		id := strings.TrimPrefix(r.URL.Path, "/token-blueprints/")
		h.delete(w, r, id)

	// 詳細: GET /token-blueprints/{id}
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/token-blueprints/"):
		id := strings.TrimPrefix(r.URL.Path, "/token-blueprints/")
		h.get(w, r, id)

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
	}
}

// POST /token-blueprints
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

	// 簡易バリデーション（詳細はドメイン層で再検証）
	if strings.TrimSpace(req.Name) == "" ||
		strings.TrimSpace(req.Symbol) == "" ||
		strings.TrimSpace(req.BrandID) == "" ||
		strings.TrimSpace(req.Description) == "" ||
		strings.TrimSpace(req.AssigneeID) == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "missing required fields"})
		return
	}

	// ActorID は暫定的にヘッダから取得（なければ空文字）
	actorID := strings.TrimSpace(r.Header.Get("X-Actor-Id"))

	tb, err := h.uc.CreateWithUploads(ctx, uc.CreateBlueprintRequest{
		Name:        strings.TrimSpace(req.Name),
		Symbol:      strings.TrimSpace(req.Symbol),
		BrandID:     strings.TrimSpace(req.BrandID),
		CompanyID:   companyID,
		Description: strings.TrimSpace(req.Description),
		AssigneeID:  strings.TrimSpace(req.AssigneeID),

		// ファイルアップロードはこのハンドラでは扱わない
		Icon:     nil,
		Contents: nil,

		ActorID: strings.TrimSpace(actorID),
	})
	if err != nil {
		writeTokenBlueprintErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(tb)
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

// PATCH/PUT /token-blueprints/{id}
func (h *TokenBlueprintHandler) update(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	var req updateTokenBlueprintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	actorID := strings.TrimSpace(r.Header.Get("X-Actor-Id"))

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

	_ = json.NewEncoder(w).Encode(tb)
}

// DELETE /token-blueprints/{id}
func (h *TokenBlueprintHandler) delete(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	if err := h.uc.Delete(ctx, id); err != nil {
		writeTokenBlueprintErr(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Error handling
func writeTokenBlueprintErr(w http.ResponseWriter, err error) {
	// Return 500 without depending on domain error types.
	w.WriteHeader(http.StatusInternalServerError)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
