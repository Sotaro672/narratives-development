// backend/internal/adapters/out/gcs/postImage_repository_gcs.go
package gcs

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iamcredentials/v1"
	"google.golang.org/api/iterator"

	postimagedom "narratives/internal/domain/postImage"
)

// PostImageRepositoryGCS is a GCS adapter for post images (object storage).
//
// ✅ Recommended layout (single bucket):
// - bucket: narratives-development-posts
// - objectPath: avatars/{avatarId}/posts/{postId or yyyyMMdd}/<fileName>
//
// Public access:
//   - If the bucket has IAM "allUsers: Storage Object Viewer" (uniform access),
//     uploaded objects become publicly readable without per-object ACL changes.
type PostImageRepositoryGCS struct {
	Client *storage.Client
	Bucket string
	// Optional: if empty, uses https://storage.googleapis.com
	PublicBaseURL string
}

func NewPostImageRepositoryGCS(client *storage.Client, bucket string) *PostImageRepositoryGCS {
	return &PostImageRepositoryGCS{
		Client:        client,
		Bucket:        strings.TrimSpace(bucket),
		PublicBaseURL: "https://storage.googleapis.com",
	}
}

func (r *PostImageRepositoryGCS) bucket(name string) (*storage.BucketHandle, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("postImage_repository_gcs: storage client is nil")
	}
	b := strings.TrimSpace(name)
	if b == "" {
		b = strings.TrimSpace(r.Bucket)
	}
	if b == "" {
		return nil, errors.New("postImage_repository_gcs: bucket is empty")
	}
	return r.Client.Bucket(b), nil
}

// EnsurePrefix creates a "folder placeholder" object (e.g. <prefix>.keep) so that
// Cloud Console shows the prefix even when no images exist.
//
// NOTE: GCS doesn't have real folders; this is only a UX/operational helper.
func (r *PostImageRepositoryGCS) EnsurePrefix(ctx context.Context, bucket, prefix string) error {
	bh, err := r.bucket(bucket)
	if err != nil {
		return err
	}
	p := strings.TrimSpace(prefix)
	if p == "" {
		return errors.New("postImage_repository_gcs: prefix is empty")
	}
	if !strings.HasSuffix(p, "/") {
		p += "/"
	}

	objPath := p + ".keep"

	oh := bh.Object(objPath)

	// If already exists, do nothing.
	_, err = oh.Attrs(ctx)
	if err == nil {
		return nil
	}
	if err != storage.ErrObjectNotExist {
		return err
	}

	w := oh.NewWriter(ctx)
	w.ContentType = "text/plain; charset=utf-8"
	// Small marker content.
	_, _ = w.Write([]byte("keep\n"))
	if cerr := w.Close(); cerr != nil {
		return cerr
	}
	return nil
}

// DeleteObjects deletes objects (best-effort, stops on first error).
func (r *PostImageRepositoryGCS) DeleteObjects(ctx context.Context, ops []postimagedom.GCSDeleteOp) error {
	if len(ops) == 0 {
		return nil
	}
	for _, op := range ops {
		bh, err := r.bucket(op.Bucket)
		if err != nil {
			return err
		}
		obj := strings.TrimSpace(op.ObjectPath)
		if obj == "" {
			continue
		}
		if err := bh.Object(obj).Delete(ctx); err != nil && err != storage.ErrObjectNotExist {
			return err
		}
	}
	return nil
}

// ListObjectPaths lists object paths under the given prefix.
// Use this for cascade delete (avatar delete) or cleanup jobs.
func (r *PostImageRepositoryGCS) ListObjectPaths(ctx context.Context, bucket, prefix string) ([]string, error) {
	bh, err := r.bucket(bucket)
	if err != nil {
		return nil, err
	}
	p := strings.TrimSpace(prefix)

	it := bh.Objects(ctx, &storage.Query{
		Prefix: p,
	})

	var out []string
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		if attrs == nil || strings.TrimSpace(attrs.Name) == "" {
			continue
		}
		out = append(out, attrs.Name)
	}
	return out, nil
}

// DeleteByPrefix deletes all objects under prefix (including ".keep").
// Useful for avatar deletion cascade: prefix = "avatars/{avatarId}/posts/".
func (r *PostImageRepositoryGCS) DeleteByPrefix(ctx context.Context, bucket, prefix string) error {
	paths, err := r.ListObjectPaths(ctx, bucket, prefix)
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		return nil
	}
	ops := make([]postimagedom.GCSDeleteOp, 0, len(paths))
	for _, p := range paths {
		ops = append(ops, postimagedom.GCSDeleteOp{
			Bucket:     strings.TrimSpace(bucketOrDefault(bucket, r.Bucket)),
			ObjectPath: p,
		})
	}
	return r.DeleteObjects(ctx, ops)
}

// Put uploads bytes to "bucket/objectPath" directly (non-signed upload).
// If you prefer signed URLs from the client, keep this unused.
func (r *PostImageRepositoryGCS) Put(
	ctx context.Context,
	bucket string,
	objectPath string,
	contentType string,
	data []byte,
) error {
	bh, err := r.bucket(bucket)
	if err != nil {
		return err
	}
	obj := strings.TrimSpace(objectPath)
	if obj == "" {
		return errors.New("postImage_repository_gcs: objectPath is empty")
	}
	w := bh.Object(obj).NewWriter(ctx)
	if ct := strings.TrimSpace(contentType); ct != "" {
		w.ContentType = ct
	}
	// Safety: avoid writer hanging forever.
	w.ChunkSize = 0
	w.Metadata = map[string]string{
		"uploadedAt": time.Now().UTC().Format(time.RFC3339),
	}
	if _, err := w.Write(data); err != nil {
		_ = w.Close()
		return err
	}
	return w.Close()
}

// PublicURL returns a public URL for the object.
// Works when the bucket is publicly readable (uniform access via IAM).
func (r *PostImageRepositoryGCS) PublicURL(bucket, objectPath string) string {
	b := strings.TrimSpace(bucket)
	if b == "" {
		b = strings.TrimSpace(r.Bucket)
	}
	base := strings.TrimSpace(r.PublicBaseURL)
	if base == "" {
		base = "https://storage.googleapis.com"
	}
	// Encode path but keep "/" separators.
	parts := strings.Split(objectPath, "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	encoded := strings.Join(parts, "/")
	return fmt.Sprintf("%s/%s/%s", strings.TrimRight(base, "/"), b, encoded)
}

// ==============================
// Signed URL (postImage_signed_url.go)
// ==============================

// IssueSignedUploadURL issues a V4 signed URL for uploading an object via HTTP PUT.
//
// NOTE:
//   - This implementation uses IAMCredentials SignBlob (no JSON private key required).
//   - You must provide the signer service account email via env.
//     Recommended: GCS_SIGNER_EMAIL (or GOOGLE_SERVICE_ACCOUNT_EMAIL / SERVICE_ACCOUNT_EMAIL).
//
// Required IAM:
//   - The runtime identity must be allowed to call iamcredentials.signBlob for that SA
//     (typically the same SA in Cloud Run).
func (r *PostImageRepositoryGCS) IssueSignedUploadURL(
	ctx context.Context,
	bucket string,
	objectPath string,
	contentType string,
	expiresIn time.Duration,
) (string, error) {
	if r == nil {
		return "", errors.New("postImage_repository_gcs: repo is nil")
	}
	b := strings.TrimSpace(bucketOrDefault(bucket, r.Bucket))
	if b == "" {
		return "", errors.New("postImage_repository_gcs: bucket is empty")
	}
	obj := strings.TrimSpace(objectPath)
	if obj == "" {
		return "", errors.New("postImage_repository_gcs: objectPath is empty")
	}

	// default / clamp
	if expiresIn <= 0 {
		expiresIn = 15 * time.Minute
	}
	if expiresIn > time.Hour {
		expiresIn = time.Hour
	}

	accessID := strings.TrimSpace(firstNonEmptyEnv(
		"GCS_SIGNER_EMAIL",
		"GOOGLE_SERVICE_ACCOUNT_EMAIL",
		"SERVICE_ACCOUNT_EMAIL",
	))
	if accessID == "" {
		return "", errors.New("postImage_repository_gcs: signer email not configured (set GCS_SIGNER_EMAIL)")
	}

	svc, err := iamcredentials.NewService(ctx)
	if err != nil {
		return "", fmt.Errorf("postImage_repository_gcs: iamcredentials init failed: %w", err)
	}

	signBytes := func(bts []byte) ([]byte, error) {
		name := fmt.Sprintf("projects/-/serviceAccounts/%s", accessID)
		req := &iamcredentials.SignBlobRequest{
			Payload: base64.StdEncoding.EncodeToString(bts),
		}
		resp, err := svc.Projects.ServiceAccounts.SignBlob(name, req).Do()
		if err != nil {
			return nil, err
		}
		sig, err := base64.StdEncoding.DecodeString(resp.SignedBlob)
		if err != nil {
			return nil, err
		}
		return sig, nil
	}

	opts := &storage.SignedURLOptions{
		Scheme:         storage.SigningSchemeV4,
		Method:         "PUT",
		GoogleAccessID: accessID,
		SignBytes:      signBytes,
		Expires:        time.Now().UTC().Add(expiresIn),
	}
	if ct := strings.TrimSpace(contentType); ct != "" {
		opts.ContentType = ct
	}

	u, err := storage.SignedURL(b, obj, opts)
	if err != nil {
		return "", err
	}
	return u, nil
}

// (Optional) Useful for private buckets: GET signed URL.
func (r *PostImageRepositoryGCS) IssueSignedDownloadURL(
	ctx context.Context,
	bucket string,
	objectPath string,
	expiresIn time.Duration,
) (string, error) {
	if r == nil {
		return "", errors.New("postImage_repository_gcs: repo is nil")
	}
	b := strings.TrimSpace(bucketOrDefault(bucket, r.Bucket))
	if b == "" {
		return "", errors.New("postImage_repository_gcs: bucket is empty")
	}
	obj := strings.TrimSpace(objectPath)
	if obj == "" {
		return "", errors.New("postImage_repository_gcs: objectPath is empty")
	}

	if expiresIn <= 0 {
		expiresIn = 10 * time.Minute
	}
	if expiresIn > time.Hour {
		expiresIn = time.Hour
	}

	accessID := strings.TrimSpace(firstNonEmptyEnv(
		"GCS_SIGNER_EMAIL",
		"GOOGLE_SERVICE_ACCOUNT_EMAIL",
		"SERVICE_ACCOUNT_EMAIL",
	))
	if accessID == "" {
		return "", errors.New("postImage_repository_gcs: signer email not configured (set GCS_SIGNER_EMAIL)")
	}

	svc, err := iamcredentials.NewService(ctx)
	if err != nil {
		return "", fmt.Errorf("postImage_repository_gcs: iamcredentials init failed: %w", err)
	}

	signBytes := func(bts []byte) ([]byte, error) {
		name := fmt.Sprintf("projects/-/serviceAccounts/%s", accessID)
		req := &iamcredentials.SignBlobRequest{
			Payload: base64.StdEncoding.EncodeToString(bts),
		}
		resp, err := svc.Projects.ServiceAccounts.SignBlob(name, req).Do()
		if err != nil {
			return nil, err
		}
		sig, err := base64.StdEncoding.DecodeString(resp.SignedBlob)
		if err != nil {
			return nil, err
		}
		return sig, nil
	}

	opts := &storage.SignedURLOptions{
		Scheme:         storage.SigningSchemeV4,
		Method:         "GET",
		GoogleAccessID: accessID,
		SignBytes:      signBytes,
		Expires:        time.Now().UTC().Add(expiresIn),
	}

	u, err := storage.SignedURL(b, obj, opts)
	if err != nil {
		return "", err
	}
	return u, nil
}

func firstNonEmptyEnv(keys ...string) string {
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}

func bucketOrDefault(v, def string) string {
	v = strings.TrimSpace(v)
	if v != "" {
		return v
	}
	return strings.TrimSpace(def)
}
