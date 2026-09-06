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
	reviewreport "narratives/internal/domain/reviewReport"
)

const adminReportsPath = "/admin/reports"

type ReportHandler struct {
	uc        *usecase.ReviewReportUsecase
	nameQuery *adminquery.ReportNameQuery
}

func NewReportHandler(
	uc *usecase.ReviewReportUsecase,
	nameQuery *adminquery.ReportNameQuery,
) http.Handler {
	return http.HandlerFunc((&ReportHandler{
		uc:        uc,
		nameQuery: nameQuery,
	}).handle)
}

type reviewReportCaseResponse struct {
	ID               string                  `json:"id"`
	TargetType       reviewreport.TargetType `json:"targetType"`
	TargetID         string                  `json:"targetId"`
	TargetParentID   string                  `json:"targetParentId"`
	TargetParentName string                  `json:"targetParentName,omitempty"`
	TargetAuthorID   string                  `json:"targetAuthorId"`
	TargetAuthorName string                  `json:"targetAuthorName,omitempty"`
	TargetAuthorType reviewreport.ActorType  `json:"targetAuthorType"`
	SnapshotTitle    string                  `json:"snapshotTitle"`
	SnapshotBody     string                  `json:"snapshotBody"`
	SnapshotRating   *int                    `json:"snapshotRating"`
	ReportCount      int                     `json:"reportCount"`
	Status           reviewreport.CaseStatus `json:"status"`
	CreatedAt        string                  `json:"createdAt"`
	UpdatedAt        string                  `json:"updatedAt"`
	DecidedAt        *string                 `json:"decidedAt"`
	DecidedBy        string                  `json:"decidedBy"`
	DecisionReason   string                  `json:"decisionReason"`
}

type reviewReportCaseListResponse struct {
	Items      []reviewReportCaseResponse `json:"items"`
	TotalCount int                        `json:"totalCount"`
	TotalPages int                        `json:"totalPages"`
	Page       int                        `json:"page"`
	PerPage    int                        `json:"perPage"`
}

type reviewReportItemResponse struct {
	ID           string                    `json:"id"`
	CaseID       string                    `json:"caseId"`
	ReporterType reviewreport.ActorType    `json:"reporterType"`
	ReporterID   string                    `json:"reporterId"`
	ReporterName string                    `json:"reporterName"`
	CompanyID    string                    `json:"companyId"`
	CompanyName  string                    `json:"companyName"`
	Reason       reviewreport.ReportReason `json:"reason"`
	Detail       string                    `json:"detail"`
	CreatedAt    string                    `json:"createdAt"`
}

type reviewReportItemsPageResponse struct {
	Items      []reviewReportItemResponse `json:"items"`
	TotalCount int                        `json:"totalCount"`
	TotalPages int                        `json:"totalPages"`
	Page       int                        `json:"page"`
	PerPage    int                        `json:"perPage"`
}

type reviewReportDetailResponse struct {
	Case    reviewReportCaseResponse      `json:"case"`
	Reports reviewReportItemsPageResponse `json:"reports"`
}

type reviewReportDecisionRequest struct {
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
		writeJSONError(w, http.StatusServiceUnavailable, "review_report_usecase_not_initialized")
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
	filter := reviewreport.CaseFilter{
		TargetID:       strings.TrimSpace(query.Get("targetId")),
		TargetParentID: strings.TrimSpace(query.Get("targetParentId")),
		TargetAuthorID: strings.TrimSpace(query.Get("targetAuthorId")),
	}

	if value := strings.TrimSpace(query.Get("status")); value != "" {
		statusValue := reviewreport.CaseStatus(strings.ToUpper(value))
		if err := statusValue.Validate(); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid_status")
			return
		}
		filter.Status = &statusValue
	}

	if value := strings.TrimSpace(query.Get("targetType")); value != "" {
		targetType := reviewreport.TargetType(strings.ToUpper(value))
		if err := targetType.Validate(); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid_target_type")
			return
		}
		filter.TargetType = &targetType
	}

	if value := strings.TrimSpace(query.Get("targetAuthorType")); value != "" {
		actorType := reviewreport.ActorType(strings.ToUpper(value))
		if err := actorType.Validate(); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid_target_author_type")
			return
		}
		filter.TargetAuthorType = &actorType
	}

	sortValue, ok := parseReviewReportCaseSort(query.Get("sort"), query.Get("order"))
	if !ok {
		writeJSONError(w, http.StatusBadRequest, "invalid_sort")
		return
	}

	page := common.Page{
		Number:  parsePositiveInt(query.Get("page"), 1),
		PerPage: reviewReportPerPage(query.Get("perPage")),
	}

	result, err := h.uc.ListReportCases(r.Context(), filter, sortValue, page)
	if err != nil {
		writeReviewReportError(w, err, "report_list_failed")
		return
	}

	items := make([]reviewReportCaseResponse, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, toReviewReportCaseResponse(item))
	}

	writeJSON(w, http.StatusOK, reviewReportCaseListResponse{
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
	caseID reviewreport.CaseID,
) {
	reportCase, err := h.uc.GetReportCase(r.Context(), caseID)
	if err != nil {
		writeReviewReportError(w, err, "report_get_failed")
		return
	}

	query := r.URL.Query()
	filter := reviewreport.ReportFilter{
		CaseID:     caseID,
		ReporterID: strings.TrimSpace(query.Get("reporterId")),
		CompanyID:  strings.TrimSpace(query.Get("companyId")),
	}

	if value := strings.TrimSpace(query.Get("reporterType")); value != "" {
		actorType := reviewreport.ActorType(strings.ToUpper(value))
		if err := actorType.Validate(); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid_reporter_type")
			return
		}
		filter.ReporterType = &actorType
	}

	if value := strings.TrimSpace(query.Get("reason")); value != "" {
		reason := reviewreport.ReportReason(strings.ToUpper(value))
		if err := reason.Validate(); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid_reason")
			return
		}
		filter.Reason = &reason
	}

	sortValue, ok := parseReviewReportItemSort(query.Get("sort"), query.Get("order"))
	if !ok {
		writeJSONError(w, http.StatusBadRequest, "invalid_sort")
		return
	}

	page := common.Page{
		Number:  parsePositiveInt(query.Get("page"), 1),
		PerPage: reviewReportPerPage(query.Get("perPage")),
	}

	reports, err := h.uc.ListReports(r.Context(), caseID, filter, sortValue, page)
	if err != nil {
		writeReviewReportError(w, err, "report_items_list_failed")
		return
	}

	items := make([]reviewReportItemResponse, 0, len(reports.Items))
	for _, item := range reports.Items {
		items = append(items, h.toReviewReportItemResponse(r.Context(), item))
	}

	writeJSON(w, http.StatusOK, reviewReportDetailResponse{
		Case: h.toReviewReportDetailCaseResponse(r.Context(), reportCase),
		Reports: reviewReportItemsPageResponse{
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
	caseID reviewreport.CaseID,
) {
	adminUID, ok := middleware.CurrentAdminUID(r)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "admin_identity_not_found")
		return
	}

	var request reviewReportDecisionRequest
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

	var decision usecase.ReviewReportDecision
	switch strings.ToUpper(strings.TrimSpace(request.Decision)) {
	case string(usecase.ReviewReportDecisionKeep):
		decision = usecase.ReviewReportDecisionKeep
	case string(usecase.ReviewReportDecisionRemove):
		decision = usecase.ReviewReportDecisionRemove
	default:
		writeJSONError(w, http.StatusBadRequest, "invalid_decision")
		return
	}

	result, err := h.uc.DecideReportCase(
		r.Context(),
		usecase.DecideReviewReportCaseInput{
			CaseID:    caseID,
			Decision:  decision,
			Reason:    reason,
			DecidedBy: adminUID,
		},
	)
	if err != nil {
		writeReviewReportError(w, err, "report_decision_failed")
		return
	}

	writeJSON(w, http.StatusOK, h.toReviewReportDetailCaseResponse(r.Context(), result))
}

func resolveReportPath(
	requestPath string,
) (reviewreport.CaseID, reportRouteKind, bool) {
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
		return reviewreport.CaseID(caseID), reportRouteDetail, true
	}

	if len(parts) == 2 {
		caseID := strings.TrimSpace(parts[0])
		action := strings.TrimSpace(parts[1])
		if caseID == "" || action != "decision" {
			return "", reportRouteList, false
		}
		return reviewreport.CaseID(caseID), reportRouteDecision, true
	}

	return "", reportRouteList, false
}

func parseReviewReportCaseSort(
	columnValue string,
	orderValue string,
) (common.Sort, bool) {
	column := strings.TrimSpace(columnValue)
	if column == "" {
		column = "updatedAt"
	}
	if _, ok := reviewreport.AllowedCaseSortColumns[column]; !ok {
		return common.Sort{}, false
	}

	order, ok := parseReviewReportSortOrder(orderValue)
	if !ok {
		return common.Sort{}, false
	}

	return common.Sort{
		Column: column,
		Order:  order,
	}, true
}

func parseReviewReportItemSort(
	columnValue string,
	orderValue string,
) (common.Sort, bool) {
	column := strings.TrimSpace(columnValue)
	if column == "" {
		column = "createdAt"
	}
	if _, ok := reviewreport.AllowedReportSortColumns[column]; !ok {
		return common.Sort{}, false
	}

	order, ok := parseReviewReportSortOrder(orderValue)
	if !ok {
		return common.Sort{}, false
	}

	return common.Sort{
		Column: column,
		Order:  order,
	}, true
}

func parseReviewReportSortOrder(value string) (common.SortOrder, bool) {
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

func reviewReportPerPage(value string) int {
	perPage := parsePositiveInt(value, 50)
	if perPage > 200 {
		return 200
	}
	return perPage
}

func toReviewReportCaseResponse(
	reportCase reviewreport.ReportCase,
) reviewReportCaseResponse {
	var decidedAt *string
	if reportCase.DecidedAt != nil && !reportCase.DecidedAt.IsZero() {
		value := reportCase.DecidedAt.UTC().Format(time.RFC3339Nano)
		decidedAt = &value
	}

	return reviewReportCaseResponse{
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
		CreatedAt:        reviewReportTimeString(reportCase.CreatedAt),
		UpdatedAt:        reviewReportTimeString(reportCase.UpdatedAt),
		DecidedAt:        decidedAt,
		DecidedBy:        reportCase.DecidedBy,
		DecisionReason:   reportCase.DecisionReason,
	}
}

func (h *ReportHandler) toReviewReportDetailCaseResponse(
	ctx context.Context,
	reportCase reviewreport.ReportCase,
) reviewReportCaseResponse {
	response := toReviewReportCaseResponse(reportCase)
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

func (h *ReportHandler) toReviewReportItemResponse(
	ctx context.Context,
	report reviewreport.Report,
) reviewReportItemResponse {
	response := reviewReportItemResponse{
		ID:           string(report.ID),
		CaseID:       string(report.CaseID),
		ReporterType: report.ReporterType,
		ReporterID:   report.ReporterID,
		CompanyID:    report.CompanyID,
		Reason:       report.Reason,
		Detail:       report.Detail,
		CreatedAt:    reviewReportTimeString(report.CreatedAt),
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

func reviewReportTimeString(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func writeReviewReportError(
	w http.ResponseWriter,
	err error,
	fallback string,
) {
	if err == nil {
		return
	}

	if errors.Is(err, usecase.ErrReviewReportUsecaseNotConfigured) {
		writeJSONError(w, http.StatusServiceUnavailable, "review_report_usecase_not_initialized")
		return
	}

	if errors.Is(err, usecase.ErrReviewReportInvalidDecision) || reviewreport.IsInvalid(err) {
		writeJSONError(w, http.StatusBadRequest, "invalid_report_request")
		return
	}

	if errors.Is(err, reviewreport.ErrCaseAlreadyRemoved) {
		writeJSONError(w, http.StatusConflict, "report_case_already_removed")
		return
	}

	if status.Code(err) == codes.NotFound {
		writeJSONError(w, http.StatusNotFound, "report_not_found")
		return
	}

	writeJSONError(w, http.StatusInternalServerError, fallback)
}
