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
	ErrConsoleCommentIDRequired                      = errors.New("console token_blueprint_review_query: commentID is required")
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

func NewTokenBlueprintReviewConsoleQuery(
	uc *usecase.TokenBlueprintReviewUsecase,
) *TokenBlueprintReviewConsoleQuery {
	return &TokenBlueprintReviewConsoleQuery{
		uc: uc,
	}
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

func (q *TokenBlueprintReviewConsoleQuery) ResolveBrandActor(
	ctx context.Context,
	tokenBlueprintID string,
) (ConsoleTokenBlueprintReviewBrandActor, error) {
	if err := q.validateConfigured(); err != nil {
		return ConsoleTokenBlueprintReviewBrandActor{}, err
	}

	tokenBlueprintID = strings.TrimSpace(tokenBlueprintID)
	if tokenBlueprintID == "" {
		return ConsoleTokenBlueprintReviewBrandActor{}, ErrConsoleTokenBlueprintIDRequired
	}

	patch, err := q.uc.GetTokenBlueprintPatchByID(ctx, tokenBlueprintID)
	if err != nil {
		return ConsoleTokenBlueprintReviewBrandActor{}, err
	}

	brandID := strings.TrimSpace(patch.BrandID)
	if brandID == "" {
		return ConsoleTokenBlueprintReviewBrandActor{}, ErrConsoleBrandIDNotFound
	}

	brandName := strings.TrimSpace(patch.BrandName)
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
	tokenBlueprintReview.TokenBlueprintReviewAggregate

	TokenBlueprintName string `json:"tokenBlueprintName"`
	BrandName          string `json:"brandName"`
}

type ConsoleTokenBlueprintReviewAggregateListReadModel struct {
	Items []ConsoleTokenBlueprintReviewAggregateItem `json:"items"`
}

type ConsoleTokenBlueprintReviewAggregateReadModel struct {
	Item ConsoleTokenBlueprintReviewAggregateItem `json:"item"`
}

type ConsoleTokenBlueprintCommentReadModel struct {
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

type ConsoleTokenBlueprintCommentListReadModel struct {
	Items []ConsoleTokenBlueprintCommentReadModel `json:"items"`

	TokenBlueprintName string `json:"tokenBlueprintName"`
	BrandName          string `json:"brandName"`

	Page       int `json:"page,omitempty"`
	PerPage    int `json:"perPage,omitempty"`
	TotalCount int `json:"totalCount,omitempty"`
}

// ============================================================
// Inputs
// ============================================================

type ListConsoleTokenBlueprintReviewAggregatesInput struct {
	CompanyID string
}

type GetConsoleTokenBlueprintReviewAggregateInput struct {
	CompanyID        string
	TokenBlueprintID string
}

type ListConsoleTokenBlueprintCommentsInput struct {
	CompanyID        string
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

type ListConsoleTokenBlueprintRepliesInput struct {
	CompanyID        string
	TokenBlueprintID string
	ParentCommentID  string

	SearchQuery string
	Sort        common.Sort
	Page        common.Page
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

	companyID := strings.TrimSpace(in.CompanyID)
	if companyID == "" {
		return ConsoleTokenBlueprintReviewAggregateListReadModel{}, ErrConsoleCompanyIDRequired
	}

	aggs, err := q.uc.ListAggregatesByCompanyTokenBlueprints(ctx, companyID)
	if err != nil {
		return ConsoleTokenBlueprintReviewAggregateListReadModel{}, err
	}

	items := make([]ConsoleTokenBlueprintReviewAggregateItem, 0, len(aggs))
	for _, agg := range aggs {
		tbName, brandName := q.resolveTokenBlueprintNameBrandName(ctx, agg.TokenBlueprintID)

		items = append(items, ConsoleTokenBlueprintReviewAggregateItem{
			TokenBlueprintReviewAggregate: agg,
			TokenBlueprintName:            tbName,
			BrandName:                     brandName,
		})
	}

	return ConsoleTokenBlueprintReviewAggregateListReadModel{
		Items: items,
	}, nil
}

func (q *TokenBlueprintReviewConsoleQuery) GetAggregateByTokenBlueprintID(
	ctx context.Context,
	in GetConsoleTokenBlueprintReviewAggregateInput,
) (ConsoleTokenBlueprintReviewAggregateReadModel, error) {
	if err := q.validateConfigured(); err != nil {
		return ConsoleTokenBlueprintReviewAggregateReadModel{}, err
	}

	companyID := strings.TrimSpace(in.CompanyID)
	if companyID == "" {
		return ConsoleTokenBlueprintReviewAggregateReadModel{}, ErrConsoleCompanyIDRequired
	}

	tokenBlueprintID := strings.TrimSpace(in.TokenBlueprintID)
	if tokenBlueprintID == "" {
		return ConsoleTokenBlueprintReviewAggregateReadModel{}, ErrConsoleTokenBlueprintIDRequired
	}

	agg, err := q.uc.GetAggregate(ctx, tokenBlueprintID)
	if err != nil {
		return ConsoleTokenBlueprintReviewAggregateReadModel{}, err
	}

	tbName, brandName := q.resolveTokenBlueprintNameBrandName(ctx, tokenBlueprintID)

	return ConsoleTokenBlueprintReviewAggregateReadModel{
		Item: ConsoleTokenBlueprintReviewAggregateItem{
			TokenBlueprintReviewAggregate: agg,
			TokenBlueprintName:            tbName,
			BrandName:                     brandName,
		},
	}, nil
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

	companyID := strings.TrimSpace(in.CompanyID)
	if companyID == "" {
		return ConsoleTokenBlueprintCommentListReadModel{}, ErrConsoleCompanyIDRequired
	}

	tokenBlueprintID := strings.TrimSpace(in.TokenBlueprintID)
	if tokenBlueprintID == "" {
		return ConsoleTokenBlueprintCommentListReadModel{}, ErrConsoleTokenBlueprintIDRequired
	}

	sort := normalizeCommentSort(in.Sort, common.SortDesc)
	page := normalizePage(in.Page, 1, 200)

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
		return ConsoleTokenBlueprintCommentListReadModel{}, err
	}

	tbName, brandName := q.resolveTokenBlueprintNameBrandName(ctx, tokenBlueprintID)

	return ConsoleTokenBlueprintCommentListReadModel{
		Items:              q.toCommentReadModels(res.Items),
		TokenBlueprintName: tbName,
		BrandName:          brandName,
		Page:               res.Page,
		PerPage:            res.PerPage,
		TotalCount:         res.TotalCount,
	}, nil
}

func (q *TokenBlueprintReviewConsoleQuery) ListRepliesByCommentID(
	ctx context.Context,
	in ListConsoleTokenBlueprintRepliesInput,
) (ConsoleTokenBlueprintCommentListReadModel, error) {
	if err := q.validateConfigured(); err != nil {
		return ConsoleTokenBlueprintCommentListReadModel{}, err
	}

	companyID := strings.TrimSpace(in.CompanyID)
	if companyID == "" {
		return ConsoleTokenBlueprintCommentListReadModel{}, ErrConsoleCompanyIDRequired
	}

	tokenBlueprintID := strings.TrimSpace(in.TokenBlueprintID)
	if tokenBlueprintID == "" {
		return ConsoleTokenBlueprintCommentListReadModel{}, ErrConsoleTokenBlueprintIDRequired
	}

	parentCommentID := strings.TrimSpace(in.ParentCommentID)
	if parentCommentID == "" {
		return ConsoleTokenBlueprintCommentListReadModel{}, ErrConsoleCommentIDRequired
	}

	sort := normalizeCommentSort(in.Sort, common.SortAsc)
	page := normalizePage(in.Page, 1, 200)

	res, err := q.uc.ListComments(ctx, usecase.ListCommentsInput{
		TokenBlueprintID: tokenBlueprintID,
		SearchQuery:      strings.TrimSpace(in.SearchQuery),
		ParentCommentID:  &parentCommentID,
		Sort:             sort,
		Page:             page,
	})
	if err != nil {
		return ConsoleTokenBlueprintCommentListReadModel{}, err
	}

	tbName, brandName := q.resolveTokenBlueprintNameBrandName(ctx, tokenBlueprintID)

	return ConsoleTokenBlueprintCommentListReadModel{
		Items:              q.toCommentReadModels(res.Items),
		TokenBlueprintName: tbName,
		BrandName:          brandName,
		Page:               res.Page,
		PerPage:            res.PerPage,
		TotalCount:         res.TotalCount,
	}, nil
}

// ============================================================
// Mapping
// ============================================================

func (q *TokenBlueprintReviewConsoleQuery) toCommentReadModels(
	views []usecase.CommentView,
) []ConsoleTokenBlueprintCommentReadModel {
	out := make([]ConsoleTokenBlueprintCommentReadModel, 0, len(views))
	for _, v := range views {
		out = append(out, q.toCommentReadModel(v))
	}
	return out
}

func (q *TokenBlueprintReviewConsoleQuery) toCommentReadModel(
	view usecase.CommentView,
) ConsoleTokenBlueprintCommentReadModel {
	c := view.Comment

	return ConsoleTokenBlueprintCommentReadModel{
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

// ============================================================
// Lightweight resolution
// ============================================================

func (q *TokenBlueprintReviewConsoleQuery) resolveTokenBlueprintNameBrandName(
	ctx context.Context,
	tokenBlueprintID string,
) (tokenBlueprintName string, brandName string) {
	tokenBlueprintID = strings.TrimSpace(tokenBlueprintID)
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
	if perPage <= 0 {
		perPage = fallbackPerPage
	}
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
