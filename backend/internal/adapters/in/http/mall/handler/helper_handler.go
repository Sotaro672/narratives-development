// backend/internal/adapters/in/http/mall/handler/helper_handler.go
package mallHandler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"narratives/internal/adapters/in/http/middleware"
)

// ============================================================
// Shared helpers
// ============================================================

// isNotFound is a package-level helper used by multiple handlers.
// It delegates to isNotFoundLike for backwards compatibility.
func isNotFound(err error) bool {
	return isNotFoundLike(err)
}

// ============================================================
// HTTP helpers
// ============================================================

func writeJSON(
	w http.ResponseWriter,
	code int,
	v any,
) {
	w.Header().Set(
		"Content-Type",
		"application/json",
	)
	w.WriteHeader(code)

	_ = json.NewEncoder(w).Encode(v)
}

func methodNotAllowed(
	w http.ResponseWriter,
) {
	writeJSON(
		w,
		http.StatusMethodNotAllowed,
		map[string]string{
			"error": "method_not_allowed",
		},
	)
}

func notFound(
	w http.ResponseWriter,
) {
	writeJSON(
		w,
		http.StatusNotFound,
		map[string]string{
			"error": "not_found",
		},
	)
}

func badRequest(
	w http.ResponseWriter,
	msg string,
) {
	writeJSON(
		w,
		http.StatusBadRequest,
		map[string]string{
			"error": msg,
		},
	)
}

func internalError(
	w http.ResponseWriter,
	msg string,
) {
	writeJSON(
		w,
		http.StatusInternalServerError,
		map[string]string{
			"error": msg,
		},
	)
}

func parseIntDefault(
	s string,
	def int,
) int {
	if s == "" {
		return def
	}

	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}

	return n
}

func ptrStr(
	p *string,
) string {
	if p == nil {
		return ""
	}

	return *p
}

func readJSON(
	r *http.Request,
	dst any,
) error {
	if dst == nil {
		return errors.New("dst is nil")
	}

	decoder := json.NewDecoder(
		http.MaxBytesReader(
			nil,
			r.Body,
			1<<20,
		),
	)
	decoder.DisallowUnknownFields()

	return decoder.Decode(dst)
}

func isNotFoundLike(
	err error,
) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.Canceled) ||
		errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	message := strings.ToLower(
		err.Error(),
	)

	return strings.Contains(
		message,
		"not_found",
	) ||
		strings.Contains(
			message,
			"not found",
		) ||
		strings.Contains(
			message,
			"404",
		) ||
		strings.Contains(
			message,
			"avatar_not_found_for_uid",
		)
}

func parsePositiveIntDefault(
	raw string,
	fallback int,
) int {
	n, err := strconv.Atoi(
		strings.TrimSpace(raw),
	)
	if err != nil || n <= 0 {
		return fallback
	}

	return n
}

func requireAvatarID(
	w http.ResponseWriter,
	r *http.Request,
) (string, bool) {
	avatarID, ok := middleware.CurrentAvatarID(r)
	avatarID = strings.TrimSpace(avatarID)

	if !ok || avatarID == "" {
		writeJSON(
			w,
			http.StatusUnauthorized,
			map[string]string{
				"error": "avatar context is required",
			},
		)
		return "", false
	}

	return avatarID, true
}
