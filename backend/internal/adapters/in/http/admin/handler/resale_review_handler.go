// backend/internal/adapters/in/http/admin/handler/resale_review_handler.go
package handler

import (
	"context"
	"errors"
	"net/http"

	usecase "narratives/internal/application/usecase"
	common "narratives/internal/domain/common"
	resaledom "narratives/internal/domain/resale"
	resalereviewdom "narratives/internal/domain/resale_review"
)

const (
	defaultAdminResaleReviewPage    = 1
	defaultAdminResaleReviewPerPage = 20
	maxAdminResaleReviewPerPage     = 100
)

type ResaleReviewResaleReader interface {
	GetByID(
		ctx context.Context,
		id string,
	) (resaledom.Resale, error)
}

type ResaleReviewReportCountReader interface {
	GetResaleCommentReportCount(
		ctx context.Context,
		resaleID string,
		commentID string,
	) (int, error)
}

type ResaleReviewHandler struct {
	resaleRepo     ResaleReviewResaleReader
	resaleReviewUC *usecase.ResaleReviewUsecase
	reportRepo     ResaleReviewReportCountReader
}

type resaleReviewItemResponse struct {
	usecase.ResaleReviewCommentListItem
	ReportCount int `json:"reportCount"`
}

type resaleReviewListResponse struct {
	Items      []resaleReviewItemResponse `json:"items"`
	TotalCount int                        `json:"totalCount"`
	TotalPages int                        `json:"totalPages"`
	Page       int                        `json:"page"`
	PerPage    int                        `json:"perPage"`
}

func NewResaleReviewHandler(
	resaleRepo ResaleReviewResaleReader,
	resaleReviewUC *usecase.ResaleReviewUsecase,
	reportRepo ResaleReviewReportCountReader,
) http.Handler {
	h := &ResaleReviewHandler{
		resaleRepo:     resaleRepo,
		resaleReviewUC: resaleReviewUC,
		reportRepo:     reportRepo,
	}

	return http.HandlerFunc(h.handle)
}

func (h *ResaleReviewHandler) handle(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}

	if h.resaleRepo == nil || h.resaleReviewUC == nil || h.reportRepo == nil {
		writeJSONError(
			w,
			http.StatusServiceUnavailable,
			"resale_review_dependencies_not_initialized",
		)
		return
	}

	avatarID := r.PathValue("avatarID")
	if avatarID == "" {
		writeJSONError(w, http.StatusBadRequest, "avatar_id_required")
		return
	}

	resaleID := r.PathValue("resaleID")
	if resaleID == "" {
		writeJSONError(w, http.StatusBadRequest, "resale_id_required")
		return
	}

	resale, err := h.resaleRepo.GetByID(r.Context(), resaleID)
	if err != nil {
		if errors.Is(err, resaledom.ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "resale_not_found")
			return
		}

		writeJSONError(w, http.StatusInternalServerError, "resale_get_failed")
		return
	}

	if resale.AvatarID != avatarID {
		writeJSONError(w, http.StatusNotFound, "resale_not_found")
		return
	}

	result, err := h.resaleReviewUC.ListComments(
		r.Context(),
		resaleID,
		buildAdminResaleReviewPage(r),
	)
	if err != nil {
		writeAdminResaleReviewError(w, err)
		return
	}

	items := make([]resaleReviewItemResponse, 0, len(result.Items))
	for _, item := range result.Items {
		reportCount, err := h.reportRepo.GetResaleCommentReportCount(
			r.Context(),
			resaleID,
			string(item.CommentID),
		)
		if err != nil {
			writeJSONError(
				w,
				http.StatusInternalServerError,
				"resale_review_report_count_failed",
			)
			return
		}

		items = append(items, resaleReviewItemResponse{
			ResaleReviewCommentListItem: item,
			ReportCount:                 reportCount,
		})
	}

	writeJSON(w, http.StatusOK, resaleReviewListResponse{
		Items:      items,
		TotalCount: result.TotalCount,
		TotalPages: result.TotalPages,
		Page:       result.Page,
		PerPage:    result.PerPage,
	})
}

func buildAdminResaleReviewPage(
	r *http.Request,
) common.Page {
	query := r.URL.Query()

	page := parsePositiveInt(
		query.Get("page"),
		defaultAdminResaleReviewPage,
	)
	perPage := parsePositiveInt(
		query.Get("perPage"),
		defaultAdminResaleReviewPerPage,
	)

	if perPage > maxAdminResaleReviewPerPage {
		perPage = maxAdminResaleReviewPerPage
	}

	return common.Page{
		Number:  page,
		PerPage: perPage,
	}
}

func writeAdminResaleReviewError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case resalereviewdom.IsInvalid(err):
		writeJSONError(w, http.StatusBadRequest, "invalid_resale_review_request")

	case resalereviewdom.IsNotFound(err):
		writeJSONError(w, http.StatusNotFound, "resale_review_not_found")

	case resalereviewdom.IsForbidden(err):
		writeJSONError(w, http.StatusForbidden, "resale_review_forbidden")

	case resalereviewdom.IsConflict(err):
		writeJSONError(w, http.StatusConflict, "resale_review_conflict")

	default:
		writeJSONError(
			w,
			http.StatusInternalServerError,
			"resale_review_list_failed",
		)
	}
}
