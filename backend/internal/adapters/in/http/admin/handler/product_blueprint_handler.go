// backend/internal/adapters/in/http/admin/handler/company_product_blueprint_handler.go
package handler

import (
	"errors"
	"net/http"
	"strings"

	adminquery "narratives/internal/application/query/admin"
	common "narratives/internal/domain/common"
	companydom "narratives/internal/domain/company"
	productblueprintdom "narratives/internal/domain/productBlueprint"
	productblueprintreviewdom "narratives/internal/domain/productBlueprintReview"
)

func (h *CompanyHandler) handleContractProductBlueprintDetail(
	w http.ResponseWriter,
	r *http.Request,
	companyID string,
	productBlueprintID string,
) {
	if r.Method != http.MethodGet {
		writeJSONError(
			w,
			http.StatusMethodNotAllowed,
			"method_not_allowed",
		)
		return
	}

	if h.contractProductBlueprintQuery == nil {
		writeJSONError(
			w,
			http.StatusServiceUnavailable,
			"contract_product_blueprint_query_not_initialized",
		)
		return
	}

	result, err := h.contractProductBlueprintQuery.Get(
		r.Context(),
		companyID,
		productBlueprintID,
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

		case errors.Is(err, productblueprintdom.ErrNotFound),
			errors.Is(err, productblueprintdom.ErrInvalidID):
			writeJSONError(
				w,
				http.StatusNotFound,
				"product_blueprint_not_found",
			)

		case errors.Is(
			err,
			adminquery.ErrContractProductBlueprintQueryNotConfigured,
		):
			writeJSONError(
				w,
				http.StatusServiceUnavailable,
				"contract_product_blueprint_query_not_initialized",
			)

		default:
			writeJSONError(
				w,
				http.StatusInternalServerError,
				"contract_product_blueprint_get_failed",
			)
		}
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *CompanyHandler) handleContractProductBlueprintReviews(
	w http.ResponseWriter,
	r *http.Request,
	companyID string,
	productBlueprintID string,
) {
	if r.Method != http.MethodGet {
		writeJSONError(
			w,
			http.StatusMethodNotAllowed,
			"method_not_allowed",
		)
		return
	}

	if h.contractProductBlueprintReviewQuery == nil {
		writeJSONError(
			w,
			http.StatusServiceUnavailable,
			"contract_product_blueprint_review_query_not_initialized",
		)
		return
	}

	query := r.URL.Query()
	status := productblueprintreviewdom.ReviewStatus(
		strings.ToUpper(
			strings.TrimSpace(query.Get("status")),
		),
	)
	page := common.Page{
		Number:  parsePositiveInt(query.Get("page"), 1),
		PerPage: parsePositiveInt(query.Get("perPage"), 20),
	}

	result, err := h.contractProductBlueprintReviewQuery.List(
		r.Context(),
		companyID,
		productBlueprintID,
		status,
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

		case errors.Is(err, productblueprintdom.ErrNotFound),
			errors.Is(err, productblueprintdom.ErrInvalidID):
			writeJSONError(
				w,
				http.StatusNotFound,
				"product_blueprint_not_found",
			)

		case errors.Is(
			err,
			productblueprintreviewdom.ErrInvalidStatus,
		):
			writeJSONError(
				w,
				http.StatusBadRequest,
				"invalid_review_status",
			)

		case errors.Is(
			err,
			adminquery.ErrContractProductBlueprintReviewQueryNotConfigured,
		):
			writeJSONError(
				w,
				http.StatusServiceUnavailable,
				"contract_product_blueprint_review_query_not_initialized",
			)

		default:
			writeJSONError(
				w,
				http.StatusInternalServerError,
				"contract_product_blueprint_review_list_failed",
			)
		}
		return
	}

	writeJSON(w, http.StatusOK, result)
}
