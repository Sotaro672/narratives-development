// backend/internal/adapters/in/http/mall/handler/brand_report_handler.go
package mallHandler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"narratives/internal/adapters/in/http/middleware"
	appusecase "narratives/internal/application/usecase"
	branddom "narratives/internal/domain/brand"
	reportdom "narratives/internal/domain/report"
)

// BrandReportService owns avatar-side reporting of Brands.
type BrandReportService interface {
	ReportBrandByAvatar(
		ctx context.Context,
		input appusecase.ReportBrandByAvatarInput,
	) (reportdom.AddReportResult, error)
}

type BrandReportHandler struct {
	reportSvc BrandReportService
}

type reportBrandRequest struct {
	Reason string `json:"reason"`
	Detail string `json:"detail"`
}

type brandReportResponse struct {
	CaseID        string               `json:"caseId"`
	ReportID      string               `json:"reportId"`
	ReportCount   int                  `json:"reportCount"`
	Status        reportdom.CaseStatus `json:"status"`
	CaseCreated   bool                 `json:"caseCreated"`
	ReportCreated bool                 `json:"reportCreated"`
}

func NewBrandReportHandler(
	reportSvc BrandReportService,
) *BrandReportHandler {
	return &BrandReportHandler{
		reportSvc: reportSvc,
	}
}

func (h *BrandReportHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	brandID, ok := parseBrandReportPath(
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

	h.handleReport(w, r, brandID)
}

func (h *BrandReportHandler) handleReport(
	w http.ResponseWriter,
	r *http.Request,
	brandID string,
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

	brandID = strings.TrimSpace(brandID)
	if brandID == "" {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"brandId is required",
		)
		return
	}

	var request reportBrandRequest

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

	result, err := h.reportSvc.ReportBrandByAvatar(
		r.Context(),
		appusecase.ReportBrandByAvatarInput{
			BrandID:  brandID,
			AvatarID: avatarID,
			Reason:   reason,
			Detail:   request.Detail,
		},
	)
	if err != nil {
		writeBrandReportError(w, err)
		return
	}

	statusCode := http.StatusCreated
	if !result.ReportCreated {
		statusCode = http.StatusOK
	}

	writeJSON(
		w,
		statusCode,
		brandReportResponse{
			CaseID:        string(result.Case.ID),
			ReportID:      string(result.Report.ID),
			ReportCount:   result.Case.ReportCount,
			Status:        result.Case.Status,
			CaseCreated:   result.CaseCreated,
			ReportCreated: result.ReportCreated,
		},
	)
}

func parseBrandReportPath(
	path string,
) (brandID string, ok bool) {
	const prefix = "/mall/me/brands/"
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

func writeBrandReportError(
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
	case errors.Is(err, branddom.ErrNotFound):
		writeJSONError(
			w,
			http.StatusNotFound,
			"brand not found",
		)

	case errors.Is(err, branddom.ErrInvalidID):
		writeJSONError(
			w,
			http.StatusBadRequest,
			err.Error(),
		)

	default:
		writeReportError(w, err)
	}
}
