// backend/internal/adapters/in/http/mall/handler/tokenBlueprint_review_handler.go
package mallHandler

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	mw "narratives/internal/adapters/in/http/middleware"
	appusecase "narratives/internal/application/usecase"
	common "narratives/internal/domain/common"
	tokenBlueprint_review "narratives/internal/domain/tokenBlueprint_review"
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
//   - response DTO mapping
//
// - usecase: application service
//   - repository orchestration
//   - aggregate/comment/reaction state transition
//   - count update
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
	uc  tokenBlueprintReviewApplicationPort
	now func() time.Time
}

type tokenBlueprintReviewApplicationPort interface {
	ListAggregates(
		ctx context.Context,
		filter tokenBlueprint_review.FilterTokenBlueprintReviewAggregate,
		sort common.Sort,
		page common.Page,
	) (common.PageResult[tokenBlueprint_review.TokenBlueprintReviewAggregate], error)

	GetAggregate(
		ctx context.Context,
		tokenBlueprintID string,
	) (tokenBlueprint_review.TokenBlueprintReviewAggregate, error)

	ListTokenBlueprintReactions(
		ctx context.Context,
		tokenBlueprintID string,
	) ([]tokenBlueprint_review.TokenBlueprintReaction, error)

	ReactToTokenBlueprintDetailed(
		ctx context.Context,
		tokenBlueprintID string,
		actorID string,
		actorType tokenBlueprint_review.ActorType,
		newType tokenBlueprint_review.ReactionType,
	) (appusecase.TokenBlueprintReactionResult, error)

	ListComments(
		ctx context.Context,
		in appusecase.ListCommentsInput,
	) (common.PageResult[tokenBlueprint_review.Comment], error)

	CreateComment(
		ctx context.Context,
		in appusecase.CreateCommentInput,
	) (tokenBlueprint_review.Comment, error)

	DeleteComment(
		ctx context.Context,
		tokenBlueprintID string,
		commentID string,
	) error

	ReactToComment(
		ctx context.Context,
		tokenBlueprintID string,
		commentID string,
		actorID string,
		actorType tokenBlueprint_review.ActorType,
		newType tokenBlueprint_review.ReactionType,
	) (tokenBlueprint_review.Comment, error)

	GetNameAndIconByID(
		ctx context.Context,
		avatarID string,
	) (name string, icon string, err error)

	GetBrandNameAndIconByID(
		ctx context.Context,
		brandID string,
	) (name string, icon string, err error)
}

func NewTokenBlueprintReviewHandler(
	uc tokenBlueprintReviewApplicationPort,
) *TokenBlueprintReviewHandler {
	return &TokenBlueprintReviewHandler{
		uc:  uc,
		now: time.Now,
	}
}

func (h *TokenBlueprintReviewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.uc == nil {
		internalError(w, "handler not configured")
		return
	}

	path := strings.TrimSuffix(r.URL.Path, "/")

	// aggregate list
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
	Type tokenBlueprint_review.ReactionType `json:"type"`
}

type createCommentRequest struct {
	CommentID       *string `json:"commentId,omitempty"`
	ParentCommentID *string `json:"parentCommentId,omitempty"`
	Body            string  `json:"body"`
}

// ============================================================
// Response DTO
// ============================================================

type CommentDTO struct {
	CommentID        string `json:"CommentID"`
	TokenBlueprintID string `json:"TokenBlueprintID"`
	ParentCommentID  string `json:"ParentCommentID"`
	RootCommentID    string `json:"RootCommentID"`
	Depth            int    `json:"Depth"`
	AuthorID         string `json:"AuthorID"`
	AuthorType       string `json:"AuthorType"`

	AuthorAvatarName string  `json:"AuthorAvatarName"`
	AuthorAvatarIcon *string `json:"AuthorAvatarIcon"`

	BrandName string  `json:"BrandName"`
	BrandIcon *string `json:"BrandIcon"`

	IsOwnerComment bool `json:"IsOwnerComment"`

	Body         string `json:"Body"`
	LikeCount    int64  `json:"LikeCount"`
	DislikeCount int64  `json:"DislikeCount"`
	ChildCount   int64  `json:"ChildCount"`
	Deleted      bool   `json:"Deleted"`

	CreatedAt string `json:"CreatedAt"`
	UpdatedAt string `json:"UpdatedAt"`
}

type aggregateListItem struct {
	tokenBlueprint_review.TokenBlueprintReviewAggregate
}

type tokenBlueprintReactionItem struct {
	TokenBlueprintID string                             `json:"TokenBlueprintID"`
	ActorID          string                             `json:"ActorID"`
	ActorType        tokenBlueprint_review.ActorType    `json:"ActorType"`
	Type             tokenBlueprint_review.ReactionType `json:"Type"`
	CreatedAt        string                             `json:"CreatedAt"`
	UpdatedAt        string                             `json:"UpdatedAt"`
	AuthorAvatarName string                             `json:"AuthorAvatarName"`
	AuthorAvatarIcon *string                            `json:"AuthorAvatarIcon"`
	BrandName        string                             `json:"BrandName"`
	BrandIcon        *string                            `json:"BrandIcon"`
}

type tokenBlueprintReactionListResponse struct {
	Items []tokenBlueprintReactionItem `json:"items"`
}

type aggregateListResponse struct {
	Items []aggregateListItem `json:"items"`
}

type commentListResponse struct {
	Items      []CommentDTO `json:"items"`
	Page       int          `json:"page"`
	PerPage    int          `json:"perPage"`
	TotalCount int          `json:"totalCount"`
}

// ============================================================
// DTO mapping / lightweight resolution
// ============================================================

func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (h *TokenBlueprintReviewHandler) resolveAvatarNameIconBestEffort(ctx context.Context, avatarID string) (string, *string) {
	if h == nil || h.uc == nil || avatarID == "" {
		return "", nil
	}

	name, icon, err := h.uc.GetNameAndIconByID(ctx, avatarID)
	if err != nil {
		return "", nil
	}

	return name, strPtrOrNil(icon)
}

func (h *TokenBlueprintReviewHandler) resolveBrandNameIconBestEffort(ctx context.Context, brandID string) (string, *string) {
	if h == nil || h.uc == nil || brandID == "" {
		return "", nil
	}

	name, icon, err := h.uc.GetBrandNameAndIconByID(ctx, brandID)
	if err != nil {
		return "", nil
	}

	return name, strPtrOrNil(icon)
}

func (h *TokenBlueprintReviewHandler) toCommentDTO(r *http.Request, c tokenBlueprint_review.Comment) CommentDTO {
	avatarName := ""
	var avatarIcon *string

	brandName := ""
	var brandIcon *string

	switch c.AuthorType {
	case tokenBlueprint_review.AuthorTypeAvatar:
		avatarName, avatarIcon = h.resolveAvatarNameIconBestEffort(r.Context(), c.AuthorID)
	case tokenBlueprint_review.AuthorTypeBrand:
		brandName, brandIcon = h.resolveBrandNameIconBestEffort(r.Context(), c.AuthorID)
	}

	return CommentDTO{
		CommentID:        c.CommentID,
		TokenBlueprintID: c.TokenBlueprintID,
		ParentCommentID:  c.ParentCommentID,
		RootCommentID:    c.RootCommentID,
		Depth:            c.Depth,
		AuthorID:         c.AuthorID,
		AuthorType:       string(c.AuthorType),

		AuthorAvatarName: avatarName,
		AuthorAvatarIcon: avatarIcon,
		BrandName:        brandName,
		BrandIcon:        brandIcon,
		IsOwnerComment:   c.IsOwnerComment,

		Body:         c.Body,
		LikeCount:    c.LikeCount,
		DislikeCount: c.DislikeCount,
		ChildCount:   c.ChildCount,
		Deleted:      c.Deleted,

		CreatedAt: c.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt: c.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func (h *TokenBlueprintReviewHandler) toTokenBlueprintReactionItem(
	r *http.Request,
	reaction tokenBlueprint_review.TokenBlueprintReaction,
) tokenBlueprintReactionItem {
	avatarName := ""
	var avatarIcon *string

	brandName := ""
	var brandIcon *string

	switch reaction.ActorType {
	case tokenBlueprint_review.ActorTypeAvatar:
		avatarName, avatarIcon = h.resolveAvatarNameIconBestEffort(r.Context(), reaction.ActorID)
	case tokenBlueprint_review.ActorTypeBrand:
		brandName, brandIcon = h.resolveBrandNameIconBestEffort(r.Context(), reaction.ActorID)
	}

	return tokenBlueprintReactionItem{
		TokenBlueprintID: reaction.TokenBlueprintID,
		ActorID:          reaction.ActorID,
		ActorType:        reaction.ActorType,
		Type:             reaction.Type,
		CreatedAt:        reaction.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:        reaction.UpdatedAt.UTC().Format(time.RFC3339Nano),
		AuthorAvatarName: avatarName,
		AuthorAvatarIcon: avatarIcon,
		BrandName:        brandName,
		BrandIcon:        brandIcon,
	}
}

// ============================================================
// Aggregate
// ============================================================

func (h *TokenBlueprintReviewHandler) listAggregates(w http.ResponseWriter, r *http.Request) {
	res, err := h.uc.ListAggregates(
		r.Context(),
		tokenBlueprint_review.FilterTokenBlueprintReviewAggregate{},
		common.Sort{
			Column: "createdAt",
			Order:  common.SortDesc,
		},
		common.Page{
			Number:  parseIntDefault(r.URL.Query().Get("page"), 1),
			PerPage: parseIntDefault(r.URL.Query().Get("perPage"), 50),
		},
	)
	if err != nil {
		internalError(w, err.Error())
		return
	}

	items := make([]aggregateListItem, 0, len(res.Items))
	for _, it := range res.Items {
		items = append(items, aggregateListItem{
			TokenBlueprintReviewAggregate: it,
		})
	}

	writeJSON(w, http.StatusOK, aggregateListResponse{Items: items})
}

func (h *TokenBlueprintReviewHandler) getAggregate(w http.ResponseWriter, r *http.Request, tokenBlueprintID string) {
	agg, err := h.uc.GetAggregate(r.Context(), tokenBlueprintID)
	if err != nil {
		if isNotFound(err) {
			notFound(w)
			return
		}
		internalError(w, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, agg)
}

func (h *TokenBlueprintReviewHandler) listTokenBlueprintReactions(
	w http.ResponseWriter,
	r *http.Request,
	tokenBlueprintID string,
) {
	items, err := h.uc.ListTokenBlueprintReactions(r.Context(), tokenBlueprintID)
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

	out := make([]tokenBlueprintReactionItem, 0, len(items))
	for _, item := range items {
		out = append(out, h.toTokenBlueprintReactionItem(r, item))
	}

	writeJSON(w, http.StatusOK, tokenBlueprintReactionListResponse{Items: out})
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

	result, err := h.uc.ReactToTokenBlueprintDetailed(
		r.Context(),
		tokenBlueprintID,
		avatarID,
		tokenBlueprint_review.ActorTypeAvatar,
		req.Type,
	)
	if err != nil {
		badRequest(w, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"TokenBlueprintID":     result.Aggregate.TokenBlueprintID,
		"ActorID":              avatarID,
		"ActorType":            tokenBlueprint_review.ActorTypeAvatar,
		"Type":                 result.Reaction.Type,
		"LikeCount":            result.Aggregate.LikeCount,
		"DislikeCount":         result.Aggregate.DislikeCount,
		"TopLevelCommentCount": result.Aggregate.TopLevelCommentCount,
		"TotalCommentCount":    result.Aggregate.TotalCommentCount,
	})
}

// ============================================================
// Comments
// ============================================================

func (h *TokenBlueprintReviewHandler) listComments(w http.ResponseWriter, r *http.Request, tokenBlueprintID string) {
	page := common.Page{
		Number:  parseIntDefault(r.URL.Query().Get("page"), 1),
		PerPage: parseIntDefault(r.URL.Query().Get("perPage"), 0),
	}

	sort := common.Sort{
		Column: r.URL.Query().Get("sort"),
		Order:  common.SortOrder(strings.ToLower(r.URL.Query().Get("order"))),
	}
	if sort.Column == "" {
		sort.Column = "createdAt"
	}
	if sort.Order != common.SortAsc && sort.Order != common.SortDesc {
		sort.Order = common.SortDesc
	}

	var parentCommentID *string
	if raw, ok := r.URL.Query()["parentCommentId"]; ok && len(raw) > 0 {
		v := raw[0]
		parentCommentID = &v
	}

	var deleted *bool
	if raw := r.URL.Query().Get("deleted"); raw != "" {
		v := strings.EqualFold(raw, "true")
		deleted = &v
	}

	var depth *int
	if raw := r.URL.Query().Get("depth"); raw != "" {
		v := parseIntDefault(raw, 0)
		depth = &v
	}

	filter := tokenBlueprint_review.FilterComment{
		FilterCommon: common.FilterCommon{
			SearchQuery: r.URL.Query().Get("q"),
		},
		TokenBlueprintID: tokenBlueprintID,
		ParentCommentID:  parentCommentID,
		RootCommentID:    r.URL.Query().Get("rootCommentId"),
		AuthorID:         r.URL.Query().Get("authorId"),
		Deleted:          deleted,
		Depth:            depth,
	}

	res, err := h.uc.ListComments(r.Context(), appusecase.ListCommentsInput{
		TokenBlueprintID: tokenBlueprintID,
		Filter:           filter,
		Sort:             sort,
		Page:             page,
	})
	if err != nil {
		internalError(w, err.Error())
		return
	}

	out := commentListResponse{
		Items:      make([]CommentDTO, 0, len(res.Items)),
		Page:       res.Page,
		PerPage:    res.PerPage,
		TotalCount: res.TotalCount,
	}
	for _, it := range res.Items {
		out.Items = append(out.Items, h.toCommentDTO(r, it))
	}

	writeJSON(w, http.StatusOK, out)
}

func (h *TokenBlueprintReviewHandler) listChildComments(
	w http.ResponseWriter,
	r *http.Request,
	tokenBlueprintID string,
	parentCommentID string,
) {
	page := common.Page{
		Number:  parseIntDefault(r.URL.Query().Get("page"), 1),
		PerPage: parseIntDefault(r.URL.Query().Get("perPage"), 0),
	}

	sort := common.Sort{
		Column: r.URL.Query().Get("sort"),
		Order:  common.SortOrder(strings.ToLower(r.URL.Query().Get("order"))),
	}
	if sort.Column == "" {
		sort.Column = "createdAt"
	}
	if sort.Order != common.SortAsc && sort.Order != common.SortDesc {
		sort.Order = common.SortAsc
	}

	filter := tokenBlueprint_review.FilterComment{
		FilterCommon: common.FilterCommon{
			SearchQuery: r.URL.Query().Get("q"),
		},
		TokenBlueprintID: tokenBlueprintID,
		ParentCommentID:  &parentCommentID,
	}

	res, err := h.uc.ListComments(r.Context(), appusecase.ListCommentsInput{
		TokenBlueprintID: tokenBlueprintID,
		Filter:           filter,
		Sort:             sort,
		Page:             page,
	})
	if err != nil {
		internalError(w, err.Error())
		return
	}

	out := commentListResponse{
		Items:      make([]CommentDTO, 0, len(res.Items)),
		Page:       res.Page,
		PerPage:    res.PerPage,
		TotalCount: res.TotalCount,
	}
	for _, it := range res.Items {
		out.Items = append(out.Items, h.toCommentDTO(r, it))
	}

	writeJSON(w, http.StatusOK, out)
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
		AuthorType:       tokenBlueprint_review.AuthorTypeAvatar,
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

	writeJSON(w, http.StatusCreated, h.toCommentDTO(r, created))
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
		AuthorType:       tokenBlueprint_review.AuthorTypeAvatar,
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

	writeJSON(w, http.StatusCreated, h.toCommentDTO(r, created))
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
// Comment reaction
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

	updated, err := h.uc.ReactToComment(
		r.Context(),
		tokenBlueprintID,
		commentID,
		avatarID,
		tokenBlueprint_review.ActorTypeAvatar,
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

	writeJSON(w, http.StatusOK, h.toCommentDTO(r, updated))
}
