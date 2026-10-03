// backend/internal/application/usecase/report_decision.go
package usecase

import (
	"context"

	reportdom "narratives/internal/domain/report"
)

type ReportDecision string

const (
	ReportDecisionKeep   ReportDecision = "KEEP"
	ReportDecisionRemove ReportDecision = "REMOVE"
)

type DecideReportCaseInput struct {
	CaseID        reportdom.CaseID
	Decision      ReportDecision
	Reason        string
	DecidedBy     string
	ContinueTrade bool
}

func (u *ReportUsecase) DecideReportCase(
	ctx context.Context,
	input DecideReportCaseInput,
) (reportdom.ReportCase, error) {
	if err := u.ensureReportRepository(); err != nil {
		return reportdom.ReportCase{}, err
	}
	if input.CaseID == "" {
		return reportdom.ReportCase{}, reportdom.ErrInvalidCaseID
	}

	switch input.Decision {
	case ReportDecisionKeep:
		if err := u.ensureDecisionNotificationRepository(); err != nil {
			return reportdom.ReportCase{}, err
		}

	case ReportDecisionRemove:
		if input.ContinueTrade {
			return reportdom.ReportCase{}, ErrReportInvalidDecision
		}
		if err := u.ensureDecisionNotificationRepository(); err != nil {
			return reportdom.ReportCase{}, err
		}

	default:
		return reportdom.ReportCase{}, ErrReportInvalidDecision
	}

	var (
		decidedCase reportdom.ReportCase
		err         error
	)

	switch input.Decision {
	case ReportDecisionKeep:
		decidedCase, err = u.keepReportCase(ctx, input)

	case ReportDecisionRemove:
		decidedCase, err = u.removeReportCase(ctx, input)
	}
	if err != nil {
		return reportdom.ReportCase{}, err
	}

	if decidedCase.Status == reportdom.CaseStatusKept ||
		decidedCase.Status == reportdom.CaseStatusRemoved {
		if err := u.createDecisionNotifications(ctx, decidedCase); err != nil {
			return reportdom.ReportCase{}, err
		}
	}

	return decidedCase, nil
}

func (u *ReportUsecase) keepReportCase(
	ctx context.Context,
	input DecideReportCaseInput,
) (reportdom.ReportCase, error) {
	reportCase, err := u.reportRepo.GetCase(ctx, input.CaseID)
	if err != nil {
		return reportdom.ReportCase{}, err
	}

	// TRADE の KEEP は返品紛争に対する「取引続行」裁定として扱う。
	// ContinueTrade=true の場合のみ、ReportCase を KEPT にした後で
	// disputed の ReturnAgreement を discussing へ戻し、運営メッセージを登録する。
	//
	// TRADE 以外では ContinueTrade を受け付けない。
	if reportCase.TargetType == reportdom.TargetTypeTrade {
		if !input.ContinueTrade {
			return reportdom.ReportCase{}, ErrReportInvalidDecision
		}
	} else if input.ContinueTrade {
		return reportdom.ReportCase{}, ErrReportInvalidDecision
	}

	now := u.now().UTC()

	if err := reportCase.Keep(
		input.Reason,
		now,
		input.DecidedBy,
	); err != nil {
		return reportdom.ReportCase{}, err
	}

	updatedCase, err := u.reportRepo.UpdateCase(
		ctx,
		reportCase.ID,
		reportdom.NewCasePatchFromEntity(reportCase),
	)
	if err != nil {
		return reportdom.ReportCase{}, err
	}

	if updatedCase.TargetType == reportdom.TargetTypeTrade &&
		input.ContinueTrade {
		if err := u.resumeTradeReturnDiscussion(
			ctx,
			updatedCase.TargetID,
			input.Reason,
			now,
		); err != nil {
			// ReportCase は既に KEPT として永続化されている。
			// resumeTradeReturnDiscussion は冪等に実装し、同じ裁定を
			// 再試行した際に ReturnAgreement とシステムメッセージの
			// 未完了処理だけを再実行できるようにする。
			return reportdom.ReportCase{}, err
		}
	}

	return updatedCase, nil
}

func (u *ReportUsecase) removeReportCase(
	ctx context.Context,
	input DecideReportCaseInput,
) (reportdom.ReportCase, error) {
	reportCase, err := u.reportRepo.GetCase(ctx, input.CaseID)
	if err != nil {
		return reportdom.ReportCase{}, err
	}

	switch reportCase.TargetType {
	case reportdom.TargetTypeTrade, reportdom.TargetTypeTradeMessage:
		return reportdom.ReportCase{}, ErrReportInvalidDecision
	}

	// REMOVED 済みの LIST / TOKEN_BLUEPRINT / AVATAR / BRAND / RESALE / ANNOUNCEMENT 裁定は、
	// 対象側の措置だけを再実行できるようにする。各 moderator は冪等に実装する。
	if reportCase.IsRemoved() {
		switch reportCase.TargetType {
		case reportdom.TargetTypeList:
			if err := u.suspendListTarget(
				ctx,
				reportCase,
				input.Reason,
				input.DecidedBy,
			); err != nil {
				return reportdom.ReportCase{}, err
			}

		case reportdom.TargetTypeTokenBlueprint:
			if err := u.hideTokenBlueprintTarget(
				ctx,
				reportCase,
				input.Reason,
				input.DecidedBy,
			); err != nil {
				return reportdom.ReportCase{}, err
			}

		case reportdom.TargetTypeAvatar:
			if err := u.suspendAvatarResaleTarget(
				ctx,
				reportCase,
				input.Reason,
				input.DecidedBy,
			); err != nil {
				return reportdom.ReportCase{}, err
			}

		case reportdom.TargetTypeBrand:
			if err := u.deactivateBrandTarget(
				ctx,
				reportCase,
				input.Reason,
				input.DecidedBy,
			); err != nil {
				return reportdom.ReportCase{}, err
			}

		case reportdom.TargetTypeResale:
			if err := u.suspendResaleTarget(
				ctx,
				reportCase,
				input.Reason,
				input.DecidedBy,
			); err != nil {
				return reportdom.ReportCase{}, err
			}

		case reportdom.TargetTypeAnnouncement:
			if err := u.removeAnnouncementTarget(
				ctx,
				reportCase,
				input.Reason,
				input.DecidedBy,
			); err != nil {
				return reportdom.ReportCase{}, err
			}
		}

		return reportCase, nil
	}

	// IMPORTANT:
	// REMOVE の対象側処理を先に完了する。
	// 商品レビュー/コメント/ANNOUNCEMENT は削除、LIST / RESALE は Mall 上で出品停止、
	// TOKEN_BLUEPRINT / BRAND は AMOL UI 上で非表示、
	// AVATAR は再販サービスのみ利用停止とする。
	// TRADE / TRADE_MESSAGE は REMOVE 対象外とし、この関数の冒頭で拒否する。
	// 対象側処理に失敗した場合、ReportCase を REMOVED にしてはいけない。
	switch reportCase.TargetType {
	case reportdom.TargetTypeProductBlueprintReview:
		if err := u.removeProductBlueprintReviewTarget(
			ctx,
			reportCase,
			input.Reason,
			input.DecidedBy,
		); err != nil {
			return reportdom.ReportCase{}, err
		}

	case reportdom.TargetTypeList:
		if err := u.suspendListTarget(
			ctx,
			reportCase,
			input.Reason,
			input.DecidedBy,
		); err != nil {
			return reportdom.ReportCase{}, err
		}

	case reportdom.TargetTypeTokenBlueprint:
		if err := u.hideTokenBlueprintTarget(
			ctx,
			reportCase,
			input.Reason,
			input.DecidedBy,
		); err != nil {
			return reportdom.ReportCase{}, err
		}

	case reportdom.TargetTypeTokenBlueprintComment:
		if err := u.removeTokenBlueprintCommentTarget(
			ctx,
			reportCase,
		); err != nil {
			return reportdom.ReportCase{}, err
		}

	case reportdom.TargetTypeAvatar:
		if err := u.suspendAvatarResaleTarget(
			ctx,
			reportCase,
			input.Reason,
			input.DecidedBy,
		); err != nil {
			return reportdom.ReportCase{}, err
		}

	case reportdom.TargetTypeBrand:
		if err := u.deactivateBrandTarget(
			ctx,
			reportCase,
			input.Reason,
			input.DecidedBy,
		); err != nil {
			return reportdom.ReportCase{}, err
		}

	case reportdom.TargetTypeResale:
		if err := u.suspendResaleTarget(
			ctx,
			reportCase,
			input.Reason,
			input.DecidedBy,
		); err != nil {
			return reportdom.ReportCase{}, err
		}

	case reportdom.TargetTypeAnnouncement:
		if err := u.removeAnnouncementTarget(
			ctx,
			reportCase,
			input.Reason,
			input.DecidedBy,
		); err != nil {
			return reportdom.ReportCase{}, err
		}

	default:
		return reportdom.ReportCase{}, reportdom.ErrInvalidTargetType
	}

	if err := reportCase.Remove(
		input.Reason,
		u.now().UTC(),
		input.DecidedBy,
	); err != nil {
		return reportdom.ReportCase{}, err
	}

	updatedCase, err := u.reportRepo.UpdateCase(
		ctx,
		reportCase.ID,
		reportdom.NewCasePatchFromEntity(reportCase),
	)
	if err != nil {
		return reportdom.ReportCase{}, err
	}

	// LIST / AVATAR / BRAND / RESALE は REMOVED を永続化した後でもう一度対象側の措置を実行する。
	// 1 回目の措置と REMOVED 永続化の間に対象が再公開・再有効化される競合を閉じるための後処理。
	// ANNOUNCEMENT は物理削除するため、この後処理は不要。
	// 失敗した場合は次回の同じ REMOVE 裁定で REMOVED 済み分岐から再試行できる。
	switch updatedCase.TargetType {
	case reportdom.TargetTypeList:
		if err := u.suspendListTarget(
			ctx,
			updatedCase,
			input.Reason,
			input.DecidedBy,
		); err != nil {
			return reportdom.ReportCase{}, err
		}

	case reportdom.TargetTypeAvatar:
		if err := u.suspendAvatarResaleTarget(
			ctx,
			updatedCase,
			input.Reason,
			input.DecidedBy,
		); err != nil {
			return reportdom.ReportCase{}, err
		}

	case reportdom.TargetTypeBrand:
		if err := u.deactivateBrandTarget(
			ctx,
			updatedCase,
			input.Reason,
			input.DecidedBy,
		); err != nil {
			return reportdom.ReportCase{}, err
		}

	case reportdom.TargetTypeResale:
		if err := u.suspendResaleTarget(
			ctx,
			updatedCase,
			input.Reason,
			input.DecidedBy,
		); err != nil {
			return reportdom.ReportCase{}, err
		}
	}

	return updatedCase, nil
}
