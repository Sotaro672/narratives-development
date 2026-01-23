// backend/internal/adapters/out/gcs/helper_repository_gcs.go
package gcs

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"path"
	"strings"
	"time"
)

// sanitizePathSegment normalizes a path segment for GCS object paths.
// - removes separators
// - trims dots/spaces
func sanitizePathSegment(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// prohibit separators
	s = strings.ReplaceAll(s, "\\", "_")
	s = strings.ReplaceAll(s, "/", "_")
	// trim dots/spaces to avoid weird paths
	s = strings.Trim(s, ". ")
	return s
}

// ensureExtensionByMIME appends an extension based on MIME when fileName has no extension.
func ensureExtensionByMIME(fileName string, mime string) string {
	lower := strings.ToLower(strings.TrimSpace(fileName))

	// If already has an extension, keep it
	if strings.Contains(path.Base(lower), ".") {
		return fileName
	}

	ext := ""
	switch strings.ToLower(strings.TrimSpace(mime)) {
	case "image/jpeg", "image/jpg":
		ext = ".jpg"
	case "image/png":
		ext = ".png"
	case "image/webp":
		ext = ".webp"
	case "image/gif":
		ext = ".gif"
	default:
		ext = ""
	}

	if ext == "" {
		return fileName
	}
	return fileName + ext
}

// newObjectID generates a random-ish id for object paths.
func newObjectID() string {
	// 12 bytes random => 24 hex chars
	b := make([]byte, 12)
	if _, err := rand.Read(b); err == nil {
		return hex.EncodeToString(b)
	}
	// fallback
	return fmt.Sprintf("%d", time.Now().UTC().UnixNano())
}

// isKeepObject returns true if the objectPath represents a ".keep" object.
// Both "xxx/.keep" and ".keep" are treated as keep objects.
func isKeepObject(objectPath string) bool {
	p := strings.TrimSpace(objectPath)
	if p == "" {
		return false
	}
	return strings.HasSuffix(p, "/.keep") || lastSegment(p) == ".keep"
}
