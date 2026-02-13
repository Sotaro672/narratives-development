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
	"sort"
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
	if listID == "" {
		return []string{}
	}

	items, err := q.imgLister.ListByListID(ctx, listID)
	if err != nil || len(items) == 0 {
		return []string{}
	}

	rows := make([]listImageURLRow, 0, len(items))

	// bucket fallback: env only (avoid domain constant dependency)
	envBucket := os.Getenv("LIST_IMAGE_BUCKET")

	for _, it := range items {
		id := it.ID
		u := it.URL
		op := trimLeftSlash(it.ObjectPath)

		// If URL missing, try to build from envBucket + objectPath
		if u == "" && op != "" && envBucket != "" {
			u = listimgdom.PublicURL(envBucket, op)
		}

		if u == "" {
			continue
		}

		order := it.DisplayOrder

		rows = append(rows, listImageURLRow{
			id:        id,
			url:       u,
			order:     order,
			createdAt: time.Time{}, // ListImage に CreatedAt が無いので安定ソート用にゼロ固定
		})
	}

	if len(rows) == 0 {
		return []string{}
	}

	// sort: displayOrder asc -> createdAt asc -> id
	// createdAt は常にゼロのため、実質 displayOrder -> id の安定ソートになる
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].order != rows[j].order {
			return rows[i].order < rows[j].order
		}
		if !rows[i].createdAt.Equal(rows[j].createdAt) {
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

	for _, r := range rows {
		u := r.url
		if u == "" || seen[u] {
			continue
		}
		seen[u] = true

		if primaryImageID != "" && r.id == primaryImageID && primaryURL == "" {
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

func trimLeftSlash(s string) string {
	for len(s) > 0 && s[0] == '/' {
		s = s[1:]
	}
	return s
}
