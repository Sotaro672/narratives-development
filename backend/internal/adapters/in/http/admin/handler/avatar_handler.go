// backend/internal/adapters/in/http/admin/handler/avatar_handler.go
package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	avatardom "narratives/internal/domain/avatar"
	userdom "narratives/internal/domain/user"
)

const adminAvatarsPath = "/admin/avatars"

type AvatarListReader interface {
	ListAll(ctx context.Context) ([]avatardom.Avatar, error)
}

type AvatarUserReader interface {
	GetByID(ctx context.Context, id string) (*userdom.User, error)
}

type AvatarReportCountReader interface {
	GetAvatarReportCount(ctx context.Context, avatarID string) (int, error)
}

type AvatarHandler struct {
	avatarRepo AvatarListReader
	userRepo   AvatarUserReader
	reportRepo AvatarReportCountReader
}

type avatarResponse struct {
	ID           string  `json:"id"`
	AvatarName   string  `json:"avatarName"`
	AvatarIcon   *string `json:"avatarIcon,omitempty"`
	Profile      *string `json:"profile,omitempty"`
	ExternalLink *string `json:"externalLink,omitempty"`
	UserName     string  `json:"userName"`
	ReportCount  int     `json:"reportCount"`
	CreatedAt    string  `json:"createdAt"`
	UpdatedAt    string  `json:"updatedAt"`
}

type avatarListResponse struct {
	Items []avatarResponse `json:"items"`
}

func NewAvatarHandler(
	avatarRepo AvatarListReader,
	userRepo AvatarUserReader,
	reportRepo AvatarReportCountReader,
) http.Handler {
	return http.HandlerFunc((&AvatarHandler{
		avatarRepo: avatarRepo,
		userRepo:   userRepo,
		reportRepo: reportRepo,
	}).handle)
}

func (h *AvatarHandler) handle(
	w http.ResponseWriter,
	r *http.Request,
) {
	path := strings.TrimSuffix(r.URL.Path, "/")

	if path != adminAvatarsPath {
		writeJSONError(w, http.StatusNotFound, "avatar_resource_not_found")
		return
	}

	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}

	if h.avatarRepo == nil || h.userRepo == nil || h.reportRepo == nil {
		writeJSONError(
			w,
			http.StatusServiceUnavailable,
			"avatar_dependencies_not_initialized",
		)
		return
	}

	avatars, err := h.avatarRepo.ListAll(r.Context())
	if err != nil {
		writeJSONError(
			w,
			http.StatusInternalServerError,
			"avatar_list_failed",
		)
		return
	}

	items := make([]avatarResponse, 0, len(avatars))
	for _, avatar := range avatars {
		userName, err := h.resolveUserName(r.Context(), avatar.UserID)
		if err != nil {
			writeJSONError(
				w,
				http.StatusInternalServerError,
				"avatar_user_resolve_failed",
			)
			return
		}

		reportCount, err := h.reportRepo.GetAvatarReportCount(
			r.Context(),
			avatar.ID,
		)
		if err != nil {
			writeJSONError(
				w,
				http.StatusInternalServerError,
				"avatar_report_count_failed",
			)
			return
		}

		items = append(items, avatarResponse{
			ID:           avatar.ID,
			AvatarName:   avatar.AvatarName,
			AvatarIcon:   avatar.AvatarIcon,
			Profile:      avatar.Profile,
			ExternalLink: avatar.ExternalLink,
			UserName:     userName,
			ReportCount:  reportCount,
			CreatedAt:    avatar.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:    avatar.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	writeJSON(
		w,
		http.StatusOK,
		avatarListResponse{Items: items},
	)
}

func (h *AvatarHandler) resolveUserName(
	ctx context.Context,
	userID string,
) (string, error) {
	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, userdom.ErrNotFound) {
			return "", nil
		}
		return "", err
	}

	return userdom.FormatName(user), nil
}
