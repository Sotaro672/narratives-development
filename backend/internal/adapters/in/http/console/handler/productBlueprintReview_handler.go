// backend/internal/adapters/in/http/console/handler/productBlueprintReview_handler.go
package consoleHandler

import (
	"encoding/json"
	"net/http"
	"strconv"
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

// usecase を注入して生成（必須）
func NewProductBlueprintReviewHandler(
	productBlueprintReviewUC *uc.ProductBlueprintReviewUsecase,
) *ProductBlueprintReviewHandler {
	return &ProductBlueprintReviewHandler{
		ProductBlueprintReviewUC: productBlueprintReviewUC,
	}
}

// ServeHTTP を実装して http.Handler を満たす（DI/Router でそのまま登録できるようにする）
//
// GET routing policy (single handler):
// - /product-blueprint-reviews/aggregates                      -> ListCompanyReviewAggregates (management page)
// - /product-blueprint-reviews?ProductBlueprintID=...          -> ListReviewsByProductBlueprintID (detail page)
//
// Query Params (PascalCase):
// - Status: PUBLISHED | HIDDEN | REMOVED (default: PUBLISHED)
// - Page: int (default: 1)
// - PerPage: int (default: 20 for reviews, 100 for aggregates)
func (h *ProductBlueprintReviewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		trimmedPath := strings.TrimRight(r.URL.Path, "/")

		// management page: aggregates only
		if strings.HasSuffix(trimmedPath, "/product-blueprint-reviews/aggregates") {
			h.ListCompanyReviewAggregates(w, r)
			return
		}

		// detail page: reviews for a single ProductBlueprintID
		if pbID := trimWS(r.URL.Query().Get("ProductBlueprintID")); pbID != "" {
			h.ListReviewsByProductBlueprintID(w, r, pbID)
			return
		}

		writeJSONResponse(w, http.StatusNotFound, map[string]any{
			"Error": "NotFound",
		})

	default:
		writeJSONResponse(w, http.StatusMethodNotAllowed, map[string]any{
			"Error": "MethodNotAllowed",
		})
	}
}

// ============================================================
// JSON helpers
// ============================================================

func writeJSONResponse(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErrorResponse(w http.ResponseWriter, status int, err string) {
	writeJSONResponse(w, status, map[string]any{
		"Error": err,
	})
}

// ============================================================
// Common helpers
// ============================================================

func (h *ProductBlueprintReviewHandler) resolveCompanyID(r *http.Request) (string, bool) {
	if m, ok := middleware.CurrentMember(r); ok && m != nil && m.CompanyID != "" {
		return m.CompanyID, true
	}
	if cid, ok := middleware.CompanyID(r); ok && cid != "" {
		return cid, true
	}
	return "", false
}

func trimWS(s string) string {
	// strings.TrimSpace を使わずに空白系を除去
	return strings.Trim(s, " \t\r\n")
}

func parseReviewStatus(raw string) (revdomain.ReviewStatus, bool) {
	s := trimWS(raw)
	if s == "" {
		return revdomain.ReviewStatusPublished, true
	}
	switch revdomain.ReviewStatus(s) {
	case revdomain.ReviewStatusPublished, revdomain.ReviewStatusHidden, revdomain.ReviewStatusRemoved:
		return revdomain.ReviewStatus(s), true
	default:
		return "", false
	}
}

func parsePositiveInt(raw string, def, max int) int {
	s := trimWS(raw)
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 {
		return def
	}
	if max > 0 && v > max {
		return max
	}
	return v
}

// ============================================================
// 1) Detail page: reviews for a single ProductBlueprintID
// - AvatarID -> AvatarName / AvatarIcon を usecase で解決して返す
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
		writeErrorResponse(w, http.StatusServiceUnavailable, "HandlerNotInitialized")
		return
	}
	if trimWS(productBlueprintID) == "" {
		writeErrorResponse(w, http.StatusBadRequest, "ProductBlueprintIDRequired")
		return
	}

	q := r.URL.Query()

	status, ok := parseReviewStatus(q.Get("Status"))
	if !ok {
		writeErrorResponse(w, http.StatusBadRequest, "InvalidStatus")
		return
	}

	pageNum := parsePositiveInt(q.Get("Page"), 1, 0)
	perPage := parsePositiveInt(q.Get("PerPage"), 20, 200)

	page := domcommon.Page{
		Number:  pageNum,
		PerPage: perPage,
	}

	out, err := h.ProductBlueprintReviewUC.ListByProductBlueprintID(
		r.Context(),
		productBlueprintID,
		status,
		page,
	)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := ListProductBlueprintReviewsResponse{
		ProductBlueprintID: productBlueprintID,
		Status:             status,
		Page:               out.Page,
		PerPage:            out.PerPage,
		Items:              out.Items,
		TotalCount:         out.TotalCount,
		TotalPages:         out.TotalPages,
	}

	writeJSONResponse(w, http.StatusOK, resp)
}

// ============================================================
// 2) Management page: aggregates (summary docId == ProductBlueprintID)
// - BrandName を usecase で解決して返す
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

func (h *ProductBlueprintReviewHandler) ListCompanyReviewAggregates(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.ProductBlueprintReviewUC == nil {
		writeErrorResponse(w, http.StatusServiceUnavailable, "HandlerNotInitialized")
		return
	}

	companyID, ok := h.resolveCompanyID(r)
	if !ok || companyID == "" {
		writeErrorResponse(w, http.StatusForbidden, "CompanyIDNotResolved")
		return
	}

	q := r.URL.Query()

	status, ok := parseReviewStatus(q.Get("Status"))
	if !ok {
		writeErrorResponse(w, http.StatusBadRequest, "InvalidStatus")
		return
	}

	pageNum := parsePositiveInt(q.Get("Page"), 1, 0)
	perPage := parsePositiveInt(q.Get("PerPage"), 100, 500)

	page := domcommon.Page{
		Number:  pageNum,
		PerPage: perPage,
	}

	out, err := h.ProductBlueprintReviewUC.ListCompanyReviewAggregatesWithNames(
		r.Context(),
		companyID,
		status,
		page,
	)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := ListCompanyReviewAggregatesResponse{
		CompanyID:  companyID,
		Status:     status,
		Page:       out.Page,
		PerPage:    out.PerPage,
		Items:      out.Items,
		TotalCount: out.TotalCount,
		TotalPages: out.TotalPages,
	}
	writeJSONResponse(w, http.StatusOK, resp)
}
