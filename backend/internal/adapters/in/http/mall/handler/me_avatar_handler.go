// backend/internal/adapters/in/http/mall/handler/me_avatar_handler.go
package mallHandler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"narratives/internal/adapters/in/http/middleware"
)

// ✅ DI で差し替えしやすいように interface を最小にする
// Repo 側 (MeAvatarRepo) が実装しているメソッド名に合わせる
type MeAvatarService interface {
	ResolveAvatarIDByUID(ctx context.Context, uid string) (string, error)
}

type MeAvatarHandler struct {
	Svc MeAvatarService
}

func NewMeAvatarHandler(svc MeAvatarService) http.Handler {
	return &MeAvatarHandler{Svc: svc}
}

func (h *MeAvatarHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	path := strings.TrimSuffix(strings.TrimSpace(r.URL.Path), "/")

	// /mall/me/avatar を想定（保険で /mall/avatar も許可）
	if path != "/mall/me/avatar" && path != "/mall/avatar" {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "method_not_allowed"})
		return
	}

	uid, ok := middleware.CurrentUserUID(r)
	if !ok || strings.TrimSpace(uid) == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized: missing uid"})
		return
	}

	if h == nil || h.Svc == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "me_avatar_service_not_initialized"})
		return
	}

	avatarId, err := h.Svc.ResolveAvatarIDByUID(r.Context(), strings.TrimSpace(uid))
	if err != nil {
		// できるだけ 404 に寄せる（循環参照を避けるため、repo の Err を直接 import しない）
		if isNotFoundLike(err) {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "avatar_not_found_for_uid"})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal_error"})
		return
	}

	avatarId = strings.TrimSpace(avatarId)
	if avatarId == "" {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "avatar_not_found_for_uid"})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]string{"avatarId": avatarId})
}
