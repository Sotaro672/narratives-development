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
	// ✅ TrimSpace を使わない
	return os.Getenv(envAvatarIconSignerEmail)
}

func avatarIconBucketName() string {
	// ✅ TrimSpace を使わない
	if v := os.Getenv(envAvatarIconBucket); v != "" {
		return v
	}
	return defaultAvatarIconBucket
}

// tokenBlueprint と揃える：固定パス（後から差し替えても規約が安定）
// ObjectPath: "{avatarId}/icon"
func avatarIconObjectPath(avatarID string) (string, error) {
	// ✅ Trim/normalize をしない方針に合わせる：sanitizePathSegment を廃止し、生値を使う
	if avatarID == "" {
		return "", avatardom.ErrInvalidID
	}
	// ただし objectPath 規約として "/" を含む avatarID は危険なので拒否する（trim はしない）
	if strings.Contains(avatarID, "/") || strings.Contains(avatarID, "..") {
		return "", avatardom.ErrInvalidID
	}
	return avatarID + "/icon", nil
}

func gcsObjectPublicURL(bucket, objectPath string) string {
	// ✅ TrimSpace を使わない（先頭 "/" だけ落とす）
	obj := strings.TrimLeft(objectPath, "/")
	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucket, obj)
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
// - SignedURL includes ContentType; frontend PUT must match exactly.
// - "毎回上書き"（固定URL）方式のため fileName は未使用。
func (u *AvatarUsecase) IssueAvatarIconUploadURL(
	ctx context.Context,
	avatarID string,
	_ string, // fileName: fixed-path overwrite policy; not used
	contentType string,
) (*IconUploadURL, error) {

	if u == nil || u.avRepo == nil {
		return nil, fmt.Errorf("avatar icon usecase/repo is nil")
	}

	// ✅ TrimSpace をしない
	id := avatarID
	if id == "" {
		return nil, avatardom.ErrInvalidID
	}

	// ensure avatar exists
	if _, err := u.avRepo.GetByID(ctx, id); err != nil {
		return nil, err
	}

	bucket := avatarIconBucketName()
	// ✅ TrimSpace をしない
	if bucket == "" {
		return nil, fmt.Errorf("avatar icon bucket is empty")
	}

	accessID := avatarIconSignerEmail()
	if accessID == "" {
		return nil, fmt.Errorf("missing %s env (signer service account email)", envAvatarIconSignerEmail)
	}

	// contentType is REQUIRED (prevents signed URL mismatch issues)
	// ✅ TrimSpace をしない（ただし lower は維持）
	ct := strings.ToLower(contentType)
	if ct == "" {
		return nil, fmt.Errorf("contentType is required (e.g. image/png)")
	}
	if !isSupportedAvatarIconMIME(ct) {
		return nil, fmt.Errorf("unsupported contentType=%q", ct)
	}

	objectPath, err := avatarIconObjectPath(id)
	if err != nil {
		return nil, err
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
		UploadURL:  uploadURL,  // ✅ TrimSpace をしない
		PublicURL:  publicURL,  // ✅ TrimSpace をしない
		Bucket:     bucket,     // ✅ TrimSpace をしない
		ObjectPath: objectPath, // ✅ TrimSpace をしない
		ExpiresAt:  ptr(expires),
	}, nil
}

func isSupportedAvatarIconMIME(mime string) bool {
	// ✅ TrimSpace を使わない（ただし lower は維持）
	switch strings.ToLower(mime) {
	case "image/jpeg", "image/jpg", "image/png", "image/webp", "image/gif":
		return true
	default:
		return false
	}
}
