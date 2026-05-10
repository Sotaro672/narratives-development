// backend/internal/adapters/in/http/mall/handler/productBlueprintReview_handler.go
package mallHandler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"narratives/internal/adapters/in/http/middleware"
	domcommon "narratives/internal/domain/common"
	pbr "narratives/internal/domain/productBlueprintReview"

	uc "narratives/internal/application/usecase"
)

// ============================================================
// Port (usecase-facing)
// ============================================================

// ProductBlueprintReviewService is the application port used by this HTTP handler.
// NOTE: この interface を満たす実装（usecase）を DI してください。
type ProductBlueprintReviewService interface {
	// List (閲覧)
	ListByProductBlueprintID(
		ctx context.Context,
		productBlueprintID string,
		status pbr.ReviewStatus,
		page domcommon.Page,
	) (domcommon.PageResult[pbr.Review], error)

	// ✅ NEW: List (閲覧) + AvatarName/Icon
	ListByProductBlueprintIDWithAvatar(
		ctx context.Context,
		productBlueprintID string,
		status pbr.ReviewStatus,
		page domcommon.Page,
	) (domcommon.PageResult[uc.ProductBlueprintReviewListItem], error)

	// VerifiedPurchase 判定（投稿可否の事前チェック用）
	IsVerifiedPurchase(
		ctx context.Context,
		avatarID string,
		productBlueprintID string,
	) (bool, error)

	// Create (投稿)
	// ✅ usecase 側の Input 型を使う（型不一致の解消）
	CreateProductBlueprintReview(
		ctx context.Context,
		in uc.CreateProductBlueprintReviewInput,
	) (pbr.Review, error)
}

// ============================================================
// Handler
// ============================================================

type ProductBlueprintReviewHandler struct {
	svc ProductBlueprintReviewService
	now func() time.Time
}

func NewProductBlueprintReviewHandler(svc ProductBlueprintReviewService) *ProductBlueprintReviewHandler {
	return &ProductBlueprintReviewHandler{
		svc: svc,
		now: time.Now,
	}
}

func (h *ProductBlueprintReviewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.svc == nil {
		writeJSONError(w, http.StatusInternalServerError, "handler not configured")
		return
	}

	// Router:
	// - /mall/catalog/**  (public) : GET only
	// - /mall/me/catalog/** (auth+avatar) : GET + POST(verified only)
	path := r.URL.Path

	isMe := strings.HasPrefix(path, "/mall/me/catalog")
	isPublic := strings.HasPrefix(path, "/mall/catalog")

	if !isMe && !isPublic {
		http.NotFound(w, r)
		return
	}

	// We handle review endpoints under catalog:
	// GET  /mall/catalog/product-blueprints/{productBlueprintId}/reviews
	// GET  /mall/me/catalog/product-blueprints/{productBlueprintId}/reviews
	// POST /mall/me/catalog/product-blueprints/{productBlueprintId}/reviews  (verified only)
	pbID, ok := extractProductBlueprintID(path, isMe)
	if !ok || pbID == "" {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleList(w, r, pbID)

	case http.MethodPost:
		if !isMe {
			// mall/catalog は閲覧のみ
			writeJSONError(w, http.StatusMethodNotAllowed, "POST not allowed on public catalog")
			return
		}
		h.handleCreateMe(w, r, pbID)

	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *ProductBlueprintReviewHandler) handleList(w http.ResponseWriter, r *http.Request, productBlueprintID string) {
	ctx := r.Context()

	page := parsePage(r)
	status := pbr.ReviewStatusPublished

	// ✅ AvatarName/Icon 付きで返す
	res, err := h.svc.ListByProductBlueprintIDWithAvatar(ctx, productBlueprintID, status, page)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	out := toCatalogReviewPageDTOWithAvatar(res)
	writeJSON(w, http.StatusOK, out)
}

func (h *ProductBlueprintReviewHandler) handleCreateMe(w http.ResponseWriter, r *http.Request, productBlueprintID string) {
	ctx := r.Context()

	// ✅ IMPORTANT:
	// avatarId は middleware が request context に積む前提。
	// ctx.Value("avatarId") のようなキー直読みは middleware 実装差分で壊れやすいので、
	// wallet handler と同様に middleware getter を正とする。
	avatarID, ok := middleware.CurrentAvatarID(r)
	if !ok || avatarID == "" {
		writeJSONError(w, http.StatusUnauthorized, "missing avatarId")
		return
	}

	// ✅ VerifiedPurchase:true のみ投稿可（事前チェック）
	ok2, err := h.svc.IsVerifiedPurchase(ctx, avatarID, productBlueprintID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	if !ok2 {
		writeJSONError(w, http.StatusForbidden, "verified purchase required")
		return
	}

	var req createProductBlueprintReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	now := h.now().UTC()

	createdAt := req.CreatedAt
	if createdAt.IsZero() {
		createdAt = now
	}
	reviewedAt := req.ReviewedAt
	if reviewedAt.IsZero() {
		reviewedAt = now
	}

	// ✅ usecase.Input 型で作る（entity.go を正としてフィールドを合わせる）
	in := uc.CreateProductBlueprintReviewInput{
		ProductBlueprintID: productBlueprintID,
		AvatarID:           avatarID,
		Rating:             pbr.Rating(req.Rating),
		Title:              req.Title,
		Body:               req.Body,
		ReviewedAt:         reviewedAt,
		CreatedAt:          createdAt,
		CreatedBy:          avatarID,
		PublishNow:         true,
	}

	created, err := h.svc.CreateProductBlueprintReview(ctx, in)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, toCatalogReviewDTO(created))
}

// ============================================================
// Response DTO (名揺れ排除: lowerCamelCase 固定)
// ============================================================

type catalogReviewPageDTO struct {
	Items   []catalogReviewDTO `json:"items"`
	Page    int                `json:"page"`
	PerPage int                `json:"perPage"`
	Total   int                `json:"total"`
	HasNext bool               `json:"hasNext"`
}

type catalogReviewDTO struct {
	ID               string `json:"id"`
	ProductBlueprint string `json:"productBlueprintId"`
	AvatarID         string `json:"avatarId"`
	Rating           int    `json:"rating"`
	Title            string `json:"title"`
	Body             string `json:"body"`
	HelpfulVotes     int    `json:"helpfulVotes"`
	TotalVotes       int    `json:"totalVotes"`
	ReviewedAt       string `json:"reviewedAt"` // RFC3339
	Status           string `json:"status"`

	// ✅ NEW: 画面に渡す
	AvatarName string `json:"avatarName"`
	AvatarIcon string `json:"avatarIcon"`
}

func toCatalogReviewPageDTOWithAvatar(res domcommon.PageResult[uc.ProductBlueprintReviewListItem]) catalogReviewPageDTO {
	items := make([]catalogReviewDTO, 0, len(res.Items))
	for _, it := range res.Items {
		items = append(items, toCatalogReviewDTOWithAvatar(it))
	}

	page := res.Page
	if page <= 0 {
		page = 1
	}
	perPage := res.PerPage
	if perPage <= 0 {
		perPage = 20
	}

	total := res.TotalCount
	hasNext := false
	if res.TotalPages > 0 {
		hasNext = page < res.TotalPages
	} else {
		// 念のためのフォールバック
		hasNext = len(items) >= perPage
	}

	return catalogReviewPageDTO{
		Items:   items,
		Page:    page,
		PerPage: perPage,
		Total:   total,
		HasNext: hasNext,
	}
}

func toCatalogReviewDTOWithAvatar(v uc.ProductBlueprintReviewListItem) catalogReviewDTO {
	reviewedAt := ""
	if !v.ReviewedAt.IsZero() {
		reviewedAt = v.ReviewedAt.UTC().Format(time.RFC3339Nano)
	}

	return catalogReviewDTO{
		ID:               string(v.ID),
		ProductBlueprint: v.ProductBlueprintID,
		AvatarID:         v.AvatarID,
		Rating:           int(v.Rating),
		Title:            v.Title,
		Body:             v.Body,
		HelpfulVotes:     v.HelpfulVotes,
		TotalVotes:       v.TotalVotes,
		ReviewedAt:       reviewedAt,
		Status:           string(v.Status),

		AvatarName: v.AvatarName,
		AvatarIcon: v.AvatarIcon,
	}
}

// 既存 Create のレスポンスはそのまま（必要ならここにも付けられるが、Create は avatarID=me なので今は不要）
func toCatalogReviewDTO(v pbr.Review) catalogReviewDTO {
	reviewedAt := ""
	if !v.ReviewedAt.IsZero() {
		reviewedAt = v.ReviewedAt.UTC().Format(time.RFC3339Nano)
	}

	return catalogReviewDTO{
		ID:               string(v.ID),
		ProductBlueprint: v.ProductBlueprintID,
		AvatarID:         v.AvatarID,
		Rating:           int(v.Rating),
		Title:            v.Title,
		Body:             v.Body,
		HelpfulVotes:     v.HelpfulVotes,
		TotalVotes:       v.TotalVotes,
		ReviewedAt:       reviewedAt,
		Status:           string(v.Status),

		AvatarName: "",
		AvatarIcon: "",
	}
}

// ============================================================
// Request DTO
// ============================================================

type createProductBlueprintReviewRequest struct {
	Rating     int       `json:"rating"` // 1..5
	Title      string    `json:"title"`
	Body       string    `json:"body"`
	ReviewedAt time.Time `json:"reviewedAt"`
	CreatedAt  time.Time `json:"createdAt"`
}

// ============================================================
// Path parsing
// ============================================================

// extractProductBlueprintID parses:
// /mall/catalog/product-blueprints/{id}/reviews
// /mall/me/catalog/product-blueprints/{id}/reviews
func extractProductBlueprintID(path string, isMe bool) (string, bool) {
	base := "/mall/catalog/"
	if isMe {
		base = "/mall/me/catalog/"
	}
	if !strings.HasPrefix(path, base) {
		return "", false
	}

	rest := path[len(base):]
	parts := splitPath(rest)
	if len(parts) < 3 {
		return "", false
	}
	if parts[0] != "product-blueprints" {
		return "", false
	}
	if parts[2] != "reviews" {
		return "", false
	}
	return parts[1], true
}

func splitPath(p string) []string {
	for len(p) > 0 && p[0] == '/' {
		p = p[1:]
	}
	for len(p) > 0 && p[len(p)-1] == '/' {
		p = p[:len(p)-1]
	}
	if p == "" {
		return nil
	}
	return strings.Split(p, "/")
}

// ============================================================
// Query parsing
// ============================================================

func parsePage(r *http.Request) domcommon.Page {
	q := r.URL.Query()
	p := domcommon.Page{Number: 1, PerPage: 20}

	if v := q.Get("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			p.Number = n
		}
	}
	if v := q.Get("perPage"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			p.PerPage = n
		}
	}
	return p
}

// ============================================================
// Context helpers
// ============================================================

func getAvatarIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v := ctx.Value("avatarId"); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	if v := ctx.Value("avatarID"); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// ============================================================
// Error handling / JSON
// ============================================================

func writeDomainError(w http.ResponseWriter, err error) {
	if err == nil {
		writeJSONError(w, http.StatusInternalServerError, "unknown error")
		return
	}

	switch {
	case errors.Is(err, pbr.ErrNotFound):
		writeJSONError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, pbr.ErrConflict):
		writeJSONError(w, http.StatusConflict, err.Error())
	case errors.Is(err, pbr.ErrInvalid):
		writeJSONError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, pbr.ErrUnauthorized):
		writeJSONError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, pbr.ErrForbidden):
		writeJSONError(w, http.StatusForbidden, err.Error())
	default:
		writeJSONError(w, http.StatusInternalServerError, err.Error())
	}
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]any{
		"error": msg,
	})
}
