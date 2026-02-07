// backend/internal/adapters/in/http/mall/handler/avatar/avatar_update.go
package avatarHandler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	avatardom "narratives/internal/domain/avatar"
)

func (h *AvatarHandler) update(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	_ = ctx

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	var raw map[string]any
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	if _, ok := raw["walletAddress"]; ok {
		log.Printf("[mall_avatar_handler] PATCH/PUT /mall/avatars/%s rejected: walletAddress field is not allowed\n", id)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "walletAddress is not allowed in update"})
		return
	}

	bs, merr := json.Marshal(raw)
	if merr != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}
	log.Printf("[mall_avatar_handler] PATCH/PUT /mall/avatars/%s raw=%q\n", id, headString(bs, 300))

	var body struct {
		AvatarName   *string `json:"avatarName,omitempty"`
		AvatarIcon   *string `json:"avatarIcon,omitempty"`
		Profile      *string `json:"profile,omitempty"`
		ExternalLink *string `json:"externalLink,omitempty"`
	}
	if err := json.Unmarshal(bs, &body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	patch := avatardom.AvatarPatch{
		AvatarName:   trimPtrNilAware(body.AvatarName),
		AvatarIcon:   trimPtrNilAware(body.AvatarIcon),
		Profile:      trimPtrNilAware(body.Profile),
		ExternalLink: trimPtrNilAware(body.ExternalLink),
	}

	log.Printf(
		"[mall_avatar_handler] PATCH/PUT /mall/avatars/%s request avatarName=%q avatarIcon=%q profile_len=%d externalLink=%q\n",
		id,
		ptrStr(patch.AvatarName),
		ptrStr(patch.AvatarIcon),
		ptrLen(patch.Profile),
		ptrStr(patch.ExternalLink),
	)

	updated, err := h.uc.Update(r.Context(), id, patch)
	if err != nil {
		log.Printf("[mall_avatar_handler] PATCH/PUT /mall/avatars/%s error=%v\n", id, err)
		writeAvatarErr(w, err)
		return
	}

	hasAvatarIconField := updated.AvatarIcon != nil && strings.TrimSpace(*updated.AvatarIcon) != ""
	log.Printf(
		"[mall_avatar_handler] PATCH/PUT /mall/avatars/%s ok avatar.avatarIcon_set=%t avatar.avatarIcon=%q\n",
		id,
		hasAvatarIconField,
		ptrStr(updated.AvatarIcon),
	)

	_ = json.NewEncoder(w).Encode(toAvatarResponse(updated))
}
