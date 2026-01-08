// backend/internal/adapters/in/http/mall/handler/me_avatar_handler.go
package mallHandler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"narratives/internal/adapters/in/http/middleware"
)

type MeAvatarResolver interface {
	ResolveAvatarIDByUID(ctx context.Context, uid string) (string, error)
}

type MeAvatarHandler struct {
	Resolver MeAvatarResolver
}

func NewMeAvatarHandler(resolver MeAvatarResolver) http.Handler {
	return &MeAvatarHandler{Resolver: resolver}
}

func (h *MeAvatarHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	path := strings.TrimSuffix(strings.TrimSpace(r.URL.Path), "/")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

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
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized: missing uid"})
		return
	}

	if h == nil || h.Resolver == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "avatar_resolver_not_initialized"})
		return
	}

	avatarId, err := h.Resolver.ResolveAvatarIDByUID(r.Context(), uid)
	if err != nil || strings.TrimSpace(avatarId) == "" {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "avatar_not_found_for_uid"})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]string{"avatarId": strings.TrimSpace(avatarId)})
}
