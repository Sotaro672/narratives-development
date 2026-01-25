// backend/internal/application/mint/helpers.go
package mint

import (
	"errors"
	"reflect"
	"sort"
	"strings"

	mintdom "narratives/internal/domain/mint"
)

// ============================================================
// Local helpers
// ============================================================

func normalizeMintProducts(raw any) []string {
	if raw == nil {
		return []string{}
	}

	rv := reflect.ValueOf(raw)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return []string{}
		}
		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		out := make([]string, 0, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			it := rv.Index(i)
			if it.Kind() == reflect.String {
				s := strings.TrimSpace(it.String())
				if s != "" {
					out = append(out, s)
				}
			}
		}
		return normalizeIDs(out)

	case reflect.Map:
		out := make([]string, 0, rv.Len())
		iter := rv.MapRange()
		for iter.Next() {
			k := iter.Key()
			if k.Kind() == reflect.String {
				s := strings.TrimSpace(k.String())
				if s != "" {
					out = append(out, s)
				}
			}
		}
		return normalizeIDs(out)

	default:
		return []string{}
	}
}

func normalizeIDs(raw []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(raw))
	for _, s := range raw {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func isNotFoundErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, mintdom.ErrNotFound) {
		return true
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(msg, "not found")
}

func isInconsistentMintErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	// 例: "mint: inconsistent minted / mintedAt"
	if strings.Contains(msg, "inconsistent minted") {
		return true
	}
	if strings.Contains(msg, "minted") && strings.Contains(msg, "mintedat") && strings.Contains(msg, "inconsistent") {
		return true
	}
	return false
}
