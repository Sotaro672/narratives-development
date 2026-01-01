package firestore

import (
	"fmt"
	"strings"
	"time"
)

func asString(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return fmt.Sprint(v)
	}
}

func asInt(v any) int {
	if v == nil {
		return 0
	}
	switch t := v.(type) {
	case int:
		return t
	case int8:
		return int(t)
	case int16:
		return int(t)
	case int32:
		return int(t)
	case int64:
		return int(t)
	case uint:
		return int(t)
	case uint8:
		return int(t)
	case uint16:
		return int(t)
	case uint32:
		return int(t)
	case uint64:
		return int(t)
	case float32:
		return int(t)
	case float64:
		return int(t)
	case string:
		tt := strings.TrimSpace(t)
		if tt == "" {
			return 0
		}
		var n int
		_, _ = fmt.Sscanf(tt, "%d", &n)
		return n
	default:
		// best-effort
		s := strings.TrimSpace(fmt.Sprint(v))
		if s == "" {
			return 0
		}
		var n int
		_, _ = fmt.Sscanf(s, "%d", &n)
		return n
	}
}

// asTime returns (time, ok)
func asTime(v any) (time.Time, bool) {
	if v == nil {
		return time.Time{}, false
	}
	switch t := v.(type) {
	case time.Time:
		return t, true
	default:
		return time.Time{}, false
	}
}
