// backend/internal/adapters/in/http/console/handler/tokenBlueprintReview_handler.go
package consoleHandler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"narratives/internal/adapters/in/http/middleware"
	appquery "narratives/internal/application/query/console"
	"narratives/internal/application/usecase"
	common "narratives/internal/domain/common"
	tbReview "narratives/internal/domain/tokenBlueprint_review"
)

var (
	errUnauthorized = errors.New("unauthorized")
)

type TokenBlueprintReviewHandler struct {
	uc    *usecase.TokenBlueprintReviewUsecase
	query *appquery.TokenBlueprintReviewConsoleQuery
}

func NewTokenBlueprintReviewHandler(
	uc *usecase.TokenBlueprintReviewUsecase,
) *TokenBlueprintReviewHandler {
	return &TokenBlueprintReviewHandler{
		uc:    uc,
		query: appquery.NewTokenBlueprintReviewConsoleQuery(uc),
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
//
// console handler では brand 側からのみ comment / reply / comment reaction を許可する。
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
	rest = strings.TrimPrefix(rest, "/")

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

type createBrandCommentResponse struct {
	Item appquery.ConsoleTokenBlueprintCommentReadModel `json:"item"`
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

func toConsoleCommentReadModel(
	view usecase.CommentView,
) appquery.ConsoleTokenBlueprintCommentReadModel {
	comment := view.Comment

	return appquery.ConsoleTokenBlueprintCommentReadModel{
		CommentID:        comment.CommentID,
		TokenBlueprintID: comment.TokenBlueprintID,
		ParentCommentID:  comment.ParentCommentID,
		RootCommentID:    comment.RootCommentID,
		Depth:            comment.Depth,
		AuthorID:         comment.AuthorID,
		AuthorType:       string(comment.AuthorType),

		AuthorAvatarName: view.AuthorAvatarName,
		AuthorAvatarIcon: view.AuthorAvatarIcon,
		BrandName:        view.BrandName,
		BrandIcon:        view.BrandIcon,
		IsOwnerComment:   comment.IsOwnerComment,

		Body:         comment.Body,
		LikeCount:    comment.LikeCount,
		DislikeCount: comment.DislikeCount,
		ChildCount:   comment.ChildCount,
		Deleted:      comment.Deleted,

		CreatedAt: formatRFC3339NanoUTC(comment.CreatedAt),
		UpdatedAt: formatRFC3339NanoUTC(comment.UpdatedAt),
	}
}

func formatRFC3339NanoUTC(value time.Time) string {
	if value.IsZero() {
		return ""
	}

	return value.UTC().Format(time.RFC3339Nano)
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
			Item: toConsoleCommentReadModel(
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
			Item: toConsoleCommentReadModel(
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
			"item": toConsoleCommentReadModel(
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
