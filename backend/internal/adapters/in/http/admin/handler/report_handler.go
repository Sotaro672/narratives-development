// backend/internal/adapters/in/http/admin/handler/report_handler.go
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"narratives/internal/adapters/in/http/middleware"
	adminquery "narratives/internal/application/query/admin"
	usecase "narratives/internal/application/usecase"
	common "narratives/internal/domain/common"
	reportdom "narratives/internal/domain/report"
)

const adminReportsPath = "/admin/reports"

type ReportHandler struct {
	uc        *usecase.ReportUsecase
	nameQuery *adminquery.ReportNameQuery
}

func NewReportHandler(
	uc *usecase.ReportUsecase,
	nameQuery *adminquery.ReportNameQuery,
) http.Handler {
	return http.HandlerFunc((&ReportHandler{
		uc:        uc,
		nameQuery: nameQuery,
	}).handle)
}

type reportCaseResponse struct {
	ID               string               `json:"id"`
	TargetType       reportdom.TargetType `json:"targetType"`
	TargetID         string               `json:"targetId"`
	TargetParentID   string               `json:"targetParentId"`
	TargetParentName string               `json:"targetParentName,omitempty"`
	TargetAuthorID   string               `json:"targetAuthorId"`
	TargetAuthorName string               `json:"targetAuthorName,omitempty"`
	TargetAuthorType reportdom.ActorType  `json:"targetAuthorType"`
	SnapshotTitle    string               `json:"snapshotTitle"`
	SnapshotBody     string               `json:"snapshotBody"`
	SnapshotRating   *int                 `json:"snapshotRating"`
	ReportCount      int                  `json:"reportCount"`
	Status           reportdom.CaseStatus `json:"status"`
	CreatedAt        string               `json:"createdAt"`
	UpdatedAt        string               `json:"updatedAt"`
	DecidedAt        *string              `json:"decidedAt"`
	DecidedBy        string               `json:"decidedBy"`
	DecisionReason   string               `json:"decisionReason"`
}

type reportCaseListResponse struct {
	Items      []reportCaseResponse `json:"items"`
	TotalCount int                  `json:"totalCount"`
	TotalPages int                  `json:"totalPages"`
	Page       int                  `json:"page"`
	PerPage    int                  `json:"perPage"`
}

type reportItemResponse struct {
	ID           string                 `json:"id"`
	CaseID       string                 `json:"caseId"`
	ReporterType reportdom.ActorType    `json:"reporterType"`
	ReporterID   string                 `json:"reporterId"`
	ReporterName string                 `json:"reporterName"`
	CompanyID    string                 `json:"companyId"`
	CompanyName  string                 `json:"companyName"`
	Reason       reportdom.ReportReason `json:"reason"`
	Detail       string                 `json:"detail"`
	CreatedAt    string                 `json:"createdAt"`
}

type reportItemsPageResponse struct {
	Items      []reportItemResponse `json:"items"`
	TotalCount int                  `json:"totalCount"`
	TotalPages int                  `json:"totalPages"`
	Page       int                  `json:"page"`
	PerPage    int                  `json:"perPage"`
}

type reportDetailResponse struct {
	Case    reportCaseResponse      `json:"case"`
	Reports reportItemsPageResponse `json:"reports"`
}

type reportDecisionRequest struct {
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
}

type reportRouteKind int

const (
	reportRouteList reportRouteKind = iota
	reportRouteDetail
	reportRouteDecision
)

func (h *ReportHandler) handle(w http.ResponseWriter, r *http.Request) {
	if h.uc == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "report_usecase_not_initialized")
		return
	}

	caseID, kind, valid := resolveReportPath(r.URL.Path)
	if !valid {
		writeJSONError(w, http.StatusNotFound, "report_not_found")
		return
	}

	switch kind {
	case reportRouteList:
		if r.Method != http.MethodGet {
			writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed")
			return
		}
		h.handleList(w, r)

	case reportRouteDetail:
		if r.Method != http.MethodGet {
			writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed")
			return
		}
		h.handleDetail(w, r, caseID)

	case reportRouteDecision:
		if r.Method != http.MethodPost {
			writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed")
			return
		}
		h.handleDecision(w, r, caseID)

	default:
		writeJSONError(w, http.StatusNotFound, "report_not_found")
	}
}

func (h *ReportHandler) handleList(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	filter := reportdom.CaseFilter{
		TargetID:       strings.TrimSpace(query.Get("targetId")),
		TargetParentID: strings.TrimSpace(query.Get("targetParentId")),
		TargetAuthorID: strings.TrimSpace(query.Get("targetAuthorId")),
	}

	if value := strings.TrimSpace(query.Get("status")); value != "" {
		statusValue := reportdom.CaseStatus(strings.ToUpper(value))
		if err := statusValue.Validate(); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid_status")
			return
		}
		filter.Status = &statusValue
	}

	if value := strings.TrimSpace(query.Get("targetType")); value != "" {
		targetType := reportdom.TargetType(strings.ToUpper(value))
		if err := targetType.Validate(); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid_target_type")
			return
		}
		filter.TargetType = &targetType
	}

	if value := strings.TrimSpace(query.Get("targetAuthorType")); value != "" {
		actorType := reportdom.ActorType(strings.ToUpper(value))
		if err := actorType.Validate(); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid_target_author_type")
			return
		}
		filter.TargetAuthorType = &actorType
	}

	sortValue, ok := parseReportCaseSort(query.Get("sort"), query.Get("order"))
	if !ok {
		writeJSONError(w, http.StatusBadRequest, "invalid_sort")
		return
	}

	page := common.Page{
		Number:  parsePositiveInt(query.Get("page"), 1),
		PerPage: reportPerPage(query.Get("perPage")),
	}

	result, err := h.uc.ListReportCases(r.Context(), filter, sortValue, page)
	if err != nil {
		writeReportError(w, err, "report_list_failed")
		return
	}

	items := make([]reportCaseResponse, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toReportCaseResponse(item))
	}

	writeJSON(w, http.StatusOK, reportCaseListResponse{
		Items:      items,
		TotalCount: result.TotalCount,
		TotalPages: result.TotalPages,
		Page:       result.Page,
		PerPage:    result.PerPage,
	})
}

func (h *ReportHandler) handleDetail(
	w http.ResponseWriter,
	r *http.Request,
	caseID reportdom.CaseID,
) {
	reportCase, err := h.uc.GetReportCase(r.Context(), caseID)
	if err != nil {
		writeReportError(w, err, "report_get_failed")
		return
	}

	query := r.URL.Query()
	filter := reportdom.ReportFilter{
		CaseID:     caseID,
		ReporterID: strings.TrimSpace(query.Get("reporterId")),
		CompanyID:  strings.TrimSpace(query.Get("companyId")),
	}

	if value := strings.TrimSpace(query.Get("reporterType")); value != "" {
		actorType := reportdom.ActorType(strings.ToUpper(value))
		if err := actorType.Validate(); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid_reporter_type")
			return
		}
		filter.ReporterType = &actorType
	}

	if value := strings.TrimSpace(query.Get("reason")); value != "" {
		reason := reportdom.ReportReason(strings.ToUpper(value))
		if err := reason.Validate(); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid_reason")
			return
		}
		filter.Reason = &reason
	}

	sortValue, ok := parseReportItemSort(query.Get("sort"), query.Get("order"))
	if !ok {
		writeJSONError(w, http.StatusBadRequest, "invalid_sort")
		return
	}

	page := common.Page{
		Number:  parsePositiveInt(query.Get("page"), 1),
		PerPage: reportPerPage(query.Get("perPage")),
	}

	reports, err := h.uc.ListReports(r.Context(), caseID, filter, sortValue, page)
	if err != nil {
		writeReportError(w, err, "report_items_list_failed")
		return
	}

	items := make([]reportItemResponse, 0, len(reports.Items))
	for _, item := range reports.Items {
		items = append(items, h.toReportItemResponse(r.Context(), item))
	}

	writeJSON(w, http.StatusOK, reportDetailResponse{
		Case: h.toReportDetailCaseResponse(r.Context(), reportCase),
		Reports: reportItemsPageResponse{
			Items:      items,
			TotalCount: reports.TotalCount,
			TotalPages: reports.TotalPages,
			Page:       reports.Page,
			PerPage:    reports.PerPage,
		},
	})
}

func (h *ReportHandler) handleDecision(
	w http.ResponseWriter,
	r *http.Request,
	caseID reportdom.CaseID,
) {
	adminUID, ok := middleware.CurrentAdminUID(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "admin_identity_not_found")
		return
	}

	var request reportDecisionRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request")
		return
	}

	reason := strings.TrimSpace(request.Reason)
	if reason == "" {
		writeJSONError(w, http.StatusBadRequest, "decision_reason_required")
		return
	}

	var decision usecase.ReportDecision
	switch strings.ToUpper(strings.TrimSpace(request.Decision)) {
	case string(usecase.ReportDecisionKeep):
		decision = usecase.ReportDecisionKeep
	case string(usecase.ReportDecisionRemove):
		decision = usecase.ReportDecisionRemove
	default:
		writeJSONError(w, http.StatusBadRequest, "invalid_decision")
		return
	}

	result, err := h.uc.DecideReportCase(
		r.Context(),
		usecase.DecideReportCaseInput{
			CaseID:    caseID,
			Decision:  decision,
			Reason:    reason,
			DecidedBy: adminUID,
		},
	)
	if err != nil {
		writeReportError(w, err, "report_decision_failed")
		return
	}

	writeJSON(w, http.StatusOK, h.toReportDetailCaseResponse(r.Context(), result))
}

func resolveReportPath(
	requestPath string,
) (reportdom.CaseID, reportRouteKind, bool) {
	if requestPath == adminReportsPath || requestPath == adminReportsPath+"/" {
		return "", reportRouteList, true
	}
	if !strings.HasPrefix(requestPath, adminReportsPath+"/") {
		return "", reportRouteList, false
	}

	remaining := strings.Trim(strings.TrimPrefix(requestPath, adminReportsPath+"/"), "/")
	if remaining == "" {
		return "", reportRouteList, true
	}

	parts := strings.Split(remaining, "/")
	if len(parts) == 1 {
		caseID := strings.TrimSpace(parts[0])
		if caseID == "" {
			return "", reportRouteList, false
		}
		return reportdom.CaseID(caseID), reportRouteDetail, true
	}

	if len(parts) == 2 {
		caseID := strings.TrimSpace(parts[0])
		action := strings.TrimSpace(parts[1])
		if caseID == "" || action != "decision" {
			return "", reportRouteList, false
		}
		return reportdom.CaseID(caseID), reportRouteDecision, true
	}

	return "", reportRouteList, false
}

func parseReportCaseSort(
	columnValue string,
	orderValue string,
) (common.Sort, bool) {
	column := strings.TrimSpace(columnValue)
	if column == "" {
		column = "updatedAt"
	}

	if _, ok := reportdom.AllowedCaseSortColumns[column]; !ok {
		return common.Sort{}, false
	}

	order, ok := parseReportSortOrder(orderValue)
	if !ok {
		return common.Sort{}, false
	}

	return common.Sort{
		Column: column,
		Order:  order,
	}, true
}

func parseReportItemSort(
	columnValue string,
	orderValue string,
) (common.Sort, bool) {
	column := strings.TrimSpace(columnValue)
	if column == "" {
		column = "createdAt"
	}

	if _, ok := reportdom.AllowedReportSortColumns[column]; !ok {
		return common.Sort{}, false
	}

	order, ok := parseReportSortOrder(orderValue)
	if !ok {
		return common.Sort{}, false
	}

	return common.Sort{
		Column: column,
		Order:  order,
	}, true
}

func parseReportSortOrder(value string) (common.SortOrder, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return common.SortDesc, true
	}

	switch common.SortOrder(value) {
	case common.SortAsc:
		return common.SortAsc, true
	case common.SortDesc:
		return common.SortDesc, true
	default:
		return "", false
	}
}

func reportPerPage(value string) int {
	perPage := parsePositiveInt(value, 50)
	if perPage > 200 {
		return 200
	}
	return perPage
}

func toReportCaseResponse(
	reportCase reportdom.ReportCase,
) reportCaseResponse {
	var decidedAt *string
	if reportCase.DecidedAt != nil && !reportCase.DecidedAt.IsZero() {
		value := reportCase.DecidedAt.UTC().Format(time.RFC3339Nano)
		decidedAt = &value
	}

	return reportCaseResponse{
		ID:               string(reportCase.ID),
		TargetType:       reportCase.TargetType,
		TargetID:         reportCase.TargetID,
		TargetParentID:   reportCase.TargetParentID,
		TargetAuthorID:   reportCase.TargetAuthorID,
		TargetAuthorType: reportCase.TargetAuthorType,
		SnapshotTitle:    reportCase.SnapshotTitle,
		SnapshotBody:     reportCase.SnapshotBody,
		SnapshotRating:   reportCase.SnapshotRating,
		ReportCount:      reportCase.ReportCount,
		Status:           reportCase.Status,
		CreatedAt:        reportTimeString(reportCase.CreatedAt),
		UpdatedAt:        reportTimeString(reportCase.UpdatedAt),
		DecidedAt:        decidedAt,
		DecidedBy:        reportCase.DecidedBy,
		DecisionReason:   reportCase.DecisionReason,
	}
}

func (h *ReportHandler) toReportDetailCaseResponse(
	ctx context.Context,
	reportCase reportdom.ReportCase,
) reportCaseResponse {
	response := toReportCaseResponse(reportCase)
	if h == nil || h.nameQuery == nil {
		return response
	}

	response.TargetParentName = h.nameQuery.ResolveTargetParentName(
		ctx,
		reportCase.TargetType,
		reportCase.TargetParentID,
	)
	response.TargetAuthorName = h.nameQuery.ResolveTargetAuthorName(
		ctx,
		reportCase.TargetAuthorType,
		reportCase.TargetAuthorID,
	)
	return response
}

func (h *ReportHandler) toReportItemResponse(
	ctx context.Context,
	report reportdom.Report,
) reportItemResponse {
	response := reportItemResponse{
		ID:           string(report.ID),
		CaseID:       string(report.CaseID),
		ReporterType: report.ReporterType,
		ReporterID:   report.ReporterID,
		CompanyID:    report.CompanyID,
		Reason:       report.Reason,
		Detail:       report.Detail,
		CreatedAt:    reportTimeString(report.CreatedAt),
	}

	if h == nil || h.nameQuery == nil {
		return response
	}

	response.ReporterName = h.nameQuery.ResolveReporterName(
		ctx,
		report.ReporterType,
		report.ReporterID,
	)
	response.CompanyName = h.nameQuery.ResolveCompanyName(ctx, report.CompanyID)
	return response
}

func reportTimeString(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func writeReportError(
	w http.ResponseWriter,
	err error,
	fallback string,
) {
	if err == nil {
		return
	}

	if errors.Is(err, usecase.ErrReportUsecaseNotConfigured) {
		writeJSONError(w, http.StatusServiceUnavailable, "report_usecase_not_initialized")
		return
	}

	if errors.Is(err, usecase.ErrReportInvalidDecision) || reportdom.IsInvalid(err) {
		writeJSONError(w, http.StatusBadRequest, "invalid_report_request")
		return
	}

	if errors.Is(err, reportdom.ErrCaseAlreadyRemoved) {
		writeJSONError(w, http.StatusConflict, "report_case_already_removed")
		return
	}

	if status.Code(err) == codes.NotFound {
		writeJSONError(w, http.StatusNotFound, "report_not_found")
		return
	}

	writeJSONError(w, http.StatusInternalServerError, fallback)
}
