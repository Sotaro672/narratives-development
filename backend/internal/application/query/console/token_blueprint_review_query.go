// backend/internal/application/query/console/token_blueprint_review_query.go
package query

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
	ErrConsoleTokenBlueprintReviewQueryNotConfigured = errors.New("console token_blueprint_review_query: service not configured")
	ErrConsoleCompanyIDRequired                      = errors.New("console token_blueprint_review_query: companyID is required")
	ErrConsoleTokenBlueprintIDRequired               = errors.New("console token_blueprint_review_query: tokenBlueprintID is required")
	ErrConsoleBrandIDNotFound                        = errors.New("console token_blueprint_review_query: brandId not found on tokenBlueprint")
)

// TokenBlueprintReviewConsoleQuery builds read models for console token blueprint review screens.
//
// Responsibility:
// - console read model composition
// - company scope handling
// - brand actor policy for console
// - brand actor resolution from tokenBlueprint
//
// Non-responsibility:
// - comment creation / deletion
// - reaction mutation
// - aggregate count update
// - low-level avatar / brand display resolution
type TokenBlueprintReviewConsoleQuery struct {
	uc *usecase.TokenBlueprintReviewUsecase
}

func NewTokenBlueprintReviewConsoleQuery(uc *usecase.TokenBlueprintReviewUsecase) *TokenBlueprintReviewConsoleQuery {
	return &TokenBlueprintReviewConsoleQuery{uc: uc}
}

// ============================================================
// Actor policy
// ============================================================

func (q *TokenBlueprintReviewConsoleQuery) ActorType() tokenBlueprintReview.ActorType {
	return tokenBlueprintReview.ActorTypeBrand
}

func (q *TokenBlueprintReviewConsoleQuery) AuthorType() tokenBlueprintReview.AuthorType {
	return tokenBlueprintReview.AuthorTypeBrand
}

type ConsoleTokenBlueprintReviewBrandActor struct {
	BrandID   string `json:"brandId"`
	BrandName string `json:"brandName"`
	BrandIcon string `json:"brandIcon"`
}

func (q *TokenBlueprintReviewConsoleQuery) ResolveBrandActor(ctx context.Context, tokenBlueprintID string) (ConsoleTokenBlueprintReviewBrandActor, error) {
	if err := q.validateConfigured(); err != nil {
		return ConsoleTokenBlueprintReviewBrandActor{}, err
	}
	if tokenBlueprintID == "" {
		return ConsoleTokenBlueprintReviewBrandActor{}, ErrConsoleTokenBlueprintIDRequired
	}

	patch, err := q.uc.GetTokenBlueprintPatchByID(ctx, tokenBlueprintID)
	if err != nil {
		return ConsoleTokenBlueprintReviewBrandActor{}, err
	}

	brandID := patch.BrandID
	if brandID == "" {
		return ConsoleTokenBlueprintReviewBrandActor{}, ErrConsoleBrandIDNotFound
	}

	brandName := patch.BrandName
	brandIcon := ""

	if brandName == "" {
		name, icon, err := q.uc.GetBrandNameAndIconByID(ctx, brandID)
		if err == nil {
			brandName = name
			brandIcon = icon
		}
	} else {
		_, icon, err := q.uc.GetBrandNameAndIconByID(ctx, brandID)
		if err == nil {
			brandIcon = icon
		}
	}

	return ConsoleTokenBlueprintReviewBrandActor{
		BrandID:   brandID,
		BrandName: brandName,
		BrandIcon: brandIcon,
	}, nil
}

// ============================================================
// Read models
// ============================================================

type ConsoleTokenBlueprintReviewAggregateItem struct {
	TokenBlueprintID     string `json:"tokenBlueprintId"`
	TokenBlueprintName   string `json:"tokenBlueprintName"`
	BrandName            string `json:"brandName"`
	LikeCount            int64  `json:"likeCount"`
	DislikeCount         int64  `json:"dislikeCount"`
	TopLevelCommentCount int64  `json:"topLevelCommentCount"`
	TotalCommentCount    int64  `json:"totalCommentCount"`
	PinnedCommentID      string `json:"pinnedCommentId"`
	CreatedAt            string `json:"createdAt"`
	UpdatedAt            string `json:"updatedAt"`
}

type ConsoleTokenBlueprintReviewAggregateListReadModel struct {
	Items []ConsoleTokenBlueprintReviewAggregateItem `json:"items"`
}

type ConsoleTokenBlueprintCommentReadModel struct {
	CommentID        string `json:"commentId"`
	TokenBlueprintID string `json:"tokenBlueprintId"`
	ParentCommentID  string `json:"parentCommentId"`
	RootCommentID    string `json:"rootCommentId"`
	Depth            int    `json:"depth"`

	AuthorID         string  `json:"authorId"`
	AuthorType       string  `json:"authorType"`
	AuthorAvatarName string  `json:"authorAvatarName"`
	AuthorAvatarIcon *string `json:"authorAvatarIcon"`
	BrandName        string  `json:"brandName"`
	BrandIcon        *string `json:"brandIcon"`
	IsOwnerComment   bool    `json:"isOwnerComment"`

	Body         string `json:"body"`
	LikeCount    int64  `json:"likeCount"`
	DislikeCount int64  `json:"dislikeCount"`
	ChildCount   int64  `json:"childCount"`
	Deleted      bool   `json:"deleted"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

type ConsoleTokenBlueprintCommentListReadModel struct {
	Items              []ConsoleTokenBlueprintCommentReadModel `json:"items"`
	TokenBlueprintName string                                  `json:"tokenBlueprintName"`
	BrandName          string                                  `json:"brandName"`
	Page               int                                     `json:"page,omitempty"`
	PerPage            int                                     `json:"perPage,omitempty"`
	TotalCount         int                                     `json:"totalCount,omitempty"`
}

// ============================================================
// Inputs
// ============================================================

type ListConsoleTokenBlueprintReviewAggregatesInput struct {
	CompanyID string
}

type ListConsoleTokenBlueprintCommentsInput struct {
	CompanyID        string
	TokenBlueprintID string
	SearchQuery      string
	ParentCommentID  *string
	RootCommentID    string
	AuthorID         string
	Deleted          *bool
	Depth            *int
	Sort             common.Sort
	Page             common.Page
}

// ============================================================
// Aggregate queries
// ============================================================

func (q *TokenBlueprintReviewConsoleQuery) ListAggregatesByCompanyTokenBlueprints(
	ctx context.Context,
	in ListConsoleTokenBlueprintReviewAggregatesInput,
) (ConsoleTokenBlueprintReviewAggregateListReadModel, error) {
	if err := q.validateConfigured(); err != nil {
		return ConsoleTokenBlueprintReviewAggregateListReadModel{}, err
	}
	if in.CompanyID == "" {
		return ConsoleTokenBlueprintReviewAggregateListReadModel{}, ErrConsoleCompanyIDRequired
	}

	aggregates, err := q.uc.ListAggregatesByCompanyTokenBlueprints(ctx, in.CompanyID)
	if err != nil {
		return ConsoleTokenBlueprintReviewAggregateListReadModel{}, err
	}

	items := make([]ConsoleTokenBlueprintReviewAggregateItem, 0, len(aggregates))
	for _, aggregate := range aggregates {
		tokenBlueprintName, brandName := q.resolveTokenBlueprintNameBrandName(ctx, aggregate.TokenBlueprintID)

		items = append(items, ConsoleTokenBlueprintReviewAggregateItem{
			TokenBlueprintID:     aggregate.TokenBlueprintID,
			TokenBlueprintName:   tokenBlueprintName,
			BrandName:            brandName,
			LikeCount:            aggregate.LikeCount,
			DislikeCount:         aggregate.DislikeCount,
			TopLevelCommentCount: aggregate.TopLevelCommentCount,
			TotalCommentCount:    aggregate.TotalCommentCount,
			PinnedCommentID:      aggregate.PinnedCommentID,
			CreatedAt:            formatRFC3339NanoUTC(aggregate.CreatedAt),
			UpdatedAt:            formatRFC3339NanoUTC(aggregate.UpdatedAt),
		})
	}

	return ConsoleTokenBlueprintReviewAggregateListReadModel{Items: items}, nil
}

// ============================================================
// Comment queries
// ============================================================

func (q *TokenBlueprintReviewConsoleQuery) ListCommentsByTokenBlueprintID(
	ctx context.Context,
	in ListConsoleTokenBlueprintCommentsInput,
) (ConsoleTokenBlueprintCommentListReadModel, error) {
	if err := q.validateConfigured(); err != nil {
		return ConsoleTokenBlueprintCommentListReadModel{}, err
	}
	if in.CompanyID == "" {
		return ConsoleTokenBlueprintCommentListReadModel{}, ErrConsoleCompanyIDRequired
	}
	if in.TokenBlueprintID == "" {
		return ConsoleTokenBlueprintCommentListReadModel{}, ErrConsoleTokenBlueprintIDRequired
	}

	res, err := q.uc.ListComments(ctx, usecase.ListCommentsInput{
		TokenBlueprintID: in.TokenBlueprintID,
		SearchQuery:      in.SearchQuery,
		ParentCommentID:  in.ParentCommentID,
		RootCommentID:    in.RootCommentID,
		AuthorID:         in.AuthorID,
		Deleted:          in.Deleted,
		Depth:            in.Depth,
		Sort:             normalizeCommentSort(in.Sort),
		Page:             normalizePage(in.Page),
	})
	if err != nil {
		return ConsoleTokenBlueprintCommentListReadModel{}, err
	}

	tokenBlueprintName, brandName := q.resolveTokenBlueprintNameBrandName(ctx, in.TokenBlueprintID)

	return ConsoleTokenBlueprintCommentListReadModel{
		Items:              q.toCommentReadModels(res.Items),
		TokenBlueprintName: tokenBlueprintName,
		BrandName:          brandName,
		Page:               res.Page,
		PerPage:            res.PerPage,
		TotalCount:         res.TotalCount,
	}, nil
}

// ============================================================
// Mapping
// ============================================================

func (q *TokenBlueprintReviewConsoleQuery) toCommentReadModels(views []usecase.CommentView) []ConsoleTokenBlueprintCommentReadModel {
	out := make([]ConsoleTokenBlueprintCommentReadModel, 0, len(views))
	for _, view := range views {
		out = append(out, q.ToCommentReadModel(view))
	}
	return out
}

func (q *TokenBlueprintReviewConsoleQuery) ToCommentReadModel(view usecase.CommentView) ConsoleTokenBlueprintCommentReadModel {
	comment := view.Comment

	return ConsoleTokenBlueprintCommentReadModel{
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
		CreatedAt:    formatRFC3339NanoUTC(comment.CreatedAt),
		UpdatedAt:    formatRFC3339NanoUTC(comment.UpdatedAt),
	}
}

// ============================================================
// Lightweight resolution
// ============================================================

func (q *TokenBlueprintReviewConsoleQuery) resolveTokenBlueprintNameBrandName(
	ctx context.Context,
	tokenBlueprintID string,
) (tokenBlueprintName string, brandName string) {
	if tokenBlueprintID == "" || q == nil || q.uc == nil {
		return "", ""
	}

	patch, err := q.uc.GetTokenBlueprintPatchByID(ctx, tokenBlueprintID)
	if err != nil {
		return "", ""
	}

	return patch.TokenName, patch.BrandName
}

// ============================================================
// Helpers
// ============================================================

func (q *TokenBlueprintReviewConsoleQuery) validateConfigured() error {
	if q == nil || q.uc == nil {
		return ErrConsoleTokenBlueprintReviewQueryNotConfigured
	}
	return nil
}

func normalizeCommentSort(sort common.Sort) common.Sort {
	column := sort.Column
	if column == "" {
		column = "createdAt"
	}

	order := common.SortOrder(strings.ToLower(string(sort.Order)))
	if order != common.SortAsc && order != common.SortDesc {
		order = common.SortDesc
	}

	return common.Sort{
		Column: column,
		Order:  order,
	}
}

func normalizePage(page common.Page) common.Page {
	number := page.Number
	if number <= 0 {
		number = 1
	}

	perPage := page.PerPage
	if perPage <= 0 {
		perPage = 200
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
