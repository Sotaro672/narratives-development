// backend/internal/adapters/out/gcs/postImage_repository_gcs.go
package gcs

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"cloud.google.com/go/storage"
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

func bucketOrDefault(v, def string) string {
	v = strings.TrimSpace(v)
	if v != "" {
		return v
	}
	return strings.TrimSpace(def)
}
