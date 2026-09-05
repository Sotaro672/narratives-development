// backend/internal/adapters/in/http/console/handler/productBlueprintReview_handler.go
package consoleHandler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"narratives/internal/adapters/in/http/middleware"
	uc "narratives/internal/application/usecase"
	domcommon "narratives/internal/domain/common"
	revdomain "narratives/internal/domain/productBlueprintReview"
	reviewreport "narratives/internal/domain/reviewReport"
)

type ProductBlueprintReviewHandler struct {
	ProductBlueprintReviewUC *uc.ProductBlueprintReviewUsecase
	ReviewReportUC           *uc.ReviewReportUsecase
}

func NewProductBlueprintReviewHandler(
	productBlueprintReviewUC *uc.ProductBlueprintReviewUsecase,
	reviewReportUC *uc.ReviewReportUsecase,
) *ProductBlueprintReviewHandler {
	return &ProductBlueprintReviewHandler{
		ProductBlueprintReviewUC: productBlueprintReviewUC,
		ReviewReportUC:           reviewReportUC,
	}
}

// Supported:
// - GET  /product-blueprint-reviews/aggregates
// - GET  /product-blueprint-reviews?ProductBlueprintID=...
// - POST /product-blueprints/{productBlueprintId}/reviews/{reviewId}/reports
//
// Query Params:
// - Status: PUBLISHED | HIDDEN | REMOVED（default: PUBLISHED）
// - Page: int（default: 1）
// - PerPage: int（default: reviews=20, aggregates=100）
func (h *ProductBlueprintReviewHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if productBlueprintID, reviewID, ok := parseProductBlueprintReviewReportPath(r.URL.Path); ok {
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}

		h.ReportReviewAsBrand(w, r, productBlueprintID, reviewID)
		return
	}

	switch r.Method {
	case http.MethodGet:
		trimmedPath := strings.TrimRight(r.URL.Path, "/")

		if strings.HasSuffix(
			trimmedPath,
			"/product-blueprint-reviews/aggregates",
		) {
			h.ListCompanyReviewAggregates(w, r)
			return
		}

		productBlueprintID := trimWS(
			r.URL.Query().Get("ProductBlueprintID"),
		)
		if productBlueprintID != "" {
			h.ListReviewsByProductBlueprintID(
				w,
				r,
				productBlueprintID,
			)
			return
		}

		writeNotFound(w)

	default:
		methodNotAllowed(w)
	}
}

// ============================================================
// Common helpers
// ============================================================

func (h *ProductBlueprintReviewHandler) resolveCompanyID(
	r *http.Request,
) (string, bool) {
	if member, ok := middleware.CurrentMember(r); ok &&
		member != nil &&
		member.CompanyID != "" {
		return member.CompanyID, true
	}

	if companyID, ok := middleware.CompanyID(r); ok &&
		companyID != "" {
		return companyID, true
	}

	return "", false
}

func (h *ProductBlueprintReviewHandler) resolveProductBlueprintBrandID(
	ctx context.Context,
	productBlueprintID string,
	companyID string,
) (string, error) {
	if h == nil ||
		h.ProductBlueprintReviewUC == nil ||
		h.ProductBlueprintReviewUC.ProductBlueprintRepo == nil {
		return "", revdomain.ErrInternal
	}

	productBlueprint, err := h.ProductBlueprintReviewUC.ProductBlueprintRepo.GetByID(
		ctx,
		productBlueprintID,
	)
	if err != nil {
		return "", err
	}

	if productBlueprint.ID != productBlueprintID ||
		productBlueprint.CompanyID != companyID ||
		productBlueprint.BrandID == "" {
		return "", uc.ErrReviewReportForbidden
	}

	return productBlueprint.BrandID, nil
}

func trimWS(value string) string {
	return strings.Trim(value, " \t\r\n")
}

func parseReviewStatus(
	raw string,
) (revdomain.ReviewStatus, bool) {
	status := trimWS(raw)
	if status == "" {
		return revdomain.ReviewStatusPublished, true
	}

	switch revdomain.ReviewStatus(status) {
	case revdomain.ReviewStatusPublished,
		revdomain.ReviewStatusHidden,
		revdomain.ReviewStatusRemoved:
		return revdomain.ReviewStatus(status), true

	default:
		return "", false
	}
}

// ============================================================
// ProductBlueprint review report
// ============================================================

type reportProductBlueprintReviewRequest struct {
	Reason string `json:"reason"`
	Detail string `json:"detail"`
}

type reportProductBlueprintReviewResponse struct {
	CaseID        string                  `json:"caseId"`
	ReportID      string                  `json:"reportId"`
	ReportCount   int                     `json:"reportCount"`
	Status        reviewreport.CaseStatus `json:"status"`
	CaseCreated   bool                    `json:"caseCreated"`
	ReportCreated bool                    `json:"reportCreated"`
}

func (h *ProductBlueprintReviewHandler) ReportReviewAsBrand(
	w http.ResponseWriter,
	r *http.Request,
	productBlueprintID string,
	reviewID string,
) {
	if h == nil || h.ReviewReportUC == nil {
		writeError(
			w,
			http.StatusServiceUnavailable,
			"ReviewReportHandlerNotInitialized",
		)
		return
	}

	if trimWS(productBlueprintID) == "" {
		writeError(
			w,
			http.StatusBadRequest,
			"ProductBlueprintIDRequired",
		)
		return
	}

	if trimWS(reviewID) == "" {
		writeError(
			w,
			http.StatusBadRequest,
			"ReviewIDRequired",
		)
		return
	}

	companyID, ok := h.resolveCompanyID(r)
	if !ok || companyID == "" {
		writeError(
			w,
			http.StatusForbidden,
			"CompanyIDNotResolved",
		)
		return
	}

	brandID, err := h.resolveProductBlueprintBrandID(
		r.Context(),
		productBlueprintID,
		companyID,
	)
	if err != nil {
		writeProductBlueprintReviewReportError(w, err)
		return
	}

	var request reportProductBlueprintReviewRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"InvalidRequest",
		)
		return
	}

	reason := reviewreport.ReportReason(
		strings.ToUpper(
			trimWS(request.Reason),
		),
	)
	if err := reason.Validate(); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"InvalidReportReason",
		)
		return
	}

	detail := trimWS(request.Detail)
	if reason == reviewreport.ReportReasonOther &&
		detail == "" {
		writeError(
			w,
			http.StatusBadRequest,
			"ReportDetailRequired",
		)
		return
	}

	result, err := h.ReviewReportUC.ReportProductBlueprintReviewByBrand(
		r.Context(),
		uc.ReportProductBlueprintReviewByBrandInput{
			ProductBlueprintID: productBlueprintID,
			ReviewID:           reviewID,
			BrandID:            brandID,
			CompanyID:          companyID,
			Reason:             reason,
			Detail:             detail,
		},
	)
	if err != nil {
		writeProductBlueprintReviewReportError(w, err)
		return
	}

	statusCode := http.StatusCreated
	if !result.ReportCreated {
		statusCode = http.StatusOK
	}

	writeJSON(
		w,
		statusCode,
		reportProductBlueprintReviewResponse{
			CaseID:        string(result.Case.ID),
			ReportID:      string(result.Report.ID),
			ReportCount:   result.Case.ReportCount,
			Status:        result.Case.Status,
			CaseCreated:   result.CaseCreated,
			ReportCreated: result.ReportCreated,
		},
	)
}

func parseProductBlueprintReviewReportPath(
	path string,
) (
	productBlueprintID string,
	reviewID string,
	ok bool,
) {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return "", "", false
	}

	parts := strings.Split(trimmed, "/")
	if len(parts) != 5 {
		return "", "", false
	}

	if parts[0] != "product-blueprints" ||
		parts[1] == "" ||
		parts[2] != "reviews" ||
		parts[3] == "" ||
		parts[4] != "reports" {
		return "", "", false
	}

	return parts[1], parts[3], true
}

func writeProductBlueprintReviewReportError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(
		err,
		uc.ErrReviewReportUsecaseNotConfigured,
	):
		writeError(
			w,
			http.StatusServiceUnavailable,
			"ReviewReportUsecaseNotConfigured",
		)

	case errors.Is(
		err,
		uc.ErrReviewReportForbidden,
	):
		writeError(
			w,
			http.StatusForbidden,
			"ReviewReportForbidden",
		)

	case errors.Is(
		err,
		uc.ErrReviewReportSelfReport,
	):
		writeError(
			w,
			http.StatusForbidden,
			"ReviewReportSelfReportNotAllowed",
		)

	case errors.Is(
		err,
		reviewreport.ErrCannotReportRemovedTarget,
	):
		writeError(
			w,
			http.StatusConflict,
			"CannotReportRemovedTarget",
		)

	case reviewreport.IsInvalid(err):
		writeError(
			w,
			http.StatusBadRequest,
			err.Error(),
		)

	case errors.Is(
		err,
		revdomain.ErrNotFound,
	):
		writeError(
			w,
			http.StatusNotFound,
			err.Error(),
		)

	case errors.Is(
		err,
		revdomain.ErrConflict,
	):
		writeError(
			w,
			http.StatusConflict,
			err.Error(),
		)

	case errors.Is(
		err,
		revdomain.ErrInvalid,
	):
		writeError(
			w,
			http.StatusBadRequest,
			err.Error(),
		)

	case errors.Is(
		err,
		revdomain.ErrUnauthorized,
	):
		writeError(
			w,
			http.StatusUnauthorized,
			err.Error(),
		)

	case errors.Is(
		err,
		revdomain.ErrForbidden,
	):
		writeError(
			w,
			http.StatusForbidden,
			err.Error(),
		)

	default:
		writeError(
			w,
			http.StatusInternalServerError,
			err.Error(),
		)
	}
}

// ============================================================
// 1) Detail page: reviews for a single ProductBlueprintID
// AvatarIDからAvatarName、AvatarIconをusecaseで解決して返す。
// ============================================================

type ListProductBlueprintReviewsResponse struct {
	ProductBlueprintID string                              `json:"ProductBlueprintID"`
	Status             revdomain.ReviewStatus              `json:"Status"`
	Page               int                                 `json:"Page"`
	PerPage            int                                 `json:"PerPage"`
	Items              []uc.ProductBlueprintReviewListItem `json:"Items"`
	TotalCount         int                                 `json:"TotalCount"`
	TotalPages         int                                 `json:"TotalPages"`
}

func (h *ProductBlueprintReviewHandler) ListReviewsByProductBlueprintID(
	w http.ResponseWriter,
	r *http.Request,
	productBlueprintID string,
) {
	if h == nil || h.ProductBlueprintReviewUC == nil {
		writeError(
			w,
			http.StatusServiceUnavailable,
			"HandlerNotInitialized",
		)
		return
	}

	if trimWS(productBlueprintID) == "" {
		writeError(
			w,
			http.StatusBadRequest,
			"ProductBlueprintIDRequired",
		)
		return
	}

	query := r.URL.Query()

	status, ok := parseReviewStatus(
		query.Get("Status"),
	)
	if !ok {
		writeError(
			w,
			http.StatusBadRequest,
			"InvalidStatus",
		)
		return
	}

	page := domcommon.Page{
		Number: parsePositiveInt(
			query.Get("Page"),
			1,
			0,
		),
		PerPage: parsePositiveInt(
			query.Get("PerPage"),
			20,
			200,
		),
	}

	result, err := h.ProductBlueprintReviewUC.ListByProductBlueprintID(
		r.Context(),
		productBlueprintID,
		status,
		page,
	)
	if err != nil {
		writeError(
			w,
			http.StatusInternalServerError,
			err.Error(),
		)
		return
	}

	response := ListProductBlueprintReviewsResponse{
		ProductBlueprintID: productBlueprintID,
		Status:             status,
		Page:               result.Page,
		PerPage:            result.PerPage,
		Items:              result.Items,
		TotalCount:         result.TotalCount,
		TotalPages:         result.TotalPages,
	}

	writeJSON(
		w,
		http.StatusOK,
		response,
	)
}

// ============================================================
// 2) Management page: aggregates
// summary docIdはProductBlueprintIDと同一。
// BrandNameをusecaseで解決して返す。
// ============================================================

type ListCompanyReviewAggregatesResponse struct {
	CompanyID  string                                   `json:"CompanyID"`
	Status     revdomain.ReviewStatus                   `json:"Status"`
	Page       int                                      `json:"Page"`
	PerPage    int                                      `json:"PerPage"`
	Items      []uc.ProductBlueprintReviewAggregateItem `json:"Items"`
	TotalCount int                                      `json:"TotalCount,omitempty"`
	TotalPages int                                      `json:"TotalPages,omitempty"`
}

func (h *ProductBlueprintReviewHandler) ListCompanyReviewAggregates(
	w http.ResponseWriter,
	r *http.Request,
) {
	if h == nil ||
		h.ProductBlueprintReviewUC == nil {
		writeError(
			w,
			http.StatusServiceUnavailable,
			"HandlerNotInitialized",
		)
		return
	}

	companyID, ok := h.resolveCompanyID(r)
	if !ok ||
		companyID == "" {
		writeError(
			w,
			http.StatusForbidden,
			"CompanyIDNotResolved",
		)
		return
	}

	query := r.URL.Query()

	status, ok := parseReviewStatus(
		query.Get("Status"),
	)
	if !ok {
		writeError(
			w,
			http.StatusBadRequest,
			"InvalidStatus",
		)
		return
	}

	page := domcommon.Page{
		Number: parsePositiveInt(
			query.Get("Page"),
			1,
			0,
		),
		PerPage: parsePositiveInt(
			query.Get("PerPage"),
			100,
			500,
		),
	}

	result, err := h.ProductBlueprintReviewUC.
		ListCompanyReviewAggregatesWithNames(
			r.Context(),
			companyID,
			status,
			page,
		)
	if err != nil {
		writeError(
			w,
			http.StatusInternalServerError,
			err.Error(),
		)
		return
	}

	response := ListCompanyReviewAggregatesResponse{
		CompanyID:  companyID,
		Status:     status,
		Page:       result.Page,
		PerPage:    result.PerPage,
		Items:      result.Items,
		TotalCount: result.TotalCount,
		TotalPages: result.TotalPages,
	}

	writeJSON(
		w,
		http.StatusOK,
		response,
	)
}
