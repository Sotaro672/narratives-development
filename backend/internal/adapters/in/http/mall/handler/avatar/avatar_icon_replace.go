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
// 方針（毎回上書き / 固定URL）:
//   - objectPath は常に "{avatarId}/icon" に固定（クライアント指定は無視）
//   - クライアント互換:
//   - avatarIcon: "gs://bucket/objectPath" を許容（bucket は読み取るが objectPath は固定）
//   - avatarIcon: "https://storage.googleapis.com/bucket/objectPath" も許容（同上）
//   - bucket:
//   - body.bucket / avatarIcon から bucket が取れればそれを採用
//   - 取れない場合は avataricon.DefaultBucket を採用（ハードコードしない）
//
// Handler responsibilities:
//   - Parse/validate HTTP request body
//   - Normalize bucket from compatibility fields
//   - Force fixed objectPath and call usecase.ReplaceAvatarIcon
//   - Return response
func (h *AvatarHandler) replaceIcon(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "avatar usecase not configured"})
		return
	}

	var body struct {
		// optional
		Bucket   *string `json:"bucket,omitempty"`
		FileName *string `json:"fileName,omitempty"`
		Size     *int64  `json:"size,omitempty"`

		// optional / compatibility:
		// - gs://bucket/objectPath
		// - https://storage.googleapis.com/bucket/objectPath
		AvatarIcon *string `json:"avatarIcon,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	// -----------------------------
	// Resolve bucket (best-effort)
	// -----------------------------
	bucket := strings.TrimSpace(ptrStr(body.Bucket))

	// If avatarIcon present, try parse bucket from gs:// or https://...
	if v := strings.TrimSpace(ptrStr(body.AvatarIcon)); v != "" {
		if b, ok := parseGSPrefixBucket(v); ok {
			bucket = b
		} else if b, _, ok2 := avataricon.ParseGCSURL(v); ok2 {
			// ParseGCSURL supports https://storage.googleapis.com/... (not gs://)
			bucket = b
		}
	}

	if bucket == "" {
		bucket = strings.TrimSpace(avataricon.DefaultBucket)
	}
	if bucket == "" {
		// domain default should exist, but guard anyway
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "bucket is empty"})
		return
	}

	// -----------------------------
	// Fixed objectPath (overwrite)
	// -----------------------------
	objectPath := strings.TrimLeft(id, "/") + "/icon" // ✅ 方針: 固定パス

	in := avataruc.ReplaceIconInput{
		Bucket:     bucket,
		ObjectPath: objectPath,
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

// parseGSPrefixBucket parses bucket from "gs://bucket/objectPath".
// Returns (bucket, ok). Object path is intentionally ignored here (fixed-path policy).
func parseGSPrefixBucket(v string) (string, bool) {
	s := strings.TrimSpace(v)
	if !strings.HasPrefix(s, "gs://") {
		return "", false
	}
	rest := strings.TrimPrefix(s, "gs://")
	rest = strings.TrimLeft(rest, "/")
	if rest == "" {
		return "", false
	}
	parts := strings.SplitN(rest, "/", 2)
	bucket := strings.TrimSpace(parts[0])
	if bucket == "" {
		return "", false
	}
	// (optional) sanity: bucket looks like a bucket name
	if strings.Contains(bucket, " ") {
		return "", false
	}
	return bucket, true
}
