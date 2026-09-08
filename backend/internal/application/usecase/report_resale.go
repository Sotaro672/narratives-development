// backend/internal/application/usecase/report_resale.go
package usecase

import (
	"context"

	reportdom "narratives/internal/domain/report"
	resaledom "narratives/internal/domain/resale"
)

// ReportResaleModerator owns Admin moderation of individual resale listings.
// RESALE + REMOVE in Report means suspending only the target resale listing.
// It must not physically delete the Resale document or its images.
type ReportResaleModerator interface {
	SuspendResaleByAdmin(
		ctx context.Context,
		input SuspendResaleByAdminInput,
	) error
}

type SuspendResaleByAdminInput struct {
	ResaleID string
	Reason   string
	AdminID  string
}

type ReportResaleByAvatarInput struct {
	ResaleID string
	AvatarID string
	Reason   reportdom.ReportReason
	Detail   string
}

func (u *ReportUsecase) ReportResaleByAvatar(
	ctx context.Context,
	input ReportResaleByAvatarInput,
) (reportdom.AddReportResult, error) {
	if err := u.ensureReportRepository(); err != nil {
		return reportdom.AddReportResult{}, err
	}

	if u.resaleRepo == nil || u.productBlueprintRepo == nil {
		return reportdom.AddReportResult{}, ErrReportUsecaseNotConfigured
	}

	if input.ResaleID == "" {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidTargetID
	}

	if input.AvatarID == "" {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidReporterID
	}

	target, err := u.resaleRepo.GetByID(ctx, input.ResaleID)
	if err != nil {
		return reportdom.AddReportResult{}, err
	}

	if target.ID != input.ResaleID {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidTargetID
	}

	if target.Status != resaledom.StatusListing {
		return reportdom.AddReportResult{}, reportdom.ErrCannotReportRemovedTarget
	}

	if target.AvatarID == "" {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidTargetAuthorID
	}

	if target.AvatarID == input.AvatarID {
		return reportdom.AddReportResult{}, ErrReportSelfReport
	}

	if target.ProductBlueprintID == "" {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidTargetParentID
	}

	productBlueprint, err := u.productBlueprintRepo.GetByID(
		ctx,
		target.ProductBlueprintID,
	)
	if err != nil {
		return reportdom.AddReportResult{}, err
	}

	if productBlueprint.ID != target.ProductBlueprintID {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidTargetParentID
	}

	return u.addResaleReport(
		ctx,
		target,
		productBlueprint.ProductName,
		input.AvatarID,
		input.Reason,
		input.Detail,
	)
}

func (u *ReportUsecase) addResaleReport(
	ctx context.Context,
	target resaledom.Resale,
	productName string,
	reporterAvatarID string,
	reason reportdom.ReportReason,
	detail string,
) (reportdom.AddReportResult, error) {
	now := u.now().UTC()

	reportCase, err := reportdom.NewReportCase(reportdom.NewReportCaseParams{
		TargetType:       reportdom.TargetTypeResale,
		TargetID:         target.ID,
		TargetParentID:   target.ProductBlueprintID,
		TargetAuthorID:   target.AvatarID,
		TargetAuthorType: reportdom.ActorTypeAvatar,
		SnapshotTitle:    productName,
		SnapshotBody:     target.Description,
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

func (u *ReportUsecase) suspendResaleTarget(
	ctx context.Context,
	reportCase reportdom.ReportCase,
	reason string,
	adminID string,
) error {
	if u.resaleModerator == nil {
		return ErrReportUsecaseNotConfigured
	}

	return u.resaleModerator.SuspendResaleByAdmin(
		ctx,
		SuspendResaleByAdminInput{
			ResaleID: reportCase.TargetID,
			Reason:   reason,
			AdminID:  adminID,
		},
	)
}
