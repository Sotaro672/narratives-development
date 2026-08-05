// backend/internal/adapters/in/http/console/handler/productBlueprintReview_handler.go
package consoleHandler

import (
	"net/http"
	"strings"

	"narratives/internal/adapters/in/http/middleware"
	uc "narratives/internal/application/usecase"
	domcommon "narratives/internal/domain/common"
	revdomain "narratives/internal/domain/productBlueprintReview"
)

type ProductBlueprintReviewHandler struct {
	// usecase（名前解決・集計・avatar解決も含めて寄せる）
	ProductBlueprintReviewUC *uc.ProductBlueprintReviewUsecase
}

// usecaseを注入して生成（必須）
func NewProductBlueprintReviewHandler(
	productBlueprintReviewUC *uc.ProductBlueprintReviewUsecase,
) *ProductBlueprintReviewHandler {
	return &ProductBlueprintReviewHandler{
		ProductBlueprintReviewUC: productBlueprintReviewUC,
	}
}

// ServeHTTPを実装してhttp.Handlerを満たす。
// DI/Routerでそのまま登録できるようにする。
//
// GET routing policy:
//   - /product-blueprint-reviews/aggregates
//     -> ListCompanyReviewAggregates
//   - /product-blueprint-reviews?ProductBlueprintID=...
//     -> ListReviewsByProductBlueprintID
//
// Query Params:
//   - Status: PUBLISHED | HIDDEN | REMOVED（default: PUBLISHED）
//   - Page: int（default: 1）
//   - PerPage: int（default: reviews=20, aggregates=100）
func (h *ProductBlueprintReviewHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
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

	status, ok := parseReviewStatus(query.Get("Status"))
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
	if h == nil || h.ProductBlueprintReviewUC == nil {
		writeError(
			w,
			http.StatusServiceUnavailable,
			"HandlerNotInitialized",
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

	query := r.URL.Query()

	status, ok := parseReviewStatus(query.Get("Status"))
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
