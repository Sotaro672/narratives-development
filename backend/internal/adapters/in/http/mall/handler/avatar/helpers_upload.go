// backend/internal/adapters/in/http/mall/handler/avatar/helpers_upload.go
package avatarHandler

import (
	"crypto/rand"
	"encoding/hex"
	"path"
	"strings"
)

func newObjectID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func guessExt(fileName string, mimeType string) string {
	ext := strings.ToLower(path.Ext(strings.TrimSpace(fileName)))
	if ext != "" && len(ext) <= 10 {
		return ext
	}

	mt := strings.ToLower(strings.TrimSpace(mimeType))
	switch mt {
	case "image/png":
		return ".png"
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	default:
		return ""
	}
}
