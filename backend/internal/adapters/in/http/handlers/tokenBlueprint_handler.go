// backend/internal/adapters/in/http/handlers/tokenBlueprint_handler.go
package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	appresolver "narratives/internal/application/resolver"
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

	// ★ iconId -> iconUrl 解決用（任意）
	tiRepo tidom.RepositoryPort

	// ★ NEW: Front から iconUrl をそのまま受け、保存用/返却用に正規化する
	imgResolver *appresolver.ImageURLResolver
}

// 後方互換: 既存のDIが壊れないように、tiRepo は nil で生成できる形を残す
func NewTokenBlueprintHandler(
	ucase *uc.TokenBlueprintUsecase,
	memSvc *memdom.Service,
	brandSvc *branddom.Service,
) http.Handler {
	return &TokenBlueprintHandler{
		uc:          ucase,
		memSvc:      memSvc,
		brandSvc:    brandSvc,
		tiRepo:      nil,
		imgResolver: appresolver.NewImageURLResolver(readTokenIconPublicBucket()),
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
		uc:          ucase,
		memSvc:      memSvc,
		brandSvc:    brandSvc,
		tiRepo:      tiRepo,
		imgResolver: appresolver.NewImageURLResolver(readTokenIconPublicBucket()),
	}
}

// env helper ---------------------------------------------------------------

// 既存環境差分に耐えるため、いくつか候補を見て bucket 名を決める
func readTokenIconPublicBucket() string {
	candidates := []string{
		"TOKEN_ICON_PUBLIC_BUCKET",
		"TOKEN_ICON_BUCKET",
		"TOKEN_ICON_BUCKET_NAME",
		"GCS_TOKEN_ICON_BUCKET",
	}
	for _, k := range candidates {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
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

	// 互換のため残しているが、方針Aでは create で iconId を受け取っても使用しない
	IconID *string `json:"iconId,omitempty"`

	// ★ NEW: Front から icon の URL をそのまま受け取る（保存時は resolver で objectPath に正規化）
	// - create では通常「署名付きPUT→update」で反映するため、ここでは互換として受け取るだけ（未使用）
	IconURL *string `json:"iconUrl,omitempty"`
}

type updateTokenBlueprintRequest struct {
	Name         *string   `json:"name,omitempty"`
	Symbol       *string   `json:"symbol,omitempty"`
	BrandID      *string   `json:"brandId,omitempty"`
	Description  *string   `json:"description,omitempty"`
	AssigneeID   *string   `json:"assigneeId,omitempty"`
	IconID       *string   `json:"iconId,omitempty"` // ★ 方針A: objectPath（例: "{docId}/icon"）を想定
	ContentFiles *[]string `json:"contentFiles,omitempty"`

	// ★ NEW: Front から iconUrl を渡された場合、backend で objectPath へ正規化して IconID に入れて保存する
	IconURL *string `json:"iconUrl,omitempty"`
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
	IconURL      string     `json:"iconUrl,omitempty"` // ★ iconId -> url（or resolver）
	ContentFiles []string   `json:"contentFiles"`
	AssigneeID   string     `json:"assigneeId"`
	AssigneeName string     `json:"assigneeName"`
	CreatedAt    time.Time  `json:"createdAt"`
	CreatedBy    string     `json:"createdBy"`
	UpdatedAt    *time.Time `json:"updatedAt,omitempty"`
	UpdatedBy    string     `json:"updatedBy"`

	// ★ Arweave メタデータ URI（公開済みの場合のみ非空）
	MetadataURI string `json:"metadataUri"`

	// ★ 署名付きURLで直接PUTするための情報（任意で返す）
	IconUpload *signedIconUploadResponse `json:"iconUpload,omitempty"`
}

type signedIconUploadResponse struct {
	UploadURL   string     `json:"uploadUrl"`
	PublicURL   string     `json:"publicUrl"`
	ObjectPath  string     `json:"objectPath"`
	ExpiresAt   *time.Time `json:"expiresAt,omitempty"`
	ContentType string     `json:"contentType"`
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

// ★ iconId -> iconUrl を解決（解決できない場合は空文字）
//
// 優先順位:
// 1) TokenIconRepository（tiRepo）があればそれで解決（既存互換）
// 2) だめなら ImageURLResolver（bucket + objectPath）で public URL を生成
func (h *TokenBlueprintHandler) resolveIconURL(ctx context.Context, iconIDPtr *string) string {
	if h == nil || iconIDPtr == nil {
		return ""
	}
	id := strings.TrimSpace(*iconIDPtr)
	if id == "" {
		return ""
	}

	// 1) 既存: TokenIconRepo で引けるならそれを優先
	if h.tiRepo != nil {
		ti, err := h.tiRepo.GetByID(ctx, id)
		if err == nil && ti != nil {
			u := strings.TrimSpace(ti.URL)
			if u != "" {
				return u
			}
		}
	}

	// 2) 新: bucket + objectPath から組み立て（iconId が objectPath の想定）
	if h.imgResolver != nil {
		return h.imgResolver.ResolveForResponse(id, "")
	}

	return ""
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
		IconURL:      h.resolveIconURL(ctx, tb.IconID),
		ContentFiles: tb.ContentFiles,
		AssigneeID:   tb.AssigneeID,
		AssigneeName: h.resolveAssigneeName(ctx, tb.AssigneeID),
		CreatedAt:    tb.CreatedAt,
		CreatedBy:    h.resolveCreatorName(ctx, tb.CreatedBy),
		UpdatedAt:    updPtr,
		UpdatedBy:    tb.UpdatedBy,
		MetadataURI:  tb.MetadataURI,
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

	// ★ 追加: 署名付きURL発行（id は /token-blueprints/{id}/icon-upload-url の {id}）
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/token-blueprints/") && strings.HasSuffix(r.URL.Path, "/icon-upload-url"):
		id := strings.TrimPrefix(r.URL.Path, "/token-blueprints/")
		id = strings.TrimSuffix(id, "/icon-upload-url")
		h.issueIconUploadURL(w, r, id)

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

	log.Printf("[tokenBlueprint_handler] create request name=%q symbol=%q brandId=%q assigneeId=%q createdBy=%q contentFiles=%d iconIdProvided=%v iconUrlProvided=%v",
		strings.TrimSpace(req.Name),
		strings.TrimSpace(req.Symbol),
		strings.TrimSpace(req.BrandID),
		strings.TrimSpace(req.AssigneeID),
		strings.TrimSpace(req.CreatedBy),
		len(req.ContentFiles),
		req.IconID != nil && strings.TrimSpace(*req.IconID) != "",
		req.IconURL != nil && strings.TrimSpace(*req.IconURL) != "",
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
		// ★ 方針A: create は icon バイナリを扱わない（署名付きURLでPUT）
	})
	if err != nil {
		log.Printf("[tokenBlueprint_handler] create failed err=%v", err)
		writeTokenBlueprintErr(w, err)
		return
	}

	// ★ 作成直後に「署名付きPUT URL」を返す（可能なら）
	uploadCT := strings.TrimSpace(r.Header.Get("X-Icon-Content-Type"))
	if uploadCT == "" {
		uploadCT = strings.TrimSpace(r.URL.Query().Get("iconContentType"))
	}
	uploadFN := strings.TrimSpace(r.Header.Get("X-Icon-File-Name"))
	if uploadFN == "" {
		uploadFN = strings.TrimSpace(r.URL.Query().Get("iconFileName"))
	}
	if uploadCT == "" {
		uploadCT = "application/octet-stream"
	}

	var iconUploadResp *signedIconUploadResponse
	upl, uerr := h.uc.IssueTokenIconUploadURL(ctx, tb.ID, uploadFN, uploadCT)
	if uerr != nil {
		log.Printf("[tokenBlueprint_handler] issue icon upload url failed id=%q err=%v", tb.ID, uerr)
	} else if upl != nil {
		iconUploadResp = &signedIconUploadResponse{
			UploadURL:   strings.TrimSpace(upl.UploadURL),
			PublicURL:   strings.TrimSpace(upl.PublicURL),
			ObjectPath:  strings.TrimSpace(upl.ObjectPath),
			ExpiresAt:   upl.ExpiresAt,
			ContentType: strings.TrimSpace(uploadCT),
		}
	}

	log.Printf("[tokenBlueprint_handler] create success id=%q companyId=%q brandId=%q assigneeId=%q minted=%v metadataUri=%q iconUploadIssued=%v",
		tb.ID, tb.CompanyID, tb.BrandID, tb.AssigneeID, tb.Minted, tb.MetadataURI, iconUploadResp != nil,
	)

	resp := h.toResponse(ctx, tb)
	resp.IconUpload = iconUploadResp
	_ = json.NewEncoder(w).Encode(resp)
}

// issueIconUploadURL -------------------------------------------------------

// GET /token-blueprints/{id}/icon-upload-url?contentType=image/png&fileName=xxx.png
// - 署名付きPUT URLを発行して返す
func (h *TokenBlueprintHandler) issueIconUploadURL(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	companyID := strings.TrimSpace(uc.CompanyIDFromContext(ctx))
	actorID := strings.TrimSpace(r.Header.Get("X-Actor-Id"))
	id = strings.TrimSpace(id)

	log.Printf("[tokenBlueprint_handler] issueIconUploadURL start id=%q companyId(ctx)=%q actorId=%q", id, companyID, actorID)

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

	// テナント境界チェック（companyId一致）
	tb, err := h.uc.GetByID(ctx, id)
	if err != nil {
		log.Printf("[tokenBlueprint_handler] issueIconUploadURL get failed id=%q err=%v", id, err)
		writeTokenBlueprintErr(w, err)
		return
	}
	if strings.TrimSpace(tb.CompanyID) != companyID {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
		return
	}

	q := r.URL.Query()
	ct := strings.TrimSpace(q.Get("contentType"))
	if ct == "" {
		ct = strings.TrimSpace(r.Header.Get("X-Icon-Content-Type"))
	}
	if ct == "" {
		ct = "application/octet-stream"
	}
	fn := strings.TrimSpace(q.Get("fileName"))
	if fn == "" {
		fn = strings.TrimSpace(r.Header.Get("X-Icon-File-Name"))
	}

	upl, err := h.uc.IssueTokenIconUploadURL(ctx, id, fn, ct)
	if err != nil {
		log.Printf("[tokenBlueprint_handler] issueIconUploadURL failed id=%q err=%v", id, err)
		writeTokenBlueprintErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(signedIconUploadResponse{
		UploadURL:   strings.TrimSpace(upl.UploadURL),
		PublicURL:   strings.TrimSpace(upl.PublicURL),
		ObjectPath:  strings.TrimSpace(upl.ObjectPath),
		ExpiresAt:   upl.ExpiresAt,
		ContentType: strings.TrimSpace(ct),
	})
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

	log.Printf("[tokenBlueprint_handler] get success id=%q companyId=%q brandId=%q assigneeId=%q minted=%v metadataUri=%q iconId=%q",
		tb.ID, tb.CompanyID, tb.BrandID, tb.AssigneeID, tb.Minted, tb.MetadataURI, strings.TrimSpace(func() string {
			if tb.IconID == nil {
				return ""
			}
			return *tb.IconID
		}()),
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

	sampleIDs := make([]string, 0, 5)
	sample := make([]map[string]interface{}, 0, 3)
	for i := range result.Items {
		if len(sampleIDs) < 5 {
			sampleIDs = append(sampleIDs, strings.TrimSpace(result.Items[i].ID))
		}
		if len(sample) < 3 {
			tb := result.Items[i]
			sample = append(sample, map[string]interface{}{
				"id":         strings.TrimSpace(tb.ID),
				"companyId":  strings.TrimSpace(tb.CompanyID),
				"brandId":    strings.TrimSpace(tb.BrandID),
				"assigneeId": strings.TrimSpace(tb.AssigneeID),
				"minted":     tb.Minted,
				"createdAt":  tb.CreatedAt,
				"updatedAt":  tb.UpdatedAt,
				"iconId": func() string {
					if tb.IconID == nil {
						return ""
					}
					return strings.TrimSpace(*tb.IconID)
				}(),
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

	iconIdValue := func() string {
		if req.IconID == nil {
			return ""
		}
		return strings.TrimSpace(*req.IconID)
	}()
	iconUrlValue := func() string {
		if req.IconURL == nil {
			return ""
		}
		return strings.TrimSpace(*req.IconURL)
	}()

	log.Printf("[tokenBlueprint_handler] update request id=%q hasName=%v hasSymbol=%v hasBrandId=%v hasDesc=%v hasAssignee=%v hasIconId=%v hasIconUrl=%v hasContentFiles=%v iconIdValue=%q iconUrlValue=%q",
		id,
		req.Name != nil,
		req.Symbol != nil,
		req.BrandID != nil,
		req.Description != nil,
		req.AssigneeID != nil,
		req.IconID != nil,
		req.IconURL != nil,
		req.ContentFiles != nil,
		iconIdValue,
		iconUrlValue,
	)

	// ★ NEW: iconUrl が来た場合は、保存用に objectPath(iconId) へ正規化して保存する
	// - 既存互換として iconId 直指定も受け付ける
	var (
		resolvedObjectPath string
		resolvedPublicURL  string
		resolvedErr        error
	)

	iconIDPtr := req.IconID
	if iconUrlValue != "" && h.imgResolver != nil {
		resolvedObjectPath, resolvedPublicURL, resolvedErr = h.imgResolver.ResolveForSave(iconUrlValue)
		if resolvedErr != nil {
			log.Printf("[tokenBlueprint_handler] update iconUrl resolve failed id=%q iconUrl=%q err=%v", id, iconUrlValue, resolvedErr)
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid iconUrl"})
			return
		}
		if strings.TrimSpace(resolvedObjectPath) != "" {
			// Update usecase へは iconId として渡す
			v := strings.TrimSpace(resolvedObjectPath)
			iconIDPtr = &v
		}
		log.Printf("[tokenBlueprint_handler] update iconUrl resolved id=%q objectPath=%q publicUrl=%q",
			id, resolvedObjectPath, resolvedPublicURL,
		)
	}

	tb, err := h.uc.Update(ctx, uc.UpdateBlueprintRequest{
		ID:           id,
		Name:         req.Name,
		Symbol:       req.Symbol,
		BrandID:      req.BrandID,
		Description:  req.Description,
		AssigneeID:   req.AssigneeID,
		IconID:       iconIDPtr,
		ContentFiles: req.ContentFiles,
		ActorID:      actorID,
	})
	if err != nil {
		log.Printf("[tokenBlueprint_handler] update failed id=%q err=%v", id, err)
		writeTokenBlueprintErr(w, err)
		return
	}

	log.Printf("[tokenBlueprint_handler] update success id=%q companyId=%q brandId=%q assigneeId=%q minted=%v metadataUri=%q iconId=%q",
		tb.ID, tb.CompanyID, tb.BrandID, tb.AssigneeID, tb.Minted, tb.MetadataURI,
		strings.TrimSpace(func() string {
			if tb.IconID == nil {
				return ""
			}
			return *tb.IconID
		}()),
	)

	resp := h.toResponse(ctx, tb)

	// ★ 要件: 「保存リクエストが来た場合は加工したURLを返す」
	// iconUrl を受け取ったケースでは、resolver が計算した publicUrl（正規化済み）を優先して返す
	if strings.TrimSpace(resolvedPublicURL) != "" {
		resp.IconURL = strings.TrimSpace(resolvedPublicURL)
	}

	_ = json.NewEncoder(w).Encode(resp)
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
