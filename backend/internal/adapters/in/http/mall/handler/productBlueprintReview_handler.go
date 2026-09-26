// backend/internal/adapters/in/http/mall/handler/productBlueprintReview_handler.go
package mallHandler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"narratives/internal/adapters/in/http/middleware"
	uc "narratives/internal/application/usecase"
	domcommon "narratives/internal/domain/common"
	pbr "narratives/internal/domain/productBlueprintReview"
	reportdom "narratives/internal/domain/report"
)

// ============================================================
// Port (usecase-facing)
// ============================================================

// ProductBlueprintReviewService is the application port used by this HTTP handler.
type ProductBlueprintReviewService interface {
	ListByProductBlueprintID(
		ctx context.Context,
		productBlueprintID string,
		status pbr.ReviewStatus,
		page domcommon.Page,
	) (domcommon.PageResult[uc.ProductBlueprintReviewListItem], error)

	IsVerifiedPurchase(
		ctx context.Context,
		avatarID string,
		productBlueprintID string,
	) (bool, error)

	CreateProductBlueprintReview(
		ctx context.Context,
		in uc.CreateProductBlueprintReviewInput,
	) (pbr.Review, error)
}

// ProductBlueprintReviewReportService owns purchaser-side reporting of
// ProductBlueprint reviews.
type ProductBlueprintReviewReportService interface {
	ReportProductBlueprintReviewByAvatar(
		ctx context.Context,
		input uc.ReportProductBlueprintReviewByAvatarInput,
	) (reportdom.AddReportResult, error)
}

// ============================================================
// Handler
// ============================================================

type ProductBlueprintReviewHandler struct {
	svc       ProductBlueprintReviewService
	reportSvc ProductBlueprintReviewReportService
	now       func() time.Time
}

func NewProductBlueprintReviewHandler(
	svc ProductBlueprintReviewService,
	reportSvc ProductBlueprintReviewReportService,
) *ProductBlueprintReviewHandler {
	return &ProductBlueprintReviewHandler{
		svc:       svc,
		reportSvc: reportSvc,
		now:       time.Now,
	}
}

func (h *ProductBlueprintReviewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.svc == nil {
		writeJSONError(w, http.StatusInternalServerError, "handler not configured")
		return
	}

	path := r.URL.Path
	isMe := strings.HasPrefix(path, "/mall/me/catalog")
	isPublic := strings.HasPrefix(path, "/mall/catalog")

	if !isMe && !isPublic {
		http.NotFound(w, r)
		return
	}

	route, ok := parseProductBlueprintReviewRoute(path, isMe)
	if !ok {
		http.NotFound(w, r)
		return
	}

	switch route.Kind {
	case productBlueprintReviewRouteCollection:
		h.handleReviewCollection(w, r, route.ProductBlueprintID, isMe)
	case productBlueprintReviewRouteReport:
		h.handleReport(w, r, route.ProductBlueprintID, route.ReviewID, isMe)
	default:
		http.NotFound(w, r)
	}
}

func (h *ProductBlueprintReviewHandler) handleReviewCollection(
	w http.ResponseWriter,
	r *http.Request,
	productBlueprintID string,
	isMe bool,
) {
	if isMe {
		if r.Method != http.MethodPost {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		h.handleCreateMe(w, r, productBlueprintID)
		return
	}

	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	h.handleList(w, r, productBlueprintID)
}

func (h *ProductBlueprintReviewHandler) handleReport(
	w http.ResponseWriter,
	r *http.Request,
	productBlueprintID string,
	reviewID string,
	isMe bool,
) {
	if !isMe {
		http.NotFound(w, r)
		return
	}

	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if h.reportSvc == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "report service not configured")
		return
	}

	h.handleReportMe(w, r, productBlueprintID, reviewID)
}

func (h *ProductBlueprintReviewHandler) handleList(
	w http.ResponseWriter,
	r *http.Request,
	productBlueprintID string,
) {
	page := parsePage(r)
	status := pbr.ReviewStatusPublished

	res, err := h.svc.ListByProductBlueprintID(
		r.Context(),
		productBlueprintID,
		status,
		page,
	)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toCatalogReviewPageDTOWithAvatar(res))
}

func (h *ProductBlueprintReviewHandler) handleCreateMe(
	w http.ResponseWriter,
	r *http.Request,
	productBlueprintID string,
) {
	ctx := r.Context()

	avatarID, ok := middleware.CurrentAvatarID(r)
	if !ok || avatarID == "" {
		writeJSONError(w, http.StatusUnauthorized, "missing avatarId")
		return
	}

	verified, err := h.svc.IsVerifiedPurchase(
		ctx,
		avatarID,
		productBlueprintID,
	)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	if !verified {
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

	in := uc.CreateProductBlueprintReviewInput{
		ProductBlueprintID: productBlueprintID,
		AvatarID:           avatarID,
		Rating:             pbr.Rating(req.Rating),
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

func (h *ProductBlueprintReviewHandler) handleReportMe(
	w http.ResponseWriter,
	r *http.Request,
	productBlueprintID string,
	reviewID string,
) {
	avatarID, ok := middleware.CurrentAvatarID(r)
	if !ok || avatarID == "" {
		writeJSONError(w, http.StatusUnauthorized, "missing avatarId")
		return
	}

	reason, detail, ok := decodeReportRequest(w, r)
	if !ok {
		return
	}

	result, err := h.reportSvc.ReportProductBlueprintReviewByAvatar(
		r.Context(),
		uc.ReportProductBlueprintReviewByAvatarInput{
			ProductBlueprintID: productBlueprintID,
			ReviewID:           reviewID,
			AvatarID:           avatarID,
			Reason:             reason,
			Detail:             detail,
		},
	)
	if err != nil {
		writeReportError(w, err)
		return
	}

	writeReportResult(w, result)
}

// ============================================================
// Response DTO
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
	Body             string `json:"body"`
	HelpfulVotes     int    `json:"helpfulVotes"`
	TotalVotes       int    `json:"totalVotes"`
	ReviewedAt       string `json:"reviewedAt"`
	Status           string `json:"status"`
	AvatarName       string `json:"avatarName"`
	AvatarIcon       string `json:"avatarIcon"`
}

func toCatalogReviewPageDTOWithAvatar(
	res domcommon.PageResult[uc.ProductBlueprintReviewListItem],
) catalogReviewPageDTO {
	items := make([]catalogReviewDTO, 0, len(res.Items))
	for _, item := range res.Items {
		items = append(items, toCatalogReviewDTOWithAvatar(item))
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

func toCatalogReviewDTOWithAvatar(
	v uc.ProductBlueprintReviewListItem,
) catalogReviewDTO {
	reviewedAt := ""
	if !v.ReviewedAt.IsZero() {
		reviewedAt = v.ReviewedAt.UTC().Format(time.RFC3339Nano)
	}

	return catalogReviewDTO{
		ID:               string(v.ID),
		ProductBlueprint: v.ProductBlueprintID,
		AvatarID:         v.AvatarID,
		Rating:           int(v.Rating),
		Body:             v.Body,
		HelpfulVotes:     v.HelpfulVotes,
		TotalVotes:       v.TotalVotes,
		ReviewedAt:       reviewedAt,
		Status:           string(v.Status),
		AvatarName:       v.AvatarName,
		AvatarIcon:       v.AvatarIcon,
	}
}

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
		Body:             v.Body,
		HelpfulVotes:     v.HelpfulVotes,
		TotalVotes:       v.TotalVotes,
		ReviewedAt:       reviewedAt,
		Status:           string(v.Status),
		AvatarName:       "",
		AvatarIcon:       "",
	}
}

// ============================================================
// Request DTO
// ============================================================

type createProductBlueprintReviewRequest struct {
	Rating     int       `json:"rating"`
	Body       string    `json:"body"`
	ReviewedAt time.Time `json:"reviewedAt"`
	CreatedAt  time.Time `json:"createdAt"`
}

// ============================================================
// Path parsing
// ============================================================

type productBlueprintReviewRouteKind int

const (
	productBlueprintReviewRouteCollection productBlueprintReviewRouteKind = iota
	productBlueprintReviewRouteReport
)

type productBlueprintReviewRoute struct {
	Kind               productBlueprintReviewRouteKind
	ProductBlueprintID string
	ReviewID           string
}

func parseProductBlueprintReviewRoute(
	path string,
	isMe bool,
) (productBlueprintReviewRoute, bool) {
	base := "/mall/catalog/"
	if isMe {
		base = "/mall/me/catalog/"
	}

	if !strings.HasPrefix(path, base) {
		return productBlueprintReviewRoute{}, false
	}

	parts := splitPath(path[len(base):])

	if len(parts) == 3 &&
		parts[0] == "product-blueprints" &&
		parts[1] != "" &&
		parts[2] == "reviews" {
		return productBlueprintReviewRoute{
			Kind:               productBlueprintReviewRouteCollection,
			ProductBlueprintID: parts[1],
		}, true
	}

	if len(parts) == 5 &&
		parts[0] == "product-blueprints" &&
		parts[1] != "" &&
		parts[2] == "reviews" &&
		parts[3] != "" &&
		parts[4] == "reports" {
		return productBlueprintReviewRoute{
			Kind:               productBlueprintReviewRouteReport,
			ProductBlueprintID: parts[1],
			ReviewID:           parts[3],
		}, true
	}

	return productBlueprintReviewRoute{}, false
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
	return parsePageFromQuery(r, 20, 100)
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
