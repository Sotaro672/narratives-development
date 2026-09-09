// backend/internal/application/query/admin/contract_tokenBlueprint_review_query.go
package query

import (
	"context"
	"errors"

	usecase "narratives/internal/application/usecase"
	common "narratives/internal/domain/common"
	companydom "narratives/internal/domain/company"
	tokenblueprintdom "narratives/internal/domain/tokenBlueprint"
)

var ErrContractTokenBlueprintReviewQueryNotConfigured = errors.New(
	"contract token blueprint review query is not configured",
)

const (
	defaultContractTokenBlueprintReviewPage    = 1
	defaultContractTokenBlueprintReviewPerPage = 20
	maxContractTokenBlueprintReviewPerPage     = 200
)

type contractTokenBlueprintReviewCompanyReader interface {
	GetByID(
		ctx context.Context,
		id string,
	) (companydom.Company, error)
}

type contractTokenBlueprintReviewTokenBlueprintReader interface {
	GetByID(
		ctx context.Context,
		id string,
	) (*tokenblueprintdom.TokenBlueprint, error)
}

type contractTokenBlueprintReviewLister interface {
	ListComments(
		ctx context.Context,
		input usecase.ListCommentsInput,
	) (common.PageResult[usecase.CommentView], error)
}

type ContractTokenBlueprintReviewQuery struct {
	companyRepo        contractTokenBlueprintReviewCompanyReader
	tokenBlueprintRepo contractTokenBlueprintReviewTokenBlueprintReader
	reviewLister       contractTokenBlueprintReviewLister
}

func NewContractTokenBlueprintReviewQuery(
	companyRepo contractTokenBlueprintReviewCompanyReader,
	tokenBlueprintRepo contractTokenBlueprintReviewTokenBlueprintReader,
	reviewLister contractTokenBlueprintReviewLister,
) *ContractTokenBlueprintReviewQuery {
	return &ContractTokenBlueprintReviewQuery{
		companyRepo:        companyRepo,
		tokenBlueprintRepo: tokenBlueprintRepo,
		reviewLister:       reviewLister,
	}
}

type ContractTokenBlueprintReviewResult struct {
	TokenBlueprintID string                            `json:"tokenBlueprintId"`
	Items            []ContractTokenBlueprintReviewRow `json:"items"`
	TotalCount       int                               `json:"totalCount"`
	TotalPages       int                               `json:"totalPages"`
	Page             int                               `json:"page"`
	PerPage          int                               `json:"perPage"`
}

type ContractTokenBlueprintReviewRow struct {
	CommentID        string `json:"commentId"`
	TokenBlueprintID string `json:"tokenBlueprintId"`
	ParentCommentID  string `json:"parentCommentId"`
	RootCommentID    string `json:"rootCommentId"`
	Depth            int    `json:"depth"`

	AuthorID       string `json:"authorId"`
	AuthorType     string `json:"authorType"`
	AuthorName     string `json:"authorName"`
	AuthorIcon     string `json:"authorIcon"`
	IsOwnerComment bool   `json:"isOwnerComment"`

	Body         string `json:"body"`
	LikeCount    int64  `json:"likeCount"`
	DislikeCount int64  `json:"dislikeCount"`
	ChildCount   int64  `json:"childCount"`
	Deleted      bool   `json:"deleted"`

	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

func (q *ContractTokenBlueprintReviewQuery) List(
	ctx context.Context,
	companyID string,
	tokenBlueprintID string,
	page common.Page,
) (ContractTokenBlueprintReviewResult, error) {
	if q == nil ||
		q.companyRepo == nil ||
		q.tokenBlueprintRepo == nil ||
		q.reviewLister == nil {
		return ContractTokenBlueprintReviewResult{},
			ErrContractTokenBlueprintReviewQueryNotConfigured
	}

	if companyID == "" {
		return ContractTokenBlueprintReviewResult{}, companydom.ErrInvalidID
	}
	if tokenBlueprintID == "" {
		return ContractTokenBlueprintReviewResult{},
			tokenblueprintdom.ErrInvalidID
	}

	if _, err := q.companyRepo.GetByID(ctx, companyID); err != nil {
		return ContractTokenBlueprintReviewResult{}, err
	}

	tokenBlueprint, err := q.tokenBlueprintRepo.GetByID(
		ctx,
		tokenBlueprintID,
	)
	if err != nil {
		return ContractTokenBlueprintReviewResult{}, err
	}
	if tokenBlueprint == nil ||
		tokenBlueprint.ID != tokenBlueprintID ||
		tokenBlueprint.CompanyID != companyID {
		return ContractTokenBlueprintReviewResult{},
			tokenblueprintdom.ErrNotFound
	}

	depth := 0

	result, err := q.reviewLister.ListComments(
		ctx,
		usecase.ListCommentsInput{
			TokenBlueprintID: tokenBlueprintID,
			Depth:            &depth,
			Sort: common.Sort{
				Column: "createdAt",
				Order:  common.SortDesc,
			},
			Page: normalizeContractTokenBlueprintReviewPage(page),
		},
	)
	if err != nil {
		return ContractTokenBlueprintReviewResult{}, err
	}

	items := make(
		[]ContractTokenBlueprintReviewRow,
		0,
		len(result.Items),
	)

	for _, view := range result.Items {
		comment := view.Comment

		authorName := ""
		authorIcon := ""

		if view.AuthorAvatarName != "" {
			authorName = view.AuthorAvatarName
			if view.AuthorAvatarIcon != nil {
				authorIcon = *view.AuthorAvatarIcon
			}
		} else if view.BrandName != "" {
			authorName = view.BrandName
			if view.BrandIcon != nil {
				authorIcon = *view.BrandIcon
			}
		}

		if authorName == "" {
			authorName = comment.AuthorID
		}

		items = append(items, ContractTokenBlueprintReviewRow{
			CommentID:        comment.CommentID,
			TokenBlueprintID: comment.TokenBlueprintID,
			ParentCommentID:  comment.ParentCommentID,
			RootCommentID:    comment.RootCommentID,
			Depth:            comment.Depth,
			AuthorID:         comment.AuthorID,
			AuthorType:       string(comment.AuthorType),
			AuthorName:       authorName,
			AuthorIcon:       authorIcon,
			IsOwnerComment:   comment.IsOwnerComment,
			Body:             comment.Body,
			LikeCount:        comment.LikeCount,
			DislikeCount:     comment.DislikeCount,
			ChildCount:       comment.ChildCount,
			Deleted:          comment.Deleted,
			CreatedAt:        formatContractDetailTime(comment.CreatedAt),
			UpdatedAt:        formatContractDetailTime(comment.UpdatedAt),
		})
	}

	return ContractTokenBlueprintReviewResult{
		TokenBlueprintID: tokenBlueprintID,
		Items:            items,
		TotalCount:       result.TotalCount,
		TotalPages:       result.TotalPages,
		Page:             result.Page,
		PerPage:          result.PerPage,
	}, nil
}

func normalizeContractTokenBlueprintReviewPage(
	page common.Page,
) common.Page {
	if page.Number <= 0 {
		page.Number = defaultContractTokenBlueprintReviewPage
	}

	if page.PerPage <= 0 {
		page.PerPage = defaultContractTokenBlueprintReviewPerPage
	}
	if page.PerPage > maxContractTokenBlueprintReviewPerPage {
		page.PerPage = maxContractTokenBlueprintReviewPerPage
	}

	return page
}
