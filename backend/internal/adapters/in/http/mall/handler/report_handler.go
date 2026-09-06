// backend\internal\adapters\in\http\mall\handler\report_handler.go
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
	reportdom "narratives/internal/domain/report"
)

const (
	defaultReportDecisionNotificationPage    = 1
	defaultReportDecisionNotificationPerPage = 20
	maxReportDecisionNotificationPerPage     = 100
)

// HTTPパスは既存クライアントとの互換性を維持するため変更しない。
const meReportDecisionNotificationsPath = "/mall/me/review-report-decision-notifications"

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
// - GET  /mall/me/review-report-decision-notifications
// - GET  /mall/me/review-report-decision-notifications?isRead=false&page=1&perPage=20
// - POST /mall/me/review-report-decision-notifications/{notificationId}/read
//
// Security:
// - avatarId はクライアント入力から受け取らない。
// - UserAuthMiddleware + AvatarContextMiddleware が解決した current avatarId を利用する。
// - Usecase 側でも通知の RecipientType / RecipientID を再検証する。
// - Admin の内部識別子 DecidedBy はレスポンスへ公開しない。
func (h *ReportDecisionNotificationHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if h == nil || h.ReportUC == nil {
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
		parseMallReportDecisionNotificationPath(r.URL.Path)
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

type reportDecisionNotificationResponse struct {
	ID               string                     `json:"id"`
	NotificationKind reportdom.NotificationKind `json:"notificationKind"`
	CaseID           string                     `json:"caseId"`
	ReportID         string                     `json:"reportId"`
	RecipientType    reportdom.ActorType        `json:"recipientType"`
	RecipientID      string                     `json:"recipientId"`
	CompanyID        string                     `json:"companyId"`
	TargetType       reportdom.TargetType       `json:"targetType"`
	TargetID         string                     `json:"targetId"`
	TargetParentID   string                     `json:"targetParentId"`
	ReportReason     reportdom.ReportReason     `json:"reportReason"`
	ReportDetail     string                     `json:"reportDetail"`
	DecisionStatus   reportdom.CaseStatus       `json:"decisionStatus"`
	DecisionReason   string                     `json:"decisionReason"`
	DecidedAt        time.Time                  `json:"decidedAt"`
	CreatedAt        time.Time                  `json:"createdAt"`
	UpdatedAt        time.Time                  `json:"updatedAt"`
	ReadAt           *time.Time                 `json:"readAt"`
	IsRead           bool                       `json:"isRead"`
}

// ============================================================
// List
// ============================================================

func (h *ReportDecisionNotificationHandler) list(
	w http.ResponseWriter,
	r *http.Request,
	avatarID string,
) {
	isRead, err := parseMallReportDecisionNotificationIsRead(
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

	page, err := parseMallReportDecisionNotificationPage(r)
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

	result, err := h.ReportUC.ListDecisionNotificationsForAvatar(
		r.Context(),
		avatarID,
		isRead,
		page,
	)
	if err != nil {
		writeMallReportDecisionNotificationError(w, err)
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
			toMallReportDecisionNotificationResponse(notification),
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
		h.ReportUC.MarkDecisionNotificationReadForAvatar(
			r.Context(),
			reportdom.DecisionNotificationID(notificationID),
			avatarID,
		)
	if err != nil {
		writeMallReportDecisionNotificationError(w, err)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		toMallReportDecisionNotificationResponse(notification),
	)
}

// ============================================================
// Path
// ============================================================

func parseMallReportDecisionNotificationPath(
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

	// 既存HTTPパスとの互換性を維持する。
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

func parseMallReportDecisionNotificationIsRead(
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

func parseMallReportDecisionNotificationPage(
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

func toMallReportDecisionNotificationResponse(
	notification reportdom.DecisionNotification,
) reportDecisionNotificationResponse {
	return reportDecisionNotificationResponse{
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

func writeMallReportDecisionNotificationError(
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
		uc.ErrReportUsecaseNotConfigured,
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
		uc.ErrReportForbidden,
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
