// backend/internal/application/query/admin/contract_productBlueprint_review_query.go
package query

import (
	"context"
	"errors"

	usecase "narratives/internal/application/usecase"
	common "narratives/internal/domain/common"
	companydom "narratives/internal/domain/company"
	productblueprintdom "narratives/internal/domain/productBlueprint"
	productblueprintreviewdom "narratives/internal/domain/productBlueprintReview"
)

var ErrContractProductBlueprintReviewQueryNotConfigured = errors.New(
	"contract product blueprint review query is not configured",
)

const (
	defaultContractProductBlueprintReviewPage    = 1
	defaultContractProductBlueprintReviewPerPage = 20
	maxContractProductBlueprintReviewPerPage     = 200
)

type contractProductBlueprintReviewCompanyReader interface {
	GetByID(
		ctx context.Context,
		id string,
	) (companydom.Company, error)
}

type contractProductBlueprintReviewProductBlueprintReader interface {
	GetByID(
		ctx context.Context,
		id string,
	) (productblueprintdom.ProductBlueprint, error)
}

type contractProductBlueprintReviewLister interface {
	ListByProductBlueprintID(
		ctx context.Context,
		productBlueprintID string,
		status productblueprintreviewdom.ReviewStatus,
		page common.Page,
	) (common.PageResult[usecase.ProductBlueprintReviewListItem], error)
}

type ContractProductBlueprintReviewQuery struct {
	companyRepo          contractProductBlueprintReviewCompanyReader
	productBlueprintRepo contractProductBlueprintReviewProductBlueprintReader
	reviewLister         contractProductBlueprintReviewLister
}

func NewContractProductBlueprintReviewQuery(
	companyRepo contractProductBlueprintReviewCompanyReader,
	productBlueprintRepo contractProductBlueprintReviewProductBlueprintReader,
	reviewLister contractProductBlueprintReviewLister,
) *ContractProductBlueprintReviewQuery {
	return &ContractProductBlueprintReviewQuery{
		companyRepo:          companyRepo,
		productBlueprintRepo: productBlueprintRepo,
		reviewLister:         reviewLister,
	}
}

type ContractProductBlueprintReviewResult struct {
	ProductBlueprintID string                                 `json:"productBlueprintId"`
	Status             productblueprintreviewdom.ReviewStatus `json:"status"`
	Items              []ContractProductBlueprintReviewRow    `json:"items"`
	TotalCount         int                                    `json:"totalCount"`
	TotalPages         int                                    `json:"totalPages"`
	Page               int                                    `json:"page"`
	PerPage            int                                    `json:"perPage"`
}

type ContractProductBlueprintReviewRow struct {
	ID                 string  `json:"id"`
	ProductBlueprintID string  `json:"productBlueprintId"`
	AvatarID           string  `json:"avatarId"`
	AvatarName         string  `json:"avatarName"`
	AvatarIcon         string  `json:"avatarIcon"`
	Rating             int     `json:"rating"`
	Title              string  `json:"title"`
	Body               string  `json:"body"`
	HelpfulVotes       int     `json:"helpfulVotes"`
	TotalVotes         int     `json:"totalVotes"`
	Status             string  `json:"status"`
	ReviewedAt         string  `json:"reviewedAt"`
	CreatedAt          string  `json:"createdAt"`
	UpdatedAt          string  `json:"updatedAt"`
	ModerationReason   *string `json:"moderationReason,omitempty"`
}

func (q *ContractProductBlueprintReviewQuery) List(
	ctx context.Context,
	companyID string,
	productBlueprintID string,
	status productblueprintreviewdom.ReviewStatus,
	page common.Page,
) (ContractProductBlueprintReviewResult, error) {
	if q == nil ||
		q.companyRepo == nil ||
		q.productBlueprintRepo == nil ||
		q.reviewLister == nil {
		return ContractProductBlueprintReviewResult{},
			ErrContractProductBlueprintReviewQueryNotConfigured
	}

	if companyID == "" {
		return ContractProductBlueprintReviewResult{}, companydom.ErrInvalidID
	}
	if productBlueprintID == "" {
		return ContractProductBlueprintReviewResult{},
			productblueprintdom.ErrInvalidID
	}

	if _, err := q.companyRepo.GetByID(ctx, companyID); err != nil {
		return ContractProductBlueprintReviewResult{}, err
	}

	productBlueprint, err := q.productBlueprintRepo.GetByID(
		ctx,
		productBlueprintID,
	)
	if err != nil {
		return ContractProductBlueprintReviewResult{}, err
	}
	if productBlueprint.ID != productBlueprintID ||
		productBlueprint.CompanyID != companyID {
		return ContractProductBlueprintReviewResult{},
			productblueprintdom.ErrNotFound
	}

	normalizedStatus, err := normalizeContractProductBlueprintReviewStatus(status)
	if err != nil {
		return ContractProductBlueprintReviewResult{}, err
	}

	normalizedPage := normalizeContractProductBlueprintReviewPage(page)

	result, err := q.reviewLister.ListByProductBlueprintID(
		ctx,
		productBlueprintID,
		normalizedStatus,
		normalizedPage,
	)
	if err != nil {
		return ContractProductBlueprintReviewResult{}, err
	}

	items := make(
		[]ContractProductBlueprintReviewRow,
		0,
		len(result.Items),
	)

	for _, item := range result.Items {
		var moderationReason *string
		if item.ModerationReason != nil {
			value := *item.ModerationReason
			moderationReason = &value
		}

		items = append(items, ContractProductBlueprintReviewRow{
			ID:                 string(item.ID),
			ProductBlueprintID: item.ProductBlueprintID,
			AvatarID:           item.AvatarID,
			AvatarName:         item.AvatarName,
			AvatarIcon:         item.AvatarIcon,
			Rating:             int(item.Rating),
			Title:              item.Title,
			Body:               item.Body,
			HelpfulVotes:       item.HelpfulVotes,
			TotalVotes:         item.TotalVotes,
			Status:             string(item.Status),
			ReviewedAt:         formatContractDetailTime(item.ReviewedAt),
			CreatedAt:          formatContractDetailTime(item.CreatedAt),
			UpdatedAt:          formatContractDetailTime(item.UpdatedAt),
			ModerationReason:   moderationReason,
		})
	}

	return ContractProductBlueprintReviewResult{
		ProductBlueprintID: productBlueprintID,
		Status:             normalizedStatus,
		Items:              items,
		TotalCount:         result.TotalCount,
		TotalPages:         result.TotalPages,
		Page:               result.Page,
		PerPage:            result.PerPage,
	}, nil
}

func normalizeContractProductBlueprintReviewStatus(
	status productblueprintreviewdom.ReviewStatus,
) (productblueprintreviewdom.ReviewStatus, error) {
	if status == "" {
		return productblueprintreviewdom.ReviewStatusPublished, nil
	}

	switch status {
	case productblueprintreviewdom.ReviewStatusPublished,
		productblueprintreviewdom.ReviewStatusHidden,
		productblueprintreviewdom.ReviewStatusRemoved:
		return status, nil

	default:
		return "", productblueprintreviewdom.ErrInvalidStatus
	}
}

func normalizeContractProductBlueprintReviewPage(
	page common.Page,
) common.Page {
	if page.Number <= 0 {
		page.Number = defaultContractProductBlueprintReviewPage
	}

	if page.PerPage <= 0 {
		page.PerPage = defaultContractProductBlueprintReviewPerPage
	}
	if page.PerPage > maxContractProductBlueprintReviewPerPage {
		page.PerPage = maxContractProductBlueprintReviewPerPage
	}

	return page
}
