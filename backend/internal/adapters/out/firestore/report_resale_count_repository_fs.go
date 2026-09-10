// backend\internal\adapters\out\firestore\report_resale_count_repository_fs.go
package firestore

import (
	"context"

	reportdom "narratives/internal/domain/report"
)

func (r *ReportRepositoryFS) GetResaleReportCount(
	ctx context.Context,
	resaleID string,
) (int, error) {
	if resaleID == "" {
		return 0, reportdom.ErrInvalidTargetID
	}

	caseID, err := reportdom.BuildCaseID(
		reportdom.TargetTypeResale,
		resaleID,
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

	if reportCase.TargetType != reportdom.TargetTypeResale {
		return 0, reportdom.ErrInvalidTargetType
	}
	if reportCase.TargetID != resaleID {
		return 0, reportdom.ErrInvalidTargetID
	}

	return reportCase.ReportCount, nil
}
