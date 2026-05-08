// backend\internal\adapters\in\http\shared\strict.go
package shared

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
)

// ------------------------------------------------------------
// Strict input helpers (NO normalization / NO trimming)
// ------------------------------------------------------------
//
// Goal:
// - Do NOT "absorb" naming variations
// - Instead, reject inputs that would require normalization.
// - Use these helpers in HTTP handlers (adapters/in) only.
//
// Policy (default):
// - Empty string is invalid for required fields.
// - Leading/trailing whitespace is invalid.
// - Any tab/newline is invalid.
// - For path: reject trailing slash (or redirect).
// ------------------------------------------------------------

var (
	ErrInvalidInput = errors.New("invalid input")
)

// HasOuterWhitespace returns true if s has leading/trailing spaces.
// It does not modify s.
func HasOuterWhitespace(s string) bool {
	return s != strings.TrimSpace(s)
}

// HasControlWhitespace returns true if s contains \t, \n, or \r.
func HasControlWhitespace(s string) bool {
	return strings.ContainsAny(s, "\t\n\r")
}

// ------------------------------------------------------------
// Path helpers
// ------------------------------------------------------------

// RejectTrailingSlash returns true if it already wrote a response (404).
// Use when you want to strictly reject "/path/".
func RejectTrailingSlash(w http.ResponseWriter, r *http.Request) bool {
	path := r.URL.Path
	if path != "/" && strings.HasSuffix(path, "/") {
		http.NotFound(w, r)
		return true
	}
	return false
}

// RedirectTrailingSlash returns true if it already redirected (308).
// Use when you want to canonicalize "/path/" -> "/path" without trimming other chars.
func RedirectTrailingSlash(w http.ResponseWriter, r *http.Request) bool {
	path := r.URL.Path
	if path != "/" && strings.HasSuffix(path, "/") {
		http.Redirect(w, r, strings.TrimSuffix(path, "/"), http.StatusPermanentRedirect)
		return true
	}
	return false
}

// ------------------------------------------------------------
// String helpers
// ------------------------------------------------------------

// StrictRequired returns raw if it is valid as a required value.
// It does NOT trim or normalize.
// If invalid, it returns ErrInvalidInput wrapped with field context.
func StrictRequired(raw string, field string) (string, error) {
	if raw == "" {
		return "", errors.New(field + " is required")
	}
	if HasOuterWhitespace(raw) {
		return "", errors.New(field + " must not have leading/trailing whitespace")
	}
	if HasControlWhitespace(raw) {
		return "", errors.New(field + " must not contain tab/newline")
	}
	return raw, nil
}

// StrictOptional validates optional input.
// - If raw is empty -> ("", false, nil)
// - If non-empty -> validates and returns (raw, true, nil)
func StrictOptional(raw string, field string) (string, bool, error) {
	if raw == "" {
		return "", false, nil
	}
	v, err := StrictRequired(raw, field)
	if err != nil {
		return "", false, err
	}
	return v, true, nil
}

// StrictID is an alias of StrictRequired, intended for path params like "{id}".
func StrictID(raw string, field string) (string, error) {
	return StrictRequired(raw, field)
}

// StrictOptionalPtr returns *string for optional field if present and valid.
func StrictOptionalPtr(raw string, field string) (*string, error) {
	v, ok, err := StrictOptional(raw, field)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	return &v, nil
}

// ------------------------------------------------------------
// Numeric helpers
// ------------------------------------------------------------

// StrictPositiveIntParam parses a positive integer parameter (e.g., page/perPage).
// - empty => (defaultValue, nil)
// - whitespace/control => error
// - <= 0 or not int => error
func StrictPositiveIntParam(raw string, field string, defaultValue int) (int, error) {
	if raw == "" {
		return defaultValue, nil
	}
	if HasOuterWhitespace(raw) {
		return 0, errors.New(field + " must not have leading/trailing whitespace")
	}
	if HasControlWhitespace(raw) {
		return 0, errors.New(field + " must not contain tab/newline")
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 0, errors.New(field + " must be a positive integer")
	}
	return n, nil
}

// StrictBoolParam parses "true" or "false" strictly.
// - empty => (defaultValue, false, nil)  (not present)
// - invalid => error
func StrictBoolParam(raw string, field string) (bool, bool, error) {
	if raw == "" {
		return false, false, nil
	}
	if HasOuterWhitespace(raw) {
		return false, false, errors.New(field + " must not have leading/trailing whitespace")
	}
	if HasControlWhitespace(raw) {
		return false, false, errors.New(field + " must not contain tab/newline")
	}
	switch raw {
	case "true":
		return true, true, nil
	case "false":
		return false, true, nil
	default:
		return false, false, errors.New(field + " must be 'true' or 'false'")
	}
}
