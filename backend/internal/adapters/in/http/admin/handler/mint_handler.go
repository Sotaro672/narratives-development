// backend\internal\adapters\in\http\admin\handler\mint_handler.go
package handler

import (
	"context"
	"net/http"

	adminquery "narratives/internal/application/query/admin"
)

type MintListQueryService interface {
	ListMints(ctx context.Context) (*adminquery.MintListResult, error)
}

type MintHandler struct {
	query MintListQueryService
}

func NewMintHandler(query MintListQueryService) http.Handler {
	return http.HandlerFunc((&MintHandler{query: query}).handle)
}

func (h *MintHandler) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}

	if h.query == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "mint_list_query_not_initialized")
		return
	}

	result, err := h.query.ListMints(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "mint_list_unavailable")
		return
	}

	writeJSON(w, http.StatusOK, result)
}
