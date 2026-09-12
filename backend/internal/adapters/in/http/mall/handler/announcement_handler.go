// backend/internal/adapters/in/http/mall/handler/announcement_handler.go
package mallHandler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"narratives/internal/adapters/in/http/middleware"
	mallquery "narratives/internal/application/query/mall"
	announcementuc "narratives/internal/application/usecase"
	ann "narratives/internal/domain/announcement"
	avatardom "narratives/internal/domain/avatar"
	reportdom "narratives/internal/domain/report"
)

// Policy (me-only):
// - uid は認証コンテキストから取得し、クライアント入力では受けない
// - avatarId はサーバで uid -> avatarId を解決する
// - GET /mall/me/announcement はログイン中 avatarId が targetAvatars に含まれる announcement を返す
// - POST /mall/me/announcement/{announcementId}/read はログイン中 avatarId で既読化する
// - POST /mall/me/announcement/{announcementId}/reports はログイン中 avatarId から announcement を通報する
//
// Endpoints:
// - GET  /mall/me/announcement
// - POST /mall/me/announcement/{announcementId}/read
// - POST /mall/me/announcement/{announcementId}/reports

type AnnouncementMeAvatarResolver interface {
	ResolveAvatarByUID(ctx context.Context, uid string) (avatarID string, walletAddress string, err error)
}

type MeAnnouncementHandler struct {
	Repo              AnnouncementMeAvatarResolver
	AnnouncementUC    *announcementuc.AnnouncementUsecase
	AnnouncementQuery *mallquery.AnnouncementQueryService
	ReportUC          *announcementuc.ReportUsecase
}

type meAnnouncementReportRequest struct {
	Reason string `json:"reason"`
	Detail string `json:"detail"`
}

type meAnnouncementReportResponse struct {
	CaseID        string               `json:"caseId"`
	ReportID      string               `json:"reportId"`
	ReportCount   int                  `json:"reportCount"`
	Status        reportdom.CaseStatus `json:"status"`
	CaseCreated   bool                 `json:"caseCreated"`
	ReportCreated bool                 `json:"reportCreated"`
}

func NewMeAnnouncementHandler(
	repo AnnouncementMeAvatarResolver,
	announcementUC *announcementuc.AnnouncementUsecase,
	announcementQuery *mallquery.AnnouncementQueryService,
	reportUC *announcementuc.ReportUsecase,
) http.Handler {
	return &MeAnnouncementHandler{
		Repo:              repo,
		AnnouncementUC:    announcementUC,
		AnnouncementQuery: announcementQuery,
		ReportUC:          reportUC,
	}
}

const meAnnouncementsPath = "/mall/me/announcement"

func (h *MeAnnouncementHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if h == nil || h.Repo == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "me_announcement_handler_not_initialized",
		})
		return
	}

	if h.AnnouncementUC == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "announcement_usecase_not_configured",
		})
		return
	}

	if h.AnnouncementQuery == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "announcement_query_not_configured",
		})
		return
	}

	uid, ok := middleware.CurrentUserUID(r)
	if !ok || uid == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "unauthorized: missing uid",
		})
		return
	}

	path0 := strings.TrimSuffix(r.URL.Path, "/")

	switch {
	case r.Method == http.MethodGet && path0 == meAnnouncementsPath:
		h.handleList(w, r, uid)
		return

	case r.Method == http.MethodPost &&
		strings.HasPrefix(path0, meAnnouncementsPath+"/") &&
		strings.HasSuffix(path0, "/read"):
		h.handleMarkRead(w, r, uid, path0)
		return

	case r.Method == http.MethodPost &&
		strings.HasPrefix(path0, meAnnouncementsPath+"/") &&
		strings.HasSuffix(path0, "/reports"):
		h.handleReport(w, r, uid, path0)
		return

	default:
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "not_found",
		})
		return
	}
}

func (h *MeAnnouncementHandler) handleList(
	w http.ResponseWriter,
	r *http.Request,
	uid string,
) {
	avatarID, err := h.resolveAvatarID(r.Context(), uid)
	if err != nil {
		writeMeAnnouncementErr(w, err)
		return
	}

	page := parsePageFromQuery(r, 50, 100)

	result, err := h.AnnouncementQuery.ListByTargetAvatar(
		r.Context(),
		avatarID,
		page,
	)
	if err != nil {
		writeMeAnnouncementErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *MeAnnouncementHandler) handleMarkRead(
	w http.ResponseWriter,
	r *http.Request,
	uid string,
	path0 string,
) {
	announcementID := extractAnnouncementIDForRead(path0)
	if announcementID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "announcementId is required",
		})
		return
	}

	avatarID, err := h.resolveAvatarID(r.Context(), uid)
	if err != nil {
		writeMeAnnouncementErr(w, err)
		return
	}

	result, err := h.AnnouncementUC.MarkRead(
		r.Context(),
		announcementID,
		avatarID,
	)
	if err != nil {
		writeMeAnnouncementErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *MeAnnouncementHandler) handleReport(
	w http.ResponseWriter,
	r *http.Request,
	uid string,
	path0 string,
) {
	if h == nil || h.ReportUC == nil {
		writeJSONError(
			w,
			http.StatusServiceUnavailable,
			"report service not configured",
		)
		return
	}

	announcementID := extractAnnouncementIDForReport(path0)
	if announcementID == "" {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"announcementId is required",
		)
		return
	}

	avatarID, err := h.resolveAvatarID(r.Context(), uid)
	if err != nil {
		writeMeAnnouncementErr(w, err)
		return
	}
	if avatarID == "" {
		writeJSONError(
			w,
			http.StatusUnauthorized,
			"missing avatarId",
		)
		return
	}

	var req meAnnouncementReportRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"invalid json body",
		)
		return
	}

	reason := reportdom.ReportReason(
		strings.ToUpper(strings.TrimSpace(req.Reason)),
	)
	if err := reason.Validate(); err != nil {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"invalid report reason",
		)
		return
	}

	req.Detail = strings.TrimSpace(req.Detail)
	if reason == reportdom.ReportReasonOther && req.Detail == "" {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"report detail required",
		)
		return
	}

	result, err := h.ReportUC.ReportAnnouncementByAvatar(
		r.Context(),
		announcementuc.ReportAnnouncementByAvatarInput{
			AnnouncementID: announcementID,
			AvatarID:       avatarID,
			Reason:         reason,
			Detail:         req.Detail,
		},
	)
	if err != nil {
		writeMeAnnouncementReportErr(w, err)
		return
	}

	statusCode := http.StatusCreated
	if !result.ReportCreated {
		statusCode = http.StatusOK
	}

	writeJSON(
		w,
		statusCode,
		meAnnouncementReportResponse{
			CaseID:        string(result.Case.ID),
			ReportID:      string(result.Report.ID),
			ReportCount:   result.Case.ReportCount,
			Status:        result.Case.Status,
			CaseCreated:   result.CaseCreated,
			ReportCreated: result.ReportCreated,
		},
	)
}

func (h *MeAnnouncementHandler) resolveAvatarID(
	ctx context.Context,
	uid string,
) (string, error) {
	if h == nil || h.Repo == nil {
		return "", errors.New("me announcement handler not configured")
	}

	avatarID, _, err := h.Repo.ResolveAvatarByUID(ctx, uid)
	if err != nil {
		return "", err
	}

	if avatarID == "" {
		return "", avatardom.ErrInvalidID
	}

	return avatarID, nil
}

func extractAnnouncementIDForRead(path0 string) string {
	trimmed := strings.Trim(path0, "/")
	parts := strings.Split(trimmed, "/")

	// Expected:
	// mall / me / announcement / {announcementId} / read
	if len(parts) != 5 {
		return ""
	}

	if parts[0] != "mall" ||
		parts[1] != "me" ||
		parts[2] != "announcement" ||
		parts[4] != "read" {
		return ""
	}

	return strings.TrimSpace(parts[3])
}

func extractAnnouncementIDForReport(path0 string) string {
	trimmed := strings.Trim(path0, "/")
	parts := strings.Split(trimmed, "/")

	// Expected:
	// mall / me / announcement / {announcementId} / reports
	if len(parts) != 5 {
		return ""
	}

	if parts[0] != "mall" ||
		parts[1] != "me" ||
		parts[2] != "announcement" ||
		parts[4] != "reports" {
		return ""
	}

	return strings.TrimSpace(parts[3])
}

func writeMeAnnouncementErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError
	message := "internal_error"

	switch {
	case err == nil:
		// Default response values are used.

	case errors.Is(err, context.Canceled),
		errors.Is(err, context.DeadlineExceeded):
		code = http.StatusRequestTimeout
		message = "request_timeout"

	case errors.Is(err, ann.ErrNotFound):
		code = http.StatusNotFound
		message = err.Error()

	case errors.Is(err, avatardom.ErrInvalidID):
		code = http.StatusNotFound
		message = "avatar_not_found_for_uid"

	case errors.Is(err, ann.ErrInvalidID),
		errors.Is(err, ann.ErrInvalidTitle),
		errors.Is(err, ann.ErrInvalidContent),
		errors.Is(err, ann.ErrInvalidCreatedBy),
		errors.Is(err, ann.ErrInvalidCreatedAt),
		errors.Is(err, ann.ErrInvalidUpdatedAt),
		errors.Is(err, ann.ErrInvalidPublishedAt),
		errors.Is(err, ann.ErrInvalidAvatarID),
		errors.Is(err, ann.ErrInvalidReadAt),
		errors.Is(err, ann.ErrInvalidAnnouncementID),
		errors.Is(err, ann.ErrInvalidFileName),
		errors.Is(err, ann.ErrInvalidFileURL),
		errors.Is(err, ann.ErrInvalidFileSize),
		errors.Is(err, ann.ErrInvalidMimeType),
		errors.Is(err, ann.ErrInvalidObjectPath):
		code = http.StatusBadRequest
		message = err.Error()

	case isNotFound(err):
		code = http.StatusNotFound
		message = err.Error()

	default:
		message = err.Error()
	}

	writeJSON(w, code, map[string]string{
		"error": message,
	})
}

func writeMeAnnouncementReportErr(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case err == nil:
		writeJSONError(
			w,
			http.StatusInternalServerError,
			"unknown error",
		)

	case errors.Is(err, context.Canceled),
		errors.Is(err, context.DeadlineExceeded):
		writeJSONError(
			w,
			http.StatusRequestTimeout,
			"request timeout",
		)

	case errors.Is(
		err,
		announcementuc.ErrReportUsecaseNotConfigured,
	):
		writeJSONError(
			w,
			http.StatusServiceUnavailable,
			"report service not configured",
		)

	case errors.Is(
		err,
		announcementuc.ErrReportForbidden,
	):
		writeJSONError(
			w,
			http.StatusForbidden,
			"report forbidden",
		)

	case errors.Is(
		err,
		reportdom.ErrCannotReportRemovedTarget,
	):
		writeJSONError(
			w,
			http.StatusConflict,
			"cannot report removed target",
		)

	case errors.Is(err, ann.ErrNotFound),
		isNotFound(err):
		writeJSONError(
			w,
			http.StatusNotFound,
			err.Error(),
		)

	case reportdom.IsInvalid(err):
		writeJSONError(
			w,
			http.StatusBadRequest,
			err.Error(),
		)

	default:
		writeJSONError(
			w,
			http.StatusInternalServerError,
			err.Error(),
		)
	}
}
