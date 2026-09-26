// backend/internal/adapters/in/http/mall/handler/trade_message_report_handler.go
package mallHandler

import (
	"errors"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	tradedom "narratives/internal/domain/trade"
)

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

	reason, detail, ok := decodeReportRequest(w, r)
	if !ok {
		return
	}

	result, err := h.reportUC.ReportTradeMessageByAvatar(
		r.Context(),
		usecase.ReportTradeMessageByAvatarInput{
			TradeID:   tradeID,
			MessageID: messageID,
			AvatarID:  avatarID,
			Reason:    reason,
			Detail:    detail,
		},
	)
	if err != nil {
		writeTradeMessageReportErr(w, err)
		return
	}

	writeReportResult(w, result)
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
