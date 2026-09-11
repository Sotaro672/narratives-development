// backend/internal/adapters/in/http/admin/handler/resale_trade_handler.go
package handler

import (
	"context"
	"net/http"

	adminquery "narratives/internal/application/query/admin"
)

type ResaleTradeQueryService interface {
	ListByResaleID(
		ctx context.Context,
		resaleID string,
	) (*adminquery.ResaleTradeListResult, error)
}

type ResaleTradeHandler struct {
	query ResaleTradeQueryService
}

func NewResaleTradeHandler(
	query ResaleTradeQueryService,
) http.Handler {
	return http.HandlerFunc((&ResaleTradeHandler{
		query: query,
	}).handle)
}

func (h *ResaleTradeHandler) handle(
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
			"resale_trade_query_not_initialized",
		)
		return
	}

	resaleID := r.PathValue("resaleID")
	if resaleID == "" {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"resale_id_required",
		)
		return
	}

	result, err := h.query.ListByResaleID(
		r.Context(),
		resaleID,
	)
	if err != nil {
		writeJSONError(
			w,
			http.StatusInternalServerError,
			"resale_trade_list_failed",
		)
		return
	}

	writeJSON(w, http.StatusOK, result)
}
