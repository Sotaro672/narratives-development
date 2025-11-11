// backend/internal/adapters/out/gcs/common/gcs_repository.go
package common

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// GCSPublicURL builds a public GCS URL.
// - bucket が空なら defaultBucket を使用
// - objectPath の先頭の "/" は除去
func GCSPublicURL(bucket, objectPath, defaultBucket string) string {
	b := strings.TrimSpace(bucket)
	if b == "" {
		b = strings.TrimSpace(defaultBucket)
	}
	obj := strings.TrimLeft(strings.TrimSpace(objectPath), "/")
	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", b, obj)
}

// ParseGCSURL parses a GCS-like URL and returns (bucket, objectPath, ok).
// 対応例:
//   - https://storage.googleapis.com/<bucket>/<object>
//   - https://storage.cloud.google.com/<bucket>/<object>
func ParseGCSURL(u string) (string, string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(u))
	if err != nil {
		return "", "", false
	}

	host := strings.ToLower(parsed.Host)
	if host != "storage.googleapis.com" && host != "storage.cloud.google.com" {
		return "", "", false
	}

	p := strings.TrimLeft(parsed.EscapedPath(), "/")
	if p == "" {
		return "", "", false
	}

	parts := strings.SplitN(p, "/", 2)
	if len(parts) < 2 {
		return "", "", false
	}

	bucket := parts[0]
	objectPath, err := url.PathUnescape(parts[1])
	if err != nil {
		return "", "", false
	}

	return bucket, objectPath, true
}

// ParseIntMeta parses an int value from metadata[key].
func ParseIntMeta(md map[string]string, key string) (int, bool) {
	if md == nil {
		return 0, false
	}
	v, ok := md[key]
	if !ok {
		return 0, false
	}
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, false
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, false
	}
	return n, true
}

// ParseInt64Meta parses an int64 value from metadata[key].
func ParseInt64Meta(md map[string]string, key string) (int64, bool) {
	if md == nil {
		return 0, false
	}
	v, ok := md[key]
	if !ok {
		return 0, false
	}
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, false
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}
