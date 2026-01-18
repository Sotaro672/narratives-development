package gcs

import (
	"strings"

	gcscommon "narratives/internal/adapters/out/gcs/common"
)

// TokenIconURLResolver resolves icon URL for mall responses.
// It prefers storedIconURL, then resolves storedObjectPath into a public URL.
//
// storedObjectPath can be:
// - http(s)://... (returned as-is)
// - gs://bucket/object or https://storage.googleapis.com/... (parsed)
// - objectPath (treated as object path within bucket)
// bucket: if empty, falls back to defaultTokenIconBucket.
type TokenIconURLResolver struct {
	Bucket string
}

func NewTokenIconURLResolver(bucket string) *TokenIconURLResolver {
	return &TokenIconURLResolver{Bucket: strings.TrimSpace(bucket)}
}

func (r *TokenIconURLResolver) ResolveForResponse(storedObjectPath string, storedIconURL string) string {
	if u := strings.TrimSpace(storedIconURL); u != "" {
		return u
	}

	p := strings.TrimSpace(storedObjectPath)
	if p == "" {
		return ""
	}

	// already absolute URL
	if strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") {
		return p
	}

	// if it's a GCS URL, use its bucket/object
	if b, obj, ok := gcscommon.ParseGCSURL(p); ok {
		return gcscommon.GCSPublicURL(b, obj, defaultTokenIconBucket)
	}

	// otherwise treat as objectPath within configured bucket
	b := strings.TrimSpace(r.Bucket)
	if b == "" {
		b = defaultTokenIconBucket
	}

	p = strings.TrimLeft(p, "/")
	return gcscommon.GCSPublicURL(b, p, defaultTokenIconBucket)
}
