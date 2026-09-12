// backend/internal/adapters/in/http/admin/handler/company_list_handler.go
package handler

import (
	"errors"
	"net/http"

	adminquery "narratives/internal/application/query/admin"
	companydom "narratives/internal/domain/company"
	inventorydom "narratives/internal/domain/inventory"
	listdom "narratives/internal/domain/list"
	productblueprintdom "narratives/internal/domain/productBlueprint"
	tokenblueprintdom "narratives/internal/domain/tokenBlueprint"
)

func (h *CompanyHandler) handleContractListDetail(
	w http.ResponseWriter,
	r *http.Request,
	companyID string,
	listID string,
) {
	if r.Method != http.MethodGet {
		writeJSONError(
			w,
			http.StatusMethodNotAllowed,
			"method_not_allowed",
		)
		return
	}

	if h.contractListQuery == nil {
		writeJSONError(
			w,
			http.StatusServiceUnavailable,
			"contract_list_query_not_initialized",
		)
		return
	}

	result, err := h.contractListQuery.Get(
		r.Context(),
		companyID,
		listID,
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

		case errors.Is(err, listdom.ErrNotFound),
			errors.Is(err, listdom.ErrInvalidID),
			errors.Is(err, inventorydom.ErrNotFound),
			errors.Is(err, inventorydom.ErrInvalidMintID),
			errors.Is(err, productblueprintdom.ErrNotFound),
			errors.Is(err, tokenblueprintdom.ErrNotFound):
			writeJSONError(
				w,
				http.StatusNotFound,
				"list_not_found",
			)

		case errors.Is(
			err,
			adminquery.ErrContractListQueryNotConfigured,
		):
			writeJSONError(
				w,
				http.StatusServiceUnavailable,
				"contract_list_query_not_initialized",
			)

		default:
			writeJSONError(
				w,
				http.StatusInternalServerError,
				"contract_list_get_failed",
			)
		}
		return
	}

	writeJSON(w, http.StatusOK, result)
}
