// backend/internal/adapters/in/http/mall/handler/avatar/avatar_errors.go
package avatarHandler

import (
	"encoding/json"
	"errors"
	avataruc "narratives/internal/application/usecase/avatar"
	avatardom "narratives/internal/domain/avatar"
	"net/http"
	"strings"
)

func writeAvatarErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	if errors.Is(err, avatardom.ErrInvalidID) ||
		errors.Is(err, avatardom.ErrInvalidUserID) ||
		errors.Is(err, avataruc.ErrInvalidUserUID) ||
		errors.Is(err, avatardom.ErrInvalidAvatarName) ||
		errors.Is(err, avatardom.ErrInvalidProfile) ||
		errors.Is(err, avatardom.ErrInvalidExternalLink) {
		code = http.StatusBadRequest
	}

	if hasErrNotFound(err) {
		code = http.StatusNotFound
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func trimPtrNilAware(p *string) *string {
	if p == nil {
		return nil
	}
	s := strings.TrimSpace(*p)
	if s == "" {
		return ptr("")
	}
	return &s
}

func ptr[T any](v T) *T { return &v }

func hasErrNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") || strings.Contains(msg, "not_found")
}
