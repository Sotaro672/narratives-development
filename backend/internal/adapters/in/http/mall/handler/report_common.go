// backend/internal/adapters/in/http/mall/handler/report_common.go
package mallHandler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	appusecase "narratives/internal/application/usecase"
	pbr "narratives/internal/domain/productBlueprintReview"
	reportdom "narratives/internal/domain/report"
)

// reportRequest はMall側の通報APIで共通利用するリクエストです。
type reportRequest struct {
	Reason string `json:"reason"`
	Detail string `json:"detail"`
}

// reportResponse はMall側の通報APIで共通利用するレスポンスです。
type reportResponse struct {
	CaseID        string               `json:"caseId"`
	ReportID      string               `json:"reportId"`
	ReportCount   int                  `json:"reportCount"`
	Status        reportdom.CaseStatus `json:"status"`
	CaseCreated   bool                 `json:"caseCreated"`
	ReportCreated bool                 `json:"reportCreated"`
}

// decodeReportRequest は通報リクエストを解析し、reasonとdetailを正規化・検証します。
//
// falseを返した場合は、この関数内でエラーレスポンスを書き込み済みです。
func decodeReportRequest(
	w http.ResponseWriter,
	r *http.Request,
) (reportdom.ReportReason, string, bool) {
	var request reportRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json body")
		return "", "", false
	}

	reason := reportdom.ReportReason(
		strings.ToUpper(strings.TrimSpace(request.Reason)),
	)
	if err := reason.Validate(); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid report reason")
		return "", "", false
	}

	detail := strings.TrimSpace(request.Detail)
	if reason == reportdom.ReportReasonOther && detail == "" {
		writeJSONError(w, http.StatusBadRequest, "report detail required")
		return "", "", false
	}

	return reason, detail, true
}

// writeReportResult は通報作成結果を共通レスポンス形式で返します。
// 新規Reportが作成された場合は201、既存Reportだった場合は200を返します。
func writeReportResult(
	w http.ResponseWriter,
	result reportdom.AddReportResult,
) {
	statusCode := http.StatusCreated
	if !result.ReportCreated {
		statusCode = http.StatusOK
	}

	writeJSON(w, statusCode, reportResponse{
		CaseID:        string(result.Case.ID),
		ReportID:      string(result.Report.ID),
		ReportCount:   result.Case.ReportCount,
		Status:        result.Case.Status,
		CaseCreated:   result.CaseCreated,
		ReportCreated: result.ReportCreated,
	})
}

// writeReportError はMall側の通報処理で共通利用するエラーレスポンスを書き込みます。
func writeReportError(
	w http.ResponseWriter,
	err error,
) {
	if err == nil {
		writeJSONError(w, http.StatusInternalServerError, "unknown error")
		return
	}

	switch {
	case errors.Is(err, appusecase.ErrReportUsecaseNotConfigured):
		writeJSONError(w, http.StatusServiceUnavailable, "report service not configured")

	case errors.Is(err, appusecase.ErrReportForbidden):
		writeJSONError(w, http.StatusForbidden, "report forbidden")

	case errors.Is(err, appusecase.ErrReportSelfReport):
		writeJSONError(w, http.StatusForbidden, "self report is not allowed")

	case errors.Is(err, reportdom.ErrCannotReportRemovedTarget):
		writeJSONError(w, http.StatusConflict, "cannot report removed target")

	case reportdom.IsInvalid(err):
		writeJSONError(w, http.StatusBadRequest, err.Error())

	case errors.Is(err, pbr.ErrNotFound):
		writeJSONError(w, http.StatusNotFound, err.Error())

	case errors.Is(err, pbr.ErrConflict):
		writeJSONError(w, http.StatusConflict, err.Error())

	case errors.Is(err, pbr.ErrInvalid):
		writeJSONError(w, http.StatusBadRequest, err.Error())

	case errors.Is(err, pbr.ErrUnauthorized):
		writeJSONError(w, http.StatusUnauthorized, err.Error())

	case errors.Is(err, pbr.ErrForbidden):
		writeJSONError(w, http.StatusForbidden, err.Error())

	default:
		writeJSONError(w, http.StatusInternalServerError, err.Error())
	}
}
