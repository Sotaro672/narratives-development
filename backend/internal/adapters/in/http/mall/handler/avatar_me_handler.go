// backend/internal/adapters/in/http/mall/handler/avatar_me_handler.go
package mallHandler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"narratives/internal/adapters/in/http/middleware"
)

// MeAvatarService resolves (avatarId, walletAddress) by Firebase uid.
// Policy A: walletAddress must be sourced from Avatar document (server-side truth).
type MeAvatarService interface {
	ResolveAvatarByUID(ctx context.Context, uid string) (avatarId string, walletAddress string, err error)
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
	if path != "/mall/me/avatar" {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_found"})
		return
	}

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "method_not_allowed"})
		return
	}

	if h == nil || h.Svc == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "me_avatar_service_not_initialized"})
		return
	}

	uid, ok := middleware.CurrentUserUID(r)
	uid = strings.TrimSpace(uid)
	if !ok || uid == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized: missing uid"})
		return
	}

	avatarId, walletAddress, err := h.Svc.ResolveAvatarByUID(r.Context(), uid)
	if err != nil {
		if isNotFoundLike(err) {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "avatar_not_found_for_uid"})
			return
		}
		// ctx canceled etc.
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			w.WriteHeader(http.StatusRequestTimeout)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "request_timeout"})
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal_error"})
		return
	}

	avatarId = strings.TrimSpace(avatarId)
	walletAddress = strings.TrimSpace(walletAddress)

	if avatarId == "" {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "avatar_not_found_for_uid"})
		return
	}

	// ✅ New contract: walletAddress is required by frontend.
	// If avatar exists but wallet not initialized yet, return 409 (client should guide user to setup).
	if walletAddress == "" {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "wallet_address_not_initialized"})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]string{
		"avatarId":      avatarId,
		"walletAddress": walletAddress,
	})
}
