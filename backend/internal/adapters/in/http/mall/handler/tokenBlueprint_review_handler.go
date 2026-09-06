// backend/internal/adapters/in/http/mall/handler/tokenBlueprint_review_handler.go
package mallHandler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	mw "narratives/internal/adapters/in/http/middleware"
	appquery "narratives/internal/application/query/mall"
	appusecase "narratives/internal/application/usecase"
	common "narratives/internal/domain/common"
	reportdom "narratives/internal/domain/report"
	tokenBlueprintReview "narratives/internal/domain/tokenBlueprint_review"
)

// 標準 net/http 前提。
// mall / console の機能差は actor 解決のみとし、mall 側は avatar actor で統一する。
//
// Hexagonal architecture policy:
// - handler: HTTP input adapter
//   - route dispatch
//   - method check
//   - auth actor resolution
//   - request/query parsing
//   - command invocation
//   - response writing
//
// - query: application read model service
//   - aggregate/comment read model composition
//   - avatar / brand lightweight display resolution
//   - mall actor policy
//
// - usecase: application command service
//   - comment creation
//   - comment deletion
//   - reaction mutation
//   - aggregate count update
//   - domain invariant execution
//
// Supported:
// - GET    /mall/me/token-blueprints/{id}/reviews/aggregate
// - POST   /mall/me/token-blueprints/{id}/reactions
// - GET    /mall/me/token-blueprints/{id}/comments
// - POST   /mall/me/token-blueprints/{id}/comments
// - DELETE /mall/me/token-blueprints/{id}/comments/{commentId}
// - POST   /mall/me/token-blueprints/{id}/comments/{commentId}/reactions
// - POST   /mall/me/token-blueprints/{id}/comments/{commentId}/replies
// - POST   /mall/me/token-blueprints/{id}/comments/{commentId}/reports
type TokenBlueprintCommentReportService interface {
	ReportTokenBlueprintCommentByAvatar(
		ctx context.Context,
		input appusecase.ReportTokenBlueprintCommentByAvatarInput,
	) (reportdom.AddReportResult, error)
}

type TokenBlueprintReviewHandler struct {
	uc        *appusecase.TokenBlueprintReviewUsecase
	query     *appquery.TokenBlueprintReviewMallQuery
	reportSvc TokenBlueprintCommentReportService
}

func NewTokenBlueprintReviewHandler(
	uc *appusecase.TokenBlueprintReviewUsecase,
	reportSvc TokenBlueprintCommentReportService,
) *TokenBlueprintReviewHandler {
	return &TokenBlueprintReviewHandler{
		uc:        uc,
		query:     appquery.NewTokenBlueprintReviewMallQuery(uc),
		reportSvc: reportSvc,
	}
}

func (h *TokenBlueprintReviewHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if h == nil || h.uc == nil || h.query == nil {
		internalError(w, "handler not configured")
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/")

	tokenBlueprintID := extractTokenBlueprintIDFromPath(path)
	if tokenBlueprintID == "" {
		notFound(w)
		return
	}

	switch {
	case strings.HasSuffix(path, "/reviews/aggregate"):
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}

		h.getAggregate(
			w,
			r,
			tokenBlueprintID,
		)
		return

	case strings.HasSuffix(path, "/reactions") &&
		isTokenBlueprintReactionPath(
			path,
			tokenBlueprintID,
		):
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}

		h.upsertTokenBlueprintReaction(
			w,
			r,
			tokenBlueprintID,
		)
		return

	case strings.Contains(path, "/comments"):
		h.dispatchComments(
			w,
			r,
			tokenBlueprintID,
		)
		return

	default:
		notFound(w)
		return
	}
}

// ============================================================
// Path helpers
// ============================================================

func extractTokenBlueprintIDFromPath(path string) string {
	const prefix = "/mall/me/token-blueprints/"

	if !strings.Contains(path, prefix) {
		return ""
	}

	index := strings.Index(path, prefix)
	if index < 0 {
		return ""
	}

	rest := path[index+len(prefix):]
	if rest == "" {
		return ""
	}

	segment := rest
	if separatorIndex := strings.Index(
		segment,
		"/",
	); separatorIndex >= 0 {
		segment = segment[:separatorIndex]
	}

	return segment
}

func isTokenBlueprintReactionPath(
	path string,
	tokenBlueprintID string,
) bool {
	if tokenBlueprintID == "" {
		return false
	}

	expectedPath :=
		"/mall/me/token-blueprints/" +
			tokenBlueprintID +
			"/reactions"

	return path == expectedPath
}

func extractCommentID(
	path string,
	tokenBlueprintID string,
) string {
	base :=
		"/mall/me/token-blueprints/" +
			tokenBlueprintID +
			"/comments/"

	if !strings.Contains(path, base) {
		return ""
	}

	index := strings.Index(path, base)
	if index < 0 {
		return ""
	}

	rest := path[index+len(base):]
	if rest == "" {
		return ""
	}

	segment := rest
	if separatorIndex := strings.Index(
		segment,
		"/",
	); separatorIndex >= 0 {
		segment = segment[:separatorIndex]
	}

	return segment
}

// ============================================================
// Comments dispatch
// ============================================================

func (h *TokenBlueprintReviewHandler) dispatchComments(
	w http.ResponseWriter,
	r *http.Request,
	tokenBlueprintID string,
) {
	path := strings.TrimSuffix(r.URL.Path, "/")
	base :=
		"/mall/me/token-blueprints/" +
			tokenBlueprintID +
			"/comments"

	// /mall/me/token-blueprints/{id}/comments
	if path == base {
		switch r.Method {
		case http.MethodGet:
			h.listComments(
				w,
				r,
				tokenBlueprintID,
			)

		case http.MethodPost:
			h.createComment(
				w,
				r,
				tokenBlueprintID,
			)

		default:
			methodNotAllowed(w)
		}

		return
	}

	commentID := extractCommentID(
		path,
		tokenBlueprintID,
	)
	if commentID == "" {
		notFound(w)
		return
	}

	// /mall/me/token-blueprints/{id}/comments/{commentId}
	if path == base+"/"+commentID {
		if r.Method != http.MethodDelete {
			methodNotAllowed(w)
			return
		}

		h.deleteComment(
			w,
			r,
			tokenBlueprintID,
			commentID,
		)
		return
	}

	// /mall/me/token-blueprints/{id}/comments/{commentId}/reactions
	if path == base+"/"+commentID+"/reactions" {
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}

		h.upsertCommentReaction(
			w,
			r,
			tokenBlueprintID,
			commentID,
		)
		return
	}

	// /mall/me/token-blueprints/{id}/comments/{commentId}/reports
	if path == base+"/"+commentID+"/reports" {
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}

		h.reportComment(
			w,
			r,
			tokenBlueprintID,
			commentID,
		)
		return
	}

	// /mall/me/token-blueprints/{id}/comments/{commentId}/replies
	if path == base+"/"+commentID+"/replies" {
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}

		h.createReplyComment(
			w,
			r,
			tokenBlueprintID,
			commentID,
		)
		return
	}

	notFound(w)
}

// ============================================================
// Request DTO
// ============================================================

type reactionRequest struct {
	Type tokenBlueprintReview.ReactionType `json:"type"`
}

type createCommentRequest struct {
	CommentID       *string `json:"commentId,omitempty"`
	ParentCommentID *string `json:"parentCommentId,omitempty"`
	Body            string  `json:"body"`
}

type reportTokenBlueprintCommentRequest struct {
	Reason string `json:"reason"`
	Detail string `json:"detail"`
}

// ============================================================
// Helpers
// ============================================================

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

// ============================================================
// Aggregate read handler
// ============================================================

func (h *TokenBlueprintReviewHandler) getAggregate(
	w http.ResponseWriter,
	r *http.Request,
	tokenBlueprintID string,
) {
	result, err :=
		h.query.GetAggregateByTokenBlueprintID(
			r.Context(),
			appquery.GetMallTokenBlueprintReviewAggregateInput{
				TokenBlueprintID: tokenBlueprintID,
			},
		)
	if err != nil {
		if isNotFound(err) {
			notFound(w)
			return
		}

		internalError(w, err.Error())
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		result,
	)
}

// ============================================================
// TokenBlueprint reaction command handler
// ============================================================

func (h *TokenBlueprintReviewHandler) upsertTokenBlueprintReaction(
	w http.ResponseWriter,
	r *http.Request,
	tokenBlueprintID string,
) {
	avatarID, ok := mw.CurrentAvatarID(r)
	if !ok || avatarID == "" {
		writeJSON(
			w,
			http.StatusUnauthorized,
			map[string]string{
				"error": "unauthorized",
			},
		)
		return
	}

	var request reactionRequest

	if err := readJSON(r, &request); err != nil {
		badRequest(w, err.Error())
		return
	}

	if err := request.Type.Validate(); err != nil {
		badRequest(w, err.Error())
		return
	}

	result, err :=
		h.uc.ReactToTokenBlueprintDetailed(
			r.Context(),
			tokenBlueprintID,
			avatarID,
			h.query.ActorType(),
			request.Type,
		)
	if err != nil {
		badRequest(w, err.Error())
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		map[string]any{
			"TokenBlueprintID": result.Aggregate.TokenBlueprintID,
			"ActorID":          avatarID,
			"ActorType":        h.query.ActorType(),
			"Type":             result.Reaction.Type,
			"LikeCount":        result.Aggregate.LikeCount,
			"DislikeCount":     result.Aggregate.DislikeCount,
			"TopLevelCommentCount": result.Aggregate.
				TopLevelCommentCount,
			"TotalCommentCount": result.Aggregate.
				TotalCommentCount,
		},
	)
}

// ============================================================
// Comment read / command handlers
// ============================================================

func (h *TokenBlueprintReviewHandler) listComments(
	w http.ResponseWriter,
	r *http.Request,
	tokenBlueprintID string,
) {
	result, err :=
		h.query.ListCommentsByTokenBlueprintID(
			r.Context(),
			appquery.ListMallTokenBlueprintCommentsInput{
				TokenBlueprintID: tokenBlueprintID,

				SearchQuery: r.URL.Query().Get("q"),
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
							r.URL.Query().Get(
								"order",
							),
						),
					),
				},
				Page: common.Page{
					Number: parseIntDefault(
						r.URL.Query().Get(
							"page",
						),
						1,
					),
					PerPage: parseIntDefault(
						r.URL.Query().Get(
							"perPage",
						),
						0,
					),
				},
			},
		)
	if err != nil {
		internalError(w, err.Error())
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		result,
	)
}

func (h *TokenBlueprintReviewHandler) createComment(
	w http.ResponseWriter,
	r *http.Request,
	tokenBlueprintID string,
) {
	authorAvatarID, ok := mw.CurrentAvatarID(r)
	if !ok || authorAvatarID == "" {
		writeJSON(
			w,
			http.StatusUnauthorized,
			map[string]string{
				"error": "unauthorized",
			},
		)
		return
	}

	var request createCommentRequest

	if err := readJSON(r, &request); err != nil {
		badRequest(w, err.Error())
		return
	}

	created, err := h.uc.CreateComment(
		r.Context(),
		appusecase.CreateCommentInput{
			CommentID: ptrStr(
				request.CommentID,
			),
			TokenBlueprintID: tokenBlueprintID,
			ParentCommentID: ptrStr(
				request.ParentCommentID,
			),
			AuthorID:       authorAvatarID,
			AuthorType:     h.query.AuthorType(),
			IsOwnerComment: false,
			Body:           request.Body,
		},
	)
	if err != nil {
		if isNotFound(err) {
			notFound(w)
			return
		}

		badRequest(w, err.Error())
		return
	}

	writeJSON(
		w,
		http.StatusCreated,
		h.query.ToCommentReadModel(
			h.uc.BuildComment(
				r.Context(),
				created,
			),
		),
	)
}

func (h *TokenBlueprintReviewHandler) createReplyComment(
	w http.ResponseWriter,
	r *http.Request,
	tokenBlueprintID string,
	parentCommentID string,
) {
	authorAvatarID, ok := mw.CurrentAvatarID(r)
	if !ok || authorAvatarID == "" {
		writeJSON(
			w,
			http.StatusUnauthorized,
			map[string]string{
				"error": "unauthorized",
			},
		)
		return
	}

	var request createCommentRequest

	if err := readJSON(r, &request); err != nil {
		badRequest(w, err.Error())
		return
	}

	created, err := h.uc.CreateComment(
		r.Context(),
		appusecase.CreateCommentInput{
			CommentID: ptrStr(
				request.CommentID,
			),
			TokenBlueprintID: tokenBlueprintID,
			ParentCommentID:  parentCommentID,
			AuthorID:         authorAvatarID,
			AuthorType:       h.query.AuthorType(),
			IsOwnerComment:   false,
			Body:             request.Body,
		},
	)
	if err != nil {
		if isNotFound(err) {
			notFound(w)
			return
		}

		badRequest(w, err.Error())
		return
	}

	writeJSON(
		w,
		http.StatusCreated,
		h.query.ToCommentReadModel(
			h.uc.BuildComment(
				r.Context(),
				created,
			),
		),
	)
}

func (h *TokenBlueprintReviewHandler) deleteComment(
	w http.ResponseWriter,
	r *http.Request,
	tokenBlueprintID string,
	commentID string,
) {
	avatarID, ok := mw.CurrentAvatarID(r)
	if !ok || avatarID == "" {
		writeJSON(
			w,
			http.StatusUnauthorized,
			map[string]string{
				"error": "unauthorized",
			},
		)
		return
	}

	err := h.uc.DeleteComment(
		r.Context(),
		appusecase.DeleteCommentInput{
			TokenBlueprintID: tokenBlueprintID,
			CommentID:        commentID,
			AuthorID:         avatarID,
			AuthorType:       h.query.AuthorType(),
		},
	)
	if err != nil {
		if errors.Is(
			err,
			appusecase.ErrCommentDeleteForbidden,
		) {
			writeJSON(
				w,
				http.StatusForbidden,
				map[string]string{
					"error": err.Error(),
				},
			)
			return
		}

		if isNotFound(err) {
			notFound(w)
			return
		}

		internalError(w, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ============================================================
// Comment report command handler
// ============================================================

func (h *TokenBlueprintReviewHandler) reportComment(
	w http.ResponseWriter,
	r *http.Request,
	tokenBlueprintID string,
	commentID string,
) {
	if h.reportSvc == nil {
		writeJSON(
			w,
			http.StatusServiceUnavailable,
			map[string]string{
				"error": "report service not configured",
			},
		)
		return
	}

	avatarID, ok := mw.CurrentAvatarID(r)
	if !ok || avatarID == "" {
		writeJSON(
			w,
			http.StatusUnauthorized,
			map[string]string{
				"error": "unauthorized",
			},
		)
		return
	}

	var request reportTokenBlueprintCommentRequest
	if err := readJSON(r, &request); err != nil {
		badRequest(w, err.Error())
		return
	}

	reason := reportdom.ReportReason(
		strings.ToUpper(strings.TrimSpace(request.Reason)),
	)
	if err := reason.Validate(); err != nil {
		badRequest(w, err.Error())
		return
	}

	detail := strings.TrimSpace(request.Detail)
	if reason == reportdom.ReportReasonOther && detail == "" {
		badRequest(w, reportdom.ErrReportDetailRequired.Error())
		return
	}

	result, err := h.reportSvc.ReportTokenBlueprintCommentByAvatar(
		r.Context(),
		appusecase.ReportTokenBlueprintCommentByAvatarInput{
			TokenBlueprintID: tokenBlueprintID,
			CommentID:        commentID,
			AvatarID:         avatarID,
			Reason:           reason,
			Detail:           detail,
		},
	)
	if err != nil {
		writeTokenBlueprintCommentReportError(w, err)
		return
	}

	statusCode := http.StatusCreated
	if !result.ReportCreated {
		statusCode = http.StatusOK
	}

	writeJSON(
		w,
		statusCode,
		map[string]any{
			"caseId":        string(result.Case.ID),
			"reportId":      string(result.Report.ID),
			"reportCount":   result.Case.ReportCount,
			"status":        result.Case.Status,
			"caseCreated":   result.CaseCreated,
			"reportCreated": result.ReportCreated,
		},
	)
}

func writeTokenBlueprintCommentReportError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(err, appusecase.ErrReportUsecaseNotConfigured):
		writeJSON(
			w,
			http.StatusServiceUnavailable,
			map[string]string{
				"error": "report service not configured",
			},
		)

	case errors.Is(err, appusecase.ErrReportForbidden):
		writeJSON(
			w,
			http.StatusForbidden,
			map[string]string{
				"error": "report forbidden",
			},
		)

	case errors.Is(err, appusecase.ErrReportSelfReport):
		writeJSON(
			w,
			http.StatusForbidden,
			map[string]string{
				"error": "self report is not allowed",
			},
		)

	case errors.Is(err, reportdom.ErrCannotReportRemovedTarget):
		writeJSON(
			w,
			http.StatusConflict,
			map[string]string{
				"error": "cannot report removed target",
			},
		)

	case reportdom.IsInvalid(err):
		badRequest(w, err.Error())

	case isNotFound(err):
		notFound(w)

	default:
		internalError(w, err.Error())
	}
}

// ============================================================
// Comment reaction command handler
// ============================================================

func (h *TokenBlueprintReviewHandler) upsertCommentReaction(
	w http.ResponseWriter,
	r *http.Request,
	tokenBlueprintID string,
	commentID string,
) {
	avatarID, ok := mw.CurrentAvatarID(r)
	if !ok || avatarID == "" {
		writeJSON(
			w,
			http.StatusUnauthorized,
			map[string]string{
				"error": "unauthorized",
			},
		)
		return
	}

	var request reactionRequest

	if err := readJSON(r, &request); err != nil {
		badRequest(w, err.Error())
		return
	}

	if err := request.Type.Validate(); err != nil {
		badRequest(w, err.Error())
		return
	}

	updated, err := h.uc.ReactToComment(
		r.Context(),
		tokenBlueprintID,
		commentID,
		avatarID,
		h.query.ActorType(),
		request.Type,
	)
	if err != nil {
		if isNotFound(err) {
			notFound(w)
			return
		}

		badRequest(w, err.Error())
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		h.query.ToCommentReadModel(
			h.uc.BuildComment(
				r.Context(),
				updated,
			),
		),
	)
}
