// backend/internal/adapters/out/firestore/report_resale_comment_count_repository_fs.go
package firestore

import (
	"context"

	reportdom "narratives/internal/domain/report"
)

func (r *ReportRepositoryFS) GetResaleCommentReportCount(
	ctx context.Context,
	resaleID string,
	commentID string,
) (int, error) {
	if resaleID == "" || commentID == "" {
		return 0, reportdom.ErrInvalidTargetID
	}

	caseID, err := reportdom.BuildCaseID(
		reportdom.TargetTypeResaleComment,
		commentID,
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

	if reportCase.TargetType != reportdom.TargetTypeResaleComment {
		return 0, reportdom.ErrInvalidTargetType
	}
	if reportCase.TargetID != commentID {
		return 0, reportdom.ErrInvalidTargetID
	}
	if reportCase.TargetParentID != resaleID {
		return 0, reportdom.ErrInvalidTargetParentID
	}

	return reportCase.ReportCount, nil
}
