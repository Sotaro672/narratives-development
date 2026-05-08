// backend/internal/application/mint/helpers.go
package mint

import (
	"errors"
	"sort"
	"strings"

	mintdom "narratives/internal/domain/mint"
)

// ============================================================
// Local helpers
// ============================================================

func normalizeIDs(raw []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(raw))
	for _, s := range raw {
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
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found")
}

func isInconsistentMintErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	// 例: "mint: inconsistent minted / mintedAt"
	if strings.Contains(msg, "inconsistent minted") {
		return true
	}
	if strings.Contains(msg, "minted") && strings.Contains(msg, "mintedat") && strings.Contains(msg, "inconsistent") {
		return true
	}
	return false
}
