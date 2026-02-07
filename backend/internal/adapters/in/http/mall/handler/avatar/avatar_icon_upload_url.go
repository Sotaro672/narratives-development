// backend/internal/adapters/in/http/mall/handler/avatar/avatar_icon_upload_url.go
package avatarHandler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

func (h *AvatarHandler) issueIconUploadURL(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()

	id = strings.TrimSpace(id)
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	// NOTE:
	// - fileName は usecase 側で現状未使用（固定パス {avatarId}/icon のため）
	// - size も現状未使用（将来バリデーションするなら usecase に渡す）
	var body struct {
		FileName *string `json:"fileName,omitempty"`
		MimeType *string `json:"mimeType,omitempty"`
		Size     *int64  `json:"size,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
		return
	}

	if h == nil || h.uc == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "avatar usecase not configured"})
		return
	}

	// usecase は contentType を SignedURL の ContentType に含めるため、PUT と一致必須
	mimeType := strings.TrimSpace(ptrStr(body.MimeType))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// fileName は現状使わないが、handler 仕様互換のため渡す（usecase 側では _ string）
	fileName := strings.TrimSpace(ptrStr(body.FileName))

	out, err := h.uc.IssueAvatarIconUploadURL(ctx, id, fileName, mimeType)
	if err != nil {
		log.Printf("[mall_avatar_handler] POST /mall/avatars/%s/icon-upload-url issue upload url error=%v\n", id, err)
		// TODO: err 種別で 400/404 を返し分けたい場合はここで分岐
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	if out == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to issue upload url"})
		return
	}

	expiresAt := ""
	if out.ExpiresAt != nil {
		expiresAt = out.ExpiresAt.UTC().Format(time.RFC3339)
	}

	// ✅ 方針: Firestore に保存するのは public https URL（長期で固定）
	// 返却も publicUrl を正として扱う
	_ = json.NewEncoder(w).Encode(map[string]any{
		"uploadUrl":  strings.TrimSpace(out.UploadURL),
		"publicUrl":  strings.TrimSpace(out.PublicURL),
		"bucket":     strings.TrimSpace(out.Bucket),                            // デバッグ用途
		"objectPath": strings.TrimLeft(strings.TrimSpace(out.ObjectPath), "/"), // デバッグ用途
		"expiresAt":  expiresAt,
	})
}
