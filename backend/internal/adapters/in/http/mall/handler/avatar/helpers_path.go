// backend/internal/adapters/in/http/mall/handler/avatar/helpers_path.go
package avatarHandler

import "strings"

func extractIDFromPath(path0 string, prefix string) (string, bool) {
	if !strings.HasPrefix(path0, prefix) {
		return "", false
	}
	rest := strings.TrimPrefix(path0, prefix)
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return "", false
	}
	parts := strings.Split(rest, "/")
	id := strings.TrimSpace(parts[0])
	if id == "" {
		return "", false
	}
	if len(parts) > 1 {
		return "", false
	}
	return id, true
}

func extractIDFromSubroute(path0 string, prefix string, suffix string) (string, bool) {
	if !strings.HasPrefix(path0, prefix) || !strings.HasSuffix(path0, suffix) {
		return "", false
	}
	rest := strings.TrimPrefix(path0, prefix)
	rest = strings.TrimSuffix(rest, suffix)
	rest = strings.Trim(rest, "/")
	if rest == "" {
		return "", false
	}
	parts := strings.Split(rest, "/")
	id := strings.TrimSpace(parts[0])
	if id == "" {
		return "", false
	}
	if len(parts) > 1 {
		return "", false
	}
	return id, true
}
