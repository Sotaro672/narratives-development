// backend/internal/adapters/in/http/mall/handler/avatar/avatar_get.go
package avatarHandler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	avatardom "narratives/internal/domain/avatar"
)

func (h *AvatarHandler) get(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	log.Printf("[mall_avatar_handler] GET /mall/avatars/%s aggregate=%q\n", id, r.URL.Query().Get("aggregate"))

	q := r.URL.Query()
	agg := strings.EqualFold(q.Get("aggregate"), "1") || strings.EqualFold(q.Get("aggregate"), "true")

	if agg {
		data, err := h.uc.GetAggregate(ctx, id)
		if err != nil {
			writeAvatarErr(w, err)
			return
		}

		icons := make([]avatarIconResponse, 0, len(data.Icons))
		for _, ic := range data.Icons {
			icons = append(icons, toAvatarIconResponse(ic, id))
		}

		out := avatarAggregateResponse{
			Avatar: toAvatarResponse(data.Avatar),
			State:  data.State,
			Icons:  icons,
		}
		_ = json.NewEncoder(w).Encode(out)
		return
	}

	avatar, err := h.uc.GetByID(ctx, id)
	if err != nil {
		writeAvatarErr(w, err)
		return
	}

	// keep log behavior
	hasAvatarIconField := avatar.AvatarIcon != nil && strings.TrimSpace(*avatar.AvatarIcon) != ""
	log.Printf(
		"[mall_avatar_handler] GET /mall/avatars/%s ok avatar.avatarIcon_set=%t avatar.avatarIcon=%q\n",
		id,
		hasAvatarIconField,
		ptrStr(avatar.AvatarIcon),
	)

	_ = json.NewEncoder(w).Encode(toAvatarResponse(avatar))
}

// compile guard for unused import in some edits
var _ = avatardom.Avatar{}
