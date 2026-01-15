// backend\internal\adapters\in\http\mall\handler\avatar_me_handler.go
package mallHandler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"narratives/internal/adapters/in/http/middleware"
)

// ✅ DI で差し替えしやすいように interface を最小にする
// 旧: ResolveAvatarIDByUID のみ
// 新: ResolveAvatarByUID (avatarId + walletAddress) を追加
type MeAvatarService interface {
	// ✅ new (preferred)
	ResolveAvatarByUID(ctx context.Context, uid string) (avatarId string, walletAddress string, err error)

	// ✅ legacy fallback (optional)
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

	uid = strings.TrimSpace(uid)

	// ------------------------------------------------------------
	// ✅ preferred: avatarId + walletAddress
	// ------------------------------------------------------------
	avatarId, walletAddress, err := h.Svc.ResolveAvatarByUID(r.Context(), uid)
	if err != nil {
		// fallback to legacy method if implemented and new method failed due to non-implemented wiring etc.
		legacyID, legacyErr := h.Svc.ResolveAvatarIDByUID(r.Context(), uid)
		if legacyErr == nil && strings.TrimSpace(legacyID) != "" {
			_ = json.NewEncoder(w).Encode(map[string]string{
				"avatarId": strings.TrimSpace(legacyID),
				// walletAddress is unknown in legacy path
				"walletAddress": "",
			})
			return
		}

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
	walletAddress = strings.TrimSpace(walletAddress)

	if avatarId == "" {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "avatar_not_found_for_uid"})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]string{
		"avatarId":      avatarId,
		"walletAddress": walletAddress, // 空でもOK（wallet未設定）
	})
}
