// backend/internal/application/query/console/list/detail/list_image_urls.go
//
// 機能: ListImage(サブコレクション) から ImageURLs を生成する
// 責任:
// - Firestore records(ListImage) を listID で取得し、URL候補を組み立てる
// - displayOrder/createdAt/id で安定ソートし、URL重複を排除する
// - primaryImageID が一致する画像があれば先頭に配置する
//
// NOTE:
// - imgLister が nil の場合は空配列を返し、画面を壊さない
package detail

import (
	"context"
	"os"
	"reflect"
	"sort"
	"strings"
	"time"

	listimgdom "narratives/internal/domain/listImage"
)

type listImageURLRow struct {
	id        string
	url       string
	order     int
	createdAt time.Time
}

func (q *ListDetailQuery) buildListImageURLs(ctx context.Context, listID string, primaryImageID string) []string {
	_ = ctx // 将来: signed-url 生成等で使う可能性があるため残す

	if q == nil || q.imgLister == nil {
		return []string{}
	}

	lid := strings.TrimSpace(listID)
	if lid == "" {
		return []string{}
	}

	items, err := q.imgLister.ListByListID(ctx, lid)
	if err != nil || len(items) == 0 {
		return []string{}
	}

	rows := make([]listImageURLRow, 0, len(items))

	// bucket fallback: env only (avoid domain constant dependency)
	envBucket := strings.TrimSpace(os.Getenv("LIST_IMAGE_BUCKET"))

	for _, it := range items {
		id := strings.TrimSpace(readStringFieldAny(it, "ID", "Id", "ImageID", "ImageId"))
		u := strings.TrimSpace(readStringFieldAny(it, "PublicURL", "PublicUrl", "URL", "Url", "SignedURL", "SignedUrl"))
		b := strings.TrimSpace(readStringFieldAny(it, "Bucket", "bucket"))
		op := strings.TrimLeft(strings.TrimSpace(readStringFieldAny(it, "ObjectPath", "objectPath", "Path", "path")), "/")

		// If URL missing, try to build from bucket+objectPath
		if u == "" && op != "" {
			if b == "" {
				b = envBucket
			}
			if b != "" {
				u = "https://storage.googleapis.com/" + b + "/" + op
			}
		}

		// If still empty, best-effort: try domain PublicURL only when bucket exists
		// (keeps compatibility if caller provides bucket but not url)
		if u == "" && op != "" && b != "" {
			u = listimgdom.PublicURL(b, op)
		}

		if u == "" {
			continue
		}

		order := readIntFieldAny(it, "DisplayOrder", "displayOrder", "Order", "order")
		ca := readTimeFieldAny(it, "CreatedAt", "createdAt")

		rows = append(rows, listImageURLRow{
			id:        id,
			url:       u,
			order:     order,
			createdAt: ca,
		})
	}

	if len(rows) == 0 {
		return []string{}
	}

	// sort: displayOrder asc -> createdAt asc -> id
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].order != rows[j].order {
			return rows[i].order < rows[j].order
		}
		if !rows[i].createdAt.Equal(rows[j].createdAt) {
			// zero time は後ろへ
			if rows[i].createdAt.IsZero() && !rows[j].createdAt.IsZero() {
				return false
			}
			if !rows[i].createdAt.IsZero() && rows[j].createdAt.IsZero() {
				return true
			}
			return rows[i].createdAt.Before(rows[j].createdAt)
		}
		return rows[i].id < rows[j].id
	})

	// dedupe by url (keep order)
	seen := map[string]bool{}
	out := make([]string, 0, len(rows))
	primaryURL := ""
	wantPrimary := strings.TrimSpace(primaryImageID)

	for _, r := range rows {
		u := strings.TrimSpace(r.url)
		if u == "" || seen[u] {
			continue
		}
		seen[u] = true

		if wantPrimary != "" && strings.TrimSpace(r.id) == wantPrimary && primaryURL == "" {
			primaryURL = u
			continue
		}
		out = append(out, u)
	}

	// primary を先頭に
	if primaryURL != "" {
		return append([]string{primaryURL}, out...)
	}
	return out
}

// --- reflection helpers (ListImage のフィールド名差分に強くする) ---

func readStringFieldAny(v any, names ...string) string {
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

	for _, name := range names {
		f := rv.FieldByName(name)
		if !f.IsValid() || !f.CanInterface() {
			continue
		}

		switch f.Kind() {
		case reflect.String:
			s := strings.TrimSpace(f.String())
			if s != "" {
				return s
			}
		case reflect.Pointer:
			if f.IsNil() {
				continue
			}
			fe := f.Elem()
			if fe.IsValid() && fe.Kind() == reflect.String {
				s := strings.TrimSpace(fe.String())
				if s != "" {
					return s
				}
			}
		}
	}

	return ""
}

func readIntFieldAny(v any, names ...string) int {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return 0
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return 0
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return 0
	}

	for _, name := range names {
		f := rv.FieldByName(name)
		if !f.IsValid() || !f.CanInterface() {
			continue
		}
		switch f.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return int(f.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return int(f.Uint())
		}
	}
	return 0
}

func readTimeFieldAny(v any, names ...string) time.Time {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return time.Time{}
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return time.Time{}
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return time.Time{}
	}

	for _, name := range names {
		f := rv.FieldByName(name)
		if !f.IsValid() || !f.CanInterface() {
			continue
		}
		if t, ok := f.Interface().(time.Time); ok {
			return t
		}
	}
	return time.Time{}
}
