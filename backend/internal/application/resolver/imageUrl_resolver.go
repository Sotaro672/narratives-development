// backend\internal\application\resolver\imageUrl_resolver.go
package resolver

import (
	"errors"
	"net/url"
	"os"
	"strings"
)

// ImageURLResolver resolves/normalizes image URLs for storage and response.
type ImageURLResolver struct {
	PublicBucketName string
}

// NewImageURLResolver creates resolver.
// env優先順:
// - TOKEN_ICON_PUBLIC_BUCKET
// - TOKEN_ICON_BUCKET              ✅ 追加（usecase と揃える）
// 引数 bucket が空なら env / fallback を使う。
func NewImageURLResolver(bucket string) *ImageURLResolver {
	b := strings.TrimSpace(bucket)
	if b == "" {
		b = strings.TrimSpace(os.Getenv("TOKEN_ICON_PUBLIC_BUCKET"))
	}
	if b == "" {
		b = strings.TrimSpace(os.Getenv("TOKEN_ICON_BUCKET")) // ✅ 追加
	}
	return &ImageURLResolver{PublicBucketName: b}
}

func (r *ImageURLResolver) ResolveForSave(rawURL string) (objectPath string, publicURL string, err error) {
	raw := strings.TrimSpace(rawURL)
	if raw == "" {
		return "", "", nil
	}

	if looksLikeURL(raw) {
		bkt, obj, e := parseGCSLikeURL(raw)
		if e != nil {
			return "", "", e
		}
		obj = normalizeObjectPath(obj)
		if obj == "" {
			return "", "", errors.New("objectPath is empty after normalize")
		}

		objectPath = obj

		if bkt == "" {
			bkt = strings.TrimSpace(r.PublicBucketName)
		}
		if bkt != "" {
			publicURL = buildGCSPublicURL(bkt, objectPath)
		}
		return objectPath, publicURL, nil
	}

	obj := normalizeObjectPath(raw)
	if obj == "" {
		return "", "", errors.New("objectPath is empty")
	}
	objectPath = obj

	bkt := strings.TrimSpace(r.PublicBucketName)
	if bkt != "" {
		publicURL = buildGCSPublicURL(bkt, objectPath)
	}

	return objectPath, publicURL, nil
}

func (r *ImageURLResolver) ResolveForResponse(storedObjectPath string, storedIconURL string) string {
	u := strings.TrimSpace(storedIconURL)
	if u != "" {
		return u
	}

	obj := normalizeObjectPath(storedObjectPath)
	if obj == "" {
		return ""
	}

	bkt := strings.TrimSpace(r.PublicBucketName)
	if bkt == "" {
		return ""
	}

	return buildGCSPublicURL(bkt, obj)
}

// ---- helpers (unchanged) ----

func looksLikeURL(s string) bool {
	ls := strings.ToLower(strings.TrimSpace(s))
	return strings.HasPrefix(ls, "http://") || strings.HasPrefix(ls, "https://") || strings.HasPrefix(ls, "gs://")
}

func parseGCSLikeURL(raw string) (bucket string, objectPath string, err error) {
	s := strings.TrimSpace(raw)

	if strings.HasPrefix(strings.ToLower(s), "gs://") {
		trim := strings.TrimPrefix(s, "gs://")
		trim = strings.TrimPrefix(trim, "GS://")

		parts := strings.SplitN(trim, "/", 2)
		if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
			return "", "", errors.New("invalid gs:// url: bucket is empty")
		}
		bucket = strings.TrimSpace(parts[0])
		if len(parts) == 1 {
			return bucket, "", errors.New("invalid gs:// url: objectPath is empty")
		}
		objectPath = parts[1]
		return bucket, objectPath, nil
	}

	u, e := url.Parse(s)
	if e != nil {
		return "", "", e
	}

	host := strings.ToLower(strings.TrimSpace(u.Host))
	path := u.Path

	if host == "storage.googleapis.com" {
		p := strings.TrimPrefix(path, "/")
		parts := strings.SplitN(p, "/", 2)
		if len(parts) < 2 {
			return "", "", errors.New("invalid storage.googleapis.com url: missing /bucket/objectPath")
		}
		bucket = strings.TrimSpace(parts[0])
		objectPath = parts[1]
		return bucket, objectPath, nil
	}

	if strings.HasSuffix(host, ".storage.googleapis.com") {
		bucket = strings.TrimSuffix(host, ".storage.googleapis.com")
		objectPath = strings.TrimPrefix(path, "/")
		if strings.TrimSpace(bucket) == "" {
			return "", "", errors.New("invalid *.storage.googleapis.com url: bucket is empty")
		}
		return bucket, objectPath, nil
	}

	return "", "", errors.New("unsupported image url host: " + host)
}

func normalizeObjectPath(p string) string {
	s := strings.TrimSpace(p)
	if s == "" {
		return ""
	}

	if i := strings.Index(s, "?"); i >= 0 {
		s = s[:i]
	}

	s = strings.ReplaceAll(s, "\\", "/")
	s = strings.TrimPrefix(s, "/")

	s = strings.ReplaceAll(s, "../", "")
	s = strings.ReplaceAll(s, "..\\", "")
	s = strings.ReplaceAll(s, "/..", "")

	for strings.Contains(s, "//") {
		s = strings.ReplaceAll(s, "//", "/")
	}

	return strings.TrimSpace(s)
}

func buildGCSPublicURL(bucket string, objectPath string) string {
	b := strings.TrimSpace(bucket)
	p := normalizeObjectPath(objectPath)
	if b == "" || p == "" {
		return ""
	}

	segs := strings.Split(p, "/")
	for i := range segs {
		segs[i] = url.PathEscape(segs[i])
	}
	encoded := strings.Join(segs, "/")

	return "https://storage.googleapis.com/" + b + "/" + encoded
}
