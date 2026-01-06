// backend\internal\application\query\console\helper_query.go
package query

import (
	"context"
	"errors"
	"reflect"
	"strings"

	querydto "narratives/internal/application/query/console/dto"
	listdom "narratives/internal/domain/list"
)

func allowedInventoryIDSetFromContext(ctx context.Context, invRows InventoryRowsLister) (map[string]struct{}, error) {
	if invRows == nil {
		return nil, errors.New("inventory rows lister is nil (company boundary via inventory_query is not configured)")
	}

	rows, err := invRows.ListByCurrentCompany(ctx)
	if err != nil {
		return nil, err
	}

	set := map[string]struct{}{}
	for _, r := range rows {
		pbID := strings.TrimSpace(r.ProductBlueprintID)
		tbID := strings.TrimSpace(r.TokenBlueprintID)
		if pbID == "" || tbID == "" {
			continue
		}
		invID := pbID + "__" + tbID
		set[invID] = struct{}{}
	}
	return set, nil
}

func inventoryAllowed(set map[string]struct{}, inventoryID string) bool {
	if len(set) == 0 {
		return false
	}
	id := strings.TrimSpace(inventoryID)
	if id == "" {
		return false
	}
	_, ok := set[id]
	return ok
}

func normalizePage(p listdom.Page) listdom.Page {
	if p.Number <= 0 {
		p.Number = 1
	}
	if p.PerPage <= 0 {
		p.PerPage = 20
	}
	return p
}

func totalPages(totalCount int, perPage int) int {
	if perPage <= 0 {
		return 0
	}
	if totalCount <= 0 {
		return 0
	}
	return (totalCount + perPage - 1) / perPage
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func nonEmpty(v string, fallback string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return fallback
	}
	return v
}

func parseInventoryIDStrict(invID string) (pbID string, tbID string, ok bool) {
	invID = strings.TrimSpace(invID)
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
	pb := strings.TrimSpace(parts[0])
	tb := strings.TrimSpace(parts[1])
	if pb == "" || tb == "" {
		return "", "", false
	}
	return pb, tb, true
}

// ------------------------------------------------------------
// PriceRows extractor (reflect)
// ------------------------------------------------------------

func extractPriceRowsFromList(it listdom.List) []any {
	rv := reflect.ValueOf(it)
	rv = deref(rv)
	if !rv.IsValid() || rv.Kind() != reflect.Struct {
		return nil
	}

	if f := rv.FieldByName("PriceRows"); f.IsValid() {
		if out := sliceToAny(f); len(out) > 0 {
			return out
		}
	}

	if f := rv.FieldByName("Prices"); f.IsValid() {
		if out := sliceToAny(f); len(out) > 0 {
			return out
		}
		if out := mapPricesToAnyRows(f); len(out) > 0 {
			return out
		}
	}

	return nil
}

func mapPricesToAnyRows(v reflect.Value) []any {
	v = deref(v)
	if !v.IsValid() || v.Kind() != reflect.Map {
		return nil
	}
	if v.Type().Key().Kind() != reflect.String {
		return nil
	}

	out := make([]any, 0, v.Len())
	iter := v.MapRange()
	for iter.Next() {
		k := iter.Key()
		val := iter.Value()

		modelID := ""
		if k.IsValid() && k.Kind() == reflect.String {
			modelID = strings.TrimSpace(k.String())
		}
		if modelID == "" {
			continue
		}

		priceInt := 0
		if n, ok := asInt(deref(val)); ok {
			priceInt = n
		}

		out = append(out, map[string]any{
			"ModelID": modelID,
			"Price":   priceInt,
		})
	}

	return out
}

func sliceToAny(v reflect.Value) []any {
	v = deref(v)
	if !v.IsValid() || v.Kind() != reflect.Slice {
		return nil
	}
	out := make([]any, 0, v.Len())
	for i := 0; i < v.Len(); i++ {
		out = append(out, v.Index(i).Interface())
	}
	return out
}

func readStringField(v any, fieldNames ...string) string {
	rv := reflect.ValueOf(v)
	rv = deref(rv)
	if !rv.IsValid() {
		return ""
	}

	if rv.Kind() == reflect.Struct {
		for _, fn := range fieldNames {
			f := rv.FieldByName(fn)
			f = deref(f)
			if f.IsValid() && f.Kind() == reflect.String {
				return f.String()
			}
		}
		return ""
	}

	if rv.Kind() == reflect.Map && rv.Type().Key().Kind() == reflect.String {
		for _, fn := range fieldNames {
			mv := rv.MapIndex(reflect.ValueOf(fn))
			mv = deref(mv)
			if mv.IsValid() && mv.Kind() == reflect.String {
				return mv.String()
			}
		}
		return ""
	}

	return ""
}

func readIntField(v any, fieldNames ...string) int {
	rv := reflect.ValueOf(v)
	rv = deref(rv)
	if !rv.IsValid() {
		return 0
	}

	if rv.Kind() == reflect.Struct {
		for _, fn := range fieldNames {
			f := rv.FieldByName(fn)
			f = deref(f)
			if f.IsValid() {
				if n, ok := asInt(f); ok {
					return n
				}
			}
		}
		return 0
	}

	if rv.Kind() == reflect.Map && rv.Type().Key().Kind() == reflect.String {
		for _, fn := range fieldNames {
			mv := rv.MapIndex(reflect.ValueOf(fn))
			mv = deref(mv)
			if mv.IsValid() {
				if n, ok := asInt(mv); ok {
					return n
				}
			}
		}
		return 0
	}

	return 0
}

func readIntPtrField(v any, fieldNames ...string) *int {
	rv := reflect.ValueOf(v)
	rv = deref(rv)
	if !rv.IsValid() {
		return nil
	}

	if rv.Kind() == reflect.Struct {
		for _, fn := range fieldNames {
			f := rv.FieldByName(fn)
			f = deref(f)
			if f.IsValid() {
				if n, ok := asInt(f); ok {
					x := n
					return &x
				}
			}
		}
		return nil
	}

	if rv.Kind() == reflect.Map && rv.Type().Key().Kind() == reflect.String {
		for _, fn := range fieldNames {
			mv := rv.MapIndex(reflect.ValueOf(fn))
			mv = deref(mv)
			if mv.IsValid() {
				if n, ok := asInt(mv); ok {
					x := n
					return &x
				}
			}
		}
		return nil
	}

	return nil
}

func deref(v reflect.Value) reflect.Value {
	if !v.IsValid() {
		return v
	}
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return reflect.Value{}
		}
		v = v.Elem()
	}
	return v
}

func asInt(v reflect.Value) (int, bool) {
	if !v.IsValid() {
		return 0, false
	}
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return int(v.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int(v.Uint()), true
	case reflect.Float32, reflect.Float64:
		return int(v.Float()), true
	default:
		return 0, false
	}
}

func bool01(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

func itoa(n int) string {
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

// dummy ref to keep imported package in this helpers file minimal
var _ = querydto.InventoryDetailDTO{}
