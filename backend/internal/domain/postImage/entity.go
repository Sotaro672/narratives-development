// backend/internal/domain/postImage/entity.go
package postImage

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidID         = errors.New("postImage: invalid id")
	ErrInvalidAvatarID   = errors.New("postImage: invalid avatarId")
	ErrInvalidBucket     = errors.New("postImage: invalid bucket")
	ErrInvalidObjectPath = errors.New("postImage: invalid objectPath")
	ErrInvalidCreatedAt  = errors.New("postImage: invalid createdAt")
)

// ✅ 1-bucket 運用（public bucket を想定）
const DefaultBucket = "narratives-development-posts"

// 公開URLのベース（GCS の一般的な形式）
// - バケットが public ならこの URL でアクセスできます
const PublicBaseURL = "https://storage.googleapis.com/"

// PostImage is a public image object metadata for a post (Mall).
//
// Storage policy (recommended):
// - bucket: narratives-development-posts (single bucket)
// - objectPath: avatars/{avatarId}/posts/{postId}/{fileName}
//
// NOTE:
// - "bucket is public" is an infra setting (GCS ACL / IAM / Public Access Prevention).
// - This entity only models metadata and generates public URLs.
type PostImage struct {
	ID         string
	AvatarID   string
	Bucket     string
	ObjectPath string

	// optional metadata (handy for UI/debug)
	FileName    *string
	ContentType *string
	Size        *int64

	CreatedAt time.Time
}

// New creates a PostImage (minimal).
func New(id, avatarID, bucket, objectPath string, createdAt time.Time) (PostImage, error) {
	pi := PostImage{
		ID:         strings.TrimSpace(id),
		AvatarID:   strings.TrimSpace(avatarID),
		Bucket:     strings.TrimSpace(bucket),
		ObjectPath: strings.TrimSpace(objectPath),
		CreatedAt:  createdAt.UTC(),
	}
	if err := pi.validate(); err != nil {
		return PostImage{}, err
	}
	return pi, nil
}

// NewPublicPostImage creates PostImage under DefaultBucket with recommended prefix.
// - postID can be any stable identifier (e.g. post docId).
// - fileName must be a plain file name (no slashes). If you pass a path, it will be normalized.
func NewPublicPostImage(
	id, avatarID, postID, fileName string,
	createdAt time.Time,
) (PostImage, error) {
	aid := strings.TrimSpace(avatarID)
	pid := strings.TrimSpace(postID)
	fn := sanitizeFileName(fileName)

	if aid == "" {
		return PostImage{}, ErrInvalidAvatarID
	}
	if pid == "" {
		return PostImage{}, errors.New("postImage: invalid postId")
	}
	if fn == "" {
		return PostImage{}, errors.New("postImage: invalid fileName")
	}

	obj := BuildObjectPath(aid, pid, fn)

	return New(id, aid, DefaultBucket, obj, createdAt)
}

// BuildObjectPath returns recommended objectPath:
// avatars/{avatarId}/posts/{postId}/{fileName}
func BuildObjectPath(avatarID, postID, fileName string) string {
	aid := strings.TrimSpace(avatarID)
	pid := strings.TrimSpace(postID)
	fn := sanitizeFileName(fileName)
	return "avatars/" + aid + "/posts/" + pid + "/" + fn
}

// PublicURL returns https://storage.googleapis.com/<bucket>/<objectPath>
// (valid if the bucket/object is publicly readable)
func (p PostImage) PublicURL() string {
	b := strings.TrimSpace(p.Bucket)
	o := strings.TrimSpace(p.ObjectPath)
	if b == "" || o == "" {
		return ""
	}
	return PublicBaseURL + b + "/" + o
}

func (p PostImage) validate() error {
	if strings.TrimSpace(p.ID) == "" {
		return ErrInvalidID
	}
	if strings.TrimSpace(p.AvatarID) == "" {
		return ErrInvalidAvatarID
	}
	if strings.TrimSpace(p.Bucket) == "" {
		return ErrInvalidBucket
	}
	if strings.TrimSpace(p.ObjectPath) == "" {
		return ErrInvalidObjectPath
	}
	if strings.Contains(p.ObjectPath, "gs://") {
		// objectPath には gs:// を入れない（bucket/object を別管理）
		return ErrInvalidObjectPath
	}
	if strings.HasPrefix(p.ObjectPath, "/") {
		return ErrInvalidObjectPath
	}
	if p.CreatedAt.IsZero() {
		return ErrInvalidCreatedAt
	}
	return nil
}

// sanitizeFileName removes any path fragments and trims.
func sanitizeFileName(s string) string {
	v := strings.TrimSpace(s)
	if v == "" {
		return ""
	}
	v = strings.ReplaceAll(v, "\\", "/")
	if i := strings.LastIndex(v, "/"); i >= 0 {
		v = v[i+1:]
	}
	v = strings.TrimSpace(v)
	// forbid empty or "." / ".."
	if v == "" || v == "." || v == ".." {
		return ""
	}
	return v
}

// Helpers for pointers (optional metadata)
func TrimPtr(p *string) *string {
	if p == nil {
		return nil
	}
	v := strings.TrimSpace(*p)
	if v == "" {
		return nil
	}
	return &v
}
