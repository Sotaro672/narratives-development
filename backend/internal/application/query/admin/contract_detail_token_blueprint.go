// backend/internal/application/query/admin/contract_detail_token_blueprint.go
package query

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	common "narratives/internal/domain/common"
	reportdom "narratives/internal/domain/report"
	tokenblueprintdom "narratives/internal/domain/tokenBlueprint"
)

func (q *ContractDetailQuery) listTokenBlueprintsByCompanyID(
	ctx context.Context,
	companyID string,
) ([]tokenblueprintdom.TokenBlueprint, error) {
	items := make(
		[]tokenblueprintdom.TokenBlueprint,
		0,
	)

	for pageNumber := 1; ; pageNumber++ {
		result, err := q.tokenBlueprintRepo.ListByCompanyID(
			ctx,
			companyID,
			common.Page{
				Number:  pageNumber,
				PerPage: contractDetailPageSize,
			},
		)
		if err != nil {
			return nil, err
		}

		items = append(
			items,
			result.Items...,
		)

		if len(result.Items) == 0 ||
			result.TotalPages <= pageNumber {
			break
		}
	}

	return items, nil
}

func (q *ContractDetailQuery) buildTokenBlueprintRows(
	ctx context.Context,
	companyID string,
	tokenBlueprints []tokenblueprintdom.TokenBlueprint,
	brandNameCache map[string]string,
	memberNameCache map[string]string,
) (
	[]ContractTokenBlueprintRow,
	map[string]tokenblueprintdom.TokenBlueprint,
	error,
) {
	rows := make(
		[]ContractTokenBlueprintRow,
		0,
		len(tokenBlueprints),
	)
	tokenByID := make(
		map[string]tokenblueprintdom.TokenBlueprint,
		len(tokenBlueprints),
	)

	for _, tokenBlueprint := range tokenBlueprints {
		if tokenBlueprint.ID == "" ||
			tokenBlueprint.CompanyID != companyID {
			continue
		}

		reportCount, err := q.resolveTokenBlueprintReportCount(
			ctx,
			tokenBlueprint.ID,
		)
		if err != nil {
			return nil, nil, err
		}

		tokenByID[tokenBlueprint.ID] = tokenBlueprint

		rows = append(
			rows,
			ContractTokenBlueprintRow{
				ID:     tokenBlueprint.ID,
				Name:   tokenBlueprint.Name,
				Symbol: tokenBlueprint.Symbol,
				BrandName: q.resolveBrandName(
					ctx,
					tokenBlueprint.BrandID,
					brandNameCache,
				),
				AssigneeName: q.resolveMemberName(
					ctx,
					tokenBlueprint.AssigneeID,
					memberNameCache,
				),
				Minted:      tokenBlueprint.Minted,
				ReportCount: reportCount,
				CreatedAt: formatContractDetailTime(
					tokenBlueprint.CreatedAt,
				),
				UpdatedAt: formatContractDetailTime(
					tokenBlueprint.UpdatedAt,
				),
			},
		)
	}

	return rows, tokenByID, nil
}

func (q *ContractDetailQuery) resolveTokenBlueprintReportCount(
	ctx context.Context,
	tokenBlueprintID string,
) (int, error) {
	caseID, err := reportdom.BuildCaseID(
		reportdom.TargetTypeTokenBlueprint,
		tokenBlueprintID,
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

	if reportCase.TargetType != reportdom.TargetTypeTokenBlueprint {
		return 0, reportdom.ErrInvalidTargetType
	}
	if reportCase.TargetID != tokenBlueprintID {
		return 0, reportdom.ErrInvalidTargetID
	}

	return reportCase.ReportCount, nil
}
