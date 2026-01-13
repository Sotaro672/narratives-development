// backend/internal/adapters/out/firestore/order_query_fs.go
package firestore

import (
	"reflect"
	"strings"
	"time"

	"cloud.google.com/go/firestore"

	uc "narratives/internal/application/usecase"
	common "narratives/internal/domain/common"
	orderdom "narratives/internal/domain/order"
)

// --- filter/sort helpers ---

func applyOrderSort(q firestore.Query, sort common.Sort) firestore.Query {
	col := strings.ToLower(strings.TrimSpace(string(sort.Column)))

	// entity.go に合わせて createdAt のみ許可
	field := ""
	switch col {
	case "createdat", "created_at", "created":
		field = "createdAt"
	default:
		// default: newest first
		return q.OrderBy("createdAt", firestore.Desc).
			OrderBy(firestore.DocumentID, firestore.Desc)
	}

	dir := firestore.Desc
	if strings.EqualFold(string(sort.Order), "asc") {
		dir = firestore.Asc
	}

	return q.OrderBy(field, dir).OrderBy(firestore.DocumentID, dir)
}

// matchOrderFilter is reflection-based so adapter compiles even if uc.OrderFilter shape changes.
// It tries to apply: ID, UserID, AvatarID, CartID, CreatedFrom/CreatedTo.
func matchOrderFilter(o orderdom.Order, f uc.OrderFilter) bool {
	return matchOrderFilterAny(o, any(f))
}

func matchOrderFilterAny(o orderdom.Order, fv any) bool {
	// ID
	if id, ok := getFilterString(fv, "ID"); ok {
		if strings.TrimSpace(id) != "" && strings.TrimSpace(o.ID) != strings.TrimSpace(id) {
			return false
		}
	}
	// UserID
	if uid, ok := getFilterString(fv, "UserID"); ok {
		if strings.TrimSpace(uid) != "" && strings.TrimSpace(o.UserID) != strings.TrimSpace(uid) {
			return false
		}
	}

	// AvatarID (filter 側の命名揺れも吸収)
	if aid, ok := getFilterString(fv, "AvatarID"); ok {
		if strings.TrimSpace(aid) != "" && strings.TrimSpace(o.AvatarID) != strings.TrimSpace(aid) {
			return false
		}
	} else if aid, ok := getFilterString(fv, "AvatarId"); ok {
		if strings.TrimSpace(aid) != "" && strings.TrimSpace(o.AvatarID) != strings.TrimSpace(aid) {
			return false
		}
	}

	// CartID
	if cid, ok := getFilterString(fv, "CartID"); ok {
		if strings.TrimSpace(cid) != "" && strings.TrimSpace(o.CartID) != strings.TrimSpace(cid) {
			return false
		}
	}

	// CreatedFrom / CreatedTo
	if from, ok := getFilterTimePtr(fv, "CreatedFrom"); ok && from != nil {
		if o.CreatedAt.IsZero() || o.CreatedAt.Before(from.UTC()) {
			return false
		}
	}
	if to, ok := getFilterTimePtr(fv, "CreatedTo"); ok && to != nil {
		// "to" は Upper bound exclusive に寄せる（以前の実装踏襲）
		if o.CreatedAt.IsZero() || !o.CreatedAt.Before(to.UTC()) {
			return false
		}
	}

	return true
}

func getFilterTimePtr(v any, field string) (*time.Time, bool) {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return nil, false
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil, false
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil, false
	}
	f := rv.FieldByName(field)
	if !f.IsValid() {
		// lowerFirst は helper_repository_fs.go の実装を利用する
		f = rv.FieldByName(lowerFirst(field))
		if !f.IsValid() {
			return nil, false
		}
	}

	// *time.Time
	if f.Kind() == reflect.Pointer {
		if f.IsNil() {
			return nil, true
		}
		if t, ok := f.Interface().(*time.Time); ok {
			return t, true
		}
	}
	// time.Time
	if f.CanInterface() {
		if t, ok := f.Interface().(time.Time); ok {
			return &t, true
		}
	}
	return nil, false
}
