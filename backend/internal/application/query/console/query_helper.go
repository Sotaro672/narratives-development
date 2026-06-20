// backend/internal/application/query/console/helper_query.go
package query

import (
	"context"
	"errors"
	"strings"

	common "narratives/internal/domain/common"
	listdom "narratives/internal/domain/list"
)

// ============================================================
// Company boundary helpers
// ============================================================

func AllowedInventoryIDsFromContext(
	ctx context.Context,
	invRows InventoryRowsLister,
) ([]string, map[string]struct{}, error) {
	if invRows == nil {
		return nil, nil, errors.New("inventory rows lister is nil (company boundary via inventory_query is not configured)")
	}

	rows, err := invRows.ListByCurrentCompany(ctx)
	if err != nil {
		return nil, nil, err
	}

	ids := make([]string, 0, len(rows))
	set := map[string]struct{}{}

	for _, r := range rows {
		invID := BuildInventoryID(r.ProductBlueprintID, r.TokenBlueprintID)
		if invID == "" {
			continue
		}

		if _, ok := set[invID]; ok {
			continue
		}

		set[invID] = struct{}{}
		ids = append(ids, invID)
	}

	return ids, set, nil
}

func AllowedInventoryIDSetFromContext(ctx context.Context, invRows InventoryRowsLister) (map[string]struct{}, error) {
	_, set, err := AllowedInventoryIDsFromContext(ctx, invRows)
	return set, err
}

func InventoryAllowed(set map[string]struct{}, inventoryID string) bool {
	if len(set) == 0 {
		return false
	}

	id := inventoryID
	if id == "" {
		return false
	}

	_, ok := set[id]
	return ok
}

// ============================================================
// Paging helpers
// ============================================================

func NormalizeCommonPage(p common.Page) common.Page {
	if p.Number <= 0 {
		p.Number = 1
	}
	if p.PerPage <= 0 {
		p.PerPage = 20
	}

	return p
}

func NormalizePage(p listdom.Page) listdom.Page {
	if p.Number <= 0 {
		p.Number = 1
	}
	if p.PerPage <= 0 {
		p.PerPage = 20
	}

	return p
}

func TotalPages(totalCount int, perPage int) int {
	if perPage <= 0 || totalCount <= 0 {
		return 0
	}

	return (totalCount + perPage - 1) / perPage
}

func MinInt(a, b int) int {
	if a < b {
		return a
	}

	return b
}

func NonEmpty(v string, fallback string) string {
	if v == "" {
		return fallback
	}

	return v
}

// ============================================================
// InventoryID helpers
// ============================================================

func BuildInventoryID(pbID string, tbID string) string {
	if pbID == "" || tbID == "" {
		return ""
	}

	return pbID + "__" + tbID
}

func ParseInventoryIDStrict(invID string) (pbID string, tbID string, ok bool) {
	if invID == "" {
		return "", "", false
	}
	if !strings.Contains(invID, "__") {
		return "", "", false
	}

	parts := strings.Split(invID, "__")
	if len(parts) < 2 {
		return "", "", false
	}

	pb := parts[0]
	tb := parts[1]
	if pb == "" || tb == "" {
		return "", "", false
	}

	return pb, tb, true
}

// ============================================================
// Small utilities
// ============================================================

func Bool01(b bool) string {
	if b {
		return "1"
	}

	return "0"
}

func Itoa(n int) string {
	if n == 0 {
		return "0"
	}

	neg := false
	if n < 0 {
		neg = true
		n = -n
	}

	var b [32]byte
	i := len(b)

	for n > 0 {
		i--
		b[i] = byte('0' + (n % 10))
		n /= 10
	}

	if neg {
		i--
		b[i] = '-'
	}

	return string(b[i:])
}

func normalizeStringSlice(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}

	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))

	for _, value := range values {
		normalized := strings.TrimSpace(value)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}

		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}

	return result
}

func int64PtrValue(value *int64) int64 {
	if value == nil {
		return 0
	}

	return *value
}
