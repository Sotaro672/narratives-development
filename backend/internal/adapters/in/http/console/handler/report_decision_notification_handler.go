// backend/internal/adapters/in/http/console/handler/report_decision_notification_handler.go
package consoleHandler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	uc "narratives/internal/application/usecase"
	domcommon "narratives/internal/domain/common"
	reportdom "narratives/internal/domain/report"
)

const (
	defaultReportDecisionNotificationPage    = 1
	defaultReportDecisionNotificationPerPage = 20
	maxReportDecisionNotificationPerPage     = 100
)

type ReportDecisionNotificationHandler struct {
	ReportUC *uc.ReportUsecase
}

func NewReportDecisionNotificationHandler(
	reportUC *uc.ReportUsecase,
) *ReportDecisionNotificationHandler {
	return &ReportDecisionNotificationHandler{
		ReportUC: reportUC,
	}
}

// Supported:
// - GET  /report-decision-notifications
// - GET  /report-decision-notifications?isRead=false&page=1&perPage=20
// - POST /report-decision-notifications/{notificationId}/read
func (h *ReportDecisionNotificationHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if h == nil || h.ReportUC == nil {
		writeError(
			w,
			http.StatusServiceUnavailable,
			"ReportDecisionNotificationHandlerNotInitialized",
		)
		return
	}

	notificationID, isReadPath, matched := parseReportDecisionNotificationPath(
		r.URL.Path,
	)
	if !matched {
		writeError(w, http.StatusNotFound, "NotFound")
		return
	}

	if isReadPath {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed")
			return
		}

		h.markRead(w, r, notificationID)
		return
	}

	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed")
		return
	}

	h.list(w, r)
}

// ============================================================
// DTO
// ============================================================

type reportDecisionNotificationResponse struct {
	ID             string                 `json:"id"`
	CaseID         string                 `json:"caseId"`
	ReportID       string                 `json:"reportId"`
	RecipientType  reportdom.ActorType    `json:"recipientType"`
	RecipientID    string                 `json:"recipientId"`
	CompanyID      string                 `json:"companyId"`
	TargetType     reportdom.TargetType   `json:"targetType"`
	TargetID       string                 `json:"targetId"`
	TargetParentID string                 `json:"targetParentId"`
	ReportReason   reportdom.ReportReason `json:"reportReason"`
	ReportDetail   string                 `json:"reportDetail"`
	DecisionStatus reportdom.CaseStatus   `json:"decisionStatus"`
	DecisionReason string                 `json:"decisionReason"`
	DecidedAt      time.Time              `json:"decidedAt"`
	CreatedAt      time.Time              `json:"createdAt"`
	UpdatedAt      time.Time              `json:"updatedAt"`
	ReadAt         *time.Time             `json:"readAt"`
	IsRead         bool                   `json:"isRead"`
}

// ============================================================
// List
// ============================================================

func (h *ReportDecisionNotificationHandler) list(
	w http.ResponseWriter,
	r *http.Request,
) {
	companyID := uc.CompanyIDFromContext(r.Context())
	if companyID == "" {
		writeError(
			w,
			http.StatusForbidden,
			"CompanyIDNotResolved",
		)
		return
	}

	isRead, err := parseReportDecisionNotificationIsRead(
		r.URL.Query().Get("isRead"),
	)
	if err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"InvalidIsRead",
		)
		return
	}

	page, err := parseReportDecisionNotificationPage(r)
	if err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"InvalidPagination",
		)
		return
	}

	result, err := h.ReportUC.ListDecisionNotificationsForCompany(
		r.Context(),
		companyID,
		isRead,
		page,
	)
	if err != nil {
		writeReportDecisionNotificationError(w, err)
		return
	}

	items := make(
		[]reportDecisionNotificationResponse,
		0,
		len(result.Items),
	)
	for _, notification := range result.Items {
		items = append(
			items,
			toReportDecisionNotificationResponse(notification),
		)
	}

	writeJSON(
		w,
		http.StatusOK,
		domcommon.PageResult[reportDecisionNotificationResponse]{
			Items:      items,
			TotalCount: result.TotalCount,
			TotalPages: result.TotalPages,
			Page:       result.Page,
			PerPage:    result.PerPage,
		},
	)
}

// ============================================================
// Mark read
// ============================================================

func (h *ReportDecisionNotificationHandler) markRead(
	w http.ResponseWriter,
	r *http.Request,
	notificationID string,
) {
	if notificationID == "" {
		writeError(
			w,
			http.StatusBadRequest,
			"NotificationIDRequired",
		)
		return
	}

	companyID := uc.CompanyIDFromContext(r.Context())
	if companyID == "" {
		writeError(
			w,
			http.StatusForbidden,
			"CompanyIDNotResolved",
		)
		return
	}

	notification, err := h.ReportUC.MarkDecisionNotificationReadForCompany(
		r.Context(),
		reportdom.DecisionNotificationID(notificationID),
		companyID,
	)
	if err != nil {
		writeReportDecisionNotificationError(w, err)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		toReportDecisionNotificationResponse(notification),
	)
}

// ============================================================
// Path
// ============================================================

func parseReportDecisionNotificationPath(
	path string,
) (
	notificationID string,
	isReadPath bool,
	matched bool,
) {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return "", false, false
	}

	parts := strings.Split(trimmed, "/")

	if len(parts) == 1 &&
		parts[0] == "report-decision-notifications" {
		return "", false, true
	}

	if len(parts) == 3 &&
		parts[0] == "report-decision-notifications" &&
		parts[1] != "" &&
		parts[2] == "read" {
		return parts[1], true, true
	}

	return "", false, false
}

// ============================================================
// Query params
// ============================================================

func parseReportDecisionNotificationIsRead(
	raw string,
) (*bool, error) {
	if raw == "" {
		return nil, nil
	}

	switch raw {
	case "true":
		value := true
		return &value, nil

	case "false":
		value := false
		return &value, nil

	default:
		return nil, errors.New("invalid isRead")
	}
}

func parseReportDecisionNotificationPage(
	r *http.Request,
) (domcommon.Page, error) {
	pageNumber := defaultReportDecisionNotificationPage
	perPage := defaultReportDecisionNotificationPerPage

	rawPage := r.URL.Query().Get("page")
	if rawPage != "" {
		parsed, err := strconv.Atoi(rawPage)
		if err != nil || parsed <= 0 {
			return domcommon.Page{}, errors.New("invalid page")
		}
		pageNumber = parsed
	}

	rawPerPage := r.URL.Query().Get("perPage")
	if rawPerPage != "" {
		parsed, err := strconv.Atoi(rawPerPage)
		if err != nil ||
			parsed <= 0 ||
			parsed > maxReportDecisionNotificationPerPage {
			return domcommon.Page{}, errors.New("invalid perPage")
		}
		perPage = parsed
	}

	return domcommon.Page{
		Number:  pageNumber,
		PerPage: perPage,
	}, nil
}

// ============================================================
// Mapping
// ============================================================

func toReportDecisionNotificationResponse(
	notification reportdom.DecisionNotification,
) reportDecisionNotificationResponse {
	return reportDecisionNotificationResponse{
		ID:             string(notification.ID),
		CaseID:         string(notification.CaseID),
		ReportID:       string(notification.ReportID),
		RecipientType:  notification.RecipientType,
		RecipientID:    notification.RecipientID,
		CompanyID:      notification.CompanyID,
		TargetType:     notification.TargetType,
		TargetID:       notification.TargetID,
		TargetParentID: notification.TargetParentID,
		ReportReason:   notification.ReportReason,
		ReportDetail:   notification.ReportDetail,
		DecisionStatus: notification.DecisionStatus,
		DecisionReason: notification.DecisionReason,
		DecidedAt:      notification.DecidedAt,
		CreatedAt:      notification.CreatedAt,
		UpdatedAt:      notification.UpdatedAt,
		ReadAt:         notification.ReadAt,
		IsRead:         notification.IsRead(),
	}
}

// ============================================================
// Error mapping
// ============================================================

func writeReportDecisionNotificationError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(
		err,
		uc.ErrReportUsecaseNotConfigured,
	):
		writeError(
			w,
			http.StatusServiceUnavailable,
			"ReportUsecaseNotConfigured",
		)

	case errors.Is(
		err,
		uc.ErrReportForbidden,
	):
		writeError(
			w,
			http.StatusForbidden,
			"ReportForbidden",
		)

	case status.Code(err) == codes.NotFound:
		writeError(
			w,
			http.StatusNotFound,
			"ReportDecisionNotificationNotFound",
		)

	case reportdom.IsInvalid(err),
		errors.Is(
			err,
			reportdom.ErrInvalidDecisionNotificationID,
		),
		errors.Is(
			err,
			reportdom.ErrInvalidDecisionNotificationKind,
		),
		errors.Is(
			err,
			reportdom.ErrDecisionNotificationCaseMismatch,
		),
		errors.Is(
			err,
			reportdom.ErrDecisionNotificationCaseNotDecided,
		),
		errors.Is(
			err,
			reportdom.ErrInvalidDecisionNotificationCreatedAt,
		),
		errors.Is(
			err,
			reportdom.ErrInvalidDecisionNotificationUpdatedAt,
		),
		errors.Is(
			err,
			reportdom.ErrInvalidDecisionNotificationReadAt,
		),
		errors.Is(
			err,
			reportdom.ErrTargetDecisionNotificationReportData,
		):
		writeError(
			w,
			http.StatusBadRequest,
			"InvalidReportDecisionNotification",
		)

	default:
		writeError(
			w,
			http.StatusInternalServerError,
			"ReportDecisionNotificationInternalError",
		)
	}
}
