// backend/internal/adapters/in/http/admin/handler/avatar_handler.go
package handler

import (
	"context"
	"net/http"
	"strings"

	avatardom "narratives/internal/domain/avatar"
)

const adminAvatarsPath = "/admin/avatars"

type AvatarListReader interface {
	ListAll(ctx context.Context) ([]avatardom.Avatar, error)
}

type AvatarHandler struct {
	avatarRepo AvatarListReader
}

type avatarListResponse struct {
	Items []avatardom.Avatar `json:"items"`
}

func NewAvatarHandler(
	avatarRepo AvatarListReader,
) http.Handler {
	return http.HandlerFunc((&AvatarHandler{
		avatarRepo: avatarRepo,
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

	if h.avatarRepo == nil {
		writeJSONError(
			w,
			http.StatusServiceUnavailable,
			"avatar_repository_not_initialized",
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

	writeJSON(
		w,
		http.StatusOK,
		avatarListResponse{Items: avatars},
	)
}
