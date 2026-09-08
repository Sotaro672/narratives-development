// backend/internal/application/usecase/report_token_blueprint_comment.go
package usecase

import (
	"context"

	reportdom "narratives/internal/domain/report"
	tokenreview "narratives/internal/domain/tokenBlueprint_review"
)

// ReportTokenCommentModerator owns Admin moderation of token comments.
type ReportTokenCommentModerator interface {
	RemoveCommentByAdmin(
		ctx context.Context,
		input RemoveCommentByAdminInput,
	) error
}

type ReportTokenBlueprintCommentByAvatarInput struct {
	TokenBlueprintID string
	CommentID        string
	AvatarID         string
	Reason           reportdom.ReportReason
	Detail           string
}

func (u *ReportUsecase) ReportTokenBlueprintCommentByAvatar(
	ctx context.Context,
	input ReportTokenBlueprintCommentByAvatarInput,
) (reportdom.AddReportResult, error) {
	if err := u.ensureReportRepository(); err != nil {
		return reportdom.AddReportResult{}, err
	}
	if u.tokenCommentRepo == nil || u.tokenAccessResolver == nil {
		return reportdom.AddReportResult{}, ErrReportUsecaseNotConfigured
	}

	comment, err := u.tokenCommentRepo.GetByParentID(
		ctx,
		input.TokenBlueprintID,
		input.CommentID,
	)
	if err != nil {
		return reportdom.AddReportResult{}, err
	}
	if comment.TokenBlueprintID != input.TokenBlueprintID {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidTargetParentID
	}
	if comment.Deleted {
		return reportdom.AddReportResult{}, reportdom.ErrCannotReportRemovedTarget
	}

	targetAuthorType, err := reportActorTypeFromCommentAuthor(comment.AuthorType)
	if err != nil {
		return reportdom.AddReportResult{}, err
	}
	if targetAuthorType == reportdom.ActorTypeAvatar && comment.AuthorID == input.AvatarID {
		return reportdom.AddReportResult{}, ErrReportSelfReport
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

	return u.addTokenBlueprintCommentReport(
		ctx,
		comment,
		targetAuthorType,
		reportdom.ActorTypeAvatar,
		input.AvatarID,
		"",
		input.Reason,
		input.Detail,
	)
}

type ReportTokenBlueprintCommentByBrandInput struct {
	TokenBlueprintID string
	CommentID        string
	BrandID          string
	CompanyID        string
	Reason           reportdom.ReportReason
	Detail           string
}

func (u *ReportUsecase) ReportTokenBlueprintCommentByBrand(
	ctx context.Context,
	input ReportTokenBlueprintCommentByBrandInput,
) (reportdom.AddReportResult, error) {
	if err := u.ensureReportRepository(); err != nil {
		return reportdom.AddReportResult{}, err
	}
	if u.tokenCommentRepo == nil || u.tokenBlueprintRepo == nil {
		return reportdom.AddReportResult{}, ErrReportUsecaseNotConfigured
	}

	tokenBlueprintEntity, err := u.tokenBlueprintRepo.GetByID(ctx, input.TokenBlueprintID)
	if err != nil {
		return reportdom.AddReportResult{}, err
	}
	if tokenBlueprintEntity == nil ||
		tokenBlueprintEntity.ID != input.TokenBlueprintID ||
		tokenBlueprintEntity.CompanyID != input.CompanyID ||
		tokenBlueprintEntity.BrandID != input.BrandID {
		return reportdom.AddReportResult{}, ErrReportForbidden
	}

	comment, err := u.tokenCommentRepo.GetByParentID(
		ctx,
		input.TokenBlueprintID,
		input.CommentID,
	)
	if err != nil {
		return reportdom.AddReportResult{}, err
	}
	if comment.TokenBlueprintID != input.TokenBlueprintID {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidTargetParentID
	}
	if comment.Deleted {
		return reportdom.AddReportResult{}, reportdom.ErrCannotReportRemovedTarget
	}

	targetAuthorType, err := reportActorTypeFromCommentAuthor(comment.AuthorType)
	if err != nil {
		return reportdom.AddReportResult{}, err
	}
	if targetAuthorType == reportdom.ActorTypeBrand && comment.AuthorID == input.BrandID {
		return reportdom.AddReportResult{}, ErrReportSelfReport
	}

	return u.addTokenBlueprintCommentReport(
		ctx,
		comment,
		targetAuthorType,
		reportdom.ActorTypeBrand,
		input.BrandID,
		input.CompanyID,
		input.Reason,
		input.Detail,
	)
}

func (u *ReportUsecase) addTokenBlueprintCommentReport(
	ctx context.Context,
	comment tokenreview.Comment,
	targetAuthorType reportdom.ActorType,
	reporterType reportdom.ActorType,
	reporterID string,
	companyID string,
	reason reportdom.ReportReason,
	detail string,
) (reportdom.AddReportResult, error) {
	now := u.now().UTC()

	reportCase, err := reportdom.NewReportCase(reportdom.NewReportCaseParams{
		TargetType:       reportdom.TargetTypeTokenBlueprintComment,
		TargetID:         comment.CommentID,
		TargetParentID:   comment.TokenBlueprintID,
		TargetAuthorID:   comment.AuthorID,
		TargetAuthorType: targetAuthorType,
		SnapshotTitle:    "",
		SnapshotBody:     comment.Body,
		SnapshotRating:   nil,
		CreatedAt:        now,
	})
	if err != nil {
		return reportdom.AddReportResult{}, err
	}

	report, err := reportdom.NewReport(reportdom.NewReportParams{
		CaseID:       reportCase.ID,
		ReporterType: reporterType,
		ReporterID:   reporterID,
		CompanyID:    companyID,
		Reason:       reason,
		Detail:       detail,
		CreatedAt:    now,
	})
	if err != nil {
		return reportdom.AddReportResult{}, err
	}

	return u.reportRepo.AddReport(ctx, reportCase, report)
}

func reportActorTypeFromCommentAuthor(
	authorType tokenreview.AuthorType,
) (reportdom.ActorType, error) {
	switch authorType {
	case tokenreview.AuthorTypeAvatar:
		return reportdom.ActorTypeAvatar, nil
	case tokenreview.AuthorTypeBrand:
		return reportdom.ActorTypeBrand, nil
	default:
		return "", reportdom.ErrInvalidActorType
	}
}

func (u *ReportUsecase) removeTokenBlueprintCommentTarget(
	ctx context.Context,
	reportCase reportdom.ReportCase,
) error {
	if u.tokenCommentModerator == nil {
		return ErrReportUsecaseNotConfigured
	}

	return u.tokenCommentModerator.RemoveCommentByAdmin(
		ctx,
		RemoveCommentByAdminInput{
			TokenBlueprintID: reportCase.TargetParentID,
			CommentID:        reportCase.TargetID,
		},
	)
}
