// backend/internal/adapters/in/http/console/handler/review_report_decision_notification_handler.go
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
	reviewreport "narratives/internal/domain/reviewReport"
)

const (
	defaultReviewReportDecisionNotificationPage    = 1
	defaultReviewReportDecisionNotificationPerPage = 20
	maxReviewReportDecisionNotificationPerPage     = 100
)

type ReviewReportDecisionNotificationHandler struct {
	ReviewReportUC *uc.ReviewReportUsecase
}

func NewReviewReportDecisionNotificationHandler(
	reviewReportUC *uc.ReviewReportUsecase,
) *ReviewReportDecisionNotificationHandler {
	return &ReviewReportDecisionNotificationHandler{
		ReviewReportUC: reviewReportUC,
	}
}

// Supported:
// - GET  /review-report-decision-notifications
// - GET  /review-report-decision-notifications?isRead=false&page=1&perPage=20
// - POST /review-report-decision-notifications/{notificationId}/read
func (h *ReviewReportDecisionNotificationHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if h == nil || h.ReviewReportUC == nil {
		writeError(
			w,
			http.StatusServiceUnavailable,
			"ReviewReportDecisionNotificationHandlerNotInitialized",
		)
		return
	}

	notificationID, isReadPath, matched := parseReviewReportDecisionNotificationPath(
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

type reviewReportDecisionNotificationResponse struct {
	ID             string                    `json:"id"`
	CaseID         string                    `json:"caseId"`
	ReportID       string                    `json:"reportId"`
	RecipientType  reviewreport.ActorType    `json:"recipientType"`
	RecipientID    string                    `json:"recipientId"`
	CompanyID      string                    `json:"companyId"`
	TargetType     reviewreport.TargetType   `json:"targetType"`
	TargetID       string                    `json:"targetId"`
	TargetParentID string                    `json:"targetParentId"`
	ReportReason   reviewreport.ReportReason `json:"reportReason"`
	ReportDetail   string                    `json:"reportDetail"`
	DecisionStatus reviewreport.CaseStatus   `json:"decisionStatus"`
	DecisionReason string                    `json:"decisionReason"`
	DecidedAt      time.Time                 `json:"decidedAt"`
	CreatedAt      time.Time                 `json:"createdAt"`
	UpdatedAt      time.Time                 `json:"updatedAt"`
	ReadAt         *time.Time                `json:"readAt"`
	IsRead         bool                      `json:"isRead"`
}

// ============================================================
// List
// ============================================================

func (h *ReviewReportDecisionNotificationHandler) list(
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

	isRead, err := parseReviewReportDecisionNotificationIsRead(
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

	page, err := parseReviewReportDecisionNotificationPage(r)
	if err != nil {
		writeError(
			w,
			http.StatusBadRequest,
			"InvalidPagination",
		)
		return
	}

	result, err := h.ReviewReportUC.ListDecisionNotificationsForCompany(
		r.Context(),
		companyID,
		isRead,
		page,
	)
	if err != nil {
		writeReviewReportDecisionNotificationError(w, err)
		return
	}

	items := make(
		[]reviewReportDecisionNotificationResponse,
		0,
		len(result.Items),
	)
	for _, notification := range result.Items {
		items = append(
			items,
			toReviewReportDecisionNotificationResponse(notification),
		)
	}

	writeJSON(
		w,
		http.StatusOK,
		domcommon.PageResult[reviewReportDecisionNotificationResponse]{
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

func (h *ReviewReportDecisionNotificationHandler) markRead(
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

	notification, err := h.ReviewReportUC.MarkDecisionNotificationReadForCompany(
		r.Context(),
		reviewreport.DecisionNotificationID(notificationID),
		companyID,
	)
	if err != nil {
		writeReviewReportDecisionNotificationError(w, err)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		toReviewReportDecisionNotificationResponse(notification),
	)
}

// ============================================================
// Path
// ============================================================

func parseReviewReportDecisionNotificationPath(
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
		parts[0] == "review-report-decision-notifications" {
		return "", false, true
	}

	if len(parts) == 3 &&
		parts[0] == "review-report-decision-notifications" &&
		parts[1] != "" &&
		parts[2] == "read" {
		return parts[1], true, true
	}

	return "", false, false
}

// ============================================================
// Query params
// ============================================================

func parseReviewReportDecisionNotificationIsRead(
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

func parseReviewReportDecisionNotificationPage(
	r *http.Request,
) (domcommon.Page, error) {
	pageNumber := defaultReviewReportDecisionNotificationPage
	perPage := defaultReviewReportDecisionNotificationPerPage

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
			parsed > maxReviewReportDecisionNotificationPerPage {
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

func toReviewReportDecisionNotificationResponse(
	notification reviewreport.DecisionNotification,
) reviewReportDecisionNotificationResponse {
	return reviewReportDecisionNotificationResponse{
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

func writeReviewReportDecisionNotificationError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(
		err,
		uc.ErrReviewReportUsecaseNotConfigured,
	):
		writeError(
			w,
			http.StatusServiceUnavailable,
			"ReviewReportUsecaseNotConfigured",
		)

	case errors.Is(
		err,
		uc.ErrReviewReportForbidden,
	):
		writeError(
			w,
			http.StatusForbidden,
			"ReviewReportForbidden",
		)

	case status.Code(err) == codes.NotFound:
		writeError(
			w,
			http.StatusNotFound,
			"ReviewReportDecisionNotificationNotFound",
		)

	case reviewreport.IsInvalid(err),
		errors.Is(
			err,
			reviewreport.ErrInvalidDecisionNotificationID,
		),
		errors.Is(
			err,
			reviewreport.ErrDecisionNotificationCaseMismatch,
		),
		errors.Is(
			err,
			reviewreport.ErrDecisionNotificationCaseNotDecided,
		),
		errors.Is(
			err,
			reviewreport.ErrInvalidDecisionNotificationCreatedAt,
		),
		errors.Is(
			err,
			reviewreport.ErrInvalidDecisionNotificationUpdatedAt,
		),
		errors.Is(
			err,
			reviewreport.ErrInvalidDecisionNotificationReadAt,
		):
		writeError(
			w,
			http.StatusBadRequest,
			"InvalidReviewReportDecisionNotification",
		)

	default:
		writeError(
			w,
			http.StatusInternalServerError,
			"ReviewReportDecisionNotificationInternalError",
		)
	}
}
