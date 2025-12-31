// backend/internal/domain/postImage/repository_port.go
package postImage

import "context"

// GCSDeleteOp represents a single delete target in GCS.
type GCSDeleteOp struct {
	Bucket     string
	ObjectPath string
}

// ObjectStoragePort is a persistence port for Post images stored in GCS.
//
// ✅ Single bucket policy:
// - bucket: narratives-development-posts
// - objectPath: avatars/{avatarId}/posts/{postId}/{fileName}
//
// NOTE:
// - "public bucket" is an infra setting (IAM / Public Access Prevention).
// - This port only handles object operations; metadata (PostImage entity) can be stored elsewhere if needed.
type ObjectStoragePort interface {
	// EnsurePrefix creates a placeholder object under prefix so that the "folder" appears in console.
	// e.g. prefix "avatars/<avatarId>/posts/" -> creates "avatars/<avatarId>/posts/.keep"
	EnsurePrefix(ctx context.Context, bucket, prefix string) error

	// DeleteObjects deletes the given objects.
	DeleteObjects(ctx context.Context, ops []GCSDeleteOp) error

	// ListObjectPaths lists object paths under prefix.
	ListObjectPaths(ctx context.Context, bucket, prefix string) ([]string, error)

	// DeleteByPrefix deletes all objects under prefix.
	DeleteByPrefix(ctx context.Context, bucket, prefix string) error

	// Put uploads bytes to bucket/objectPath (server-side upload).
	// If you use signed URLs from the client, this can be unused.
	Put(ctx context.Context, bucket, objectPath, contentType string, data []byte) error

	// PublicURL returns a publicly accessible URL if the bucket is public.
	PublicURL(bucket, objectPath string) string
}
