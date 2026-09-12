// backend/internal/application/usecase/report_announcement.go
package usecase

import (
	"context"

	announcementdom "narratives/internal/domain/announcement"
	reportdom "narratives/internal/domain/report"
	tokenblueprintdom "narratives/internal/domain/tokenBlueprint"
)

// ReportAnnouncementModerator owns Admin moderation of announcements.
// ANNOUNCEMENT + REMOVE in Report means deleting the reported announcement.
// This is intentionally separate from the normal Console delete flow because
// reported announcements are already published.
type ReportAnnouncementModerator interface {
	DeleteAnnouncementByAdmin(
		ctx context.Context,
		input DeleteAnnouncementByAdminInput,
	) error
}

type DeleteAnnouncementByAdminInput struct {
	AnnouncementID string
	Reason         string
	AdminID        string
}

type ReportAnnouncementByAvatarInput struct {
	AnnouncementID string
	AvatarID       string
	Reason         reportdom.ReportReason
	Detail         string
}

type reportAnnouncementTargetContext struct {
	Announcement   announcementdom.Announcement
	TokenBlueprint tokenblueprintdom.TokenBlueprint
	BrandID        string
	CompanyID      string
}

func (u *ReportUsecase) ReportAnnouncementByAvatar(
	ctx context.Context,
	input ReportAnnouncementByAvatarInput,
) (reportdom.AddReportResult, error) {
	if err := u.ensureReportRepository(); err != nil {
		return reportdom.AddReportResult{}, err
	}
	if u.announcementRepo == nil || u.tokenBlueprintRepo == nil {
		return reportdom.AddReportResult{}, ErrReportUsecaseNotConfigured
	}
	if input.AnnouncementID == "" {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidTargetID
	}
	if input.AvatarID == "" {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidReporterID
	}

	targetContext, err := u.resolveAnnouncementTargetContext(
		ctx,
		input.AnnouncementID,
	)
	if err != nil {
		return reportdom.AddReportResult{}, err
	}

	if !targetContext.Announcement.Published {
		return reportdom.AddReportResult{}, reportdom.ErrCannotReportRemovedTarget
	}

	// Mall の announcement は対象 Avatar のみ閲覧・通報できる。
	// URL を直接指定して他 Avatar 宛てのお知らせを通報することを禁止する。
	if !targetContext.Announcement.IsTargetAvatar(input.AvatarID) {
		return reportdom.AddReportResult{}, ErrReportForbidden
	}

	return u.addAnnouncementReport(
		ctx,
		targetContext,
		input.AvatarID,
		input.Reason,
		input.Detail,
	)
}

func (u *ReportUsecase) addAnnouncementReport(
	ctx context.Context,
	targetContext reportAnnouncementTargetContext,
	reporterAvatarID string,
	reason reportdom.ReportReason,
	detail string,
) (reportdom.AddReportResult, error) {
	now := u.now().UTC()
	target := targetContext.Announcement

	reportCase, err := reportdom.NewReportCase(reportdom.NewReportCaseParams{
		TargetType:       reportdom.TargetTypeAnnouncement,
		TargetID:         target.ID,
		TargetParentID:   targetContext.TokenBlueprint.ID,
		TargetAuthorID:   targetContext.BrandID,
		TargetAuthorType: reportdom.ActorTypeBrand,
		SnapshotTitle:    target.Title,
		SnapshotBody:     target.Content,
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

	return u.reportRepo.AddReport(
		ctx,
		reportCase,
		report,
	)
}

func (u *ReportUsecase) resolveAnnouncementTargetContext(
	ctx context.Context,
	announcementID string,
) (reportAnnouncementTargetContext, error) {
	if u == nil || u.announcementRepo == nil || u.tokenBlueprintRepo == nil {
		return reportAnnouncementTargetContext{}, ErrReportUsecaseNotConfigured
	}
	if announcementID == "" {
		return reportAnnouncementTargetContext{}, reportdom.ErrInvalidTargetID
	}

	target, err := u.announcementRepo.GetByID(
		ctx,
		announcementID,
	)
	if err != nil {
		return reportAnnouncementTargetContext{}, err
	}
	if target.ID != announcementID {
		return reportAnnouncementTargetContext{}, reportdom.ErrInvalidTargetID
	}

	if target.TargetToken == nil || *target.TargetToken == "" {
		return reportAnnouncementTargetContext{}, reportdom.ErrInvalidTargetParentID
	}

	tokenBlueprintID := *target.TargetToken

	tokenBlueprint, err := u.tokenBlueprintRepo.GetByID(
		ctx,
		tokenBlueprintID,
	)
	if err != nil {
		return reportAnnouncementTargetContext{}, err
	}
	if tokenBlueprint == nil || tokenBlueprint.ID != tokenBlueprintID {
		return reportAnnouncementTargetContext{}, reportdom.ErrInvalidTargetParentID
	}
	if tokenBlueprint.BrandID == "" {
		return reportAnnouncementTargetContext{}, reportdom.ErrInvalidTargetAuthorID
	}
	if tokenBlueprint.CompanyID == "" {
		return reportAnnouncementTargetContext{}, reportdom.ErrInvalidCompanyID
	}

	return reportAnnouncementTargetContext{
		Announcement:   target,
		TokenBlueprint: *tokenBlueprint,
		BrandID:        tokenBlueprint.BrandID,
		CompanyID:      tokenBlueprint.CompanyID,
	}, nil
}

func (u *ReportUsecase) removeAnnouncementTarget(
	ctx context.Context,
	reportCase reportdom.ReportCase,
	reason string,
	adminID string,
) error {
	if u.announcementModerator == nil {
		return ErrReportUsecaseNotConfigured
	}

	return u.announcementModerator.DeleteAnnouncementByAdmin(
		ctx,
		DeleteAnnouncementByAdminInput{
			AnnouncementID: reportCase.TargetID,
			Reason:         reason,
			AdminID:        adminID,
		},
	)
}
