// backend/internal/adapters/in/http/mall/handler/list_report_handler.go
package mallHandler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"narratives/internal/adapters/in/http/middleware"
	appusecase "narratives/internal/application/usecase"
	inventorydom "narratives/internal/domain/inventory"
	listdom "narratives/internal/domain/list"
	productblueprintdom "narratives/internal/domain/productBlueprint"
	reportdom "narratives/internal/domain/report"
)

// ListReportService owns avatar-side reporting of Lists.
type ListReportService interface {
	ReportListByAvatar(
		ctx context.Context,
		input appusecase.ReportListByAvatarInput,
	) (reportdom.AddReportResult, error)
}

type ListReportHandler struct {
	reportSvc ListReportService
}

type reportListRequest struct {
	Reason string `json:"reason"`
	Detail string `json:"detail"`
}

type listReportResponse struct {
	CaseID        string               `json:"caseId"`
	ReportID      string               `json:"reportId"`
	ReportCount   int                  `json:"reportCount"`
	Status        reportdom.CaseStatus `json:"status"`
	CaseCreated   bool                 `json:"caseCreated"`
	ReportCreated bool                 `json:"reportCreated"`
}

func NewListReportHandler(
	reportSvc ListReportService,
) *ListReportHandler {
	return &ListReportHandler{
		reportSvc: reportSvc,
	}
}

func (h *ListReportHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	listID, ok := parseListReportPath(
		strings.TrimSuffix(r.URL.Path, "/"),
	)
	if !ok {
		http.NotFound(w, r)
		return
	}

	if r.Method != http.MethodPost {
		writeJSONError(
			w,
			http.StatusMethodNotAllowed,
			"method not allowed",
		)
		return
	}

	if h == nil || h.reportSvc == nil {
		writeJSONError(
			w,
			http.StatusServiceUnavailable,
			"report service not configured",
		)
		return
	}

	h.handleReport(w, r, listID)
}

func (h *ListReportHandler) handleReport(
	w http.ResponseWriter,
	r *http.Request,
	listID string,
) {
	avatarID, ok := middleware.CurrentAvatarID(r)
	if !ok || avatarID == "" {
		writeJSONError(
			w,
			http.StatusUnauthorized,
			"missing avatarId",
		)
		return
	}

	listID = strings.TrimSpace(listID)
	if listID == "" {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"listId is required",
		)
		return
	}

	var request reportListRequest

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

	result, err := h.reportSvc.ReportListByAvatar(
		r.Context(),
		appusecase.ReportListByAvatarInput{
			ListID:   listID,
			AvatarID: avatarID,
			Reason:   reason,
			Detail:   request.Detail,
		},
	)
	if err != nil {
		writeListReportError(w, err)
		return
	}

	statusCode := http.StatusCreated
	if !result.ReportCreated {
		statusCode = http.StatusOK
	}

	writeJSON(
		w,
		statusCode,
		listReportResponse{
			CaseID:        string(result.Case.ID),
			ReportID:      string(result.Report.ID),
			ReportCount:   result.Case.ReportCount,
			Status:        result.Case.Status,
			CaseCreated:   result.CaseCreated,
			ReportCreated: result.ReportCreated,
		},
	)
}

func parseListReportPath(
	path string,
) (listID string, ok bool) {
	const prefix = "/mall/me/lists/"
	const suffix = "/reports"

	if !strings.HasPrefix(path, prefix) ||
		!strings.HasSuffix(path, suffix) {
		return "", false
	}

	relative := strings.TrimPrefix(path, prefix)
	relative = strings.TrimSuffix(relative, suffix)
	relative = strings.TrimSuffix(relative, "/")

	if relative == "" ||
		strings.Contains(relative, "/") {
		return "", false
	}

	return relative, true
}

func writeListReportError(
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
	case errors.Is(err, listdom.ErrNotFound):
		writeJSONError(
			w,
			http.StatusNotFound,
			"list not found",
		)

	case errors.Is(err, inventorydom.ErrNotFound):
		writeJSONError(
			w,
			http.StatusNotFound,
			"inventory not found",
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
