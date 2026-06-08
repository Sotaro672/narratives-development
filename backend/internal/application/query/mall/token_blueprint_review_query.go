// backend/internal/application/query/mall/token_blueprint_review_query.go
package mall

import (
	"context"
	"errors"
	"strings"
	"time"

	"narratives/internal/application/usecase"
	common "narratives/internal/domain/common"
	tokenBlueprintReview "narratives/internal/domain/tokenBlueprint_review"
)

var (
	ErrMallTokenBlueprintReviewQueryNotConfigured = errors.New("mall token_blueprint_review_query: service not configured")
	ErrMallTokenBlueprintIDRequired               = errors.New("mall token_blueprint_review_query: tokenBlueprintID is required")
	ErrMallAvatarIDRequired                       = errors.New("mall token_blueprint_review_query: avatarID is required")
	ErrMallCommentIDRequired                      = errors.New("mall token_blueprint_review_query: commentID is required")
)

// TokenBlueprintReviewMallQuery builds read models for mall token blueprint review screens.
//
// Responsibility:
// - mall read model composition
// - avatar actor policy for mall
// - comment / reaction display model composition
//
// Non-responsibility:
// - comment creation / deletion
// - reaction mutation
// - aggregate count update
// - HTTP request parsing
type TokenBlueprintReviewMallQuery struct {
	uc *usecase.TokenBlueprintReviewUsecase
}

func NewTokenBlueprintReviewMallQuery(
	uc *usecase.TokenBlueprintReviewUsecase,
) *TokenBlueprintReviewMallQuery {
	return &TokenBlueprintReviewMallQuery{
		uc: uc,
	}
}

// ============================================================
// Actor policy
// ============================================================

func (q *TokenBlueprintReviewMallQuery) ActorType() tokenBlueprintReview.ActorType {
	return tokenBlueprintReview.ActorTypeAvatar
}

func (q *TokenBlueprintReviewMallQuery) AuthorType() tokenBlueprintReview.AuthorType {
	return tokenBlueprintReview.AuthorTypeAvatar
}

// ============================================================
// Read models
// ============================================================

type MallTokenBlueprintReviewAggregateItem struct {
	tokenBlueprintReview.TokenBlueprintReviewAggregate
}

type MallTokenBlueprintReviewAggregateListReadModel struct {
	Items      []MallTokenBlueprintReviewAggregateItem `json:"items"`
	Page       int                                     `json:"page,omitempty"`
	PerPage    int                                     `json:"perPage,omitempty"`
	TotalCount int                                     `json:"totalCount,omitempty"`
}

type MallTokenBlueprintReviewAggregateReadModel struct {
	tokenBlueprintReview.TokenBlueprintReviewAggregate
}

type MallTokenBlueprintReactionReadModel struct {
	TokenBlueprintID string                            `json:"TokenBlueprintID"`
	ActorID          string                            `json:"ActorID"`
	ActorType        tokenBlueprintReview.ActorType    `json:"ActorType"`
	Type             tokenBlueprintReview.ReactionType `json:"Type"`
	CreatedAt        string                            `json:"CreatedAt"`
	UpdatedAt        string                            `json:"UpdatedAt"`

	AuthorAvatarName string  `json:"AuthorAvatarName"`
	AuthorAvatarIcon *string `json:"AuthorAvatarIcon"`

	BrandName string  `json:"BrandName"`
	BrandIcon *string `json:"BrandIcon"`
}

type MallTokenBlueprintReactionListReadModel struct {
	Items []MallTokenBlueprintReactionReadModel `json:"items"`
}

type MallTokenBlueprintCommentReadModel struct {
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

type MallTokenBlueprintCommentListReadModel struct {
	Items      []MallTokenBlueprintCommentReadModel `json:"items"`
	Page       int                                  `json:"page"`
	PerPage    int                                  `json:"perPage"`
	TotalCount int                                  `json:"totalCount"`
}

// ============================================================
// Inputs
// ============================================================

type ListMallTokenBlueprintReviewAggregatesInput struct {
	Page common.Page
	Sort common.Sort
}

type GetMallTokenBlueprintReviewAggregateInput struct {
	TokenBlueprintID string
}

type ListMallTokenBlueprintReactionsInput struct {
	TokenBlueprintID string
}

type ListMallTokenBlueprintCommentsInput struct {
	TokenBlueprintID string

	SearchQuery     string
	ParentCommentID *string
	RootCommentID   string
	AuthorID        string
	Deleted         *bool
	Depth           *int

	Sort common.Sort
	Page common.Page
}

type ListMallTokenBlueprintRepliesInput struct {
	TokenBlueprintID string
	ParentCommentID  string

	SearchQuery string
	Sort        common.Sort
	Page        common.Page
}

// ============================================================
// Aggregate queries
// ============================================================

func (q *TokenBlueprintReviewMallQuery) ListAggregates(
	ctx context.Context,
	in ListMallTokenBlueprintReviewAggregatesInput,
) (MallTokenBlueprintReviewAggregateListReadModel, error) {
	if err := q.validateConfigured(); err != nil {
		return MallTokenBlueprintReviewAggregateListReadModel{}, err
	}

	sort := normalizeAggregateSort(in.Sort)
	page := normalizePage(in.Page, 1, 50)

	res, err := q.uc.ListAggregates(
		ctx,
		tokenBlueprintReview.FilterTokenBlueprintReviewAggregate{},
		sort,
		page,
	)
	if err != nil {
		return MallTokenBlueprintReviewAggregateListReadModel{}, err
	}

	items := make([]MallTokenBlueprintReviewAggregateItem, 0, len(res.Items))
	for _, item := range res.Items {
		items = append(items, MallTokenBlueprintReviewAggregateItem{
			TokenBlueprintReviewAggregate: item,
		})
	}

	return MallTokenBlueprintReviewAggregateListReadModel{
		Items:      items,
		Page:       res.Page,
		PerPage:    res.PerPage,
		TotalCount: res.TotalCount,
	}, nil
}

func (q *TokenBlueprintReviewMallQuery) GetAggregateByTokenBlueprintID(
	ctx context.Context,
	in GetMallTokenBlueprintReviewAggregateInput,
) (MallTokenBlueprintReviewAggregateReadModel, error) {
	if err := q.validateConfigured(); err != nil {
		return MallTokenBlueprintReviewAggregateReadModel{}, err
	}

	tokenBlueprintID := strings.TrimSpace(in.TokenBlueprintID)
	if tokenBlueprintID == "" {
		return MallTokenBlueprintReviewAggregateReadModel{}, ErrMallTokenBlueprintIDRequired
	}

	agg, err := q.uc.GetAggregate(ctx, tokenBlueprintID)
	if err != nil {
		return MallTokenBlueprintReviewAggregateReadModel{}, err
	}

	return MallTokenBlueprintReviewAggregateReadModel{
		TokenBlueprintReviewAggregate: agg,
	}, nil
}

// ============================================================
// TokenBlueprint reaction queries
// ============================================================

func (q *TokenBlueprintReviewMallQuery) ListTokenBlueprintReactions(
	ctx context.Context,
	in ListMallTokenBlueprintReactionsInput,
) (MallTokenBlueprintReactionListReadModel, error) {
	if err := q.validateConfigured(); err != nil {
		return MallTokenBlueprintReactionListReadModel{}, err
	}

	tokenBlueprintID := strings.TrimSpace(in.TokenBlueprintID)
	if tokenBlueprintID == "" {
		return MallTokenBlueprintReactionListReadModel{}, ErrMallTokenBlueprintIDRequired
	}

	views, err := q.uc.ListTokenBlueprintReactions(ctx, tokenBlueprintID)
	if err != nil {
		return MallTokenBlueprintReactionListReadModel{}, err
	}

	return MallTokenBlueprintReactionListReadModel{
		Items: q.toReactionReadModels(views),
	}, nil
}

// ============================================================
// Comment queries
// ============================================================

func (q *TokenBlueprintReviewMallQuery) ListCommentsByTokenBlueprintID(
	ctx context.Context,
	in ListMallTokenBlueprintCommentsInput,
) (MallTokenBlueprintCommentListReadModel, error) {
	if err := q.validateConfigured(); err != nil {
		return MallTokenBlueprintCommentListReadModel{}, err
	}

	tokenBlueprintID := strings.TrimSpace(in.TokenBlueprintID)
	if tokenBlueprintID == "" {
		return MallTokenBlueprintCommentListReadModel{}, ErrMallTokenBlueprintIDRequired
	}

	sort := normalizeCommentSort(in.Sort, common.SortDesc)
	page := normalizePage(in.Page, 1, 0)

	res, err := q.uc.ListComments(ctx, usecase.ListCommentsInput{
		TokenBlueprintID: tokenBlueprintID,
		SearchQuery:      strings.TrimSpace(in.SearchQuery),
		ParentCommentID:  in.ParentCommentID,
		RootCommentID:    strings.TrimSpace(in.RootCommentID),
		AuthorID:         strings.TrimSpace(in.AuthorID),
		Deleted:          in.Deleted,
		Depth:            in.Depth,
		Sort:             sort,
		Page:             page,
	})
	if err != nil {
		return MallTokenBlueprintCommentListReadModel{}, err
	}

	return MallTokenBlueprintCommentListReadModel{
		Items:      q.toCommentReadModels(res.Items),
		Page:       res.Page,
		PerPage:    res.PerPage,
		TotalCount: res.TotalCount,
	}, nil
}

func (q *TokenBlueprintReviewMallQuery) ListRepliesByCommentID(
	ctx context.Context,
	in ListMallTokenBlueprintRepliesInput,
) (MallTokenBlueprintCommentListReadModel, error) {
	if err := q.validateConfigured(); err != nil {
		return MallTokenBlueprintCommentListReadModel{}, err
	}

	tokenBlueprintID := strings.TrimSpace(in.TokenBlueprintID)
	if tokenBlueprintID == "" {
		return MallTokenBlueprintCommentListReadModel{}, ErrMallTokenBlueprintIDRequired
	}

	parentCommentID := strings.TrimSpace(in.ParentCommentID)
	if parentCommentID == "" {
		return MallTokenBlueprintCommentListReadModel{}, ErrMallCommentIDRequired
	}

	sort := normalizeCommentSort(in.Sort, common.SortAsc)
	page := normalizePage(in.Page, 1, 0)

	res, err := q.uc.ListComments(ctx, usecase.ListCommentsInput{
		TokenBlueprintID: tokenBlueprintID,
		SearchQuery:      strings.TrimSpace(in.SearchQuery),
		ParentCommentID:  &parentCommentID,
		Sort:             sort,
		Page:             page,
	})
	if err != nil {
		return MallTokenBlueprintCommentListReadModel{}, err
	}

	return MallTokenBlueprintCommentListReadModel{
		Items:      q.toCommentReadModels(res.Items),
		Page:       res.Page,
		PerPage:    res.PerPage,
		TotalCount: res.TotalCount,
	}, nil
}

// ============================================================
// Mapping
// ============================================================

func (q *TokenBlueprintReviewMallQuery) toCommentReadModels(
	views []usecase.CommentView,
) []MallTokenBlueprintCommentReadModel {
	out := make([]MallTokenBlueprintCommentReadModel, 0, len(views))
	for _, view := range views {
		out = append(out, q.toCommentReadModel(view))
	}
	return out
}

func (q *TokenBlueprintReviewMallQuery) toCommentReadModel(
	view usecase.CommentView,
) MallTokenBlueprintCommentReadModel {
	c := view.Comment

	return MallTokenBlueprintCommentReadModel{
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

func (q *TokenBlueprintReviewMallQuery) toReactionReadModels(
	views []usecase.TokenBlueprintReactionView,
) []MallTokenBlueprintReactionReadModel {
	out := make([]MallTokenBlueprintReactionReadModel, 0, len(views))
	for _, view := range views {
		out = append(out, q.toReactionReadModel(view))
	}
	return out
}

func (q *TokenBlueprintReviewMallQuery) toReactionReadModel(
	view usecase.TokenBlueprintReactionView,
) MallTokenBlueprintReactionReadModel {
	r := view.TokenBlueprintReaction

	return MallTokenBlueprintReactionReadModel{
		TokenBlueprintID: r.TokenBlueprintID,
		ActorID:          r.ActorID,
		ActorType:        r.ActorType,
		Type:             r.Type,
		CreatedAt:        formatRFC3339NanoUTC(r.CreatedAt),
		UpdatedAt:        formatRFC3339NanoUTC(r.UpdatedAt),

		AuthorAvatarName: view.AuthorAvatarName,
		AuthorAvatarIcon: view.AuthorAvatarIcon,
		BrandName:        view.BrandName,
		BrandIcon:        view.BrandIcon,
	}
}

// ============================================================
// Helpers
// ============================================================

func (q *TokenBlueprintReviewMallQuery) validateConfigured() error {
	if q == nil || q.uc == nil {
		return ErrMallTokenBlueprintReviewQueryNotConfigured
	}
	return nil
}

func normalizeAggregateSort(sort common.Sort) common.Sort {
	column := strings.TrimSpace(sort.Column)
	if column == "" {
		column = "createdAt"
	}

	order := common.SortOrder(strings.ToLower(strings.TrimSpace(string(sort.Order))))
	if order != common.SortAsc && order != common.SortDesc {
		order = common.SortDesc
	}

	return common.Sort{
		Column: column,
		Order:  order,
	}
}

func normalizeCommentSort(sort common.Sort, fallbackOrder common.SortOrder) common.Sort {
	column := strings.TrimSpace(sort.Column)
	if column == "" {
		column = "createdAt"
	}

	order := common.SortOrder(strings.ToLower(strings.TrimSpace(string(sort.Order))))
	if order != common.SortAsc && order != common.SortDesc {
		order = fallbackOrder
	}
	if order != common.SortAsc && order != common.SortDesc {
		order = common.SortDesc
	}

	return common.Sort{
		Column: column,
		Order:  order,
	}
}

func normalizePage(page common.Page, fallbackNumber int, fallbackPerPage int) common.Page {
	number := page.Number
	if number <= 0 {
		number = fallbackNumber
	}
	if number <= 0 {
		number = 1
	}

	perPage := page.PerPage
	if perPage < 0 {
		perPage = fallbackPerPage
	}

	return common.Page{
		Number:  number,
		PerPage: perPage,
	}
}

func formatRFC3339NanoUTC(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}
