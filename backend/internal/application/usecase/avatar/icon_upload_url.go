// backend/internal/application/usecase/avatar/icon_upload_url.go
package avatar

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iamcredentials/v1"

	avatardom "narratives/internal/domain/avatar"
)

// ============================================================
// Config (avatar icon)
// ============================================================

// 署名に使うサービスアカウントメール（必須）
// Cloud Run では env で明示推奨
const envAvatarIconSignerEmail = "GCS_SIGNER_EMAIL"

// bucket env（任意）
const envAvatarIconBucket = "AVATAR_ICON_BUCKET"

// デフォルト bucket（旧 handler と同じ）
const defaultAvatarIconBucket = "narratives-development_avatar_icon"

// 署名付きURLの有効期限（PUT）
const avatarIconSignedURLTTL = 15 * time.Minute

func avatarIconSignerEmail() string {
	return strings.TrimSpace(os.Getenv(envAvatarIconSignerEmail))
}

func avatarIconBucketName() string {
	if v := strings.TrimSpace(os.Getenv(envAvatarIconBucket)); v != "" {
		return v
	}
	return defaultAvatarIconBucket
}

// tokenBlueprint と揃える：固定パス（後から差し替えても規約が安定）
func avatarIconObjectPath(avatarID string) string {
	id := strings.Trim(strings.TrimSpace(avatarID), "/")
	return id + "/icon"
}

func gcsObjectPublicURL(bucket, objectPath string) string {
	obj := strings.TrimLeft(strings.TrimSpace(objectPath), "/")
	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", strings.TrimSpace(bucket), obj)
}

func ptr[T any](v T) *T { return &v }

// ============================================================
// Usecase: Icon Upload URL (Signed URL)
// ============================================================

// IconUploadURL is returned to front for direct PUT.
type IconUploadURL struct {
	UploadURL  string     `json:"uploadUrl"`
	PublicURL  string     `json:"publicUrl"`
	Bucket     string     `json:"bucket"`
	ObjectPath string     `json:"objectPath"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"`
}

// IssueAvatarIconUploadURL issues V4 signed PUT URL for "{avatarId}/icon".
//
// Required:
// - env GCS_SIGNER_EMAIL set
// - Cloud Run runtime SA has iam.serviceAccounts.signBlob
//
// Note:
// - SignedURL includes ContentType; frontend PUT must match.
func (u *AvatarUsecase) IssueAvatarIconUploadURL(
	ctx context.Context,
	avatarID string,
	_ string, // fileName: not persisted; kept only to match handler signature
	contentType string,
) (*IconUploadURL, error) {

	if u == nil || u.avRepo == nil {
		return nil, fmt.Errorf("avatar icon usecase/repo is nil")
	}

	id := strings.TrimSpace(avatarID)
	if id == "" {
		return nil, avatardom.ErrInvalidID
	}

	// ensure avatar exists
	if _, err := u.avRepo.GetByID(ctx, id); err != nil {
		return nil, err
	}

	bucket := avatarIconBucketName()
	if bucket == "" {
		return nil, fmt.Errorf("avatar icon bucket is empty")
	}

	accessID := avatarIconSignerEmail()
	if accessID == "" {
		return nil, fmt.Errorf("missing %s env (signer service account email)", envAvatarIconSignerEmail)
	}

	objectPath := avatarIconObjectPath(id)

	ct := strings.TrimSpace(contentType)
	if ct == "" {
		ct = "application/octet-stream"
	}

	iamSvc, err := iamcredentials.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("create iamcredentials service: %w", err)
	}

	signBytes := func(b []byte) ([]byte, error) {
		name := "projects/-/serviceAccounts/" + accessID
		req := &iamcredentials.SignBlobRequest{
			Payload: base64.StdEncoding.EncodeToString(b),
		}
		resp, err := iamSvc.Projects.ServiceAccounts.SignBlob(name, req).Do()
		if err != nil {
			return nil, err
		}
		return base64.StdEncoding.DecodeString(resp.SignedBlob)
	}

	expires := time.Now().UTC().Add(avatarIconSignedURLTTL)

	uploadURL, err := storage.SignedURL(bucket, objectPath, &storage.SignedURLOptions{
		Scheme:         storage.SigningSchemeV4,
		Method:         "PUT",
		GoogleAccessID: accessID,
		SignBytes:      signBytes,
		Expires:        expires,
		ContentType:    ct,
	})
	if err != nil {
		return nil, fmt.Errorf("sign gcs url: %w", err)
	}

	publicURL := gcsObjectPublicURL(bucket, objectPath)

	return &IconUploadURL{
		UploadURL:  strings.TrimSpace(uploadURL),
		PublicURL:  strings.TrimSpace(publicURL),
		Bucket:     strings.TrimSpace(bucket),
		ObjectPath: strings.TrimSpace(objectPath),
		ExpiresAt:  ptr(expires),
	}, nil
}
