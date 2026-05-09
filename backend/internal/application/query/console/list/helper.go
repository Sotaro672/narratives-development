// backend/internal/application/query/console/list/helper.go
package list

import (
	"context"
	"errors"
	"strings"

	querydto "narratives/internal/application/query/console/dto"
	listdom "narratives/internal/domain/list"
	pbpdom "narratives/internal/domain/productBlueprint"
	tbdom "narratives/internal/domain/tokenBlueprint"
)

// ============================================================
// Shared Ports (read-only) - used by detail/management
// ============================================================

type ProductBlueprintGetter interface {
	GetByID(ctx context.Context, id string) (pbpdom.ProductBlueprint, error)
}

type TokenBlueprintGetter interface {
	GetByID(ctx context.Context, id string) (*tbdom.TokenBlueprint, error)
}

type InventoryRowsLister interface {
	ListByCurrentCompany(ctx context.Context) ([]querydto.InventoryManagementRowDTO, error)
}

// productBlueprintPatch reader.
type ProductBlueprintPatchReader interface {
	GetPatchByID(ctx context.Context, id string) (pbpdom.Patch, error)
}

// ============================================================
// Company boundary helpers (exported)
// ============================================================

func AllowedInventoryIDSetFromContext(ctx context.Context, invRows InventoryRowsLister) (map[string]struct{}, error) {
	if invRows == nil {
		return nil, errors.New("inventory rows lister is nil (company boundary via inventory_query is not configured)")
	}

	rows, err := invRows.ListByCurrentCompany(ctx)
	if err != nil {
		return nil, err
	}

	set := map[string]struct{}{}
	for _, r := range rows {
		pbID := r.ProductBlueprintID
		tbID := r.TokenBlueprintID
		if pbID == "" || tbID == "" {
			continue
		}
		invID := pbID + "__" + tbID
		set[invID] = struct{}{}
	}

	return set, nil
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
// Paging helpers (exported)
// ============================================================

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
// InventoryID helpers (exported)
// ============================================================

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
// PriceRows extractor (exported)
// ============================================================

func ExtractPriceRowsFromList(it listdom.List) []any {
	if len(it.Prices) == 0 {
		return nil
	}

	out := make([]any, 0, len(it.Prices))
	for _, row := range it.Prices {
		if row.ModelID == "" {
			continue
		}
		out = append(out, row)
	}

	if len(out) == 0 {
		return nil
	}
	return out
}

// ============================================================
// Small utilities (exported)
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
