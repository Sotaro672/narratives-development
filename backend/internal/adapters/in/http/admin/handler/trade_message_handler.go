// backend/internal/adapters/in/http/admin/handler/trade_message_handler.go
package handler

import (
	"context"
	"net/http"

	adminquery "narratives/internal/application/query/admin"
)

type TradeMessageQueryService interface {
	ListByTradeID(
		ctx context.Context,
		tradeID string,
	) (*adminquery.TradeMessageListResult, error)
}

type TradeMessageHandler struct {
	query TradeMessageQueryService
}

func NewTradeMessageHandler(
	query TradeMessageQueryService,
) http.Handler {
	return http.HandlerFunc((&TradeMessageHandler{
		query: query,
	}).handle)
}

func (h *TradeMessageHandler) handle(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}

	if h.query == nil {
		writeJSONError(
			w,
			http.StatusServiceUnavailable,
			"trade_message_query_not_initialized",
		)
		return
	}

	tradeID := r.PathValue("tradeID")
	if tradeID == "" {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"trade_id_required",
		)
		return
	}

	result, err := h.query.ListByTradeID(
		r.Context(),
		tradeID,
	)
	if err != nil {
		writeJSONError(
			w,
			http.StatusInternalServerError,
			"trade_message_list_failed",
		)
		return
	}

	writeJSON(w, http.StatusOK, result)
}
