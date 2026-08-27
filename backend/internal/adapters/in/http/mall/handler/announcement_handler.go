// backend/internal/adapters/in/http/mall/handler/announcement_handler.go
package mallHandler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"narratives/internal/adapters/in/http/middleware"
	mallquery "narratives/internal/application/query/mall"
	announcementuc "narratives/internal/application/usecase"
	ann "narratives/internal/domain/announcement"
	avatardom "narratives/internal/domain/avatar"
	common "narratives/internal/domain/common"
)

// Policy (me-only):
// - uid は認証コンテキストから取得し、クライアント入力では受けない
// - avatarId はサーバで uid -> avatarId を解決する
// - GET /mall/me/announcement はログイン中 avatarId が targetAvatars に含まれる announcement を返す
// - POST /mall/me/announcement/{announcementId}/read はログイン中 avatarId で既読化する
//
// Endpoints:
// - GET  /mall/me/announcement
// - POST /mall/me/announcement/{announcementId}/read

type AnnouncementMeAvatarResolver interface {
	ResolveAvatarByUID(ctx context.Context, uid string) (avatarID string, walletAddress string, err error)
}

type MeAnnouncementHandler struct {
	Repo              AnnouncementMeAvatarResolver
	AnnouncementUC    *announcementuc.AnnouncementUsecase
	AnnouncementQuery *mallquery.AnnouncementQueryService
}

func NewMeAnnouncementHandler(
	repo AnnouncementMeAvatarResolver,
	announcementUC *announcementuc.AnnouncementUsecase,
	announcementQuery *mallquery.AnnouncementQueryService,
) http.Handler {
	return &MeAnnouncementHandler{
		Repo:              repo,
		AnnouncementUC:    announcementUC,
		AnnouncementQuery: announcementQuery,
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

	pageNumber := parsePositiveIntDefault(r.URL.Query().Get("page"), 1)
	perPage := parsePositiveIntDefault(r.URL.Query().Get("perPage"), 50)

	result, err := h.AnnouncementQuery.ListByTargetAvatar(
		r.Context(),
		avatarID,
		common.Page{
			Number:  pageNumber,
			PerPage: perPage,
		},
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

	result, err := h.AnnouncementUC.MarkRead(r.Context(), announcementID, avatarID)
	if err != nil {
		writeMeAnnouncementErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
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

	return parts[3]
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
