// backend/internal/application/usecase/report_trade_message.go
package usecase

import (
	"context"

	reportdom "narratives/internal/domain/report"
	tradedom "narratives/internal/domain/trade"
)

type ReportTradeMessageByAvatarInput struct {
	TradeID   string
	MessageID string
	AvatarID  string
	Reason    reportdom.ReportReason
	Detail    string
}

func (u *ReportUsecase) ReportTradeMessageByAvatar(
	ctx context.Context,
	input ReportTradeMessageByAvatarInput,
) (reportdom.AddReportResult, error) {
	if err := u.ensureReportRepository(); err != nil {
		return reportdom.AddReportResult{}, err
	}
	if u.tradeRepo == nil || u.tradeMessageRepo == nil {
		return reportdom.AddReportResult{}, ErrReportUsecaseNotConfigured
	}
	if input.TradeID == "" {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidTargetParentID
	}
	if input.MessageID == "" {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidTargetID
	}
	if input.AvatarID == "" {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidReporterID
	}

	trade, err := u.tradeRepo.GetByID(ctx, input.TradeID)
	if err != nil {
		return reportdom.AddReportResult{}, err
	}
	if trade.ID != input.TradeID {
		return reportdom.AddReportResult{}, tradedom.ErrNotFound
	}
	if trade.SellerType != tradedom.SellerTypeAvatar || trade.SellerAvatarID == "" {
		return reportdom.AddReportResult{}, ErrReportForbidden
	}

	if _, err := resolveTradeMessageParticipantSide(trade, input.AvatarID); err != nil {
		return reportdom.AddReportResult{}, err
	}

	message, err := u.tradeMessageRepo.GetByID(
		ctx,
		input.TradeID,
		input.MessageID,
	)
	if err != nil {
		return reportdom.AddReportResult{}, err
	}
	if message.ID != input.MessageID {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidTargetID
	}
	if message.TradeID != input.TradeID {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidTargetParentID
	}
	if message.SenderSide == tradedom.MessageSenderSideSystem ||
		message.SenderType == tradedom.MessageSenderTypeSystem {
		return reportdom.AddReportResult{}, ErrReportForbidden
	}
	if message.SenderType != tradedom.MessageSenderTypeAvatar {
		return reportdom.AddReportResult{}, reportdom.ErrInvalidActorType
	}

	targetAuthorID, err := resolveTradeMessageReportAuthor(trade, message)
	if err != nil {
		return reportdom.AddReportResult{}, err
	}
	if targetAuthorID == input.AvatarID {
		return reportdom.AddReportResult{}, ErrReportSelfReport
	}

	return u.addTradeMessageReport(
		ctx,
		message,
		targetAuthorID,
		input.AvatarID,
		input.Reason,
		input.Detail,
	)
}

func (u *ReportUsecase) addTradeMessageReport(
	ctx context.Context,
	message tradedom.Message,
	targetAuthorID string,
	reporterAvatarID string,
	reason reportdom.ReportReason,
	detail string,
) (reportdom.AddReportResult, error) {
	now := u.now().UTC()

	reportCase, err := reportdom.NewReportCase(reportdom.NewReportCaseParams{
		TargetType:       reportdom.TargetTypeTradeMessage,
		TargetID:         message.ID,
		TargetParentID:   message.TradeID,
		TargetAuthorID:   targetAuthorID,
		TargetAuthorType: reportdom.ActorTypeAvatar,
		SnapshotTitle:    "",
		SnapshotBody:     message.Content,
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

func resolveTradeMessageReportAuthor(
	trade tradedom.Trade,
	message tradedom.Message,
) (string, error) {
	switch message.SenderSide {
	case tradedom.MessageSenderSideBuyer:
		if message.SenderID == "" || message.SenderID != trade.BuyerAvatarID {
			return "", reportdom.ErrInvalidTargetAuthorID
		}
		return message.SenderID, nil

	case tradedom.MessageSenderSideSeller:
		if trade.SellerType != tradedom.SellerTypeAvatar ||
			trade.SellerAvatarID == "" ||
			message.SenderID == "" ||
			message.SenderID != trade.SellerAvatarID {
			return "", reportdom.ErrInvalidTargetAuthorID
		}
		return message.SenderID, nil

	case tradedom.MessageSenderSideSystem:
		return "", ErrReportForbidden

	default:
		return "", reportdom.ErrInvalidTargetAuthorID
	}
}
