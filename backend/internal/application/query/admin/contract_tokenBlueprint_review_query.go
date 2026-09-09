// backend/internal/application/query/admin/contract_tokenBlueprint_review_query.go
package query

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	usecase "narratives/internal/application/usecase"
	common "narratives/internal/domain/common"
	companydom "narratives/internal/domain/company"
	reportdom "narratives/internal/domain/report"
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

type contractTokenBlueprintReviewReportCaseReader interface {
	GetCase(
		ctx context.Context,
		caseID reportdom.CaseID,
	) (reportdom.ReportCase, error)
}

type ContractTokenBlueprintReviewQuery struct {
	companyRepo        contractTokenBlueprintReviewCompanyReader
	tokenBlueprintRepo contractTokenBlueprintReviewTokenBlueprintReader
	reviewLister       contractTokenBlueprintReviewLister
	reportCaseRepo     contractTokenBlueprintReviewReportCaseReader
}

func NewContractTokenBlueprintReviewQuery(
	companyRepo contractTokenBlueprintReviewCompanyReader,
	tokenBlueprintRepo contractTokenBlueprintReviewTokenBlueprintReader,
	reviewLister contractTokenBlueprintReviewLister,
	reportCaseRepo contractTokenBlueprintReviewReportCaseReader,
) *ContractTokenBlueprintReviewQuery {
	return &ContractTokenBlueprintReviewQuery{
		companyRepo:        companyRepo,
		tokenBlueprintRepo: tokenBlueprintRepo,
		reviewLister:       reviewLister,
		reportCaseRepo:     reportCaseRepo,
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

	AuthorID   string `json:"authorId"`
	AuthorType string `json:"authorType"`
	AuthorName string `json:"authorName"`
	AuthorIcon string `json:"authorIcon"`

	Body         string `json:"body"`
	LikeCount    int64  `json:"likeCount"`
	DislikeCount int64  `json:"dislikeCount"`
	ChildCount   int64  `json:"childCount"`
	ReportCount  int    `json:"reportCount"`
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
		q.reviewLister == nil ||
		q.reportCaseRepo == nil {
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

	tokenBlueprint, err := q.tokenBlueprintRepo.GetByID(ctx, tokenBlueprintID)
	if err != nil {
		return ContractTokenBlueprintReviewResult{}, err
	}
	if tokenBlueprint == nil ||
		tokenBlueprint.ID != tokenBlueprintID ||
		tokenBlueprint.CompanyID != companyID {
		return ContractTokenBlueprintReviewResult{},
			tokenblueprintdom.ErrNotFound
	}

	parentCommentID := ""

	result, err := q.reviewLister.ListComments(
		ctx,
		usecase.ListCommentsInput{
			TokenBlueprintID: tokenBlueprintID,
			ParentCommentID:  &parentCommentID,
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

		reportCount, err := q.resolveReportCount(
			ctx,
			tokenBlueprintID,
			comment.CommentID,
		)
		if err != nil {
			return ContractTokenBlueprintReviewResult{}, err
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
			Body:             comment.Body,
			LikeCount:        comment.LikeCount,
			DislikeCount:     comment.DislikeCount,
			ChildCount:       comment.ChildCount,
			ReportCount:      reportCount,
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

func (q *ContractTokenBlueprintReviewQuery) resolveReportCount(
	ctx context.Context,
	tokenBlueprintID string,
	commentID string,
) (int, error) {
	caseID, err := reportdom.BuildCaseID(
		reportdom.TargetTypeTokenBlueprintComment,
		commentID,
	)
	if err != nil {
		return 0, err
	}

	reportCase, err := q.reportCaseRepo.GetCase(ctx, caseID)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return 0, nil
		}
		return 0, err
	}

	if reportCase.TargetType != reportdom.TargetTypeTokenBlueprintComment {
		return 0, reportdom.ErrInvalidTargetType
	}
	if reportCase.TargetID != commentID {
		return 0, reportdom.ErrInvalidTargetID
	}
	if reportCase.TargetParentID != tokenBlueprintID {
		return 0, reportdom.ErrInvalidTargetParentID
	}

	return reportCase.ReportCount, nil
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
