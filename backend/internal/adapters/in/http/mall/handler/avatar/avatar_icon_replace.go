// backend/internal/adapters/in/http/mall/handler/avatar/avatar_icon_replace.go
package avatarHandler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	avataruc "narratives/internal/application/usecase/avatar"
	avataricon "narratives/internal/domain/avatarIcon"
)

// -----------------------------------------------------------------------------
// POST /mall/avatars/{id}/icon
// -----------------------------------------------------------------------------
//
// Handler responsibilities (after cleanup):
//   - Parse/validate HTTP request body
//   - Compatibility: accept gs://... in avatarIcon and normalize to bucket/objectPath
//   - Call usecase.ReplaceAvatarIcon (which is responsible for persisting icon metadata,
//     deleting old objects (best-effort), and patching avatars.avatarIcon (best-effort)
//   - Return response
func (h *AvatarHandler) replaceIcon(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	var body struct {
		Bucket     *string `json:"bucket,omitempty"`
		ObjectPath *string `json:"objectPath,omitempty"`
		FileName   *string `json:"fileName,omitempty"`
		Size       *int64  `json:"size,omitempty"`

		// compatibility: client may send gs://... in avatarIcon
		AvatarIcon *string `json:"avatarIcon,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	bucket := strings.TrimSpace(ptrStr(body.Bucket))
	obj := strings.TrimSpace(ptrStr(body.ObjectPath))

	// if avatarIcon is gs://..., parse and override
	if v := strings.TrimSpace(ptrStr(body.AvatarIcon)); v != "" {
		if b, o, ok := avataricon.ParseGCSURL(v); ok {
			bucket = b
			obj = o
		}
	}

	if bucket == "" {
		bucket = "narratives-development_avatar_icon"
	}
	if obj == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "objectPath is required"})
		return
	}

	in := avataruc.ReplaceIconInput{
		Bucket:     bucket,
		ObjectPath: strings.TrimLeft(obj, "/"),
		FileName:   trimPtr(body.FileName),
		Size:       body.Size,
	}

	log.Printf(
		"[mall_avatar_handler] POST /mall/avatars/%s/icon request bucket=%q objectPath=%q fileName=%q size=%v\n",
		id,
		in.Bucket,
		in.ObjectPath,
		ptrStr(in.FileName),
		in.Size,
	)

	ic, err := h.uc.ReplaceAvatarIcon(ctx, id, in)
	if err != nil {
		log.Printf("[mall_avatar_handler] POST /mall/avatars/%s/icon error=%v\n", id, err)
		writeAvatarErr(w, err)
		return
	}

	hasURL := strings.TrimSpace(ic.URL) != ""
	log.Printf(
		"[mall_avatar_handler] POST /mall/avatars/%s/icon ok iconId=%q url_set=%t url=%q\n",
		id,
		ic.ID,
		hasURL,
		ic.URL,
	)

	_ = json.NewEncoder(w).Encode(toAvatarIconResponse(ic, id))
}
