// backend/internal/adapters/in/http/mall/handler/resale_report_handler.go
package mallHandler

import (
	"errors"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	productblueprintdom "narratives/internal/domain/productBlueprint"
	resaledom "narratives/internal/domain/resale"
)

func (h *ResaleHandler) reportResale(
	w http.ResponseWriter,
	r *http.Request,
	resaleID string,
) {
	if h == nil || h.reportUC == nil {
		writeJSONError(
			w,
			http.StatusServiceUnavailable,
			"report service not configured",
		)
		return
	}

	avatarID, ok := requireAvatarID(w, r)
	if !ok {
		return
	}

	resaleID = strings.TrimSpace(resaleID)
	if resaleID == "" {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"resaleId is required",
		)
		return
	}

	reason, detail, ok := decodeReportRequest(w, r)
	if !ok {
		return
	}

	result, err := h.reportUC.ReportResaleByAvatar(
		r.Context(),
		usecase.ReportResaleByAvatarInput{
			ResaleID: resaleID,
			AvatarID: avatarID,
			Reason:   reason,
			Detail:   detail,
		},
	)
	if err != nil {
		writeResaleReportError(w, err)
		return
	}

	writeReportResult(w, result)
}

func writeResaleReportError(
	w http.ResponseWriter,
	err error,
) {
	if err == nil {
		writeJSONError(
			w,
			http.StatusInternalServerError,
			"unknown error",
		)
		return
	}

	switch {
	case errors.Is(err, resaledom.ErrNotFound):
		writeJSONError(
			w,
			http.StatusNotFound,
			"resale not found",
		)

	case productblueprintdom.IsNotFound(err):
		writeJSONError(
			w,
			http.StatusNotFound,
			"product blueprint not found",
		)

	default:
		writeReportError(w, err)
	}
}
