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
	reviewreport "narratives/internal/domain/reviewReport"
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
	) (reviewreport.AddReportResult, error)
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
		h.handleReviewReport(w, r, route.ProductBlueprintID, route.ReviewID, isMe)
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
	switch r.Method {
	case http.MethodGet:
		h.handleList(w, r, productBlueprintID)
	case http.MethodPost:
		if !isMe {
			writeJSONError(w, http.StatusMethodNotAllowed, "POST not allowed on public catalog")
			return
		}
		h.handleCreateMe(w, r, productBlueprintID)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *ProductBlueprintReviewHandler) handleReviewReport(
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
		writeJSONError(w, http.StatusServiceUnavailable, "review report service not configured")
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

	var req reportProductBlueprintReviewRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	reason := reviewreport.ReportReason(
		strings.ToUpper(strings.TrimSpace(req.Reason)),
	)
	if err := reason.Validate(); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid report reason")
		return
	}

	if reason == reviewreport.ReportReasonOther && strings.TrimSpace(req.Detail) == "" {
		writeJSONError(w, http.StatusBadRequest, "report detail required")
		return
	}

	result, err := h.reportSvc.ReportProductBlueprintReviewByAvatar(
		r.Context(),
		uc.ReportProductBlueprintReviewByAvatarInput{
			ProductBlueprintID: productBlueprintID,
			ReviewID:           reviewID,
			AvatarID:           avatarID,
			Reason:             reason,
			Detail:             req.Detail,
		},
	)
	if err != nil {
		writeReviewReportError(w, err)
		return
	}

	statusCode := http.StatusCreated
	if !result.ReportCreated {
		statusCode = http.StatusOK
	}

	writeJSON(w, statusCode, toProductBlueprintReviewReportResponse(result))
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
	Title            string `json:"title"`
	Body             string `json:"body"`
	HelpfulVotes     int    `json:"helpfulVotes"`
	TotalVotes       int    `json:"totalVotes"`
	ReviewedAt       string `json:"reviewedAt"`
	Status           string `json:"status"`
	AvatarName       string `json:"avatarName"`
	AvatarIcon       string `json:"avatarIcon"`
}

type productBlueprintReviewReportResponse struct {
	CaseID        string                  `json:"caseId"`
	ReportID      string                  `json:"reportId"`
	ReportCount   int                     `json:"reportCount"`
	Status        reviewreport.CaseStatus `json:"status"`
	CaseCreated   bool                    `json:"caseCreated"`
	ReportCreated bool                    `json:"reportCreated"`
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
		Title:            v.Title,
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
		Title:            v.Title,
		Body:             v.Body,
		HelpfulVotes:     v.HelpfulVotes,
		TotalVotes:       v.TotalVotes,
		ReviewedAt:       reviewedAt,
		Status:           string(v.Status),
		AvatarName:       "",
		AvatarIcon:       "",
	}
}

func toProductBlueprintReviewReportResponse(
	result reviewreport.AddReportResult,
) productBlueprintReviewReportResponse {
	return productBlueprintReviewReportResponse{
		CaseID:        string(result.Case.ID),
		ReportID:      string(result.Report.ID),
		ReportCount:   result.Case.ReportCount,
		Status:        result.Case.Status,
		CaseCreated:   result.CaseCreated,
		ReportCreated: result.ReportCreated,
	}
}

// ============================================================
// Request DTO
// ============================================================

type createProductBlueprintReviewRequest struct {
	Rating     int       `json:"rating"`
	Title      string    `json:"title"`
	Body       string    `json:"body"`
	ReviewedAt time.Time `json:"reviewedAt"`
	CreatedAt  time.Time `json:"createdAt"`
}

type reportProductBlueprintReviewRequest struct {
	Reason string `json:"reason"`
	Detail string `json:"detail"`
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
	q := r.URL.Query()

	page := parsePositiveIntDefault(q.Get("page"), 1)
	perPage := parsePositiveIntDefault(q.Get("perPage"), 20)

	if perPage > 100 {
		perPage = 100
	}

	return domcommon.Page{
		Number:  page,
		PerPage: perPage,
	}
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

func writeReviewReportError(w http.ResponseWriter, err error) {
	if err == nil {
		writeJSONError(w, http.StatusInternalServerError, "unknown error")
		return
	}

	switch {
	case errors.Is(err, uc.ErrReviewReportUsecaseNotConfigured):
		writeJSONError(w, http.StatusServiceUnavailable, "review report service not configured")
	case errors.Is(err, uc.ErrReviewReportForbidden):
		writeJSONError(w, http.StatusForbidden, "review report forbidden")
	case errors.Is(err, uc.ErrReviewReportSelfReport):
		writeJSONError(w, http.StatusForbidden, "self report is not allowed")
	case errors.Is(err, reviewreport.ErrCannotReportRemovedTarget):
		writeJSONError(w, http.StatusConflict, "cannot report removed target")
	case reviewreport.IsInvalid(err):
		writeJSONError(w, http.StatusBadRequest, err.Error())
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
