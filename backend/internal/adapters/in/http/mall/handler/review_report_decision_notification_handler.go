// backend/internal/adapters/in/http/mall/handler/review_report_decision_notification_handler.go
package mallHandler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"narratives/internal/adapters/in/http/middleware"
	uc "narratives/internal/application/usecase"
	domcommon "narratives/internal/domain/common"
	reviewreport "narratives/internal/domain/reviewReport"
)

const (
	defaultReviewReportDecisionNotificationPage    = 1
	defaultReviewReportDecisionNotificationPerPage = 20
	maxReviewReportDecisionNotificationPerPage     = 100
)

const meReviewReportDecisionNotificationsPath = "/mall/me/review-report-decision-notifications"

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
// - GET  /mall/me/review-report-decision-notifications
// - GET  /mall/me/review-report-decision-notifications?isRead=false&page=1&perPage=20
// - POST /mall/me/review-report-decision-notifications/{notificationId}/read
//
// Security:
// - avatarId はクライアント入力から受け取らない。
// - UserAuthMiddleware + AvatarContextMiddleware が解決した current avatarId を利用する。
// - Usecase 側でも通知の RecipientType / RecipientID を再検証する。
// - Admin の内部識別子 DecidedBy はレスポンスへ公開しない。
func (h *ReviewReportDecisionNotificationHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if h == nil || h.ReviewReportUC == nil {
		writeJSON(
			w,
			http.StatusServiceUnavailable,
			map[string]string{
				"error": "review_report_decision_notification_handler_not_initialized",
			},
		)
		return
	}

	notificationID, isReadPath, matched :=
		parseMallReviewReportDecisionNotificationPath(r.URL.Path)
	if !matched {
		writeJSON(
			w,
			http.StatusNotFound,
			map[string]string{
				"error": "not_found",
			},
		)
		return
	}

	avatarID, ok := middleware.CurrentAvatarID(r)
	if !ok || avatarID == "" {
		writeJSON(
			w,
			http.StatusUnauthorized,
			map[string]string{
				"error": "avatar_context_required",
			},
		)
		return
	}

	if isReadPath {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeJSON(
				w,
				http.StatusMethodNotAllowed,
				map[string]string{
					"error": "method_not_allowed",
				},
			)
			return
		}

		h.markRead(
			w,
			r,
			avatarID,
			notificationID,
		)
		return
	}

	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSON(
			w,
			http.StatusMethodNotAllowed,
			map[string]string{
				"error": "method_not_allowed",
			},
		)
		return
	}

	h.list(
		w,
		r,
		avatarID,
	)
}

// ============================================================
// DTO
// ============================================================

type reviewReportDecisionNotificationResponse struct {
	ID               string                        `json:"id"`
	NotificationKind reviewreport.NotificationKind `json:"notificationKind"`
	CaseID           string                        `json:"caseId"`
	ReportID         string                        `json:"reportId"`
	RecipientType    reviewreport.ActorType        `json:"recipientType"`
	RecipientID      string                        `json:"recipientId"`
	CompanyID        string                        `json:"companyId"`
	TargetType       reviewreport.TargetType       `json:"targetType"`
	TargetID         string                        `json:"targetId"`
	TargetParentID   string                        `json:"targetParentId"`
	ReportReason     reviewreport.ReportReason     `json:"reportReason"`
	ReportDetail     string                        `json:"reportDetail"`
	DecisionStatus   reviewreport.CaseStatus       `json:"decisionStatus"`
	DecisionReason   string                        `json:"decisionReason"`
	DecidedAt        time.Time                     `json:"decidedAt"`
	CreatedAt        time.Time                     `json:"createdAt"`
	UpdatedAt        time.Time                     `json:"updatedAt"`
	ReadAt           *time.Time                    `json:"readAt"`
	IsRead           bool                          `json:"isRead"`
}

// ============================================================
// List
// ============================================================

func (h *ReviewReportDecisionNotificationHandler) list(
	w http.ResponseWriter,
	r *http.Request,
	avatarID string,
) {
	isRead, err := parseMallReviewReportDecisionNotificationIsRead(
		r.URL.Query().Get("isRead"),
	)
	if err != nil {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "invalid_is_read",
			},
		)
		return
	}

	page, err := parseMallReviewReportDecisionNotificationPage(r)
	if err != nil {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "invalid_pagination",
			},
		)
		return
	}

	result, err := h.ReviewReportUC.ListDecisionNotificationsForAvatar(
		r.Context(),
		avatarID,
		isRead,
		page,
	)
	if err != nil {
		writeMallReviewReportDecisionNotificationError(w, err)
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
			toMallReviewReportDecisionNotificationResponse(notification),
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
	avatarID string,
	notificationID string,
) {
	if notificationID == "" {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "notification_id_required",
			},
		)
		return
	}

	notification, err :=
		h.ReviewReportUC.MarkDecisionNotificationReadForAvatar(
			r.Context(),
			reviewreport.DecisionNotificationID(notificationID),
			avatarID,
		)
	if err != nil {
		writeMallReviewReportDecisionNotificationError(w, err)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		toMallReviewReportDecisionNotificationResponse(notification),
	)
}

// ============================================================
// Path
// ============================================================

func parseMallReviewReportDecisionNotificationPath(
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

	// mall / me / review-report-decision-notifications
	if len(parts) == 3 &&
		parts[0] == "mall" &&
		parts[1] == "me" &&
		parts[2] == "review-report-decision-notifications" {
		return "", false, true
	}

	// mall / me / review-report-decision-notifications / {notificationId} / read
	if len(parts) == 5 &&
		parts[0] == "mall" &&
		parts[1] == "me" &&
		parts[2] == "review-report-decision-notifications" &&
		parts[3] != "" &&
		parts[4] == "read" {
		return parts[3], true, true
	}

	return "", false, false
}

// ============================================================
// Query params
// ============================================================

func parseMallReviewReportDecisionNotificationIsRead(
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

func parseMallReviewReportDecisionNotificationPage(
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

func toMallReviewReportDecisionNotificationResponse(
	notification reviewreport.DecisionNotification,
) reviewReportDecisionNotificationResponse {
	return reviewReportDecisionNotificationResponse{
		ID:               string(notification.ID),
		NotificationKind: notification.Kind(),
		CaseID:           string(notification.CaseID),
		ReportID:         string(notification.ReportID),
		RecipientType:    notification.RecipientType,
		RecipientID:      notification.RecipientID,
		CompanyID:        notification.CompanyID,
		TargetType:       notification.TargetType,
		TargetID:         notification.TargetID,
		TargetParentID:   notification.TargetParentID,
		ReportReason:     notification.ReportReason,
		ReportDetail:     notification.ReportDetail,
		DecisionStatus:   notification.DecisionStatus,
		DecisionReason:   notification.DecisionReason,
		DecidedAt:        notification.DecidedAt,
		CreatedAt:        notification.CreatedAt,
		UpdatedAt:        notification.UpdatedAt,
		ReadAt:           notification.ReadAt,
		IsRead:           notification.IsRead(),
	}
}

// ============================================================
// Error mapping
// ============================================================

func writeMallReviewReportDecisionNotificationError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case err == nil:
		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{
				"error": "review_report_decision_notification_internal_error",
			},
		)

	case errors.Is(err, context.Canceled),
		errors.Is(err, context.DeadlineExceeded):
		writeJSON(
			w,
			http.StatusRequestTimeout,
			map[string]string{
				"error": "request_timeout",
			},
		)

	case errors.Is(
		err,
		uc.ErrReviewReportUsecaseNotConfigured,
	):
		writeJSON(
			w,
			http.StatusServiceUnavailable,
			map[string]string{
				"error": "review_report_usecase_not_configured",
			},
		)

	case errors.Is(
		err,
		uc.ErrReviewReportForbidden,
	):
		writeJSON(
			w,
			http.StatusForbidden,
			map[string]string{
				"error": "review_report_forbidden",
			},
		)

	case status.Code(err) == codes.NotFound:
		writeJSON(
			w,
			http.StatusNotFound,
			map[string]string{
				"error": "review_report_decision_notification_not_found",
			},
		)

	case reviewreport.IsInvalid(err),
		errors.Is(
			err,
			reviewreport.ErrInvalidDecisionNotificationID,
		),
		errors.Is(
			err,
			reviewreport.ErrInvalidDecisionNotificationKind,
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
		),
		errors.Is(
			err,
			reviewreport.ErrTargetDecisionNotificationReportData,
		):
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "invalid_review_report_decision_notification",
			},
		)

	case isNotFound(err):
		writeJSON(
			w,
			http.StatusNotFound,
			map[string]string{
				"error": "review_report_decision_notification_not_found",
			},
		)

	default:
		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{
				"error": "review_report_decision_notification_internal_error",
			},
		)
	}
}
