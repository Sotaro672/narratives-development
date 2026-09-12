// backend/internal/application/usecase/report_decision_notification.go
package usecase

import (
	"context"
	"time"

	common "narratives/internal/domain/common"
	reportdom "narratives/internal/domain/report"
)

func (u *ReportUsecase) ListDecisionNotificationsForAvatar(
	ctx context.Context,
	avatarID string,
	isRead *bool,
	page common.Page,
) (common.PageResult[reportdom.DecisionNotification], error) {
	if err := u.ensureDecisionNotificationRepository(); err != nil {
		return common.PageResult[reportdom.DecisionNotification]{}, err
	}
	if avatarID == "" {
		return common.PageResult[reportdom.DecisionNotification]{}, reportdom.ErrInvalidReporterID
	}

	recipientType := reportdom.ActorTypeAvatar
	return u.decisionNotificationRepo.List(ctx, reportdom.DecisionNotificationFilter{
		RecipientType: &recipientType,
		RecipientID:   avatarID,
		IsRead:        isRead,
	}, common.Sort{
		Column: "createdAt",
		Order:  common.SortDesc,
	}, page)
}

func (u *ReportUsecase) GetDecisionNotificationForAvatar(
	ctx context.Context,
	notificationID reportdom.DecisionNotificationID,
	avatarID string,
) (reportdom.DecisionNotification, error) {
	if err := u.ensureDecisionNotificationRepository(); err != nil {
		return reportdom.DecisionNotification{}, err
	}
	if notificationID == "" {
		return reportdom.DecisionNotification{}, reportdom.ErrInvalidDecisionNotificationID
	}
	if avatarID == "" {
		return reportdom.DecisionNotification{}, reportdom.ErrInvalidReporterID
	}

	notification, err := u.decisionNotificationRepo.GetByID(ctx, notificationID)
	if err != nil {
		return reportdom.DecisionNotification{}, err
	}
	if notification.RecipientType != reportdom.ActorTypeAvatar || notification.RecipientID != avatarID {
		return reportdom.DecisionNotification{}, ErrReportForbidden
	}

	return notification, nil
}

func (u *ReportUsecase) MarkDecisionNotificationReadForAvatar(
	ctx context.Context,
	notificationID reportdom.DecisionNotificationID,
	avatarID string,
) (reportdom.DecisionNotification, error) {
	if err := u.ensureDecisionNotificationRepository(); err != nil {
		return reportdom.DecisionNotification{}, err
	}
	if notificationID == "" {
		return reportdom.DecisionNotification{}, reportdom.ErrInvalidDecisionNotificationID
	}
	if avatarID == "" {
		return reportdom.DecisionNotification{}, reportdom.ErrInvalidReporterID
	}

	notification, err := u.GetDecisionNotificationForAvatar(ctx, notificationID, avatarID)
	if err != nil {
		return reportdom.DecisionNotification{}, err
	}

	return u.decisionNotificationRepo.MarkRead(
		ctx,
		notificationID,
		reportdom.ActorTypeAvatar,
		notification.RecipientID,
		u.now().UTC(),
	)
}

func (u *ReportUsecase) ListDecisionNotificationsForCompany(
	ctx context.Context,
	companyID string,
	isRead *bool,
	page common.Page,
) (common.PageResult[reportdom.DecisionNotification], error) {
	if err := u.ensureDecisionNotificationRepository(); err != nil {
		return common.PageResult[reportdom.DecisionNotification]{}, err
	}
	if companyID == "" {
		return common.PageResult[reportdom.DecisionNotification]{}, reportdom.ErrInvalidCompanyID
	}

	recipientType := reportdom.ActorTypeBrand
	return u.decisionNotificationRepo.List(ctx, reportdom.DecisionNotificationFilter{
		RecipientType: &recipientType,
		CompanyID:     companyID,
		IsRead:        isRead,
	}, common.Sort{
		Column: "createdAt",
		Order:  common.SortDesc,
	}, page)
}

func (u *ReportUsecase) GetDecisionNotificationForCompany(
	ctx context.Context,
	notificationID reportdom.DecisionNotificationID,
	companyID string,
) (reportdom.DecisionNotification, error) {
	if err := u.ensureDecisionNotificationRepository(); err != nil {
		return reportdom.DecisionNotification{}, err
	}
	if notificationID == "" {
		return reportdom.DecisionNotification{}, reportdom.ErrInvalidDecisionNotificationID
	}
	if companyID == "" {
		return reportdom.DecisionNotification{}, reportdom.ErrInvalidCompanyID
	}

	notification, err := u.decisionNotificationRepo.GetByID(ctx, notificationID)
	if err != nil {
		return reportdom.DecisionNotification{}, err
	}
	if notification.RecipientType != reportdom.ActorTypeBrand || notification.CompanyID != companyID {
		return reportdom.DecisionNotification{}, ErrReportForbidden
	}

	return notification, nil
}

func (u *ReportUsecase) MarkDecisionNotificationReadForCompany(
	ctx context.Context,
	notificationID reportdom.DecisionNotificationID,
	companyID string,
) (reportdom.DecisionNotification, error) {
	if err := u.ensureDecisionNotificationRepository(); err != nil {
		return reportdom.DecisionNotification{}, err
	}
	if notificationID == "" {
		return reportdom.DecisionNotification{}, reportdom.ErrInvalidDecisionNotificationID
	}
	if companyID == "" {
		return reportdom.DecisionNotification{}, reportdom.ErrInvalidCompanyID
	}

	notification, err := u.GetDecisionNotificationForCompany(ctx, notificationID, companyID)
	if err != nil {
		return reportdom.DecisionNotification{}, err
	}

	return u.decisionNotificationRepo.MarkRead(
		ctx,
		notificationID,
		reportdom.ActorTypeBrand,
		notification.RecipientID,
		u.now().UTC(),
	)
}

func (u *ReportUsecase) createDecisionNotifications(
	ctx context.Context,
	reportCase reportdom.ReportCase,
) error {
	if err := u.ensureReportRepository(); err != nil {
		return err
	}
	if err := u.ensureDecisionNotificationRepository(); err != nil {
		return err
	}
	if reportCase.DecidedAt == nil || reportCase.DecidedAt.IsZero() {
		return reportdom.ErrDecisionNotificationCaseNotDecided
	}

	decidedAt := reportCase.DecidedAt.UTC()
	filter := reportdom.ReportFilter{
		CaseID: reportCase.ID,
		CreatedAt: common.TimeRange{
			To: &decidedAt,
		},
	}
	sort := common.Sort{
		Column: "createdAt",
		Order:  common.SortAsc,
	}

	const perPage = 100
	for pageNumber := 1; ; pageNumber++ {
		result, err := u.reportRepo.ListReports(ctx, reportCase.ID, filter, sort, common.Page{
			Number:  pageNumber,
			PerPage: perPage,
		})
		if err != nil {
			return err
		}

		for _, report := range result.Items {
			notification, err := reportdom.NewDecisionNotification(reportCase, report, decidedAt)
			if err != nil {
				return err
			}
			if _, err := u.decisionNotificationRepo.CreateIfAbsent(ctx, notification); err != nil {
				return err
			}
		}

		if pageNumber >= result.TotalPages {
			break
		}
	}

	return u.createTargetEnforcementDecisionNotification(ctx, reportCase, decidedAt)
}

func (u *ReportUsecase) createTargetEnforcementDecisionNotification(
	ctx context.Context,
	reportCase reportdom.ReportCase,
	createdAt time.Time,
) error {
	if reportCase.Status != reportdom.CaseStatusRemoved {
		return nil
	}

	companyID := ""
	notificationCase := reportCase

	// PRODUCT_BLUEPRINT_REVIEW はレビュー投稿Avatar、AVATAR は対象Avatar自身、
	// RESALE は出品Avatar、LIST は出品元Brand、TOKEN_BLUEPRINT はそのTokenBlueprintを
	// 所有するBrand、ANNOUNCEMENT は配信元Brandへ措置通知を送る。
	// TOKEN_BLUEPRINT_COMMENT は現時点では対象者通知の対象外とする。
	switch reportCase.TargetType {
	case reportdom.TargetTypeProductBlueprintReview,
		reportdom.TargetTypeAvatar,
		reportdom.TargetTypeResale:

	case reportdom.TargetTypeList:
		targetContext, err := u.resolveListTargetContext(ctx, reportCase.TargetID)
		if err != nil {
			return err
		}

		notificationCase.TargetAuthorID = targetContext.BrandID
		notificationCase.TargetAuthorType = reportdom.ActorTypeBrand
		companyID = targetContext.CompanyID

	case reportdom.TargetTypeTokenBlueprint,
		reportdom.TargetTypeAnnouncement:
		if u.tokenBlueprintRepo == nil {
			return ErrReportUsecaseNotConfigured
		}

		tokenBlueprintID := reportCase.TargetID
		if reportCase.TargetType == reportdom.TargetTypeAnnouncement {
			tokenBlueprintID = reportCase.TargetParentID
		}
		if tokenBlueprintID == "" {
			return reportdom.ErrInvalidTargetParentID
		}

		target, err := u.tokenBlueprintRepo.GetByID(ctx, tokenBlueprintID)
		if err != nil {
			return err
		}
		if target == nil || target.ID != tokenBlueprintID {
			return reportdom.ErrInvalidTargetParentID
		}
		if target.BrandID == "" {
			return reportdom.ErrInvalidTargetAuthorID
		}
		if target.CompanyID == "" {
			return reportdom.ErrInvalidCompanyID
		}

		// Announcement本体はREMOVE時点で物理削除済みのため、
		// TargetParentIDに保存したTokenBlueprintから現在のBrand/Companyを解決する。
		// TokenBlueprint通報についても、旧ReportCaseのTargetAuthorIDではなく
		// 現在のTokenBlueprintのBrandIDを正しい通知先として利用する。
		notificationCase.TargetAuthorID = target.BrandID
		notificationCase.TargetAuthorType = reportdom.ActorTypeBrand
		companyID = target.CompanyID

	default:
		return nil
	}

	notification, err := reportdom.NewTargetEnforcementNotification(
		notificationCase,
		companyID,
		createdAt,
	)
	if err != nil {
		return err
	}

	_, err = u.decisionNotificationRepo.CreateIfAbsent(ctx, notification)
	return err
}
