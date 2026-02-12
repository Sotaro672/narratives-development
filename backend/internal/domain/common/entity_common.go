// backend/internal/domain/common/entity_common.go
package common

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// ParseTime parses various time formats and returns UTC time.
func ParseTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, errors.New("empty time")
	}

	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), nil
	}

	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t.UTC(), nil
		}
	}

	return time.Time{}, fmt.Errorf("cannot parse time: %q", s)
}

// NormalizeStringPtr trims whitespace and converts empty string to nil.
// - nil -> nil
// - "   " -> nil
// - " abc " -> ptr("abc")
func NormalizeStringPtr(s *string) *string {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil
	}
	return &v
}
