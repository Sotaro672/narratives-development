// backend/internal/adapters/in/http/admin/handler/company_token_blueprint_handler.go
package handler

import (
	"errors"
	"net/http"

	adminquery "narratives/internal/application/query/admin"
	common "narratives/internal/domain/common"
	companydom "narratives/internal/domain/company"
	tokenblueprintdom "narratives/internal/domain/tokenBlueprint"
)

func (h *CompanyHandler) handleContractTokenBlueprintDetail(
	w http.ResponseWriter,
	r *http.Request,
	companyID string,
	tokenBlueprintID string,
) {
	if r.Method != http.MethodGet {
		writeJSONError(
			w,
			http.StatusMethodNotAllowed,
			"method_not_allowed",
		)
		return
	}

	if h.contractTokenBlueprintQuery == nil {
		writeJSONError(
			w,
			http.StatusServiceUnavailable,
			"contract_token_blueprint_query_not_initialized",
		)
		return
	}

	result, err := h.contractTokenBlueprintQuery.Get(
		r.Context(),
		companyID,
		tokenBlueprintID,
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

		case errors.Is(err, tokenblueprintdom.ErrNotFound),
			errors.Is(err, tokenblueprintdom.ErrInvalidID):
			writeJSONError(
				w,
				http.StatusNotFound,
				"token_blueprint_not_found",
			)

		case errors.Is(
			err,
			adminquery.ErrContractTokenBlueprintQueryNotConfigured,
		):
			writeJSONError(
				w,
				http.StatusServiceUnavailable,
				"contract_token_blueprint_query_not_initialized",
			)

		default:
			writeJSONError(
				w,
				http.StatusInternalServerError,
				"contract_token_blueprint_get_failed",
			)
		}
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *CompanyHandler) handleContractTokenBlueprintReviews(
	w http.ResponseWriter,
	r *http.Request,
	companyID string,
	tokenBlueprintID string,
) {
	if r.Method != http.MethodGet {
		writeJSONError(
			w,
			http.StatusMethodNotAllowed,
			"method_not_allowed",
		)
		return
	}

	if h.contractTokenBlueprintReviewQuery == nil {
		writeJSONError(
			w,
			http.StatusServiceUnavailable,
			"contract_token_blueprint_review_query_not_initialized",
		)
		return
	}

	query := r.URL.Query()
	page := common.Page{
		Number:  parsePositiveInt(query.Get("page"), 1),
		PerPage: parsePositiveInt(query.Get("perPage"), 20),
	}

	result, err := h.contractTokenBlueprintReviewQuery.List(
		r.Context(),
		companyID,
		tokenBlueprintID,
		page,
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

		case errors.Is(err, tokenblueprintdom.ErrNotFound),
			errors.Is(err, tokenblueprintdom.ErrInvalidID):
			writeJSONError(
				w,
				http.StatusNotFound,
				"token_blueprint_not_found",
			)

		case errors.Is(
			err,
			adminquery.ErrContractTokenBlueprintReviewQueryNotConfigured,
		):
			writeJSONError(
				w,
				http.StatusServiceUnavailable,
				"contract_token_blueprint_review_query_not_initialized",
			)

		default:
			writeJSONError(
				w,
				http.StatusInternalServerError,
				"contract_token_blueprint_review_list_failed",
			)
		}
		return
	}

	writeJSON(w, http.StatusOK, result)
}
