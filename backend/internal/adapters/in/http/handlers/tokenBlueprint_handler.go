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
	branddom "narratives/internal/domain/brand"
	memdom "narratives/internal/domain/member"
	tbdom "narratives/internal/domain/tokenBlueprint"
	tidom "narratives/internal/domain/tokenIcon"
)

// TokenBlueprintHandler handles /token-blueprints endpoints.
type TokenBlueprintHandler struct {
	uc       *uc.TokenBlueprintUsecase
	memSvc   *memdom.Service
	brandSvc *branddom.Service

	// ★ 追加: iconId -> iconUrl 解決用
	tiRepo tidom.RepositoryPort
}

// 後方互換: 既存のDIが壊れないように、tiRepo は nil で生成できる形を残す
func NewTokenBlueprintHandler(
	ucase *uc.TokenBlueprintUsecase,
	memSvc *memdom.Service,
	brandSvc *branddom.Service,
) http.Handler {
	return &TokenBlueprintHandler{
		uc:       ucase,
		memSvc:   memSvc,
		brandSvc: brandSvc,
		tiRepo:   nil,
	}
}

// ★ 推奨: TokenIconRepo を渡して iconUrl を解決できるコンストラクタ
func NewTokenBlueprintHandlerWithTokenIconRepo(
	ucase *uc.TokenBlueprintUsecase,
	memSvc *memdom.Service,
	brandSvc *branddom.Service,
	tiRepo tidom.RepositoryPort,
) http.Handler {
	return &TokenBlueprintHandler{
		uc:       ucase,
		memSvc:   memSvc,
		brandSvc: brandSvc,
		tiRepo:   tiRepo,
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
	IconURL      string     `json:"iconUrl,omitempty"` // ★ 追加: iconId -> url を解決した結果
	ContentFiles []string   `json:"contentFiles"`
	AssigneeID   string     `json:"assigneeId"`
	AssigneeName string     `json:"assigneeName"`
	CreatedAt    time.Time  `json:"createdAt"`
	CreatedBy    string     `json:"createdBy"`
	UpdatedAt    *time.Time `json:"updatedAt,omitempty"`
	UpdatedBy    string     `json:"updatedBy"`
	// ★ 追加: Arweave メタデータ URI（公開済みの場合のみ非空）
	MetadataURI string `json:"metadataUri"`
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

// ★ iconId -> iconUrl を解決（解決できない場合は空文字）
func (h *TokenBlueprintHandler) resolveIconURL(ctx context.Context, iconIDPtr *string) string {
	if h == nil || h.tiRepo == nil || iconIDPtr == nil {
		return ""
	}
	id := strings.TrimSpace(*iconIDPtr)
	if id == "" {
		return ""
	}

	ti, err := h.tiRepo.GetByID(ctx, id)
	if err != nil || ti == nil {
		return ""
	}
	return strings.TrimSpace(ti.URL)
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
		IconURL:      h.resolveIconURL(ctx, tb.IconID), // ★ ここで解決
		ContentFiles: tb.ContentFiles,
		AssigneeID:   tb.AssigneeID,
		AssigneeName: h.resolveAssigneeName(ctx, tb.AssigneeID),
		CreatedAt:    tb.CreatedAt,
		CreatedBy:    h.resolveCreatorName(ctx, tb.CreatedBy),
		UpdatedAt:    updPtr,
		UpdatedBy:    tb.UpdatedBy,
		MetadataURI:  tb.MetadataURI, // ★ ドメインからそのまま返す
	}
}

// ServeHTTP ---------------------------------------------------------------

func (h *TokenBlueprintHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/token-blueprints":
		h.create(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/token-blueprints":
		// ★ 一覧取得（クエリにより ListByCompanyID / ListByBrandID / ListMintedNotYet / ListMintedCompleted を切り替え）
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
	actorID := strings.TrimSpace(r.Header.Get("X-Actor-Id"))

	log.Printf("[tokenBlueprint_handler] create start companyId=%q actorId=%q", companyID, actorID)

	if companyID == "" {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "companyId not found in context"})
		return
	}

	var req createTokenBlueprintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[tokenBlueprint_handler] create decode failed err=%v", err)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	log.Printf("[tokenBlueprint_handler] create request name=%q symbol=%q brandId=%q assigneeId=%q createdBy=%q hasIcon=%v contentFiles=%d",
		strings.TrimSpace(req.Name),
		strings.TrimSpace(req.Symbol),
		strings.TrimSpace(req.BrandID),
		strings.TrimSpace(req.AssigneeID),
		strings.TrimSpace(req.CreatedBy),
		req.IconID != nil && strings.TrimSpace(*req.IconID) != "",
		len(req.ContentFiles),
	)

	if strings.TrimSpace(req.Name) == "" ||
		strings.TrimSpace(req.Symbol) == "" ||
		strings.TrimSpace(req.BrandID) == "" ||
		strings.TrimSpace(req.AssigneeID) == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "missing required fields"})
		return
	}

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
		log.Printf("[tokenBlueprint_handler] create failed err=%v", err)
		writeTokenBlueprintErr(w, err)
		return
	}

	log.Printf("[tokenBlueprint_handler] create success id=%q companyId=%q brandId=%q assigneeId=%q minted=%v metadataUri=%q",
		tb.ID, tb.CompanyID, tb.BrandID, tb.AssigneeID, tb.Minted, tb.MetadataURI,
	)

	_ = json.NewEncoder(w).Encode(h.toResponse(ctx, tb))
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
// クエリパラメータで挙動が変わる:
// - ?brandId=xxxx                        → ListByBrandID
// - ?minted=notYet                      → ListMintedNotYet
// - ?minted=minted                      → ListMintedCompleted
// - いずれも指定なし                     → ListByCompanyID (従来動作)
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

	// ★ ログ: 入力条件
	log.Printf("[tokenBlueprint_handler] list start companyId=%q page=%d perPage=%d brandId=%q minted=%q",
		companyID, pageNum, perPage, brandID, mintedFilter,
	)

	var (
		result tbdom.PageResult
		err    error
		mode   string
	)

	switch {
	// ★ brandId ごとの一覧
	case brandID != "" && mintedFilter == "":
		mode = "ListByBrandID"
		result, err = h.uc.ListByBrandID(ctx, brandID, page)

	// ★ minted = notYet のみ
	case mintedFilter == "notYet":
		mode = "ListMintedNotYet"
		result, err = h.uc.ListMintedNotYet(ctx, page)

	// ★ minted = minted のみ
	case mintedFilter == "minted":
		mode = "ListMintedCompleted"
		result, err = h.uc.ListMintedCompleted(ctx, page)

	// ★ デフォルト: companyId 単位の一覧（従来挙動）
	default:
		mode = "ListByCompanyID"
		result, err = h.uc.ListByCompanyID(ctx, companyID, page)
	}

	if err != nil {
		log.Printf("[tokenBlueprint_handler] list failed mode=%s companyId=%q err=%v", mode, companyID, err)
		writeTokenBlueprintErr(w, err)
		return
	}

	// ★ ログ: 取得結果（件数 + サンプル）
	sampleIDs := make([]string, 0, 5)
	sample := make([]map[string]any, 0, 3)
	for i := range result.Items {
		if len(sampleIDs) < 5 {
			sampleIDs = append(sampleIDs, strings.TrimSpace(result.Items[i].ID))
		}
		if len(sample) < 3 {
			tb := result.Items[i]
			sample = append(sample, map[string]any{
				"id":         strings.TrimSpace(tb.ID),
				"companyId":  strings.TrimSpace(tb.CompanyID),
				"brandId":    strings.TrimSpace(tb.BrandID),
				"assigneeId": strings.TrimSpace(tb.AssigneeID),
				"minted":     tb.Minted,
				"createdAt":  tb.CreatedAt,
				"updatedAt":  tb.UpdatedAt,
			})
		}
	}

	log.Printf("[tokenBlueprint_handler] list success mode=%s companyId=%q totalCount=%d page=%d perPage=%d items=%d sampleIds=%v sample=%v",
		mode, companyID, result.TotalCount, result.Page, result.PerPage, len(result.Items), sampleIDs, sample,
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

	// 入力の概要だけログ（値そのものは必要最低限）
	log.Printf("[tokenBlueprint_handler] update request id=%q hasName=%v hasSymbol=%v hasBrandId=%v hasDesc=%v hasAssignee=%v hasIconId=%v hasContentFiles=%v",
		id,
		req.Name != nil,
		req.Symbol != nil,
		req.BrandID != nil,
		req.Description != nil,
		req.AssigneeID != nil,
		req.IconID != nil,
		req.ContentFiles != nil,
	)

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
		log.Printf("[tokenBlueprint_handler] update failed id=%q err=%v", id, err)
		writeTokenBlueprintErr(w, err)
		return
	}

	log.Printf("[tokenBlueprint_handler] update success id=%q companyId=%q brandId=%q assigneeId=%q minted=%v metadataUri=%q",
		tb.ID, tb.CompanyID, tb.BrandID, tb.AssigneeID, tb.Minted, tb.MetadataURI,
	)

	_ = json.NewEncoder(w).Encode(h.toResponse(ctx, tb))
}

// delete ------------------------------------------------------------------

func (h *TokenBlueprintHandler) delete(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	id = strings.TrimSpace(id)

	companyID := strings.TrimSpace(uc.CompanyIDFromContext(ctx))
	actorID := strings.TrimSpace(r.Header.Get("X-Actor-Id"))

	log.Printf("[tokenBlueprint_handler] delete start id=%q companyId(ctx)=%q actorId=%q", id, companyID, actorID)

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
	w.WriteHeader(http.StatusInternalServerError)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
