// backend/internal/application/query/admin/contract_detail_announcement.go
package query

import (
	"context"

	announcementdom "narratives/internal/domain/announcement"
	tokenblueprintdom "narratives/internal/domain/tokenBlueprint"
)

func (q *ContractDetailQuery) buildAnnouncementRows(
	ctx context.Context,
	companyID string,
	tokenBlueprints []tokenblueprintdom.TokenBlueprint,
) ([]ContractAnnouncementRow, error) {
	rows := make(
		[]ContractAnnouncementRow,
		0,
	)
	seenAnnouncementIDs := make(
		map[string]struct{},
	)

	for _, tokenBlueprint := range tokenBlueprints {
		if tokenBlueprint.ID == "" ||
			tokenBlueprint.CompanyID != companyID {
			continue
		}

		for pageNumber := 1; ; pageNumber++ {
			result, err := q.announcementRepo.ListByTargetToken(
				ctx,
				tokenBlueprint.ID,
				announcementdom.Page{
					Number:  pageNumber,
					PerPage: contractDetailPageSize,
				},
			)
			if err != nil {
				return nil, err
			}

			for _, announcement := range result.Items {
				if announcement.ID == "" {
					continue
				}
				if announcement.TargetToken == nil ||
					*announcement.TargetToken != tokenBlueprint.ID {
					continue
				}
				if _, exists := seenAnnouncementIDs[announcement.ID]; exists {
					continue
				}
				seenAnnouncementIDs[announcement.ID] = struct{}{}

				rows = append(
					rows,
					ContractAnnouncementRow{
						ID:                announcement.ID,
						Title:             announcement.Title,
						TokenBlueprintID:  tokenBlueprint.ID,
						TokenName:         tokenBlueprint.Name,
						Published:         announcement.Published,
						TargetAvatarCount: len(announcement.TargetAvatars),
						CreatedAt: formatContractDetailTime(
							announcement.CreatedAt,
						),
						UpdatedAt: formatOptionalContractDetailTime(
							announcement.UpdatedAt,
						),
					},
				)
			}

			if len(result.Items) == 0 ||
				result.TotalPages <= pageNumber {
				break
			}
		}
	}

	return rows, nil
}
