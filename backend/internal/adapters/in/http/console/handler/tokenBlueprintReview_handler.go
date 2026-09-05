// backend/internal/adapters/in/http/console/handler/tokenBlueprintReview_handler.go
package consoleHandler

import (
	"errors"
	"net/http"
	"strings"

	"narratives/internal/adapters/in/http/middleware"
	appquery "narratives/internal/application/query/console"
	"narratives/internal/application/usecase"
	common "narratives/internal/domain/common"
	reviewreport "narratives/internal/domain/reviewReport"
	tbReview "narratives/internal/domain/tokenBlueprint_review"
)

var (
	errUnauthorized = errors.New("unauthorized")
)

type TokenBlueprintReviewHandler struct {
	uc             *usecase.TokenBlueprintReviewUsecase
	query          *appquery.TokenBlueprintReviewConsoleQuery
	reviewReportUC *usecase.ReviewReportUsecase
}

func NewTokenBlueprintReviewHandler(
	uc *usecase.TokenBlueprintReviewUsecase,
	reviewReportUC *usecase.ReviewReportUsecase,
) *TokenBlueprintReviewHandler {
	return &TokenBlueprintReviewHandler{
		uc:             uc,
		query:          appquery.NewTokenBlueprintReviewConsoleQuery(uc),
		reviewReportUC: reviewReportUC,
	}
}

// ================================
// Routing
// ================================
//
// Supported:
// - GET    /token-blueprint-reviews
// - GET    /token-blueprint-reviews/{tokenBlueprintId}/comments
// - POST   /token-blueprint-reviews/{tokenBlueprintId}/comments
// - DELETE /token-blueprint-reviews/{tokenBlueprintId}/comments/{commentId}
// - POST   /token-blueprint-reviews/{tokenBlueprintId}/comments/{commentId}/reactions
// - POST   /token-blueprint-reviews/{tokenBlueprintId}/comments/{commentId}/replies
// - POST   /token-blueprint-reviews/{tokenBlueprintId}/comments/{commentId}/reports
//
// console handler では brand 側からのみ comment / reply / comment reaction / report を許可する。
func (h *TokenBlueprintReviewHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if h == nil || h.uc == nil || h.query == nil {
		writeError(w, http.StatusInternalServerError, "handler not configured")
		return
	}

	rest := strings.TrimPrefix(
		r.URL.Path,
		"/token-blueprint-reviews",
	)
	rest = strings.Trim(rest, "/")

	if rest == "" {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}

		h.ListAggregatesByCompanyTokenBlueprints(w, r)
		return
	}

	parts := strings.Split(rest, "/")
	tbID := parts[0]

	if tbID == "" {
		writeError(
			w,
			http.StatusBadRequest,
			"tokenBlueprintId is required",
		)
		return
	}

	if len(parts) == 1 {
		writeNotFound(w)
		return
	}

	switch parts[1] {
	case "comments":
		if len(parts) == 2 {
			switch r.Method {
			case http.MethodGet:
				h.ListCommentsByTokenBlueprintID(
					w,
					r,
					tbID,
				)

			case http.MethodPost:
				h.CreateCommentAsBrand(
					w,
					r,
					tbID,
				)

			default:
				methodNotAllowed(w)
			}

			return
		}

		commentID := parts[2]

		if commentID == "" {
			writeError(
				w,
				http.StatusBadRequest,
				"commentId is required",
			)
			return
		}

		if len(parts) == 3 {
			if r.Method != http.MethodDelete {
				methodNotAllowed(w)
				return
			}

			h.DeleteComment(
				w,
				r,
				tbID,
				commentID,
			)
			return
		}

		if len(parts) == 4 &&
			parts[3] == "reactions" {
			if r.Method != http.MethodPost {
				methodNotAllowed(w)
				return
			}

			h.ReactToCommentAsBrand(
				w,
				r,
				tbID,
				commentID,
			)
			return
		}

		if len(parts) == 4 &&
			parts[3] == "replies" {
			if r.Method != http.MethodPost {
				methodNotAllowed(w)
				return
			}

			h.CreateBrandReply(
				w,
				r,
				tbID,
				commentID,
			)
			return
		}

		if len(parts) == 4 &&
			parts[3] == "reports" {
			if r.Method != http.MethodPost {
				methodNotAllowed(w)
				return
			}

			h.ReportCommentAsBrand(
				w,
				r,
				tbID,
				commentID,
			)
			return
		}

		writeNotFound(w)
		return

	default:
		writeNotFound(w)
		return
	}
}

// ================================
// Requests / Responses
// ================================

type createBrandCommentRequest struct {
	CommentID       *string `json:"commentId,omitempty"`
	ParentCommentID *string `json:"parentCommentId,omitempty"`
	Body            string  `json:"body"`
}

type reactAsBrandRequest struct {
	Type tbReview.ReactionType `json:"type"`
}

type reportCommentAsBrandRequest struct {
	Reason string `json:"reason"`
	Detail string `json:"detail"`
}

type createBrandCommentResponse struct {
	Item appquery.ConsoleTokenBlueprintCommentReadModel `json:"item"`
}

type reportCommentAsBrandResponse struct {
	CaseID        string                  `json:"caseId"`
	ReportID      string                  `json:"reportId"`
	ReportCount   int                     `json:"reportCount"`
	Status        reviewreport.CaseStatus `json:"status"`
	CaseCreated   bool                    `json:"caseCreated"`
	ReportCreated bool                    `json:"reportCreated"`
}

// ================================
// Helpers
// ================================

func ptrStr(value *string) string {
	if value == nil {
		return ""
	}

	return strings.Trim(*value, " \t\r\n")
}

func queryStringPtr(
	r *http.Request,
	key string,
) *string {
	if r == nil {
		return nil
	}

	raw, ok := r.URL.Query()[key]
	if !ok || len(raw) == 0 {
		return nil
	}

	value := raw[0]

	return &value
}

func queryBoolPtr(
	r *http.Request,
	key string,
) *bool {
	if r == nil {
		return nil
	}

	raw := r.URL.Query().Get(key)
	if raw == "" {
		return nil
	}

	value := strings.EqualFold(raw, "true")

	return &value
}

func queryIntPtr(
	r *http.Request,
	key string,
) *int {
	if r == nil {
		return nil
	}

	raw := r.URL.Query().Get(key)
	if raw == "" {
		return nil
	}

	value := parseIntDefault(raw, 0)

	return &value
}

// ================================
// Read handlers
// ================================

func (h *TokenBlueprintReviewHandler) ListAggregatesByCompanyTokenBlueprints(
	w http.ResponseWriter,
	r *http.Request,
) {
	companyID, ok := middleware.CompanyID(r)
	if !ok || companyID == "" {
		writeError(
			w,
			http.StatusUnauthorized,
			errUnauthorized.Error(),
		)
		return
	}

	result, err := h.query.ListAggregatesByCompanyTokenBlueprints(
		r.Context(),
		appquery.ListConsoleTokenBlueprintReviewAggregatesInput{
			CompanyID: companyID,
		},
	)
	if err != nil {
		writeError(
			w,
			http.StatusInternalServerError,
			err.Error(),
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		result,
	)
}

func (h *TokenBlueprintReviewHandler) ListCommentsByTokenBlueprintID(
	w http.ResponseWriter,
	r *http.Request,
	tokenBlueprintID string,
) {
	companyID, ok := middleware.CompanyID(r)
	if !ok || companyID == "" {
		writeError(
			w,
			http.StatusUnauthorized,
			errUnauthorized.Error(),
		)
		return
	}

	result, err := h.query.ListCommentsByTokenBlueprintID(
		r.Context(),
		appquery.ListConsoleTokenBlueprintCommentsInput{
			CompanyID:        companyID,
			TokenBlueprintID: tokenBlueprintID,
			SearchQuery:      r.URL.Query().Get("q"),
			ParentCommentID: queryStringPtr(
				r,
				"parentCommentId",
			),
			RootCommentID: r.URL.Query().Get(
				"rootCommentId",
			),
			AuthorID: r.URL.Query().Get(
				"authorId",
			),
			Deleted: queryBoolPtr(
				r,
				"deleted",
			),
			Depth: queryIntPtr(
				r,
				"depth",
			),
			Sort: common.Sort{
				Column: r.URL.Query().Get(
					"sort",
				),
				Order: common.SortOrder(
					strings.ToLower(
						r.URL.Query().Get("order"),
					),
				),
			},
			Page: common.Page{
				Number: parseIntDefault(
					r.URL.Query().Get("page"),
					1,
				),
				PerPage: parseIntDefault(
					r.URL.Query().Get("perPage"),
					200,
				),
			},
		},
	)
	if err != nil {
		writeError(
			w,
			http.StatusInternalServerError,
			err.Error(),
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		result,
	)
}

// ================================
// Command handlers
// ================================

func (h *TokenBlueprintReviewHandler) CreateCommentAsBrand(
	w http.ResponseWriter,
	r *http.Request,
	tokenBlueprintID string,
) {
	companyID, ok := middleware.CompanyID(r)
	if !ok || companyID == "" {
		writeError(
			w,
			http.StatusUnauthorized,
			errUnauthorized.Error(),
		)
		return
	}

	var request createBrandCommentRequest
	if err := decodeStrictJSON(r, &request); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid_json",
		)
		return
	}

	actor, err := h.query.ResolveBrandActor(
		r.Context(),
		tokenBlueprintID,
	)
	if err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	created, err := h.uc.CreateComment(
		r.Context(),
		usecase.CreateCommentInput{
			CommentID: ptrStr(
				request.CommentID,
			),
			TokenBlueprintID: tokenBlueprintID,
			ParentCommentID: ptrStr(
				request.ParentCommentID,
			),
			AuthorID:       actor.BrandID,
			AuthorType:     h.query.AuthorType(),
			IsOwnerComment: true,
			Body:           request.Body,
		},
	)
	if err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	writeJSON(
		w,
		http.StatusCreated,
		createBrandCommentResponse{
			Item: h.query.ToCommentReadModel(
				h.uc.BuildComment(
					r.Context(),
					created,
				),
			),
		},
	)
}

func (h *TokenBlueprintReviewHandler) CreateBrandReply(
	w http.ResponseWriter,
	r *http.Request,
	tokenBlueprintID string,
	parentCommentID string,
) {
	companyID, ok := middleware.CompanyID(r)
	if !ok || companyID == "" {
		writeError(
			w,
			http.StatusUnauthorized,
			errUnauthorized.Error(),
		)
		return
	}

	var request createBrandCommentRequest
	if err := decodeStrictJSON(r, &request); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid_json",
		)
		return
	}

	actor, err := h.query.ResolveBrandActor(
		r.Context(),
		tokenBlueprintID,
	)
	if err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	created, err := h.uc.CreateComment(
		r.Context(),
		usecase.CreateCommentInput{
			CommentID:        ptrStr(request.CommentID),
			TokenBlueprintID: tokenBlueprintID,
			ParentCommentID:  parentCommentID,
			AuthorID:         actor.BrandID,
			AuthorType:       h.query.AuthorType(),
			IsOwnerComment:   true,
			Body:             request.Body,
		},
	)
	if err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	writeJSON(
		w,
		http.StatusCreated,
		createBrandCommentResponse{
			Item: h.query.ToCommentReadModel(
				h.uc.BuildComment(
					r.Context(),
					created,
				),
			),
		},
	)
}

func (h *TokenBlueprintReviewHandler) DeleteComment(
	w http.ResponseWriter,
	r *http.Request,
	tokenBlueprintID string,
	commentID string,
) {
	companyID, ok := middleware.CompanyID(r)
	if !ok || companyID == "" {
		writeError(
			w,
			http.StatusUnauthorized,
			errUnauthorized.Error(),
		)
		return
	}

	actor, err := h.query.ResolveBrandActor(
		r.Context(),
		tokenBlueprintID,
	)
	if err != nil {
		writeError(
			w,
			http.StatusForbidden,
			err.Error(),
		)
		return
	}

	err = h.uc.DeleteComment(
		r.Context(),
		usecase.DeleteCommentInput{
			TokenBlueprintID: tokenBlueprintID,
			CommentID:        commentID,
			AuthorID:         actor.BrandID,
			AuthorType:       h.query.AuthorType(),
		},
	)
	if err != nil {
		if errors.Is(
			err,
			usecase.ErrCommentDeleteForbidden,
		) {
			writeError(
				w,
				http.StatusForbidden,
				err.Error(),
			)
			return
		}

		writeError(
			w,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *TokenBlueprintReviewHandler) ReactToCommentAsBrand(
	w http.ResponseWriter,
	r *http.Request,
	tokenBlueprintID string,
	commentID string,
) {
	companyID, ok := middleware.CompanyID(r)
	if !ok || companyID == "" {
		writeError(
			w,
			http.StatusUnauthorized,
			errUnauthorized.Error(),
		)
		return
	}

	var request reactAsBrandRequest
	if err := decodeStrictJSON(r, &request); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid_json",
		)
		return
	}

	if err := request.Type.Validate(); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	actor, err := h.query.ResolveBrandActor(
		r.Context(),
		tokenBlueprintID,
	)
	if err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	updated, err := h.uc.ReactToComment(
		r.Context(),
		tokenBlueprintID,
		commentID,
		actor.BrandID,
		h.query.ActorType(),
		request.Type,
	)
	if err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			err.Error(),
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		map[string]any{
			"item": h.query.ToCommentReadModel(
				h.uc.BuildComment(
					r.Context(),
					updated,
				),
			),
			"actor": map[string]any{
				"actorType":  h.query.ActorType(),
				"authorType": h.query.AuthorType(),
				"brandId":    actor.BrandID,
				"brandName":  actor.BrandName,
				"brandIcon":  actor.BrandIcon,
			},
		},
	)
}

func (h *TokenBlueprintReviewHandler) ReportCommentAsBrand(
	w http.ResponseWriter,
	r *http.Request,
	tokenBlueprintID string,
	commentID string,
) {
	if h == nil || h.reviewReportUC == nil {
		writeError(
			w,
			http.StatusServiceUnavailable,
			"review_report_usecase_not_configured",
		)
		return
	}

	companyID, ok := middleware.CompanyID(r)
	if !ok || companyID == "" {
		writeError(
			w,
			http.StatusUnauthorized,
			errUnauthorized.Error(),
		)
		return
	}

	actor, err := h.query.ResolveBrandActor(
		r.Context(),
		tokenBlueprintID,
	)
	if err != nil {
		writeError(
			w,
			http.StatusForbidden,
			err.Error(),
		)
		return
	}

	var request reportCommentAsBrandRequest
	if err := decodeStrictJSON(r, &request); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid_json",
		)
		return
	}

	reason := reviewreport.ReportReason(
		strings.ToUpper(
			strings.TrimSpace(request.Reason),
		),
	)
	if err := reason.Validate(); err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid_report_reason",
		)
		return
	}

	detail := strings.TrimSpace(request.Detail)
	if reason == reviewreport.ReportReasonOther &&
		detail == "" {
		writeError(
			w,
			http.StatusBadRequest,
			"report_detail_required",
		)
		return
	}

	result, err := h.reviewReportUC.ReportTokenBlueprintCommentByBrand(
		r.Context(),
		usecase.ReportTokenBlueprintCommentByBrandInput{
			TokenBlueprintID: tokenBlueprintID,
			CommentID:        commentID,
			BrandID:          actor.BrandID,
			CompanyID:        companyID,
			Reason:           reason,
			Detail:           detail,
		},
	)
	if err != nil {
		writeTokenBlueprintReviewReportError(
			w,
			err,
		)
		return
	}

	statusCode := http.StatusCreated
	if !result.ReportCreated {
		statusCode = http.StatusOK
	}

	writeJSON(
		w,
		statusCode,
		reportCommentAsBrandResponse{
			CaseID:        string(result.Case.ID),
			ReportID:      string(result.Report.ID),
			ReportCount:   result.Case.ReportCount,
			Status:        result.Case.Status,
			CaseCreated:   result.CaseCreated,
			ReportCreated: result.ReportCreated,
		},
	)
}

func writeTokenBlueprintReviewReportError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(
		err,
		usecase.ErrReviewReportUsecaseNotConfigured,
	):
		writeError(
			w,
			http.StatusServiceUnavailable,
			"review_report_usecase_not_configured",
		)

	case errors.Is(
		err,
		usecase.ErrReviewReportForbidden,
	):
		writeError(
			w,
			http.StatusForbidden,
			"review_report_forbidden",
		)

	case errors.Is(
		err,
		usecase.ErrReviewReportSelfReport,
	):
		writeError(
			w,
			http.StatusForbidden,
			"self_report_not_allowed",
		)

	case errors.Is(
		err,
		reviewreport.ErrCannotReportRemovedTarget,
	):
		writeError(
			w,
			http.StatusConflict,
			"cannot_report_removed_target",
		)

	case reviewreport.IsInvalid(err):
		writeError(
			w,
			http.StatusBadRequest,
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
