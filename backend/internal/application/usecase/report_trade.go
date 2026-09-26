// backend/internal/application/usecase/report_trade.go

package usecase

import (
	"context"
	"strings"

	reportdom "narratives/internal/domain/report"
	tradedom "narratives/internal/domain/trade"
)

type ReportTradeReturnDisputeByAvatarInput struct {
	TradeID  string
	AvatarID string
	Reason   reportdom.ReportReason
	Detail   string
}

func (u *ReportUsecase) ReportTradeReturnDisputeByAvatar(
	ctx context.Context,
	input ReportTradeReturnDisputeByAvatarInput,
) (reportdom.AddReportResult, error) {
	if err := u.ensureReportRepository(); err != nil {
		return reportdom.AddReportResult{}, err
	}
	if u.tradeRepo == nil || u.returnAgreementRepo == nil {
		return reportdom.AddReportResult{}, ErrReportUsecaseNotConfigured
	}

	tradeID := strings.TrimSpace(input.TradeID)
	avatarID := strings.TrimSpace(input.AvatarID)
	detail := strings.TrimSpace(input.Detail)

	if tradeID == "" {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidTargetID
	}
	if avatarID == "" {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidReporterID
	}
	if detail == "" {
		return reportdom.AddReportResult{}, reportdom.ErrReportDetailRequired
	}

	trade, err := u.tradeRepo.GetByID(ctx, tradeID)
	if err != nil {
		return reportdom.AddReportResult{}, err
	}

	if trade.ID != tradeID {
		return reportdom.AddReportResult{}, tradedom.ErrNotFound
	}
	if trade.BuyerAvatarID != avatarID {
		return reportdom.AddReportResult{}, ErrReportForbidden
	}
	if trade.SellerType != tradedom.SellerTypeAvatar ||
		trade.SellerAvatarID == "" {
		return reportdom.AddReportResult{}, ErrReportForbidden
	}

	agreement, err := u.returnAgreementRepo.GetByTradeID(
		ctx,
		tradeID,
	)
	if err != nil {
		return reportdom.AddReportResult{}, err
	}

	if agreement.Status != tradedom.ReturnStatusDiscussing ||
		agreement.Proposal == nil ||
		agreement.Proposal.Agreement !=
			tradedom.ReturnProposalAgreementDisagree {
		return reportdom.AddReportResult{},
			tradedom.ErrReturnDisputeNotAllowed
	}

	now := u.now().UTC()

	reportCase, err := reportdom.NewReportCase(
		reportdom.NewReportCaseParams{
			TargetType:       reportdom.TargetTypeTrade,
			TargetID:         trade.ID,
			TargetParentID:   trade.OrderID,
			TargetAuthorID:   trade.SellerAvatarID,
			TargetAuthorType: reportdom.ActorTypeAvatar,
			SnapshotTitle:    "返品トラブル",
			SnapshotBody:     agreement.Consultation.Detail,
			SnapshotRating:   nil,
			CreatedAt:        now,
		},
	)
	if err != nil {
		return reportdom.AddReportResult{}, err
	}

	report, err := reportdom.NewReport(
		reportdom.NewReportParams{
			CaseID:       reportCase.ID,
			ReporterType: reportdom.ActorTypeAvatar,
			ReporterID:   avatarID,
			CompanyID:    "",
			Reason:       input.Reason,
			Detail:       detail,
			CreatedAt:    now,
		},
	)
	if err != nil {
		return reportdom.AddReportResult{}, err
	}

	result, err := u.reportRepo.AddReport(
		ctx,
		reportCase,
		report,
	)
	if err != nil {
		return reportdom.AddReportResult{}, err
	}

	if err := agreement.MarkDisputed(now); err != nil {
		return reportdom.AddReportResult{}, err
	}

	if _, err := u.returnAgreementRepo.Update(
		ctx,
		tradeID,
		agreement,
	); err != nil {
		return reportdom.AddReportResult{}, err
	}

	return result, nil
}
