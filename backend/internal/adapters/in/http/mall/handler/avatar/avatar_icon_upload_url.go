// backend/internal/adapters/in/http/mall/handler/avatar/avatar_icon_upload_url.go
package avatarHandler

import (
	"encoding/json"
	"fmt"
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

	fileName := strings.TrimSpace(ptrStr(body.FileName))
	mimeType := strings.TrimSpace(ptrStr(body.MimeType))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	out, err := h.uc.IssueAvatarIconUploadURL(ctx, id, fileName, mimeType)
	if err != nil {
		log.Printf("[mall_avatar_handler] POST /mall/avatars/%s/icon-upload-url issue upload url error=%v\n", id, err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	expiresAt := ""
	if out != nil && out.ExpiresAt != nil {
		expiresAt = out.ExpiresAt.UTC().Format(time.RFC3339)
	}

	bucket := strings.TrimSpace(out.Bucket)
	obj := strings.TrimLeft(strings.TrimSpace(out.ObjectPath), "/")

	gsURL := ""
	if bucket != "" && obj != "" {
		gsURL = fmt.Sprintf("gs://%s/%s", bucket, obj)
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"uploadUrl":  strings.TrimSpace(out.UploadURL),
		"publicUrl":  strings.TrimSpace(out.PublicURL),
		"bucket":     bucket,
		"objectPath": obj,
		"gsUrl":      gsURL,
		"expiresAt":  expiresAt,
	})
}
