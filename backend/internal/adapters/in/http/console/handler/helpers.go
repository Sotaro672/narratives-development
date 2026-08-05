// backend\internal\adapters\in\http\console\handler\helpers.go
package consoleHandler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	listdom "narratives/internal/domain/list"
)

func methodNotAllowed(w http.ResponseWriter) {
	writeError(w, http.StatusMethodNotAllowed, "method_not_allowed")
}

// usecase.ErrNotSupported は型が見えないので message 判定
func isNotSupported(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())

	return strings.Contains(msg, "not supported") ||
		strings.Contains(msg, "not_supported") ||
		strings.Contains(msg, "notsupported")
}

func parseIntDefault(s string, def int) int {
	return parsePositiveInt(s, def, 0)
}

func parsePositiveInt(raw string, def int, max int) int {
	value := strings.Trim(raw, " \t\r\n")
	if value == "" {
		return def
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return def
	}

	if max > 0 && parsed > max {
		return max
	}

	return parsed
}

// splitCSV parses "a,b,c" / "a, b, c" into []string.
// Empty items are removed.
func splitCSV(s string) []string {
	if s == "" {
		return nil
	}

	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))

	for _, part := range parts {
		value := strings.Trim(part, " \t\r\n")
		if value != "" {
			out = append(out, value)
		}
	}

	return out
}

// ------------------------------------------------------------
// Request helpers
// ------------------------------------------------------------

func decodeStrictJSON(r *http.Request, destination any) error {
	if r == nil || r.Body == nil {
		return errors.New("request body is required")
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		return err
	}

	var trailingValue any
	if err := decoder.Decode(&trailingValue); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple json values are not allowed")
		}

		return err
	}

	return nil
}

// ------------------------------------------------------------
// Response helpers
// ------------------------------------------------------------

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{
		"error": message,
	})
}

func writeNotFound(w http.ResponseWriter) {
	writeError(w, http.StatusNotFound, "not_found")
}

func writeConsoleListErr(w http.ResponseWriter, err error) {
	code := http.StatusInternalServerError

	switch {
	case errors.Is(err, listdom.ErrNotFound),
		errors.Is(err, listdom.ErrListImageNotFound):
		code = http.StatusNotFound

	case errors.Is(err, listdom.ErrConflict),
		errors.Is(err, listdom.ErrListImageConflict):
		code = http.StatusConflict

	default:
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "invalid") ||
			strings.Contains(msg, "required") ||
			strings.Contains(msg, "must") {
			code = http.StatusBadRequest
		}
	}

	writeError(w, code, err.Error())
}
