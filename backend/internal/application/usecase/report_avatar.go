// backend/internal/application/usecase/report_avatar.go
package usecase

import (
	"context"

	avatar "narratives/internal/domain/avatar"
	reportdom "narratives/internal/domain/report"
)

// ReportAvatarResaleModerator owns Admin moderation of avatar resale access.
// AVATAR + REMOVE in Report means resale-service suspension only.
// It must not delete or disable the avatar itself.
type ReportAvatarResaleModerator interface {
	SuspendAvatarResaleByAdmin(
		ctx context.Context,
		input SuspendAvatarResaleByAdminInput,
	) error
}

type SuspendAvatarResaleByAdminInput struct {
	AvatarID string
	Reason   string
	AdminID  string
}

type ReportAvatarByAvatarInput struct {
	TargetAvatarID   string
	ReporterAvatarID string
	Reason           reportdom.ReportReason
	Detail           string
}

func (u *ReportUsecase) ReportAvatarByAvatar(
	ctx context.Context,
	input ReportAvatarByAvatarInput,
) (reportdom.AddReportResult, error) {
	if err := u.ensureReportRepository(); err != nil {
		return reportdom.AddReportResult{}, err
	}
	if u.avatarRepo == nil {
		return reportdom.AddReportResult{}, ErrReportUsecaseNotConfigured
	}
	if input.TargetAvatarID == "" {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidTargetID
	}
	if input.ReporterAvatarID == "" {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidReporterID
	}
	if input.TargetAvatarID == input.ReporterAvatarID {
		return reportdom.AddReportResult{}, ErrReportSelfReport
	}

	targetAvatar, err := u.avatarRepo.GetByID(ctx, input.TargetAvatarID)
	if err != nil {
		return reportdom.AddReportResult{}, err
	}
	if targetAvatar.ID != input.TargetAvatarID {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidTargetID
	}

	return u.addAvatarReport(
		ctx,
		targetAvatar,
		input.ReporterAvatarID,
		input.Reason,
		input.Detail,
	)
}

func (u *ReportUsecase) addAvatarReport(
	ctx context.Context,
	target avatar.Avatar,
	reporterAvatarID string,
	reason reportdom.ReportReason,
	detail string,
) (reportdom.AddReportResult, error) {
	now := u.now().UTC()
	snapshotBody := ""
	if target.Profile != nil {
		snapshotBody = *target.Profile
	}

	reportCase, err := reportdom.NewReportCase(reportdom.NewReportCaseParams{
		TargetType:       reportdom.TargetTypeAvatar,
		TargetID:         target.ID,
		TargetParentID:   target.ID,
		TargetAuthorID:   target.ID,
		TargetAuthorType: reportdom.ActorTypeAvatar,
		SnapshotTitle:    target.AvatarName,
		SnapshotBody:     snapshotBody,
		SnapshotRating:   nil,
		CreatedAt:        now,
	})
	if err != nil {
		return reportdom.AddReportResult{}, err
	}

	report, err := reportdom.NewReport(reportdom.NewReportParams{
		CaseID:       reportCase.ID,
		ReporterType: reportdom.ActorTypeAvatar,
		ReporterID:   reporterAvatarID,
		CompanyID:    "",
		Reason:       reason,
		Detail:       detail,
		CreatedAt:    now,
	})
	if err != nil {
		return reportdom.AddReportResult{}, err
	}

	return u.reportRepo.AddReport(ctx, reportCase, report)
}

func (u *ReportUsecase) suspendAvatarResaleTarget(
	ctx context.Context,
	reportCase reportdom.ReportCase,
	reason string,
	adminID string,
) error {
	if u.avatarResaleModerator == nil {
		return ErrReportUsecaseNotConfigured
	}

	return u.avatarResaleModerator.SuspendAvatarResaleByAdmin(
		ctx,
		SuspendAvatarResaleByAdminInput{
			AvatarID: reportCase.TargetID,
			Reason:   reason,
			AdminID:  adminID,
		},
	)
}
