// backend/internal/adapters/in/http/mall/handler/list_report_handler.go
package mallHandler

import (
	"context"
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

	reason, detail, ok := decodeReportRequest(w, r)
	if !ok {
		return
	}

	result, err := h.reportSvc.ReportListByAvatar(
		r.Context(),
		appusecase.ReportListByAvatarInput{
			ListID:   listID,
			AvatarID: avatarID,
			Reason:   reason,
			Detail:   detail,
		},
	)
	if err != nil {
		writeListReportError(w, err)
		return
	}

	writeReportResult(w, result)
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
