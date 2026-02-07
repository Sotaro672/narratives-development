// backend/internal/adapters/in/http/mall/handler/avatar/avatar_post.go
package avatarHandler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	uc "narratives/internal/application/usecase"
)

func (h *AvatarHandler) post(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var body struct {
		UserID       string  `json:"userId"`
		UserUID      string  `json:"userUid"`
		AvatarName   string  `json:"avatarName"`
		AvatarIcon   *string `json:"avatarIcon,omitempty"`
		Profile      *string `json:"profile,omitempty"`
		ExternalLink *string `json:"externalLink,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	in := uc.CreateAvatarInput{
		UserID:       strings.TrimSpace(body.UserID),
		UserUID:      strings.TrimSpace(body.UserUID),
		AvatarName:   strings.TrimSpace(body.AvatarName),
		AvatarIcon:   trimPtr(body.AvatarIcon),
		Profile:      trimPtr(body.Profile),
		ExternalLink: trimPtr(body.ExternalLink),
	}

	log.Printf(
		"[mall_avatar_handler] POST /mall/avatars request userId=%q userUid=%q avatarName=%q avatarIcon=%q profile_len=%d externalLink=%q\n",
		in.UserID,
		maskUID(in.UserUID),
		in.AvatarName,
		ptrStr(in.AvatarIcon),
		ptrLen(in.Profile),
		ptrStr(in.ExternalLink),
	)

	created, err := h.uc.Create(ctx, in)
	if err != nil {
		log.Printf("[mall_avatar_handler] POST /mall/avatars error=%v\n", err)
		writeAvatarErr(w, err)
		return
	}

	hasAvatarIconField := created.AvatarIcon != nil && strings.TrimSpace(*created.AvatarIcon) != ""
	log.Printf(
		"[mall_avatar_handler] POST /mall/avatars ok avatarId=%q walletAddress=%q avatar.avatarIcon_set=%t avatar.avatarIcon=%q\n",
		created.ID,
		ptrStr(created.WalletAddress),
		hasAvatarIconField,
		ptrStr(created.AvatarIcon),
	)

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(toAvatarResponse(created))
}
