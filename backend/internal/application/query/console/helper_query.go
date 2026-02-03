// backend/internal/application/query/console/helper_query.go
package query

import (
	"reflect"
	"strings"

	querydto "narratives/internal/application/query/console/dto"
)

// ✅ listDetail / listManagement 向けに移管した exported 関数は削除
// - AllowedInventoryIDSetFromContext
// - InventoryAllowed
// - ParseInventoryIDStrict
// - ExtractPriceRowsFromList
// - ReadStringField / ReadIntField / ReadIntPtrField
// - Bool01 / Itoa
// - normalizePage / totalPages / minInt / nonEmpty（listManagement 側で list/helper.go に移管）
//
// このファイルには「query パッケージ内でのみ使う汎用ヘルパ」を残す。

// dummy ref to keep imported package in this helpers file minimal
var _ = querydto.InventoryDetailDTO{}

// (optional) generic getter used by some query files; keep here if referenced elsewhere
func getStringFieldAny(v any, names ...string) string {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return ""
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return ""
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return ""
	}

	for _, n := range names {
		f := rv.FieldByName(n)
		if !f.IsValid() {
			continue
		}

		switch f.Kind() {
		case reflect.String:
			return strings.TrimSpace(f.String())

		case reflect.Pointer:
			if f.IsNil() {
				continue
			}
			fe := f.Elem()
			if fe.IsValid() && fe.Kind() == reflect.String {
				return strings.TrimSpace(fe.String())
			}
		}
	}

	return ""
}
