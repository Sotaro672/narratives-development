// backend/internal/application/usecase/report_trade.go

package usecase

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	reportdom "narratives/internal/domain/report"
	tradedom "narratives/internal/domain/trade"
)

const (
	reportTradeReturnDisputeSystemMessageID                 = "return-dispute-reported"
	reportTradeReturnDiscussionResumedSystemMessageIDPrefix = "return-dispute-resumed-"
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
	if u.tradeRepo == nil ||
		u.tradeMessageRepo == nil ||
		u.returnAgreementRepo == nil {
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

	agreement, err := u.returnAgreementRepo.GetByTradeID(ctx, tradeID)
	if err != nil {
		return reportdom.AddReportResult{}, err
	}

	if agreement.Status != tradedom.ReturnStatusDiscussing ||
		agreement.Proposal == nil ||
		agreement.Proposal.Agreement != tradedom.ReturnProposalAgreementDisagree {
		return reportdom.AddReportResult{}, tradedom.ErrReturnDisputeNotAllowed
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

	result, err := u.reportRepo.AddReport(ctx, reportCase, report)
	if err != nil {
		return reportdom.AddReportResult{}, err
	}

	message, err := tradedom.NewSystemMessageForCreate(
		reportTradeReturnDisputeSystemMessageID,
		tradeID,
		"購入者が運営へ返品問題を報告しました。",
	)
	if err != nil {
		return reportdom.AddReportResult{}, err
	}

	message.CreatedAt = now

	if _, err := u.tradeMessageRepo.Create(ctx, message); err != nil &&
		!errors.Is(err, tradedom.ErrMessageAlreadyExists) {
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

// resumeTradeReturnDiscussion applies the Admin "continue trade" decision to a
// disputed Avatar-to-Avatar resale return.
//
// The ReportCase decision itself is persisted by report_decision.go. This
// function owns only the Trade-side effects:
//   - confirm the target Trade is still active
//   - confirm the return is currently disputed after a seller disagreement
//   - create an AMOL system message containing the Admin decision reason
//   - move ReturnAgreement from disputed back to discussing
//
// The system-message ID is derived from DisputedAt so retries for the same
// dispute remain idempotent while a later dispute can create a different
// message.
func (u *ReportUsecase) resumeTradeReturnDiscussion(
	ctx context.Context,
	tradeID string,
	reason string,
	at time.Time,
) error {
	if u == nil ||
		u.tradeRepo == nil ||
		u.tradeMessageRepo == nil ||
		u.returnAgreementRepo == nil {
		return ErrReportUsecaseNotConfigured
	}

	tradeID = strings.TrimSpace(tradeID)
	reason = strings.TrimSpace(reason)

	if tradeID == "" {
		return reportdom.ErrInvalidTargetID
	}
	if reason == "" {
		return reportdom.ErrDecisionReasonRequired
	}
	if at.IsZero() {
		return tradedom.ErrInvalidReturnAgreementUpdatedAt
	}

	at = at.UTC()

	trade, err := u.tradeRepo.GetByID(ctx, tradeID)
	if err != nil {
		return err
	}

	if trade.ID != tradeID {
		return tradedom.ErrNotFound
	}
	if trade.Status == tradedom.StatusClosed {
		return tradedom.ErrTradeAlreadyClosed
	}
	if trade.Status != tradedom.StatusActive {
		return tradedom.ErrInvalidStatus
	}
	if trade.SellerType != tradedom.SellerTypeAvatar ||
		strings.TrimSpace(trade.SellerAvatarID) == "" ||
		strings.TrimSpace(trade.BuyerAvatarID) == "" {
		return ErrReportInvalidDecision
	}

	agreement, err := u.returnAgreementRepo.GetByTradeID(ctx, tradeID)
	if err != nil {
		return err
	}

	if agreement.ID != tradeID ||
		agreement.TradeID != tradeID {
		return tradedom.ErrReturnAgreementConflict
	}

	// Successful retries after the dispute has already been resumed are treated
	// as idempotent success. ResumeDiscussion intentionally preserves the latest
	// seller disagreement, so this state uniquely represents the completed
	// "continue trade" transition for this workflow.
	if agreement.Status == tradedom.ReturnStatusDiscussing &&
		agreement.DisputedAt == nil &&
		agreement.Proposal != nil &&
		agreement.Proposal.Agreement == tradedom.ReturnProposalAgreementDisagree {
		return nil
	}

	if agreement.Status != tradedom.ReturnStatusDisputed ||
		agreement.DisputedAt == nil ||
		agreement.DisputedAt.IsZero() ||
		agreement.Proposal == nil ||
		agreement.Proposal.Agreement != tradedom.ReturnProposalAgreementDisagree {
		return tradedom.ErrReturnDiscussionResumeNotAllowed
	}

	disputedAt := agreement.DisputedAt.UTC()

	if at.Before(disputedAt) ||
		(!agreement.UpdatedAt.IsZero() && at.Before(agreement.UpdatedAt)) {
		return tradedom.ErrInvalidReturnAgreementUpdatedAt
	}

	messageID :=
		reportTradeReturnDiscussionResumedSystemMessageIDPrefix +
			strconv.FormatInt(disputedAt.UnixNano(), 10)

	messageContent :=
		"運営が返品相談の続行を案内しました。\n" +
			reason

	message, err := tradedom.NewSystemMessageForCreate(
		messageID,
		tradeID,
		messageContent,
	)
	if err != nil {
		return err
	}

	message.CreatedAt = at

	// Create the deterministic message before clearing DisputedAt. If the
	// following ReturnAgreement update fails, a retry still has the same
	// DisputedAt and therefore attempts the same message ID rather than creating
	// a duplicate timeline entry.
	if _, err := u.tradeMessageRepo.Create(ctx, message); err != nil &&
		!errors.Is(err, tradedom.ErrMessageAlreadyExists) {
		return err
	}

	if err := agreement.ResumeDiscussion(at); err != nil {
		return err
	}

	if _, err := u.returnAgreementRepo.Update(
		ctx,
		tradeID,
		agreement,
	); err != nil {
		return err
	}

	return nil
}
