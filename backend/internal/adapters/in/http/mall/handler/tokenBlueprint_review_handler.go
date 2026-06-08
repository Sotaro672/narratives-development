// backend/internal/adapters/in/http/mall/handler/tokenBlueprint_review_handler.go
package mallHandler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	mw "narratives/internal/adapters/in/http/middleware"
	appquery "narratives/internal/application/query/mall"
	appusecase "narratives/internal/application/usecase"
	common "narratives/internal/domain/common"
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
//   - aggregate/comment/reaction read model composition
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
// - GET    /mall/me/token-blueprints
// - GET    /mall/me/token-blueprints/{id}/reviews/aggregate
// - GET    /mall/me/token-blueprints/{id}/reactions
// - POST   /mall/me/token-blueprints/{id}/reactions
// - GET    /mall/me/token-blueprints/{id}/comments
// - POST   /mall/me/token-blueprints/{id}/comments
// - DELETE /mall/me/token-blueprints/{id}/comments/{commentId}
// - POST   /mall/me/token-blueprints/{id}/comments/{commentId}/reactions
// - GET    /mall/me/token-blueprints/{id}/comments/{commentId}/replies
// - POST   /mall/me/token-blueprints/{id}/comments/{commentId}/replies
type TokenBlueprintReviewHandler struct {
	uc    *appusecase.TokenBlueprintReviewUsecase
	query *appquery.TokenBlueprintReviewMallQuery
}

func NewTokenBlueprintReviewHandler(
	uc *appusecase.TokenBlueprintReviewUsecase,
) *TokenBlueprintReviewHandler {
	return &TokenBlueprintReviewHandler{
		uc:    uc,
		query: appquery.NewTokenBlueprintReviewMallQuery(uc),
	}
}

func (h *TokenBlueprintReviewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.uc == nil || h.query == nil {
		internalError(w, "handler not configured")
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/")

	// GET /mall/me/token-blueprints
	if path == "/mall/me/token-blueprints" {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		h.listAggregates(w, r)
		return
	}

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
		h.getAggregate(w, r, tokenBlueprintID)
		return

	case strings.HasSuffix(path, "/reactions") && isTokenBlueprintReactionPath(path, tokenBlueprintID):
		switch r.Method {
		case http.MethodGet:
			h.listTokenBlueprintReactions(w, r, tokenBlueprintID)
		case http.MethodPost:
			h.upsertTokenBlueprintReaction(w, r, tokenBlueprintID)
		default:
			methodNotAllowed(w)
		}
		return

	case strings.Contains(path, "/comments"):
		h.dispatchComments(w, r, tokenBlueprintID)
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
	const p = "/mall/me/token-blueprints/"

	if !strings.Contains(path, p) {
		return ""
	}

	idx := strings.Index(path, p)
	if idx < 0 {
		return ""
	}

	rest := path[idx+len(p):]
	if rest == "" {
		return ""
	}

	seg := rest
	if i := strings.Index(seg, "/"); i >= 0 {
		seg = seg[:i]
	}

	return seg
}

func isTokenBlueprintReactionPath(path, tokenBlueprintID string) bool {
	if tokenBlueprintID == "" {
		return false
	}

	want := "/mall/me/token-blueprints/" + tokenBlueprintID + "/reactions"
	return path == want
}

func extractCommentID(path, tokenBlueprintID string) string {
	base := "/mall/me/token-blueprints/" + tokenBlueprintID + "/comments/"
	if !strings.Contains(path, base) {
		return ""
	}

	idx := strings.Index(path, base)
	if idx < 0 {
		return ""
	}

	rest := path[idx+len(base):]
	if rest == "" {
		return ""
	}

	seg := rest
	if i := strings.Index(seg, "/"); i >= 0 {
		seg = seg[:i]
	}

	return seg
}

// ============================================================
// Comments dispatch
// ============================================================

func (h *TokenBlueprintReviewHandler) dispatchComments(w http.ResponseWriter, r *http.Request, tokenBlueprintID string) {
	path := strings.TrimSuffix(r.URL.Path, "/")
	base := "/mall/me/token-blueprints/" + tokenBlueprintID + "/comments"

	// /mall/me/token-blueprints/{id}/comments
	if path == base {
		switch r.Method {
		case http.MethodGet:
			h.listComments(w, r, tokenBlueprintID)
		case http.MethodPost:
			h.createComment(w, r, tokenBlueprintID)
		default:
			methodNotAllowed(w)
		}
		return
	}

	commentID := extractCommentID(path, tokenBlueprintID)
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
		h.deleteComment(w, r, tokenBlueprintID, commentID)
		return
	}

	// /mall/me/token-blueprints/{id}/comments/{commentId}/reactions
	if path == base+"/"+commentID+"/reactions" {
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		h.upsertCommentReaction(w, r, tokenBlueprintID, commentID)
		return
	}

	// /mall/me/token-blueprints/{id}/comments/{commentId}/replies
	if path == base+"/"+commentID+"/replies" {
		switch r.Method {
		case http.MethodGet:
			h.listChildComments(w, r, tokenBlueprintID, commentID)
		case http.MethodPost:
			h.createReplyComment(w, r, tokenBlueprintID, commentID)
		default:
			methodNotAllowed(w)
		}
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

// ============================================================
// Helpers
// ============================================================

func queryStringPtr(r *http.Request, key string) *string {
	if r == nil {
		return nil
	}

	raw, ok := r.URL.Query()[key]
	if !ok || len(raw) == 0 {
		return nil
	}

	v := raw[0]
	return &v
}

func queryBoolPtr(r *http.Request, key string) *bool {
	if r == nil {
		return nil
	}

	raw := r.URL.Query().Get(key)
	if raw == "" {
		return nil
	}

	v := strings.EqualFold(raw, "true")
	return &v
}

func queryIntPtr(r *http.Request, key string) *int {
	if r == nil {
		return nil
	}

	raw := r.URL.Query().Get(key)
	if raw == "" {
		return nil
	}

	v := parseIntDefault(raw, 0)
	return &v
}

func toMallCommentReadModel(
	view appusecase.CommentView,
) appquery.MallTokenBlueprintCommentReadModel {
	c := view.Comment

	return appquery.MallTokenBlueprintCommentReadModel{
		CommentID:        c.CommentID,
		TokenBlueprintID: c.TokenBlueprintID,
		ParentCommentID:  c.ParentCommentID,
		RootCommentID:    c.RootCommentID,
		Depth:            c.Depth,
		AuthorID:         c.AuthorID,
		AuthorType:       string(c.AuthorType),

		AuthorAvatarName: view.AuthorAvatarName,
		AuthorAvatarIcon: view.AuthorAvatarIcon,
		BrandName:        view.BrandName,
		BrandIcon:        view.BrandIcon,
		IsOwnerComment:   c.IsOwnerComment,

		Body:         c.Body,
		LikeCount:    c.LikeCount,
		DislikeCount: c.DislikeCount,
		ChildCount:   c.ChildCount,
		Deleted:      c.Deleted,

		CreatedAt: formatRFC3339NanoUTC(c.CreatedAt),
		UpdatedAt: formatRFC3339NanoUTC(c.UpdatedAt),
	}
}

func formatRFC3339NanoUTC(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

// ============================================================
// Aggregate read handlers
// ============================================================

func (h *TokenBlueprintReviewHandler) listAggregates(w http.ResponseWriter, r *http.Request) {
	res, err := h.query.ListAggregates(
		r.Context(),
		appquery.ListMallTokenBlueprintReviewAggregatesInput{
			Sort: common.Sort{
				Column: "createdAt",
				Order:  common.SortDesc,
			},
			Page: common.Page{
				Number:  parseIntDefault(r.URL.Query().Get("page"), 1),
				PerPage: parseIntDefault(r.URL.Query().Get("perPage"), 50),
			},
		},
	)
	if err != nil {
		internalError(w, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, res)
}

func (h *TokenBlueprintReviewHandler) getAggregate(w http.ResponseWriter, r *http.Request, tokenBlueprintID string) {
	res, err := h.query.GetAggregateByTokenBlueprintID(
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

	writeJSON(w, http.StatusOK, res)
}

// ============================================================
// TokenBlueprint reaction read / command handlers
// ============================================================

func (h *TokenBlueprintReviewHandler) listTokenBlueprintReactions(
	w http.ResponseWriter,
	r *http.Request,
	tokenBlueprintID string,
) {
	res, err := h.query.ListTokenBlueprintReactions(
		r.Context(),
		appquery.ListMallTokenBlueprintReactionsInput{
			TokenBlueprintID: tokenBlueprintID,
		},
	)
	if err != nil {
		if errors.Is(err, appusecase.ErrTokenBlueprintReactionsListNotImplemented) {
			writeJSON(w, http.StatusNotImplemented, map[string]any{
				"error": "token blueprint reactions list not implemented",
			})
			return
		}

		internalError(w, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, res)
}

func (h *TokenBlueprintReviewHandler) upsertTokenBlueprintReaction(
	w http.ResponseWriter,
	r *http.Request,
	tokenBlueprintID string,
) {
	avatarID, ok := mw.CurrentAvatarID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req reactionRequest
	if err := readJSON(r, &req); err != nil {
		badRequest(w, err.Error())
		return
	}
	if err := req.Type.Validate(); err != nil {
		badRequest(w, err.Error())
		return
	}

	result, err := h.uc.ReactToTokenBlueprintDetailed(
		r.Context(),
		tokenBlueprintID,
		avatarID,
		h.query.ActorType(),
		req.Type,
	)
	if err != nil {
		badRequest(w, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"TokenBlueprintID":     result.Aggregate.TokenBlueprintID,
		"ActorID":              avatarID,
		"ActorType":            h.query.ActorType(),
		"Type":                 result.Reaction.Type,
		"LikeCount":            result.Aggregate.LikeCount,
		"DislikeCount":         result.Aggregate.DislikeCount,
		"TopLevelCommentCount": result.Aggregate.TopLevelCommentCount,
		"TotalCommentCount":    result.Aggregate.TotalCommentCount,
	})
}

// ============================================================
// Comment read / command handlers
// ============================================================

func (h *TokenBlueprintReviewHandler) listComments(w http.ResponseWriter, r *http.Request, tokenBlueprintID string) {
	res, err := h.query.ListCommentsByTokenBlueprintID(
		r.Context(),
		appquery.ListMallTokenBlueprintCommentsInput{
			TokenBlueprintID: tokenBlueprintID,

			SearchQuery:     r.URL.Query().Get("q"),
			ParentCommentID: queryStringPtr(r, "parentCommentId"),
			RootCommentID:   r.URL.Query().Get("rootCommentId"),
			AuthorID:        r.URL.Query().Get("authorId"),
			Deleted:         queryBoolPtr(r, "deleted"),
			Depth:           queryIntPtr(r, "depth"),

			Sort: common.Sort{
				Column: r.URL.Query().Get("sort"),
				Order:  common.SortOrder(strings.ToLower(r.URL.Query().Get("order"))),
			},
			Page: common.Page{
				Number:  parseIntDefault(r.URL.Query().Get("page"), 1),
				PerPage: parseIntDefault(r.URL.Query().Get("perPage"), 0),
			},
		},
	)
	if err != nil {
		internalError(w, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, res)
}

func (h *TokenBlueprintReviewHandler) listChildComments(
	w http.ResponseWriter,
	r *http.Request,
	tokenBlueprintID string,
	parentCommentID string,
) {
	res, err := h.query.ListRepliesByCommentID(
		r.Context(),
		appquery.ListMallTokenBlueprintRepliesInput{
			TokenBlueprintID: tokenBlueprintID,
			ParentCommentID:  parentCommentID,
			SearchQuery:      r.URL.Query().Get("q"),
			Sort: common.Sort{
				Column: r.URL.Query().Get("sort"),
				Order:  common.SortOrder(strings.ToLower(r.URL.Query().Get("order"))),
			},
			Page: common.Page{
				Number:  parseIntDefault(r.URL.Query().Get("page"), 1),
				PerPage: parseIntDefault(r.URL.Query().Get("perPage"), 0),
			},
		},
	)
	if err != nil {
		internalError(w, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, res)
}

func (h *TokenBlueprintReviewHandler) createComment(w http.ResponseWriter, r *http.Request, tokenBlueprintID string) {
	authorAvatarID, ok := mw.CurrentAvatarID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req createCommentRequest
	if err := readJSON(r, &req); err != nil {
		badRequest(w, err.Error())
		return
	}

	created, err := h.uc.CreateComment(r.Context(), appusecase.CreateCommentInput{
		CommentID:        ptrStr(req.CommentID),
		TokenBlueprintID: tokenBlueprintID,
		ParentCommentID:  ptrStr(req.ParentCommentID),
		AuthorID:         authorAvatarID,
		AuthorType:       h.query.AuthorType(),
		IsOwnerComment:   false,
		Body:             req.Body,
	})
	if err != nil {
		if isNotFound(err) {
			notFound(w)
			return
		}
		badRequest(w, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, toMallCommentReadModel(h.uc.BuildComment(r.Context(), created)))
}

func (h *TokenBlueprintReviewHandler) createReplyComment(
	w http.ResponseWriter,
	r *http.Request,
	tokenBlueprintID string,
	parentCommentID string,
) {
	authorAvatarID, ok := mw.CurrentAvatarID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req createCommentRequest
	if err := readJSON(r, &req); err != nil {
		badRequest(w, err.Error())
		return
	}

	created, err := h.uc.CreateComment(r.Context(), appusecase.CreateCommentInput{
		CommentID:        ptrStr(req.CommentID),
		TokenBlueprintID: tokenBlueprintID,
		ParentCommentID:  parentCommentID,
		AuthorID:         authorAvatarID,
		AuthorType:       h.query.AuthorType(),
		IsOwnerComment:   false,
		Body:             req.Body,
	})
	if err != nil {
		if isNotFound(err) {
			notFound(w)
			return
		}
		badRequest(w, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, toMallCommentReadModel(h.uc.BuildComment(r.Context(), created)))
}

func (h *TokenBlueprintReviewHandler) deleteComment(w http.ResponseWriter, r *http.Request, tokenBlueprintID, commentID string) {
	if _, ok := mw.CurrentAvatarID(r); !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	if err := h.uc.DeleteComment(r.Context(), tokenBlueprintID, commentID); err != nil {
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
// Comment reaction command handler
// ============================================================

func (h *TokenBlueprintReviewHandler) upsertCommentReaction(
	w http.ResponseWriter,
	r *http.Request,
	tokenBlueprintID string,
	commentID string,
) {
	avatarID, ok := mw.CurrentAvatarID(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req reactionRequest
	if err := readJSON(r, &req); err != nil {
		badRequest(w, err.Error())
		return
	}
	if err := req.Type.Validate(); err != nil {
		badRequest(w, err.Error())
		return
	}

	updated, err := h.uc.ReactToComment(
		r.Context(),
		tokenBlueprintID,
		commentID,
		avatarID,
		h.query.ActorType(),
		req.Type,
	)
	if err != nil {
		if isNotFound(err) {
			notFound(w)
			return
		}
		badRequest(w, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, toMallCommentReadModel(h.uc.BuildComment(r.Context(), updated)))
}
