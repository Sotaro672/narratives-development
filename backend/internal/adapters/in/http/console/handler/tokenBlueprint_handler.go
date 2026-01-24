// backend/internal/adapters/in/http/console/handler/tokenBlueprint_handler.go
package consoleHandler

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"
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

// path helpers -------------------------------------------------------------

func trimSlashSuffix(p string) string {
	p = strings.TrimSpace(p)
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
	return strings.TrimSpace(parts[0])
}

// GCS URL helpers ----------------------------------------------------------

// defaults (あなたの実パスに合わせる)
const (
	defaultTokenIconBucket     = "narratives-development_token_icon"
	defaultTokenContentsBucket = "narratives-development-token-contents"
)

func tokenIconBucketName() string {
	if v := strings.TrimSpace(os.Getenv("TOKEN_ICON_BUCKET")); v != "" {
		return v
	}
	return defaultTokenIconBucket
}

func tokenContentsBucketName() string {
	if v := strings.TrimSpace(os.Getenv("TOKEN_CONTENTS_BUCKET")); v != "" {
		return v
	}
	return defaultTokenContentsBucket
}

func gcsObjectPublicURL(bucket, objectPath string) string {
	b := strings.Trim(strings.TrimSpace(bucket), "/")
	p := strings.TrimLeft(strings.TrimSpace(objectPath), "/")
	if b == "" || p == "" {
		return ""
	}
	u := url.URL{
		Scheme: "https",
		Host:   "storage.googleapis.com",
		Path:   "/" + b + "/" + p,
	}
	return u.String()
}

func withCacheBuster(u string, t time.Time) string {
	u = strings.TrimSpace(u)
	if u == "" {
		return ""
	}
	if t.IsZero() {
		return u
	}
	sep := "?"
	if strings.Contains(u, "?") {
		sep = "&"
	}
	return u + sep + "v=" + strconv.FormatInt(t.UnixNano(), 10)
}

// iconUrl:  https://storage.googleapis.com/narratives-development_token_icon/{id}/icon?v=...
func (h *TokenBlueprintHandler) resolveTokenIconURL(tb *tbdom.TokenBlueprint) string {
	if tb == nil {
		return ""
	}
	id := strings.TrimSpace(tb.ID)
	if id == "" {
		return ""
	}

	objectPath := id + "/icon"
	base := gcsObjectPublicURL(tokenIconBucketName(), objectPath)
	if base == "" {
		return ""
	}

	ver := tb.UpdatedAt
	if ver.IsZero() {
		ver = tb.CreatedAt
	}
	return withCacheBuster(base, ver)
}

// contentsUrl: https://storage.googleapis.com/narratives-development-token-contents/{id}
// NOTE: バケットが private の場合、このURL自体は 403 になる（安定識別子としてのみ利用）。
func (h *TokenBlueprintHandler) resolveTokenContentsURL(tb *tbdom.TokenBlueprint) string {
	if tb == nil {
		return ""
	}
	id := strings.TrimSpace(tb.ID)
	if id == "" {
		return ""
	}
	return gcsObjectPublicURL(tokenContentsBucketName(), id)
}

// DTO --------------------------------------------------------------------

type createTokenBlueprintRequest struct {
	Name        string `json:"name"`
	Symbol      string `json:"symbol"`
	BrandID     string `json:"brandId"`
	Description string `json:"description,omitempty"`
	AssigneeID  string `json:"assigneeId"`

	HasIconFile     bool   `json:"hasIconFile"`
	IconContentType string `json:"iconContentType,omitempty"`
}

type updateTokenBlueprintRequest struct {
	Name        *string `json:"name,omitempty"`
	Symbol      *string `json:"symbol,omitempty"`
	BrandID     *string `json:"brandId,omitempty"`
	Description *string `json:"description,omitempty"`
	AssigneeID  *string `json:"assigneeId,omitempty"`

	ContentFiles *[]tbdom.ContentFile `json:"contentFiles,omitempty"`

	HasIconFile     bool   `json:"hasIconFile"`
	IconContentType string `json:"iconContentType,omitempty"`
}

// token-contents: signed PUT URLs issuance
type issueTokenContentsUploadURLsRequest struct {
	Files []issueTokenContentsUploadURLsFile `json:"files"`
}

type issueTokenContentsUploadURLsFile struct {
	ContentID   string `json:"contentId"`             // required (uuid等)
	Name        string `json:"name"`                  // required
	Type        string `json:"type"`                  // required: "image"|"video"|"pdf"|"document"
	ContentType string `json:"contentType,omitempty"` // optional: e.g. "image/png"
	Size        int64  `json:"size"`                  // required (>=0)
	Visibility  string `json:"visibility,omitempty"`  // optional: "private"|"public" (default private)
}

// ★ A. 構造体ごと返す（推奨）
// upload: 署名付きPUT URL + public URL + view URL + objectPath + expiresAt
type tokenContentUploadItemResponse struct {
	ContentID string                     `json:"contentId"`
	URL       string                     `json:"url"` // 表示用（private bucket は viewUrl を返す）
	Upload    *uc.TokenContentsUploadURL `json:"upload"`
	Content   tbdom.ContentFile          `json:"contentFile"`
}

type issueTokenContentsUploadURLsResponse struct {
	Items []tokenContentUploadItemResponse `json:"items"`
}

// ★ 追加: GET 詳細/更新レスポンスで contentFiles[].url を返すためのラッパー
// - tbdom.ContentFile に url フィールドが無くても返せる
// - 署名URLには cache buster を付けない（署名が壊れるため）
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

	// ★ 変更: contentFiles に url を付与できる型へ
	ContentFiles []contentFileResponse `json:"contentFiles"`

	AssigneeID   string `json:"assigneeId"`
	AssigneeName string `json:"assigneeName"`

	CreatedAt time.Time `json:"createdAt"`

	CreatedByID   string `json:"createdById"`
	CreatedByName string `json:"createdByName"`

	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
	UpdatedBy string     `json:"updatedBy"`

	MetadataURI string `json:"metadataUri"`

	// ★docId から生成
	IconURL     string `json:"iconUrl,omitempty"`
	ContentsURL string `json:"contentsUrl,omitempty"`

	IconUpload *uc.TokenIconUploadURL `json:"iconUpload,omitempty"`
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

	// ★docId から生成
	IconURL     string `json:"iconUrl,omitempty"`
	ContentsURL string `json:"contentsUrl,omitempty"`
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

// ★ 追加: private bucket 表示用の viewUrl を解決する
// NOTE(短期): IssueTokenContentUploadURL を流用して ViewURL を得る（UploadURL はレスポンスに載せない）。
// 将来的には GET専用の署名発行（view専用）を usecase に用意するのが望ましい。
func (h *TokenBlueprintHandler) resolveTokenContentViewURL(ctx context.Context, tokenBlueprintID string, f tbdom.ContentFile) string {
	if h == nil || h.uc == nil {
		return ""
	}
	id := strings.TrimSpace(tokenBlueprintID)
	cid := strings.TrimSpace(f.ID)
	if id == "" || cid == "" {
		return ""
	}

	ct := strings.TrimSpace(f.ContentType)
	if ct == "" {
		ct = "application/octet-stream"
	}

	issued, err := h.uc.IssueTokenContentUploadURL(ctx, id, cid, ct)
	if err != nil || issued == nil {
		return ""
	}

	// private bucket 前提では ViewURL を返す
	if v := strings.TrimSpace(issued.ViewURL); v != "" {
		return v
	}

	// フォールバック（public bucket の場合のみ意味がある）
	if p := strings.TrimSpace(issued.PublicURL); p != "" {
		return p
	}
	return ""
}

func (h *TokenBlueprintHandler) toContentFilesResponse(ctx context.Context, tb *tbdom.TokenBlueprint, includeViewURL bool) []contentFileResponse {
	if tb == nil || len(tb.ContentFiles) == 0 {
		return []contentFileResponse{}
	}

	out := make([]contentFileResponse, 0, len(tb.ContentFiles))
	for _, f := range tb.ContentFiles {
		var viewURL string
		if includeViewURL {
			viewURL = h.resolveTokenContentViewURL(ctx, tb.ID, f)
		}
		out = append(out, contentFileResponse{
			ContentFile: f,
			URL:         strings.TrimSpace(viewURL),
		})
	}
	return out
}

// ★ 変更: includeContentViewURL を追加（list では署名発行を避ける）
func (h *TokenBlueprintHandler) toResponse(ctx context.Context, tb *tbdom.TokenBlueprint, includeContentViewURL bool) tokenBlueprintResponse {
	if tb == nil {
		return tokenBlueprintResponse{}
	}

	var updPtr *time.Time
	if !tb.UpdatedAt.IsZero() {
		t := tb.UpdatedAt
		updPtr = &t
	}

	createdByID := strings.TrimSpace(tb.CreatedBy)

	return tokenBlueprintResponse{
		ID:           strings.TrimSpace(tb.ID),
		Name:         strings.TrimSpace(tb.Name),
		Symbol:       strings.TrimSpace(tb.Symbol),
		BrandID:      strings.TrimSpace(tb.BrandID),
		BrandName:    h.resolveBrandName(ctx, tb.BrandID),
		CompanyID:    strings.TrimSpace(tb.CompanyID),
		Description:  strings.TrimSpace(tb.Description),
		ContentFiles: h.toContentFilesResponse(ctx, tb, includeContentViewURL),

		AssigneeID:   strings.TrimSpace(tb.AssigneeID),
		AssigneeName: h.resolveAssigneeName(ctx, tb.AssigneeID),
		CreatedAt:    tb.CreatedAt,

		CreatedByID:   createdByID,
		CreatedByName: h.resolveCreatorName(ctx, createdByID),

		UpdatedAt:   updPtr,
		UpdatedBy:   strings.TrimSpace(tb.UpdatedBy),
		MetadataURI: strings.TrimSpace(tb.MetadataURI),

		IconURL:     h.resolveTokenIconURL(tb),
		ContentsURL: h.resolveTokenContentsURL(tb),

		IconUpload: nil,
	}
}

func (h *TokenBlueprintHandler) toPatchResponse(ctx context.Context, tb *tbdom.TokenBlueprint) tokenBlueprintPatchResponse {
	if tb == nil {
		return tokenBlueprintPatchResponse{}
	}
	return tokenBlueprintPatchResponse{
		ID:          strings.TrimSpace(tb.ID),
		TokenName:   strings.TrimSpace(tb.Name),
		Symbol:      strings.TrimSpace(tb.Symbol),
		BrandID:     strings.TrimSpace(tb.BrandID),
		BrandName:   h.resolveBrandName(ctx, tb.BrandID),
		Description: strings.TrimSpace(tb.Description),
		Minted:      tb.Minted,
		MetadataURI: strings.TrimSpace(tb.MetadataURI),

		IconURL:     h.resolveTokenIconURL(tb),
		ContentsURL: h.resolveTokenContentsURL(tb),
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

	// TokenBlueprintCard 用 patch
	case r.Method == http.MethodGet &&
		strings.HasPrefix(path, "/token-blueprints/") &&
		strings.HasSuffix(path, "/patch"):
		id := strings.TrimSuffix(strings.TrimPrefix(path, "/token-blueprints/"), "/patch")
		id = strings.Trim(id, "/")
		h.getPatch(w, r, id)
		return

	// ★ token-contents: signed upload urls
	case r.Method == http.MethodPost &&
		strings.HasPrefix(path, "/token-blueprints/") &&
		strings.HasSuffix(path, "/contents/upload-urls"):
		id := strings.TrimSuffix(strings.TrimPrefix(path, "/token-blueprints/"), "/contents/upload-urls")
		id = strings.Trim(id, "/")
		h.issueContentsUploadURLs(w, r, id)
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

	companyID := strings.TrimSpace(uc.CompanyIDFromContext(ctx))
	actorID := strings.TrimSpace(r.Header.Get("X-Actor-Id"))

	log.Printf("[tokenBlueprint_handler] create start companyId=%q actorId=%q", companyID, actorID)

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

	log.Printf("[tokenBlueprint_handler] create request name=%q symbol=%q brandId=%q assigneeId=%q hasIconFile=%v iconContentType=%q",
		strings.TrimSpace(req.Name),
		strings.TrimSpace(req.Symbol),
		strings.TrimSpace(req.BrandID),
		strings.TrimSpace(req.AssigneeID),
		req.HasIconFile,
		strings.TrimSpace(req.IconContentType),
	)

	if strings.TrimSpace(req.Name) == "" ||
		strings.TrimSpace(req.Symbol) == "" ||
		strings.TrimSpace(req.BrandID) == "" ||
		strings.TrimSpace(req.AssigneeID) == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "missing required fields"})
		return
	}

	tb, err := h.uc.Create(ctx, uc.CreateBlueprintRequest{
		Name:        strings.TrimSpace(req.Name),
		Symbol:      strings.TrimSpace(req.Symbol),
		BrandID:     strings.TrimSpace(req.BrandID),
		CompanyID:   companyID,
		Description: strings.TrimSpace(req.Description),
		AssigneeID:  strings.TrimSpace(req.AssigneeID),
		CreatedBy:   actorID,
		ActorID:     actorID,
	})
	if err != nil {
		log.Printf("[tokenBlueprint_handler] create failed err=%v", err)
		writeTokenBlueprintErr(w, err)
		return
	}

	// ★ create も detail と同様に viewUrl を含めて返す（コンテンツが無ければ空）
	resp := h.toResponse(ctx, tb, true)

	if req.HasIconFile {
		ct := strings.TrimSpace(req.IconContentType)
		if ct == "" {
			ct = "application/octet-stream"
		}

		iconUpload, err := h.uc.IssueTokenIconUploadURL(ctx, tb.ID, "", ct)
		if err != nil {
			log.Printf("[tokenBlueprint_handler] iconUpload issue FAILED id=%q err=%v", tb.ID, err)
		} else {
			resp.IconUpload = iconUpload
		}
	}

	_ = json.NewEncoder(w).Encode(resp)
}

// issueContentsUploadURLs --------------------------------------------------
// POST /token-blueprints/{id}/contents/upload-urls
//
// Response で返すもの（フロント実装を最短化）:
// - upload: 署名付き PUT URL / publicUrl / viewUrl / objectPath / expiresAt（構造体ごと）
// - url: 表示用（private bucket は viewUrl を返す。署名URLに cache buster は付けない）
// - contentFile: そのまま PATCH contentFiles に投入できる形（createdAt/By もサーバ統一）
func (h *TokenBlueprintHandler) issueContentsUploadURLs(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	companyID := strings.TrimSpace(uc.CompanyIDFromContext(ctx))
	actorID := strings.TrimSpace(r.Header.Get("X-Actor-Id"))
	id = strings.TrimSpace(id)

	log.Printf("[tokenBlueprint_handler] issueContentsUploadURLs start id=%q companyId(ctx)=%q actorId=%q", id, companyID, actorID)

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
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "id is empty"})
		return
	}

	// tenant boundary check
	tb, err := h.uc.GetByID(ctx, id)
	if err != nil {
		log.Printf("[tokenBlueprint_handler] issueContentsUploadURLs get failed id=%q err=%v", id, err)
		writeTokenBlueprintErr(w, err)
		return
	}
	if strings.TrimSpace(tb.CompanyID) != companyID {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
		return
	}

	var req issueTokenContentsUploadURLsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[tokenBlueprint_handler] issueContentsUploadURLs decode failed id=%q err=%v", id, err)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}
	if len(req.Files) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "files is empty"})
		return
	}

	now := time.Now().UTC()

	items := make([]tokenContentUploadItemResponse, 0, len(req.Files))
	for _, f := range req.Files {
		contentID := strings.TrimSpace(f.ContentID)
		name := strings.TrimSpace(f.Name)
		typ := tbdom.ContentFileType(strings.TrimSpace(f.Type))
		size := f.Size

		if contentID == "" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "contentId is required"})
			return
		}
		if name == "" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "name is required"})
			return
		}
		if !tbdom.IsValidContentType(typ) {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid type"})
			return
		}
		if size < 0 {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid size"})
			return
		}

		ct := strings.TrimSpace(f.ContentType)
		if ct == "" {
			ct = "application/octet-stream"
		}

		vis := tbdom.VisibilityPrivate
		if v := strings.TrimSpace(f.Visibility); v != "" {
			vis = tbdom.ContentVisibility(v)
		}
		if !tbdom.IsValidVisibility(vis) {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid visibility"})
			return
		}

		// ★ Usecase から *TokenContentsUploadURL（構造体）を返す（PUT + GET(view)）
		upload, err := h.uc.IssueTokenContentUploadURL(ctx, id, contentID, ct)
		if err != nil {
			log.Printf("[tokenBlueprint_handler] issueContentsUploadURLs signedUrl FAILED id=%q contentId=%q ct=%q err=%v", id, contentID, ct, err)
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to issue upload url"})
			return
		}
		if upload == nil || strings.TrimSpace(upload.UploadURL) == "" {
			log.Printf("[tokenBlueprint_handler] issueContentsUploadURLs signedUrl EMPTY id=%q contentId=%q", id, contentID)
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to issue upload url"})
			return
		}

		// objectPath は Usecase 側の返却を正とする（空の場合はフォールバック）
		objectPath := strings.TrimSpace(upload.ObjectPath)
		if objectPath == "" {
			objectPath = id + "/" + contentID
		}

		contentFile := tbdom.ContentFile{
			ID:          contentID,
			Name:        name,
			Type:        typ,
			ContentType: ct,
			Size:        size,
			ObjectPath:  objectPath,
			Visibility:  vis,
			CreatedAt:   now,
			CreatedBy:   actorID,
			UpdatedAt:   now,
			UpdatedBy:   actorID,
		}
		if err := contentFile.Validate(); err != nil {
			log.Printf("[tokenBlueprint_handler] issueContentsUploadURLs contentFile invalid id=%q contentId=%q err=%v", id, contentID, err)
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid contentFile"})
			return
		}

		// ★ 表示URL:
		// - private bucket: upload.ViewURL（GET署名）を返す（cache buster は付けない）
		// - fallback: publicUrl + cache buster（public bucket の場合）
		displayURL := strings.TrimSpace(upload.ViewURL)
		if displayURL == "" {
			basePublic := strings.TrimSpace(upload.PublicURL)
			if basePublic == "" {
				basePublic = gcsObjectPublicURL(tokenContentsBucketName(), objectPath)
			}
			displayURL = withCacheBuster(basePublic, now)
		}

		items = append(items, tokenContentUploadItemResponse{
			ContentID: contentID,
			URL:       displayURL,
			Upload:    upload,
			Content:   contentFile,
		})
	}

	log.Printf("[tokenBlueprint_handler] issueContentsUploadURLs success id=%q items=%d", id, len(items))
	_ = json.NewEncoder(w).Encode(issueTokenContentsUploadURLsResponse{Items: items})
}

// getPatch -----------------------------------------------------------------

func (h *TokenBlueprintHandler) getPatch(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	companyID := strings.TrimSpace(uc.CompanyIDFromContext(ctx))
	id = strings.TrimSpace(id)

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

	if strings.TrimSpace(tb.CompanyID) != companyID {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
		return
	}

	_ = json.NewEncoder(w).Encode(h.toPatchResponse(ctx, tb))
}

// get ---------------------------------------------------------------------

func (h *TokenBlueprintHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	companyID := strings.TrimSpace(uc.CompanyIDFromContext(ctx))
	if companyID == "" {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "companyId not found in context"})
		return
	}

	tb, err := h.uc.GetByID(ctx, strings.TrimSpace(id))
	if err != nil {
		writeTokenBlueprintErr(w, err)
		return
	}

	// ★ 追加: tenant boundary check（越境参照防止）
	if strings.TrimSpace(tb.CompanyID) != companyID {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
		return
	}

	// ★ 詳細は viewUrl を含めて返す
	_ = json.NewEncoder(w).Encode(h.toResponse(ctx, tb, true))
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

	q := r.URL.Query()
	brandID := strings.TrimSpace(q.Get("brandId"))
	mintedFilter := strings.TrimSpace(q.Get("minted"))

	page := tbdom.Page{Number: pageNum, PerPage: perPage}

	var (
		result tbdom.PageResult
		err    error
	)

	switch {
	case brandID != "" && mintedFilter == "":
		result, err = h.uc.ListByBrandID(ctx, brandID, page)
	case mintedFilter == "notYet":
		result, err = h.uc.ListMintedNotYet(ctx, page)
	case mintedFilter == "minted":
		result, err = h.uc.ListMintedCompleted(ctx, page)
	default:
		result, err = h.uc.ListByCompanyID(ctx, companyID, page)
	}

	if err != nil {
		writeTokenBlueprintErr(w, err)
		return
	}

	items := make([]tokenBlueprintResponse, 0, len(result.Items))
	for i := range result.Items {
		// ★ list では署名発行を避ける（重いので url は空）
		items = append(items, h.toResponse(ctx, &result.Items[i], false))
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

	if companyID == "" {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "companyId not found in context"})
		return
	}
	// ★ 追加: 更新は actor 必須
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
	if strings.TrimSpace(tb.CompanyID) != companyID {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
		return
	}

	updated, err := h.uc.Update(ctx, uc.UpdateBlueprintRequest{
		ID:           id,
		Name:         req.Name,
		Symbol:       req.Symbol,
		BrandID:      req.BrandID,
		Description:  req.Description,
		AssigneeID:   req.AssigneeID,
		ContentFiles: req.ContentFiles,
		ActorID:      actorID,
	})
	if err != nil {
		writeTokenBlueprintErr(w, err)
		return
	}

	// ★ update は viewUrl を含めて返す（直後の画面表示のため）
	resp := h.toResponse(ctx, updated, true)

	if req.HasIconFile {
		ct := strings.TrimSpace(req.IconContentType)
		if ct == "" {
			ct = "application/octet-stream"
		}

		iconUpload, err := h.uc.IssueTokenIconUploadURL(ctx, updated.ID, "", ct)
		if err != nil {
			log.Printf("[tokenBlueprint_handler] iconUpload issue FAILED (update) id=%q err=%v", updated.ID, err)
		} else {
			resp.IconUpload = iconUpload
		}
	}

	_ = json.NewEncoder(w).Encode(resp)
}

// delete ------------------------------------------------------------------

func (h *TokenBlueprintHandler) delete(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	id = strings.TrimSpace(id)

	companyID := strings.TrimSpace(uc.CompanyIDFromContext(ctx))

	tb, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeTokenBlueprintErr(w, err)
		return
	}
	if strings.TrimSpace(tb.CompanyID) != companyID {
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
