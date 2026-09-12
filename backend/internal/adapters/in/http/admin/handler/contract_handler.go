// backend/internal/adapters/in/http/admin/handler/company_contract_handler.go
package handler

import (
	"errors"
	"net/http"

	adminquery "narratives/internal/application/query/admin"
	companydom "narratives/internal/domain/company"
)

func (h *CompanyHandler) handleContractDetail(
	w http.ResponseWriter,
	r *http.Request,
	companyID string,
) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}

	if h.contractDetailQuery == nil {
		writeJSONError(
			w,
			http.StatusServiceUnavailable,
			"contract_detail_query_not_initialized",
		)
		return
	}

	result, err := h.contractDetailQuery.Get(
		r.Context(),
		companyID,
	)
	if err != nil {
		switch {
		case errors.Is(err, companydom.ErrNotFound),
			errors.Is(err, companydom.ErrInvalidID):
			writeJSONError(
				w,
				http.StatusNotFound,
				"company_not_found",
			)
		case errors.Is(
			err,
			adminquery.ErrContractDetailQueryNotConfigured,
		):
			writeJSONError(
				w,
				http.StatusServiceUnavailable,
				"contract_detail_query_not_initialized",
			)
		default:
			writeJSONError(
				w,
				http.StatusInternalServerError,
				"contract_detail_get_failed",
			)
		}
		return
	}

	writeJSON(w, http.StatusOK, result)
}
