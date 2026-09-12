// backend/internal/adapters/in/http/admin/handler/company_announcement_handler.go
package handler

import (
	"errors"
	"net/http"

	adminquery "narratives/internal/application/query/admin"
	announcementdom "narratives/internal/domain/announcement"
	companydom "narratives/internal/domain/company"
)

func (h *CompanyHandler) handleContractAnnouncementDetail(
	w http.ResponseWriter,
	r *http.Request,
	companyID string,
	announcementID string,
) {
	if r.Method != http.MethodGet {
		writeJSONError(
			w,
			http.StatusMethodNotAllowed,
			"method_not_allowed",
		)
		return
	}

	if h.contractAnnouncementDetailQuery == nil {
		writeJSONError(
			w,
			http.StatusServiceUnavailable,
			"contract_announcement_detail_query_not_initialized",
		)
		return
	}

	result, err := h.contractAnnouncementDetailQuery.Get(
		r.Context(),
		companyID,
		announcementID,
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

		case errors.Is(err, announcementdom.ErrNotFound),
			errors.Is(err, announcementdom.ErrInvalidID):
			writeJSONError(
				w,
				http.StatusNotFound,
				"announcement_not_found",
			)

		case errors.Is(
			err,
			adminquery.ErrContractAnnouncementDetailQueryNotConfigured,
		):
			writeJSONError(
				w,
				http.StatusServiceUnavailable,
				"contract_announcement_detail_query_not_initialized",
			)

		default:
			writeJSONError(
				w,
				http.StatusInternalServerError,
				"contract_announcement_detail_get_failed",
			)
		}
		return
	}

	writeJSON(w, http.StatusOK, result)
}
