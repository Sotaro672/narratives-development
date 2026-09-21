// backend/internal/adapters/in/http/mall/handler/trade_message_report_handler.go
package mallHandler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	reportdom "narratives/internal/domain/report"
	tradedom "narratives/internal/domain/trade"
)

type reportTradeMessageRequest struct {
	Reason string `json:"reason"`
	Detail string `json:"detail"`
}

type reportTradeMessageResponse struct {
	CaseID        string               `json:"caseId"`
	ReportID      string               `json:"reportId"`
	ReportCount   int                  `json:"reportCount"`
	Status        reportdom.CaseStatus `json:"status"`
	CaseCreated   bool                 `json:"caseCreated"`
	ReportCreated bool                 `json:"reportCreated"`
}

// POST /mall/me/trades/{tradeId}/messages/{messageId}/reports
//
// Reports one user-authored Trade message. The authenticated Avatar must be a
// participant of the Trade and cannot report their own message or a system
// message.
func (h *TradeHandler) reportMessage(
	w http.ResponseWriter,
	r *http.Request,
	tradeID string,
	messageID string,
) {
	if h == nil || h.reportUC == nil {
		writeJSONError(
			w,
			http.StatusServiceUnavailable,
			"report service not configured",
		)
		return
	}

	avatarID, ok := requireAvatarID(w, r)
	if !ok {
		return
	}

	tradeID = strings.TrimSpace(tradeID)
	if tradeID == "" {
		badRequest(w, "tradeId is required")
		return
	}

	messageID = strings.TrimSpace(messageID)
	if messageID == "" {
		badRequest(w, "messageId is required")
		return
	}

	var req reportTradeMessageRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		badRequest(w, "invalid json body")
		return
	}

	reason := reportdom.ReportReason(
		strings.ToUpper(strings.TrimSpace(req.Reason)),
	)
	if err := reason.Validate(); err != nil {
		badRequest(w, "invalid report reason")
		return
	}

	req.Detail = strings.TrimSpace(req.Detail)
	if reason == reportdom.ReportReasonOther && req.Detail == "" {
		badRequest(w, "report detail required")
		return
	}

	result, err := h.reportUC.ReportTradeMessageByAvatar(
		r.Context(),
		usecase.ReportTradeMessageByAvatarInput{
			TradeID:   tradeID,
			MessageID: messageID,
			AvatarID:  avatarID,
			Reason:    reason,
			Detail:    req.Detail,
		},
	)
	if err != nil {
		writeTradeMessageReportErr(w, err)
		return
	}

	statusCode := http.StatusCreated
	if !result.ReportCreated {
		statusCode = http.StatusOK
	}

	writeJSON(w, statusCode, reportTradeMessageResponse{
		CaseID:        string(result.Case.ID),
		ReportID:      string(result.Report.ID),
		ReportCount:   result.Case.ReportCount,
		Status:        result.Case.Status,
		CaseCreated:   result.CaseCreated,
		ReportCreated: result.ReportCreated,
	})
}

func writeTradeMessageReportErr(
	w http.ResponseWriter,
	err error,
) {
	if err == nil {
		internalError(w, "unknown error")
		return
	}

	switch {
	case errors.Is(err, tradedom.ErrNotFound),
		errors.Is(err, tradedom.ErrMessageNotFound):
		notFound(w)

	default:
		writeReportError(w, err)
	}
}
