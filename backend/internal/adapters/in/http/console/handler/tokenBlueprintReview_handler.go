// backend/internal/adapters/in/http/console/handler/tokenBlueprintReview_handler.go
package consoleHandler

import (
	"encoding/json"
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
// - GET    /token-blueprint-reviews/{tokenBlueprintId}
// - GET    /token-blueprint-reviews/{tokenBlueprintId}/comments
// - POST   /token-blueprint-reviews/{tokenBlueprintId}/comments
// - DELETE /token-blueprint-reviews/{tokenBlueprintId}/comments/{commentId}
// - POST   /token-blueprint-reviews/{tokenBlueprintId}/comments/{commentId}/reactions
// - GET    /token-blueprint-reviews/{tokenBlueprintId}/comments/{commentId}/replies
// - POST   /token-blueprint-reviews/{tokenBlueprintId}/comments/{commentId}/replies
//
// console handler では brand 側からのみ comment / reply / comment reaction を許可する。
// aggregate への reaction は扱わない。
func (h *TokenBlueprintReviewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.uc == nil || h.query == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "handler not configured"})
		return
	}

	rest := strings.TrimPrefix(r.URL.Path, "/token-blueprint-reviews")
	rest = strings.TrimPrefix(rest, "/")
	if rest == "" {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.ListAggregatesByCompanyTokenBlueprints(w, r)
		return
	}

	parts := strings.Split(rest, "/")
	tbID := parts[0]
	if tbID == "" {
		http.Error(w, "tokenBlueprintId is required", http.StatusBadRequest)
		return
	}

	// /{id}
	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.GetAggregateByID(w, r, tbID)
		return
	}

	switch parts[1] {
	case "comments":
		if len(parts) == 2 {
			switch r.Method {
			case http.MethodGet:
				h.ListCommentsByTokenBlueprintID(w, r, tbID)
			case http.MethodPost:
				h.CreateCommentAsBrand(w, r, tbID)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
			return
		}

		commentID := parts[2]
		if commentID == "" {
			http.Error(w, "commentId is required", http.StatusBadRequest)
			return
		}

		// DELETE /{id}/comments/{commentId}
		if len(parts) == 3 {
			if r.Method != http.MethodDelete {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			h.DeleteComment(w, r, tbID, commentID)
			return
		}

		// POST /{id}/comments/{commentId}/reactions
		if len(parts) == 4 && parts[3] == "reactions" {
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			h.ReactToCommentAsBrand(w, r, tbID, commentID)
			return
		}

		// GET|POST /{id}/comments/{commentId}/replies
		if len(parts) == 4 && parts[3] == "replies" {
			switch r.Method {
			case http.MethodGet:
				h.ListChildCommentsByTokenBlueprintID(w, r, tbID, commentID)
			case http.MethodPost:
				h.CreateBrandReply(w, r, tbID, commentID)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
			return
		}

		http.Error(w, "not found", http.StatusNotFound)
		return

	default:
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
}

// ================================
// Requests / Responses
// ================================

type createBrandReplyRequest struct {
	CommentID       *string `json:"commentId,omitempty"`
	ParentCommentID *string `json:"parentCommentId,omitempty"`
	Body            string  `json:"body"`
}

type reactAsBrandRequest struct {
	Type tbReview.ReactionType `json:"type"`
}

type createBrandReplyResponse struct {
	Item appquery.ConsoleTokenBlueprintCommentReadModel `json:"item"`
}

// ================================
// Helpers
// ================================

func decodeJSONBody(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func ptrStr(v *string) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(*v)
}

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

func toConsoleCommentReadModel(
	view usecase.CommentView,
) appquery.ConsoleTokenBlueprintCommentReadModel {
	c := view.Comment

	return appquery.ConsoleTokenBlueprintCommentReadModel{
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

// ================================
// Read handlers
// ================================

func (h *TokenBlueprintReviewHandler) ListAggregatesByCompanyTokenBlueprints(w http.ResponseWriter, r *http.Request) {
	companyID, ok := middleware.CompanyID(r)
	if !ok || companyID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": errUnauthorized.Error()})
		return
	}

	res, err := h.query.ListAggregatesByCompanyTokenBlueprints(
		r.Context(),
		appquery.ListConsoleTokenBlueprintReviewAggregatesInput{
			CompanyID: companyID,
		},
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, res)
}

func (h *TokenBlueprintReviewHandler) GetAggregateByID(w http.ResponseWriter, r *http.Request, tokenBlueprintID string) {
	companyID, ok := middleware.CompanyID(r)
	if !ok || companyID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": errUnauthorized.Error()})
		return
	}

	res, err := h.query.GetAggregateByTokenBlueprintID(
		r.Context(),
		appquery.GetConsoleTokenBlueprintReviewAggregateInput{
			CompanyID:        companyID,
			TokenBlueprintID: tokenBlueprintID,
		},
	)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}

	writeJSON(w, http.StatusOK, res)
}

func (h *TokenBlueprintReviewHandler) ListCommentsByTokenBlueprintID(w http.ResponseWriter, r *http.Request, tokenBlueprintID string) {
	companyID, ok := middleware.CompanyID(r)
	if !ok || companyID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": errUnauthorized.Error()})
		return
	}

	res, err := h.query.ListCommentsByTokenBlueprintID(
		r.Context(),
		appquery.ListConsoleTokenBlueprintCommentsInput{
			CompanyID:        companyID,
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
				PerPage: parseIntDefault(r.URL.Query().Get("perPage"), 200),
			},
		},
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, res)
}

func (h *TokenBlueprintReviewHandler) ListChildCommentsByTokenBlueprintID(
	w http.ResponseWriter,
	r *http.Request,
	tokenBlueprintID string,
	parentCommentID string,
) {
	companyID, ok := middleware.CompanyID(r)
	if !ok || companyID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": errUnauthorized.Error()})
		return
	}

	res, err := h.query.ListRepliesByCommentID(
		r.Context(),
		appquery.ListConsoleTokenBlueprintRepliesInput{
			CompanyID:        companyID,
			TokenBlueprintID: tokenBlueprintID,
			ParentCommentID:  parentCommentID,
			SearchQuery:      r.URL.Query().Get("q"),
			Sort: common.Sort{
				Column: r.URL.Query().Get("sort"),
				Order:  common.SortOrder(strings.ToLower(r.URL.Query().Get("order"))),
			},
			Page: common.Page{
				Number:  parseIntDefault(r.URL.Query().Get("page"), 1),
				PerPage: parseIntDefault(r.URL.Query().Get("perPage"), 200),
			},
		},
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, res)
}

// ================================
// Command handlers
// ================================

func (h *TokenBlueprintReviewHandler) CreateCommentAsBrand(w http.ResponseWriter, r *http.Request, tokenBlueprintID string) {
	companyID, ok := middleware.CompanyID(r)
	if !ok || companyID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": errUnauthorized.Error()})
		return
	}
	_ = companyID

	var req createBrandReplyRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	actor, err := h.query.ResolveBrandActor(r.Context(), tokenBlueprintID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	created, err := h.uc.CreateComment(r.Context(), usecase.CreateCommentInput{
		CommentID:        ptrStr(req.CommentID),
		TokenBlueprintID: tokenBlueprintID,
		ParentCommentID:  ptrStr(req.ParentCommentID),
		AuthorID:         actor.BrandID,
		AuthorType:       h.query.AuthorType(),
		IsOwnerComment:   true,
		Body:             req.Body,
	})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, createBrandReplyResponse{
		Item: toConsoleCommentReadModel(h.uc.BuildComment(r.Context(), created)),
	})
}

// CreateBrandReply
//
// POST /token-blueprint-reviews/{tokenBlueprintId}/comments/{commentId}/replies
func (h *TokenBlueprintReviewHandler) CreateBrandReply(
	w http.ResponseWriter,
	r *http.Request,
	tokenBlueprintID string,
	parentCommentID string,
) {
	companyID, ok := middleware.CompanyID(r)
	if !ok || companyID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": errUnauthorized.Error()})
		return
	}
	_ = companyID

	var req createBrandReplyRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	actor, err := h.query.ResolveBrandActor(r.Context(), tokenBlueprintID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	created, err := h.uc.CreateComment(r.Context(), usecase.CreateCommentInput{
		CommentID:        ptrStr(req.CommentID),
		TokenBlueprintID: tokenBlueprintID,
		ParentCommentID:  parentCommentID,
		AuthorID:         actor.BrandID,
		AuthorType:       h.query.AuthorType(),
		IsOwnerComment:   true,
		Body:             req.Body,
	})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, createBrandReplyResponse{
		Item: toConsoleCommentReadModel(h.uc.BuildComment(r.Context(), created)),
	})
}

func (h *TokenBlueprintReviewHandler) DeleteComment(
	w http.ResponseWriter,
	r *http.Request,
	tokenBlueprintID string,
	commentID string,
) {
	companyID, ok := middleware.CompanyID(r)
	if !ok || companyID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": errUnauthorized.Error()})
		return
	}
	_ = companyID

	if err := h.uc.DeleteComment(r.Context(), tokenBlueprintID, commentID); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ReactToCommentAsBrand
//
// POST /token-blueprint-reviews/{tokenBlueprintId}/comments/{commentId}/reactions
func (h *TokenBlueprintReviewHandler) ReactToCommentAsBrand(
	w http.ResponseWriter,
	r *http.Request,
	tokenBlueprintID string,
	commentID string,
) {
	companyID, ok := middleware.CompanyID(r)
	if !ok || companyID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": errUnauthorized.Error()})
		return
	}
	_ = companyID

	var req reactAsBrandRequest
	if err := decodeJSONBody(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	if err := req.Type.Validate(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	actor, err := h.query.ResolveBrandActor(r.Context(), tokenBlueprintID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	updated, err := h.uc.ReactToComment(
		r.Context(),
		tokenBlueprintID,
		commentID,
		actor.BrandID,
		h.query.ActorType(),
		req.Type,
	)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"item": toConsoleCommentReadModel(h.uc.BuildComment(r.Context(), updated)),
		"actor": map[string]any{
			"actorType":  h.query.ActorType(),
			"authorType": h.query.AuthorType(),
			"brandId":    actor.BrandID,
			"brandName":  actor.BrandName,
			"brandIcon":  actor.BrandIcon,
		},
	})
}
