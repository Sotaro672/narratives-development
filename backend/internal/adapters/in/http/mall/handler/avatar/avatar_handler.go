// backend/internal/adapters/in/http/mall/handler/avatar/avatar_handler.go
package avatarHandler

import (
	"encoding/json"
	"net/http"
	"strings"

	avataruc "narratives/internal/application/usecase/avatar"
)

type AvatarHandler struct {
	uc *avataruc.AvatarUsecase
}

func NewAvatarHandler(avatarUC *avataruc.AvatarUsecase) http.Handler {
	return &AvatarHandler{uc: avatarUC}
}

func (h *AvatarHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	path0 := strings.TrimSuffix(r.URL.Path, "/")

	switch {
	case r.Method == http.MethodGet && path0 == "/mall/avatars":
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_implemented"})
		return

	case r.Method == http.MethodPost && path0 == "/mall/avatars":
		h.post(w, r)
		return

	case r.Method == http.MethodPost && strings.HasPrefix(path0, "/mall/avatars/") && strings.HasSuffix(path0, "/wallet"):
		id, ok := extractIDFromSubroute(path0, "/mall/avatars/", "/wallet")
		if !ok {
			notFound(w)
			return
		}
		h.openWallet(w, r, id)
		return

	case r.Method == http.MethodPost && strings.HasPrefix(path0, "/mall/avatars/") && strings.HasSuffix(path0, "/icon-upload-url"):
		id, ok := extractIDFromSubroute(path0, "/mall/avatars/", "/icon-upload-url")
		if !ok {
			notFound(w)
			return
		}
		h.issueIconUploadURL(w, r, id)
		return

	case r.Method == http.MethodPost && strings.HasPrefix(path0, "/mall/avatars/") && strings.HasSuffix(path0, "/icon"):
		id, ok := extractIDFromSubroute(path0, "/mall/avatars/", "/icon")
		if !ok {
			notFound(w)
			return
		}
		h.replaceIcon(w, r, id)
		return

	case r.Method == http.MethodGet && strings.HasPrefix(path0, "/mall/avatars/"):
		id, ok := extractIDFromPath(path0, "/mall/avatars/")
		if !ok {
			notFound(w)
			return
		}
		h.get(w, r, id)
		return

	case (r.Method == http.MethodPatch || r.Method == http.MethodPut) && strings.HasPrefix(path0, "/mall/avatars/"):
		id, ok := extractIDFromPath(path0, "/mall/avatars/")
		if !ok {
			notFound(w)
			return
		}
		h.update(w, r, id)
		return

	default:
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}
}
