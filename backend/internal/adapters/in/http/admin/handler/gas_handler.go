// backend/internal/adapters/in/http/admin/handler/gas_handler.go
package handler

import (
	"context"
	"net/http"

	adminquery "narratives/internal/application/query/admin"
)

type GasBalanceQueryService interface {
	GetGasBalance(ctx context.Context) (*adminquery.GasBalanceResult, error)
}

type GasHandler struct {
	query GasBalanceQueryService
}

func NewGasHandler(query GasBalanceQueryService) http.Handler {
	return http.HandlerFunc((&GasHandler{query: query}).handle)
}

func (h *GasHandler) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	if h.query == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "gas_balance_query_not_initialized")
		return
	}

	result, err := h.query.GetGasBalance(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusServiceUnavailable, "gas_balance_unavailable")
		return
	}

	writeJSON(w, http.StatusOK, result)
}
