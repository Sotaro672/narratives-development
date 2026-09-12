// backend/internal/application/usecase/report_brand.go
package usecase

import (
	"context"

	branddom "narratives/internal/domain/brand"
	reportdom "narratives/internal/domain/report"
)

// ReportBrandModerator owns Admin moderation of brands.
// BRAND + REMOVE in Report means deactivating the reported brand.
// It must not physically delete the brand.
type ReportBrandModerator interface {
	DeactivateBrandByAdmin(
		ctx context.Context,
		input DeactivateBrandByAdminInput,
	) error
}

type DeactivateBrandByAdminInput struct {
	BrandID string
	Reason  string
	AdminID string
}

type ReportBrandByAvatarInput struct {
	BrandID  string
	AvatarID string
	Reason   reportdom.ReportReason
	Detail   string
}

func (u *ReportUsecase) ReportBrandByAvatar(
	ctx context.Context,
	input ReportBrandByAvatarInput,
) (reportdom.AddReportResult, error) {
	if err := u.ensureReportRepository(); err != nil {
		return reportdom.AddReportResult{}, err
	}
	if u.brandRepo == nil {
		return reportdom.AddReportResult{}, ErrReportUsecaseNotConfigured
	}
	if input.BrandID == "" {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidTargetID
	}
	if input.AvatarID == "" {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidReporterID
	}

	target, err := u.brandRepo.GetByID(ctx, input.BrandID)
	if err != nil {
		return reportdom.AddReportResult{}, err
	}
	if target.ID != input.BrandID {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidTargetID
	}
	if target.CompanyID == "" {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidCompanyID
	}
	if !target.IsActive {
		return reportdom.AddReportResult{}, reportdom.ErrCannotReportRemovedTarget
	}

	return u.addBrandReport(
		ctx,
		target,
		input.AvatarID,
		input.Reason,
		input.Detail,
	)
}

func (u *ReportUsecase) addBrandReport(
	ctx context.Context,
	target branddom.Brand,
	reporterAvatarID string,
	reason reportdom.ReportReason,
	detail string,
) (reportdom.AddReportResult, error) {
	now := u.now().UTC()

	reportCase, err := reportdom.NewReportCase(reportdom.NewReportCaseParams{
		TargetType:       reportdom.TargetTypeBrand,
		TargetID:         target.ID,
		TargetParentID:   target.ID,
		TargetAuthorID:   target.ID,
		TargetAuthorType: reportdom.ActorTypeBrand,
		SnapshotTitle:    target.Name,
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

func (u *ReportUsecase) deactivateBrandTarget(
	ctx context.Context,
	reportCase reportdom.ReportCase,
	reason string,
	adminID string,
) error {
	if u.brandModerator == nil {
		return ErrReportUsecaseNotConfigured
	}

	return u.brandModerator.DeactivateBrandByAdmin(
		ctx,
		DeactivateBrandByAdminInput{
			BrandID: reportCase.TargetID,
			Reason:  reason,
			AdminID: adminID,
		},
	)
}
