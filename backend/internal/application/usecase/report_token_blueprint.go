// backend/internal/application/usecase/report_token_blueprint.go
package usecase

import (
	"context"

	reportdom "narratives/internal/domain/report"
	tokenblueprint "narratives/internal/domain/tokenBlueprint"
)

// ReportTokenBlueprintModerator owns Admin moderation of token blueprints.
// TOKEN_BLUEPRINT + REMOVE in Report means hiding the token blueprint from AMOL UI.
// It must not delete on-chain metadata or physically delete the token blueprint.
type ReportTokenBlueprintModerator interface {
	HideTokenBlueprintByAdmin(
		ctx context.Context,
		input HideTokenBlueprintByAdminInput,
	) error
}

type HideTokenBlueprintByAdminInput struct {
	TokenBlueprintID string
	Reason           string
	AdminID          string
}

type ReportTokenBlueprintByAvatarInput struct {
	TokenBlueprintID string
	AvatarID         string
	Reason           reportdom.ReportReason
	Detail           string
}

func (u *ReportUsecase) ReportTokenBlueprintByAvatar(
	ctx context.Context,
	input ReportTokenBlueprintByAvatarInput,
) (reportdom.AddReportResult, error) {
	if err := u.ensureReportRepository(); err != nil {
		return reportdom.AddReportResult{}, err
	}
	if u.tokenBlueprintRepo == nil || u.tokenAccessResolver == nil {
		return reportdom.AddReportResult{}, ErrReportUsecaseNotConfigured
	}
	if input.TokenBlueprintID == "" {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidTargetID
	}
	if input.AvatarID == "" {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidReporterID
	}

	tokenBlueprintEntity, err := u.tokenBlueprintRepo.GetByID(ctx, input.TokenBlueprintID)
	if err != nil {
		return reportdom.AddReportResult{}, err
	}
	if tokenBlueprintEntity == nil || tokenBlueprintEntity.ID != input.TokenBlueprintID {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidTargetID
	}
	if tokenBlueprintEntity.IsHiddenByModeration() {
		return reportdom.AddReportResult{}, reportdom.ErrCannotReportRemovedTarget
	}

	allowed, err := u.tokenAccessResolver.CanReportTokenBlueprint(
		ctx,
		input.AvatarID,
		input.TokenBlueprintID,
	)
	if err != nil {
		return reportdom.AddReportResult{}, err
	}
	if !allowed {
		return reportdom.AddReportResult{}, ErrReportForbidden
	}

	return u.addTokenBlueprintReport(
		ctx,
		*tokenBlueprintEntity,
		input.AvatarID,
		input.Reason,
		input.Detail,
	)
}

func (u *ReportUsecase) addTokenBlueprintReport(
	ctx context.Context,
	target tokenblueprint.TokenBlueprint,
	reporterAvatarID string,
	reason reportdom.ReportReason,
	detail string,
) (reportdom.AddReportResult, error) {
	now := u.now().UTC()

	reportCase, err := reportdom.NewReportCase(reportdom.NewReportCaseParams{
		TargetType:       reportdom.TargetTypeTokenBlueprint,
		TargetID:         target.ID,
		TargetParentID:   target.ID,
		TargetAuthorID:   target.BrandID,
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

func (u *ReportUsecase) hideTokenBlueprintTarget(
	ctx context.Context,
	reportCase reportdom.ReportCase,
	reason string,
	adminID string,
) error {
	if u.tokenBlueprintModerator == nil {
		return ErrReportUsecaseNotConfigured
	}

	return u.tokenBlueprintModerator.HideTokenBlueprintByAdmin(
		ctx,
		HideTokenBlueprintByAdminInput{
			TokenBlueprintID: reportCase.TargetID,
			Reason:           reason,
			AdminID:          adminID,
		},
	)
}
