// backend\internal\adapters\out\firestore\report_avatar_count_repository_fs.go
package firestore

import (
	"context"

	reportdom "narratives/internal/domain/report"
)

func (r *ReportRepositoryFS) GetAvatarReportCount(
	ctx context.Context,
	avatarID string,
) (int, error) {
	if avatarID == "" {
		return 0, reportdom.ErrInvalidTargetID
	}

	caseID, err := reportdom.BuildCaseID(
		reportdom.TargetTypeAvatar,
		avatarID,
	)
	if err != nil {
		return 0, err
	}

	reportCase, err := r.GetCase(ctx, caseID)
	if err != nil {
		if reportIsNotFound(err) {
			return 0, nil
		}
		return 0, err
	}

	if reportCase.TargetType != reportdom.TargetTypeAvatar {
		return 0, reportdom.ErrInvalidTargetType
	}
	if reportCase.TargetID != avatarID {
		return 0, reportdom.ErrInvalidTargetID
	}

	return reportCase.ReportCount, nil
}
