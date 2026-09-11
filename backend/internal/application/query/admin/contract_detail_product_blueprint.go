// backend/internal/application/query/admin/contract_detail_product_blueprint.go
package query

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	common "narratives/internal/domain/common"
	productblueprintdom "narratives/internal/domain/productBlueprint"
	productblueprintreviewdom "narratives/internal/domain/productBlueprintReview"
	reportdom "narratives/internal/domain/report"
)

func (q *ContractDetailQuery) buildProductBlueprintRows(
	ctx context.Context,
	companyID string,
	productBlueprints []productblueprintdom.ProductBlueprint,
	brandNameCache map[string]string,
	memberNameCache map[string]string,
) ([]ContractProductBlueprintRow, error) {
	rows := make(
		[]ContractProductBlueprintRow,
		0,
		len(productBlueprints),
	)

	for _, productBlueprint := range productBlueprints {
		if productBlueprint.ID == "" ||
			productBlueprint.CompanyID != companyID {
			continue
		}

		reportCount, err := q.resolveProductBlueprintReviewReportCount(
			ctx,
			productBlueprint.ID,
		)
		if err != nil {
			return nil, err
		}

		rows = append(
			rows,
			ContractProductBlueprintRow{
				ID:          productBlueprint.ID,
				ProductName: productBlueprint.ProductName,
				BrandName: q.resolveBrandName(
					ctx,
					productBlueprint.BrandID,
					brandNameCache,
				),
				AssigneeName: q.resolveMemberName(
					ctx,
					productBlueprint.AssigneeID,
					memberNameCache,
				),
				Printed:     productBlueprint.Printed,
				ReportCount: reportCount,
				CreatedAt: formatContractDetailTime(
					productBlueprint.CreatedAt,
				),
				UpdatedAt: formatContractDetailTime(
					productBlueprint.UpdatedAt,
				),
			},
		)
	}

	return rows, nil
}

func (q *ContractDetailQuery) resolveProductBlueprintReviewReportCount(
	ctx context.Context,
	productBlueprintID string,
) (int, error) {
	statuses := []productblueprintreviewdom.ReviewStatus{
		productblueprintreviewdom.ReviewStatusPublished,
		productblueprintreviewdom.ReviewStatusHidden,
		productblueprintreviewdom.ReviewStatusRemoved,
	}

	const perPage = 100

	reportCount := 0
	seenReviewIDs := make(map[productblueprintreviewdom.ReviewID]struct{})

	for _, reviewStatus := range statuses {
		pageNumber := 1

		for {
			result, err := q.productBlueprintReviewRepo.ListByProductBlueprintID(
				ctx,
				productBlueprintID,
				reviewStatus,
				common.Page{
					Number:  pageNumber,
					PerPage: perPage,
				},
			)
			if err != nil {
				return 0, err
			}

			for _, review := range result.Items {
				if review.ID == "" {
					continue
				}
				if _, exists := seenReviewIDs[review.ID]; exists {
					continue
				}
				seenReviewIDs[review.ID] = struct{}{}

				caseID, err := reportdom.BuildCaseID(
					reportdom.TargetTypeProductBlueprintReview,
					string(review.ID),
				)
				if err != nil {
					return 0, err
				}

				reportCase, err := q.reportCaseRepo.GetCase(ctx, caseID)
				if err != nil {
					if status.Code(err) == codes.NotFound {
						continue
					}
					return 0, err
				}

				if reportCase.TargetType != reportdom.TargetTypeProductBlueprintReview {
					return 0, reportdom.ErrInvalidTargetType
				}
				if reportCase.TargetID != string(review.ID) {
					return 0, reportdom.ErrInvalidTargetID
				}
				if reportCase.TargetParentID != productBlueprintID {
					return 0, reportdom.ErrInvalidTargetParentID
				}
				if reportCase.ReportCount <= 0 {
					continue
				}

				reportCount += reportCase.ReportCount
			}

			if pageNumber >= result.TotalPages {
				break
			}
			pageNumber++
		}
	}

	return reportCount, nil
}
