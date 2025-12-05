// backend/internal/adapters/in/http/handlers/tokenBlueprint_handler.go
package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	uc "narratives/internal/application/usecase"
	memdom "narratives/internal/domain/member"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

// TokenBlueprintHandler handles /token-blueprints endpoints (list/get/create/update/delete).
type TokenBlueprintHandler struct {
	uc     *uc.TokenBlueprintUsecase
	memSvc *memdom.Service
}

// NewTokenBlueprintHandler initializes the HTTP handler.
func NewTokenBlueprintHandler(ucase *uc.TokenBlueprintUsecase, memSvc *memdom.Service) http.Handler {
	return &TokenBlueprintHandler{
		uc:     ucase,
		memSvc: memSvc,
	}
}

// リクエスト DTO

type createTokenBlueprintRequest struct {
	Name         string   `json:"name"`
	Symbol       string   `json:"symbol"`
	BrandID      string   `json:"brandId"`
	Description  string   `json:"description"`
	AssigneeID   string   `json:"assigneeId"`
	CreatedBy    string   `json:"createdBy"` // ★ 追加: 作成者（memberId）
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

// レスポンス DTO（assigneeName を含めて画面に渡す）
type tokenBlueprintResponse struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Symbol       string   `json:"symbol"`
	BrandID      string   `json:"brandId"`
	CompanyID    string   `json:"companyId"`
	Description  string   `json:"description"`
	IconID       *string  `json:"iconId,omitempty"`
	ContentFiles []string `json:"contentFiles"`
	AssigneeID   string   `json:"assigneeId"`
	AssigneeName string   `json:"assigneeName"` // ★ member.Service で解決した表示名

	CreatedAt time.Time `json:"createdAt"`
	CreatedBy string    `json:"createdBy"`
	UpdatedAt time.Time `json:"updatedAt"`
	UpdatedBy string    `json:"updatedBy"`
}

type tokenBlueprintPageResponse struct {
	Items      []tokenBlueprintResponse `json:"items"`
	TotalCount int                      `json:"totalCount"`
	TotalPages int                      `json:"totalPages"`
	Page       int                      `json:"page"`
	PerPage    int                      `json:"perPage"`
}

// assigneeId → assigneeName 解決ヘルパー
func (h *TokenBlueprintHandler) resolveAssigneeName(ctx context.Context, assigneeID string) string {
	id := strings.TrimSpace(assigneeID)
	if id == "" || h.memSvc == nil {
		return ""
	}
	name, err := h.memSvc.GetNameLastFirstByID(ctx, id)
	if err != nil {
		// 名前が取れなくても致命的ではないので空文字で返す
		return ""
	}
	return name
}

// ドメイン TokenBlueprint → レスポンス DTO 変換
func (h *TokenBlueprintHandler) toResponse(ctx context.Context, tb *tbdom.TokenBlueprint) tokenBlueprintResponse {
	if tb == nil {
		return tokenBlueprintResponse{}
	}

	assigneeName := h.resolveAssigneeName(ctx, tb.AssigneeID)

	return tokenBlueprintResponse{
		ID:           tb.ID,
		Name:         tb.Name,
		Symbol:       tb.Symbol,
		BrandID:      tb.BrandID,
		CompanyID:    tb.CompanyID,
		Description:  tb.Description,
		IconID:       tb.IconID,
		ContentFiles: tb.ContentFiles,
		AssigneeID:   tb.AssigneeID,
		AssigneeName: assigneeName,

		CreatedAt: tb.CreatedAt,
		CreatedBy: tb.CreatedBy,
		UpdatedAt: tb.UpdatedAt,
		UpdatedBy: tb.UpdatedBy,
	}
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

	// ★ デバッグ用ログ：実際に backend でどう見えているか確認
	log.Printf(
		"[TokenBlueprintHandler.create] raw req: name=%q symbol=%q brandId=%q assigneeId=%q createdBy=%q",
		req.Name, req.Symbol, req.BrandID, req.AssigneeID, req.CreatedBy,
	)

	// -----------------------------------------
	// description を必須チェックから除外したバリデーション
	// -----------------------------------------
	if strings.TrimSpace(req.Name) == "" ||
		strings.TrimSpace(req.Symbol) == "" ||
		strings.TrimSpace(req.BrandID) == "" ||
		strings.TrimSpace(req.AssigneeID) == "" {
		log.Printf(
			"[TokenBlueprintHandler.create] missing required fields: name=%q symbol=%q brandId=%q assigneeId=%q",
			strings.TrimSpace(req.Name),
			strings.TrimSpace(req.Symbol),
			strings.TrimSpace(req.BrandID),
			strings.TrimSpace(req.AssigneeID),
		)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "missing required fields"})
		return
	}

	// ActorID / CreatedBy の解決
	actorID := strings.TrimSpace(r.Header.Get("X-Actor-Id"))
	createdBy := strings.TrimSpace(req.CreatedBy)
	if createdBy == "" {
		// ★ フロントから createdBy が来ていない場合は、暫定的に actorID を使う
		createdBy = actorID
	}

	tb, err := h.uc.CreateWithUploads(ctx, uc.CreateBlueprintRequest{
		Name:        strings.TrimSpace(req.Name),
		Symbol:      strings.TrimSpace(req.Symbol),
		BrandID:     strings.TrimSpace(req.BrandID),
		CompanyID:   companyID,
		Description: strings.TrimSpace(req.Description), // ← 空でも OK
		AssigneeID:  strings.TrimSpace(req.AssigneeID),

		CreatedBy: createdBy,
		ActorID:   actorID,

		// ファイルアップロードはこのハンドラでは扱わない
		Icon:     nil,
		Contents: nil,
	})
	if err != nil {
		writeTokenBlueprintErr(w, err)
		return
	}

	resp := h.toResponse(ctx, tb)

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
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

	resp := h.toResponse(ctx, tb)
	_ = json.NewEncoder(w).Encode(resp)
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

	items := make([]tokenBlueprintResponse, 0, len(result.Items))
	for i := range result.Items {
		tb := &result.Items[i]
		items = append(items, h.toResponse(ctx, tb))
	}

	resp := tokenBlueprintPageResponse{
		Items:      items,
		TotalCount: result.TotalCount,
		TotalPages: result.TotalPages,
		Page:       result.Page,
		PerPage:    result.PerPage,
	}

	_ = json.NewEncoder(w).Encode(resp)
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

	resp := h.toResponse(ctx, tb)
	_ = json.NewEncoder(w).Encode(resp)
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
	w.WriteHeader(http.StatusInternalServerError)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
