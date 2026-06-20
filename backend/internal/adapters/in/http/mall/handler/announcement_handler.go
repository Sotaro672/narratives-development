// backend/internal/adapters/in/http/mall/handler/announcement_handler.go
package mallHandler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
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

const (
	meAnnouncementsPath = "/mall/me/announcement"
)

func (h *MeAnnouncementHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if h == nil || h.Repo == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "me_announcement_handler_not_initialized"})
		return
	}

	if h.AnnouncementUC == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "announcement_usecase_not_configured"})
		return
	}

	if h.AnnouncementQuery == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "announcement_query_not_configured"})
		return
	}

	uid, ok := middleware.CurrentUserUID(r)
	if !ok || uid == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized: missing uid"})
		return
	}

	path0 := strings.TrimSuffix(r.URL.Path, "/")

	switch {
	case r.Method == http.MethodGet && path0 == meAnnouncementsPath:
		h.handleList(w, r, uid)
		return

	case r.Method == http.MethodPost && strings.HasPrefix(path0, meAnnouncementsPath+"/") && strings.HasSuffix(path0, "/read"):
		h.handleMarkRead(w, r, uid, path0)
		return

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}
}

func (h *MeAnnouncementHandler) handleList(w http.ResponseWriter, r *http.Request, uid string) {
	avatarID, err := h.resolveAvatarID(r.Context(), uid)
	if err != nil {
		writeMeAnnouncementErr(w, err)
		return
	}

	pageNumber := parseMeAnnouncementPositiveInt(r.URL.Query().Get("page"), 1)
	perPage := parseMeAnnouncementPositiveInt(r.URL.Query().Get("perPage"), 50)

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

	_ = json.NewEncoder(w).Encode(result)
}

func (h *MeAnnouncementHandler) handleMarkRead(w http.ResponseWriter, r *http.Request, uid string, path0 string) {
	announcementID := extractAnnouncementIDForRead(path0)
	if announcementID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "announcementId is required"})
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

	_ = json.NewEncoder(w).Encode(result)
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
	if parts[0] != "mall" || parts[1] != "me" || parts[2] != "announcement" || parts[4] != "read" {
		return ""
	}

	return parts[3]
}

func parseMeAnnouncementPositiveInt(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}

	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return fallback
	}

	return n
}

func writeMeAnnouncementErr(w http.ResponseWriter, err error) {
	if err == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal_error"})
		return
	}

	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		w.WriteHeader(http.StatusRequestTimeout)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "request_timeout"})
		return

	case errors.Is(err, ann.ErrNotFound):
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return

	case errors.Is(err, avatardom.ErrInvalidID):
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "avatar_not_found_for_uid"})
		return

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
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return

	case isNotFoundLike(err):
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return

	default:
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
}
