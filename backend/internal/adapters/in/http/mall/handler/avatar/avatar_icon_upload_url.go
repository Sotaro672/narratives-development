// backend/internal/adapters/in/http/mall/handler/avatar/avatar_icon_upload_url.go
package avatarHandler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/storage"
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

	fileName := strings.TrimSpace(ptrStr(body.FileName))
	mimeType := strings.TrimSpace(ptrStr(body.MimeType))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	bucket := strings.TrimSpace(os.Getenv("AVATAR_ICON_BUCKET"))
	if bucket == "" {
		bucket = "narratives-development_avatar_icon"
	}

	oid := newObjectID()
	ext := guessExt(fileName, mimeType)
	objectPath := fmt.Sprintf("%s/%s%s", id, oid, ext)

	signerEmail := strings.TrimSpace(os.Getenv("GCS_SIGNER_EMAIL"))

	credPath := strings.TrimSpace(os.Getenv("GCS_SIGNER_CREDENTIALS"))
	if credPath == "" {
		credPath = strings.TrimSpace(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"))
	}

	exp := time.Now().Add(15 * time.Minute)

	// A) keyfile signing (local)
	if credPath != "" {
		email, pk, err := loadServiceAccountKey(credPath)
		if err != nil {
			log.Printf("[mall_avatar_handler] POST /mall/avatars/%s/icon-upload-url load key error=%v credPath=%q\n", id, err, credPath)
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to load signer key"})
			return
		}

		uploadURL, err := storage.SignedURL(bucket, objectPath, &storage.SignedURLOptions{
			GoogleAccessID: email,
			PrivateKey:     pk,
			Method:         http.MethodPut,
			Expires:        exp,
			ContentType:    mimeType,
		})
		if err != nil {
			log.Printf("[mall_avatar_handler] POST /mall/avatars/%s/icon-upload-url sign (keyfile) error=%v\n", id, err)
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to sign url"})
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"uploadUrl":  uploadURL,
			"bucket":     bucket,
			"objectPath": objectPath,
			"gsUrl":      fmt.Sprintf("gs://%s/%s", bucket, objectPath),
			"expiresAt":  exp.UTC().Format(time.RFC3339),
		})
		return
	}

	// B) IAM Credentials signing (Cloud Run)
	if signerEmail == "" {
		log.Printf("[mall_avatar_handler] POST /mall/avatars/%s/icon-upload-url missing signer email (set GCS_SIGNER_EMAIL)\n", id)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "missing signer email (set GCS_SIGNER_EMAIL)"})
		return
	}

	uploadURL, err := storage.SignedURL(bucket, objectPath, &storage.SignedURLOptions{
		GoogleAccessID: signerEmail,
		Method:         http.MethodPut,
		Expires:        exp,
		ContentType:    mimeType,
		SignBytes: func(b []byte) ([]byte, error) {
			return signBytesWithIAM(ctx, signerEmail, b)
		},
	})
	if err != nil {
		log.Printf("[mall_avatar_handler] POST /mall/avatars/%s/icon-upload-url sign (iam) error=%v signer=%q bucket=%q objectPath=%q\n", id, err, signerEmail, bucket, objectPath)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to sign url"})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"uploadUrl":  uploadURL,
		"bucket":     bucket,
		"objectPath": objectPath,
		"gsUrl":      fmt.Sprintf("gs://%s/%s", bucket, objectPath),
		"expiresAt":  exp.UTC().Format(time.RFC3339),
	})
}
