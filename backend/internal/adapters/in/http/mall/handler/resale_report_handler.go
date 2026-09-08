// backend/internal/adapters/in/http/mall/handler/resale_report_handler.go
package mallHandler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	usecase "narratives/internal/application/usecase"
	productblueprintdom "narratives/internal/domain/productBlueprint"
	reportdom "narratives/internal/domain/report"
	resaledom "narratives/internal/domain/resale"
)

type resaleReportRequest struct {
	Reason string `json:"reason"`
	Detail string `json:"detail"`
}

type resaleReportResponse struct {
	CaseID        string               `json:"caseId"`
	ReportID      string               `json:"reportId"`
	ReportCount   int                  `json:"reportCount"`
	Status        reportdom.CaseStatus `json:"status"`
	CaseCreated   bool                 `json:"caseCreated"`
	ReportCreated bool                 `json:"reportCreated"`
}

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

	var request resaleReportRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"invalid json body",
		)
		return
	}

	reason := reportdom.ReportReason(
		strings.ToUpper(
			strings.TrimSpace(request.Reason),
		),
	)
	if err := reason.Validate(); err != nil {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"invalid report reason",
		)
		return
	}

	request.Detail = strings.TrimSpace(request.Detail)
	if reason == reportdom.ReportReasonOther &&
		request.Detail == "" {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"report detail required",
		)
		return
	}

	result, err := h.reportUC.ReportResaleByAvatar(
		r.Context(),
		usecase.ReportResaleByAvatarInput{
			ResaleID: resaleID,
			AvatarID: avatarID,
			Reason:   reason,
			Detail:   request.Detail,
		},
	)
	if err != nil {
		writeResaleReportError(w, err)
		return
	}

	statusCode := http.StatusCreated
	if !result.ReportCreated {
		statusCode = http.StatusOK
	}

	writeJSON(
		w,
		statusCode,
		resaleReportResponse{
			CaseID:        string(result.Case.ID),
			ReportID:      string(result.Report.ID),
			ReportCount:   result.Case.ReportCount,
			Status:        result.Case.Status,
			CaseCreated:   result.CaseCreated,
			ReportCreated: result.ReportCreated,
		},
	)
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
