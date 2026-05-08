// backend/internal/adapters/in/http/console/handler/tokenBlueprintReview_handler.go
package consoleHandler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"narratives/internal/adapters/in/http/middleware"
	"narratives/internal/application/usecase"
	common "narratives/internal/domain/common"
	tbReview "narratives/internal/domain/tokenBlueprint_review"
)

var (
	errUnauthorized = errors.New("unauthorized")
)

type TokenBlueprintReviewHandler struct {
	uc *usecase.TokenBlueprintReviewUsecase
}

func NewTokenBlueprintReviewHandler(
	uc *usecase.TokenBlueprintReviewUsecase,
) *TokenBlueprintReviewHandler {
	return &TokenBlueprintReviewHandler{
		uc: uc,
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

type tokenBlueprintAggregateItem struct {
	tbReview.TokenBlueprintReviewAggregate
	TokenBlueprintName string `json:"tokenBlueprintName"`
	BrandName          string `json:"brandName"`
}

type listTokenBlueprintAggregatesResponse struct {
	Items []tokenBlueprintAggregateItem `json:"items"`
}

type getTokenBlueprintAggregateResponse struct {
	Item tokenBlueprintAggregateItem `json:"item"`
}

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

type listTokenBlueprintCommentsResponse struct {
	Items []CommentDTO `json:"items"`

	TokenBlueprintName string `json:"tokenBlueprintName"`
	BrandName          string `json:"brandName"`
}

type createBrandReplyResponse struct {
	Item CommentDTO `json:"item"`
}

// ================================
// Helpers
// ================================

func decodeJSONBody(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func ptrStr(v *string) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(*v)
}

// responseWriterKey is only used internally to avoid repeating unauthorized branches in helpers.
type responseWriterKey struct{}

func withResponseWriter(r *http.Request, w http.ResponseWriter) *http.Request {
	ctx := context.WithValue(r.Context(), responseWriterKey{}, w)
	return r.WithContext(ctx)
}

func (h *TokenBlueprintReviewHandler) resolveBrandActor(
	ctx context.Context,
	tokenBlueprintID string,
) (brandID string, brandName string, brandIcon string, err error) {
	patch, err := h.uc.GetTokenBlueprintPatchByID(ctx, tokenBlueprintID)
	if err != nil {
		return "", "", "", err
	}

	brandID = strings.TrimSpace(patch.BrandID)
	if brandID == "" {
		return "", "", "", errors.New("brandId not found on tokenBlueprint")
	}

	if n, ic, berr := h.uc.GetBrandNameAndIconByID(ctx, brandID); berr == nil {
		brandName = n
		brandIcon = ic
	}
	return brandID, brandName, brandIcon, nil
}

func (h *TokenBlueprintReviewHandler) toCommentDTO(ctx context.Context, c tbReview.Comment) CommentDTO {
	dto := CommentDTO{
		CommentID:        c.CommentID,
		TokenBlueprintID: c.TokenBlueprintID,
		ParentCommentID:  c.ParentCommentID,
		RootCommentID:    c.RootCommentID,
		Depth:            c.Depth,
		AuthorID:         c.AuthorID,
		AuthorType:       string(c.AuthorType),
		IsOwnerComment:   c.IsOwnerComment,
		Body:             c.Body,
		LikeCount:        c.LikeCount,
		DislikeCount:     c.DislikeCount,
		ChildCount:       c.ChildCount,
		Deleted:          c.Deleted,
		CreatedAt:        c.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:        c.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}

	switch c.AuthorType {
	case tbReview.AuthorTypeAvatar:
		if n, ic, err := h.uc.GetNameAndIconByID(ctx, c.AuthorID); err == nil {
			dto.AuthorAvatarName = n
			dto.AuthorAvatarIcon = strPtrOrNil(ic)
		}
	case tbReview.AuthorTypeBrand:
		if n, ic, err := h.uc.GetBrandNameAndIconByID(ctx, c.AuthorID); err == nil {
			dto.BrandName = n
			dto.BrandIcon = strPtrOrNil(ic)
		}
	}

	return dto
}

// ================================
// Handlers
// ================================

func (h *TokenBlueprintReviewHandler) ListAggregatesByCompanyTokenBlueprints(w http.ResponseWriter, r *http.Request) {
	r = withResponseWriter(r, w)

	companyID, ok := middleware.CompanyID(r)
	if !ok || companyID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": errUnauthorized.Error()})
		return
	}

	aggs, err := h.uc.ListAggregatesByCompanyTokenBlueprints(r.Context(), companyID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	items := make([]tokenBlueprintAggregateItem, 0, len(aggs))
	for _, agg := range aggs {
		tbName := ""
		brandName := ""
		if agg.TokenBlueprintID != "" {
			if p, perr := h.uc.GetTokenBlueprintPatchByID(r.Context(), agg.TokenBlueprintID); perr == nil {
				tbName = p.TokenName
				brandName = p.BrandName
			}
		}
		items = append(items, tokenBlueprintAggregateItem{
			TokenBlueprintReviewAggregate: agg,
			TokenBlueprintName:            tbName,
			BrandName:                     brandName,
		})
	}

	writeJSON(w, http.StatusOK, listTokenBlueprintAggregatesResponse{Items: items})
}

func (h *TokenBlueprintReviewHandler) GetAggregateByID(w http.ResponseWriter, r *http.Request, tokenBlueprintID string) {
	r = withResponseWriter(r, w)

	companyID, ok := middleware.CompanyID(r)
	if !ok || companyID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": errUnauthorized.Error()})
		return
	}
	_ = companyID

	agg, err := h.uc.GetAggregate(r.Context(), tokenBlueprintID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}

	tbName := ""
	brandName := ""
	if p, perr := h.uc.GetTokenBlueprintPatchByID(r.Context(), tokenBlueprintID); perr == nil {
		tbName = p.TokenName
		brandName = p.BrandName
	}

	writeJSON(w, http.StatusOK, getTokenBlueprintAggregateResponse{
		Item: tokenBlueprintAggregateItem{
			TokenBlueprintReviewAggregate: agg,
			TokenBlueprintName:            tbName,
			BrandName:                     brandName,
		},
	})
}

func (h *TokenBlueprintReviewHandler) ListCommentsByTokenBlueprintID(w http.ResponseWriter, r *http.Request, tokenBlueprintID string) {
	r = withResponseWriter(r, w)

	companyID, ok := middleware.CompanyID(r)
	if !ok || companyID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": errUnauthorized.Error()})
		return
	}
	_ = companyID

	page := common.Page{
		Number:  parseIntDefault(r.URL.Query().Get("page"), 1),
		PerPage: parseIntDefault(r.URL.Query().Get("perPage"), 200),
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

	res, err := h.uc.ListComments(r.Context(), usecase.ListCommentsInput{
		TokenBlueprintID: tokenBlueprintID,
		Filter: tbReview.FilterComment{
			FilterCommon: common.FilterCommon{
				SearchQuery: r.URL.Query().Get("q"),
			},
			TokenBlueprintID: tokenBlueprintID,
			ParentCommentID:  parentCommentID,
			RootCommentID:    r.URL.Query().Get("rootCommentId"),
			AuthorID:         r.URL.Query().Get("authorId"),
			Deleted:          deleted,
			Depth:            depth,
		},
		Sort: sort,
		Page: page,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	items := make([]CommentDTO, 0, len(res.Items))
	for _, item := range res.Items {
		items = append(items, h.toCommentDTO(r.Context(), item))
	}

	tbName := ""
	brandName := ""
	if p, perr := h.uc.GetTokenBlueprintPatchByID(r.Context(), tokenBlueprintID); perr == nil {
		tbName = p.TokenName
		brandName = p.BrandName
	}

	writeJSON(w, http.StatusOK, listTokenBlueprintCommentsResponse{
		Items:              items,
		TokenBlueprintName: tbName,
		BrandName:          brandName,
	})
}

func (h *TokenBlueprintReviewHandler) ListChildCommentsByTokenBlueprintID(w http.ResponseWriter, r *http.Request, tokenBlueprintID, parentCommentID string) {
	r = withResponseWriter(r, w)

	companyID, ok := middleware.CompanyID(r)
	if !ok || companyID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": errUnauthorized.Error()})
		return
	}
	_ = companyID

	page := common.Page{
		Number:  parseIntDefault(r.URL.Query().Get("page"), 1),
		PerPage: parseIntDefault(r.URL.Query().Get("perPage"), 200),
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

	res, err := h.uc.ListComments(r.Context(), usecase.ListCommentsInput{
		TokenBlueprintID: tokenBlueprintID,
		Filter: tbReview.FilterComment{
			FilterCommon: common.FilterCommon{
				SearchQuery: r.URL.Query().Get("q"),
			},
			TokenBlueprintID: tokenBlueprintID,
			ParentCommentID:  &parentCommentID,
		},
		Sort: sort,
		Page: page,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	items := make([]CommentDTO, 0, len(res.Items))
	for _, item := range res.Items {
		items = append(items, h.toCommentDTO(r.Context(), item))
	}

	tbName := ""
	brandName := ""
	if p, perr := h.uc.GetTokenBlueprintPatchByID(r.Context(), tokenBlueprintID); perr == nil {
		tbName = p.TokenName
		brandName = p.BrandName
	}

	writeJSON(w, http.StatusOK, listTokenBlueprintCommentsResponse{
		Items:              items,
		TokenBlueprintName: tbName,
		BrandName:          brandName,
	})
}

func (h *TokenBlueprintReviewHandler) CreateCommentAsBrand(w http.ResponseWriter, r *http.Request, tokenBlueprintID string) {
	r = withResponseWriter(r, w)

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

	brandID, _, _, err := h.resolveBrandActor(r.Context(), tokenBlueprintID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	commentID := ptrStr(req.CommentID)
	if commentID == "" {
		commentID = "cm_brand_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	}

	parentCommentID := ptrStr(req.ParentCommentID)

	created, err := h.uc.CreateComment(r.Context(), usecase.CreateCommentInput{
		CommentID:        commentID,
		TokenBlueprintID: tokenBlueprintID,
		ParentCommentID:  parentCommentID,
		AuthorID:         brandID,
		AuthorType:       tbReview.AuthorTypeBrand,
		IsOwnerComment:   true,
		Body:             req.Body,
	})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, createBrandReplyResponse{
		Item: h.toCommentDTO(r.Context(), created),
	})
}

// CreateBrandReply
//
// POST /token-blueprint-reviews/{tokenBlueprintId}/comments/{commentId}/replies
func (h *TokenBlueprintReviewHandler) CreateBrandReply(w http.ResponseWriter, r *http.Request, tokenBlueprintID, parentCommentID string) {
	r = withResponseWriter(r, w)

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

	brandID, _, _, err := h.resolveBrandActor(r.Context(), tokenBlueprintID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	commentID := ptrStr(req.CommentID)
	if commentID == "" {
		commentID = "cm_brand_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	}

	created, err := h.uc.CreateComment(r.Context(), usecase.CreateCommentInput{
		CommentID:        commentID,
		TokenBlueprintID: tokenBlueprintID,
		ParentCommentID:  parentCommentID,
		AuthorID:         brandID,
		AuthorType:       tbReview.AuthorTypeBrand,
		IsOwnerComment:   true,
		Body:             req.Body,
	})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, createBrandReplyResponse{
		Item: h.toCommentDTO(r.Context(), created),
	})
}

func (h *TokenBlueprintReviewHandler) DeleteComment(w http.ResponseWriter, r *http.Request, tokenBlueprintID, commentID string) {
	r = withResponseWriter(r, w)

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
func (h *TokenBlueprintReviewHandler) ReactToCommentAsBrand(w http.ResponseWriter, r *http.Request, tokenBlueprintID, commentID string) {
	r = withResponseWriter(r, w)

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

	brandID, brandName, brandIcon, err := h.resolveBrandActor(r.Context(), tokenBlueprintID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	updated, err := h.uc.ReactToComment(
		r.Context(),
		tokenBlueprintID,
		commentID,
		brandID,
		tbReview.ActorTypeBrand,
		req.Type,
	)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"item": h.toCommentDTO(r.Context(), updated),
		"actor": map[string]any{
			"actorType":  tbReview.ActorTypeBrand,
			"authorType": tbReview.AuthorTypeBrand,
			"brandId":    brandID,
			"brandName":  brandName,
			"brandIcon":  brandIcon,
		},
	})
}
