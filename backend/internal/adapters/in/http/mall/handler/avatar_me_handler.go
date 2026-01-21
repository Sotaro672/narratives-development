// backend/internal/adapters/in/http/mall/handler/avatar_me_handler.go
package mallHandler

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	avatardom "narratives/internal/domain/avatar"

	"narratives/internal/adapters/in/http/middleware"
)

// MeAvatarService resolves avatar patch by Firebase uid.
// Policy A: walletAddress must be sourced from Avatar document (server-side truth).
//
// Return:
// - avatarId: required (empty => not found)
// - patch: AvatarPatch (contains userId + optional fields)
// - err: not found / internal / ctx canceled etc.
type MeAvatarService interface {
	ResolveAvatarPatchByUID(ctx context.Context, uid string) (avatarId string, patch avatardom.AvatarPatch, err error)
}

type MeAvatarHandler struct {
	Svc MeAvatarService
}

func NewMeAvatarHandler(svc MeAvatarService) http.Handler {
	return &MeAvatarHandler{Svc: svc}
}

func (h *MeAvatarHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// always JSON
	w.Header().Set("Content-Type", "application/json")

	// ---- logging: entrypoint (must show if routing reaches here) ----
	log.Printf(
		"[mall_me_avatar_handler] HIT method=%s path=%s rawPath=%s query=%q origin=%q",
		r.Method,
		r.URL.Path,
		r.URL.EscapedPath(),
		r.URL.RawQuery,
		strings.TrimSpace(r.Header.Get("Origin")),
	)

	// CORS preflight
	if r.Method == http.MethodOptions {
		log.Printf("[mall_me_avatar_handler] OPTIONS -> 204")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// only GET
	if r.Method != http.MethodGet {
		log.Printf("[mall_me_avatar_handler] method_not_allowed method=%s", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "method_not_allowed"})
		return
	}

	if h == nil || h.Svc == nil {
		log.Printf("[mall_me_avatar_handler] service_not_initialized (h_nil=%t svc_nil=%t)", h == nil, h == nil || h.Svc == nil)
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "me_avatar_service_not_initialized"})
		return
	}

	uid, ok := middleware.CurrentUserUID(r)
	uid = strings.TrimSpace(uid)
	if !ok || uid == "" {
		log.Printf("[mall_me_avatar_handler] unauthorized: missing uid (ok=%t uid=%q)", ok, uid)
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized: missing uid"})
		return
	}

	// ---- resolve ----
	log.Printf("[mall_me_avatar_handler] resolve start uid=%q", maskUID(uid))
	avatarId, patch, err := h.Svc.ResolveAvatarPatchByUID(r.Context(), uid)
	if err != nil {
		if isNotFoundLike(err) {
			log.Printf("[mall_me_avatar_handler] resolve not_found uid=%q err=%v", maskUID(uid), err)
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "avatar_not_found_for_uid"})
			return
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			log.Printf("[mall_me_avatar_handler] resolve timeout uid=%q err=%v", maskUID(uid), err)
			w.WriteHeader(http.StatusRequestTimeout)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "request_timeout"})
			return
		}

		log.Printf("[mall_me_avatar_handler] resolve internal_error uid=%q err=%v", maskUID(uid), err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal_error"})
		return
	}

	avatarId = strings.TrimSpace(avatarId)
	if avatarId == "" {
		log.Printf("[mall_me_avatar_handler] resolve empty avatarId uid=%q", maskUID(uid))
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "avatar_not_found_for_uid"})
		return
	}

	// sanitize patch (trim + empty -> nil)
	patch.Sanitize()

	// walletAddress required by frontend
	if patch.WalletAddress == nil || strings.TrimSpace(*patch.WalletAddress) == "" {
		log.Printf("[mall_me_avatar_handler] wallet_not_initialized avatarId=%q uid=%q", avatarId, maskUID(uid))
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "wallet_address_not_initialized"})
		return
	}

	// Response:
	// - avatarId をトップレベルに追加
	// - patch の全フィールドをそのまま返す（= “avatar patch 全体”）
	type meAvatarPatchResponse struct {
		AvatarID      string     `json:"avatarId"`
		UserID        string     `json:"userId"`
		AvatarName    *string    `json:"avatarName,omitempty"`
		AvatarIcon    *string    `json:"avatarIcon,omitempty"`
		WalletAddress *string    `json:"walletAddress,omitempty"`
		Profile       *string    `json:"profile,omitempty"`
		ExternalLink  *string    `json:"externalLink,omitempty"`
		DeletedAt     *time.Time `json:"deletedAt,omitempty"`
	}

	out := meAvatarPatchResponse{
		AvatarID:      avatarId,
		UserID:        strings.TrimSpace(patch.UserID),
		AvatarName:    patch.AvatarName,
		AvatarIcon:    patch.AvatarIcon,
		WalletAddress: patch.WalletAddress,
		Profile:       patch.Profile,
		ExternalLink:  patch.ExternalLink,
		DeletedAt:     patch.DeletedAt,
	}

	log.Printf(
		"[mall_me_avatar_handler] OK avatarId=%q userId=%q avatarName_set=%t avatarIcon_set=%t wallet_set=%t profile_set=%t externalLink_set=%t deletedAt_set=%t",
		out.AvatarID,
		out.UserID,
		out.AvatarName != nil && strings.TrimSpace(*out.AvatarName) != "",
		out.AvatarIcon != nil && strings.TrimSpace(*out.AvatarIcon) != "",
		out.WalletAddress != nil && strings.TrimSpace(*out.WalletAddress) != "",
		out.Profile != nil && strings.TrimSpace(*out.Profile) != "",
		out.ExternalLink != nil && strings.TrimSpace(*out.ExternalLink) != "",
		out.DeletedAt != nil,
	)

	// IMPORTANT: always write body
	_ = json.NewEncoder(w).Encode(out)
}
