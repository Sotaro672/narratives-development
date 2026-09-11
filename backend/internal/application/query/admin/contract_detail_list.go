// backend/internal/application/query/admin/contract_detail_list.go
package query

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	productblueprintdom "narratives/internal/domain/productBlueprint"
	reportdom "narratives/internal/domain/report"
	tokenblueprintdom "narratives/internal/domain/tokenBlueprint"
)

func (q *ContractDetailQuery) buildListRows(
	ctx context.Context,
	companyID string,
	productBlueprints []productblueprintdom.ProductBlueprint,
	tokenByID map[string]tokenblueprintdom.TokenBlueprint,
	brandNameCache map[string]string,
	memberNameCache map[string]string,
) ([]ContractListRow, error) {
	rows := make(
		[]ContractListRow,
		0,
	)

	seenInventoryID := make(
		map[string]struct{},
	)
	seenListID := make(
		map[string]struct{},
	)

	for _, productBlueprint := range productBlueprints {
		if productBlueprint.ID == "" ||
			productBlueprint.CompanyID != companyID {
			continue
		}

		inventories, err := q.inventoryRepo.ListByProductBlueprintID(
			ctx,
			productBlueprint.ID,
		)
		if err != nil {
			return nil, err
		}

		for _, inventory := range inventories {
			if inventory.ID == "" ||
				inventory.ProductBlueprintID != productBlueprint.ID {
				continue
			}

			tokenBlueprint, ok := tokenByID[inventory.TokenBlueprintID]
			if !ok {
				continue
			}

			if _, exists := seenInventoryID[inventory.ID]; exists {
				continue
			}
			seenInventoryID[inventory.ID] = struct{}{}

			lists, err := q.listRepo.ListByInventoryID(
				ctx,
				inventory.ID,
			)
			if err != nil {
				return nil, err
			}

			for _, item := range lists {
				if item.ID == "" ||
					item.InventoryID != inventory.ID {
					continue
				}

				if _, exists := seenListID[item.ID]; exists {
					continue
				}
				seenListID[item.ID] = struct{}{}

				updatedAt := ""
				if item.UpdatedAt != nil {
					updatedAt = formatContractDetailTime(
						*item.UpdatedAt,
					)
				}

				reportCount, err := q.resolveListReportCount(
					ctx,
					item.ID,
				)
				if err != nil {
					return nil, err
				}

				rows = append(
					rows,
					ContractListRow{
						ID:          item.ID,
						ReadableID:  item.ReadableID,
						InventoryID: item.InventoryID,
						Title:       item.Title,
						ProductName: productBlueprint.ProductName,
						TokenName:   tokenBlueprint.Name,
						BrandName: q.resolveBrandName(
							ctx,
							productBlueprint.BrandID,
							brandNameCache,
						),
						AssigneeName: q.resolveMemberName(
							ctx,
							item.AssigneeID,
							memberNameCache,
						),
						Status:      string(item.Status),
						ReportCount: reportCount,
						CreatedAt: formatContractDetailTime(
							item.CreatedAt,
						),
						UpdatedAt: updatedAt,
					},
				)
			}
		}
	}

	return rows, nil
}

func (q *ContractDetailQuery) resolveListReportCount(
	ctx context.Context,
	listID string,
) (int, error) {
	caseID, err := reportdom.BuildCaseID(
		reportdom.TargetTypeList,
		listID,
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

	if reportCase.TargetType != reportdom.TargetTypeList {
		return 0, reportdom.ErrInvalidTargetType
	}
	if reportCase.TargetID != listID {
		return 0, reportdom.ErrInvalidTargetID
	}

	return reportCase.ReportCount, nil
}
