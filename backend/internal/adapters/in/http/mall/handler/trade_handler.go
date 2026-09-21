// backend/internal/adapters/in/http/mall/handler/trade_handler.go
package mallHandler

import (
	"net/http"
	"strings"

	mallquery "narratives/internal/application/query/mall"
	usecase "narratives/internal/application/usecase"
)

// TradeHandler handles private Resale Trade communication in Mall.
//
// Trade is limited to secondary-market transactions:
//
//	buyer Avatar <-> seller Avatar
//
// Avatar identity is always resolved from AvatarContextMiddleware and is never
// accepted from request body or query parameters.
type TradeHandler struct {
	query                *mallquery.TradeQuery
	messageUC            *usecase.TradeMessageUsecase
	reportUC             *usecase.ReportUsecase
	dispatchUC           *usecase.ResaleTradeDispatchUsecase
	returnConsultationUC *usecase.ResaleTradeReturnConsultationUsecase
	returnProposalUC     *usecase.ResaleTradeReturnProposalUsecase
	returnReceiptUC      *usecase.ResaleTradeReturnReceiptUsecase
}

func NewTradeHandler(
	query *mallquery.TradeQuery,
	messageUC *usecase.TradeMessageUsecase,
	reportUC *usecase.ReportUsecase,
	dispatchUC *usecase.ResaleTradeDispatchUsecase,
	returnConsultationUC *usecase.ResaleTradeReturnConsultationUsecase,
	returnProposalUC *usecase.ResaleTradeReturnProposalUsecase,
	returnReceiptUC *usecase.ResaleTradeReturnReceiptUsecase,
) http.Handler {
	return &TradeHandler{
		query:                query,
		messageUC:            messageUC,
		reportUC:             reportUC,
		dispatchUC:           dispatchUC,
		returnConsultationUC: returnConsultationUC,
		returnProposalUC:     returnProposalUC,
		returnReceiptUC:      returnReceiptUC,
	}
}

// ServeHTTP is the routing entry point.
//
// Supported:
//
//	GET  /mall/me/trades
//	GET  /mall/me/trades/order-items/{orderId}/{itemIndex}
//	GET  /mall/me/trades/{tradeId}
//	POST /mall/me/trades/{tradeId}/messages
//	POST /mall/me/trades/{tradeId}/messages/{messageId}/reports
//	POST /mall/me/trades/{tradeId}/read
//	GET  /mall/me/trades/{tradeId}/unread-count
//	POST /mall/me/trades/{tradeId}/dispatch
//	POST /mall/me/trades/{tradeId}/return-consultations
//	POST /mall/me/trades/{tradeId}/return-proposals
//	POST /mall/me/trades/{tradeId}/receive-return
func (h *TradeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if h == nil || h.query == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "trade query is nil",
		})
		return
	}
	if h.messageUC == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "trade message usecase is nil",
		})
		return
	}

	if r.URL.Path == "/mall/me/trades" || r.URL.Path == "/mall/me/trades/" {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}

		h.list(w, r)
		return
	}

	if strings.HasPrefix(r.URL.Path, "/mall/me/trades/order-items/") {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}

		h.getByOrderItem(w, r)
		return
	}

	if !strings.HasPrefix(r.URL.Path, "/mall/me/trades/") {
		notFound(w)
		return
	}

	rest := strings.TrimPrefix(r.URL.Path, "/mall/me/trades/")
	parts := strings.Split(rest, "/")
	if len(parts) == 0 || parts[0] == "" {
		notFound(w)
		return
	}

	tradeID := parts[0]

	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}

		h.getByID(w, r, tradeID)
		return
	}

	if len(parts) == 4 &&
		parts[1] == "messages" &&
		parts[2] != "" &&
		parts[3] == "reports" {
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}

		h.reportMessage(w, r, tradeID, parts[2])
		return
	}

	if len(parts) != 2 || parts[1] == "" {
		notFound(w)
		return
	}

	switch parts[1] {
	case "messages":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}

		h.createMessage(w, r, tradeID)

	case "read":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}

		h.markRead(w, r, tradeID)

	case "unread-count":
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}

		h.countUnread(w, r, tradeID)

	case "dispatch":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}

		h.dispatch(w, r, tradeID)

	case "return-consultations":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}

		h.createReturnConsultation(w, r, tradeID)

	case "return-proposals":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}

		h.createReturnProposal(w, r, tradeID)

	case "receive-return":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}

		h.receiveReturn(w, r, tradeID)

	default:
		notFound(w)
	}
}
