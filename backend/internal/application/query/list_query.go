package query

import (
	"context"
	"log"
	"strings"

	resolver "narratives/internal/application/resolver"
	listdom "narratives/internal/domain/list"
)

// ============================================================
// Ports (read-only)
// ============================================================

// ListLister は lists の一覧取得（ページング付き）を行う最小契約です。
type ListLister interface {
	List(ctx context.Context, filter listdom.Filter, sort listdom.Sort, page listdom.Page) (listdom.PageResult[listdom.List], error)
}

// ============================================================
// DTO (query -> handler)
// ============================================================

// ListRowDTO はフロントの listManagement に渡す 1 行分（最小）
// - 画面要件: 左から productName, tokenName, assigneeName, status(出品中|保留中)
// - status はフロントで normalize できるように raw のまま返します
type ListRowDTO struct {
	ID           string `json:"id"`
	ProductName  string `json:"productName"`
	TokenName    string `json:"tokenName"`
	AssigneeName string `json:"assigneeName"`
	Status       string `json:"status"`
}

// ============================================================
// ListQuery
// ============================================================

type ListQuery struct {
	lister       ListLister
	nameResolver *resolver.NameResolver
}

func NewListQuery(lister ListLister, nameResolver *resolver.NameResolver) *ListQuery {
	return &ListQuery{lister: lister, nameResolver: nameResolver}
}

// ListRows は lists 一覧を取得し、tokenName / assigneeName / productName を解決して返します。
func (q *ListQuery) ListRows(ctx context.Context, filter listdom.Filter, sort listdom.Sort, page listdom.Page) (listdom.PageResult[ListRowDTO], error) {
	// nil ガード
	if q == nil || q.lister == nil {
		log.Printf("[ListQuery] WARN ListRows called but q or lister is nil (q=%v listerNil=%v)", q != nil, q == nil || q.lister == nil)
		return listdom.PageResult[ListRowDTO]{
			Items:      []ListRowDTO{},
			Page:       page.Number,
			PerPage:    page.PerPage,
			TotalCount: 0,
			TotalPages: 0,
		}, nil
	}

	// ✅ 入口ログ（filter/page）
	// NOTE: listdom.Filter には InventoryIDs が無いので参照しない
	log.Printf("[ListQuery] ListRows ENTER page=%d perPage=%d filter={q:%q assigneeID:%v status:%v statuses:%d deleted:%v modelNumbers:%d}",
		page.Number,
		page.PerPage,
		strings.TrimSpace(filter.SearchQuery),
		ptrStr(filter.AssigneeID),
		ptrStatus(filter.Status),
		len(filter.Statuses),
		ptrBool(filter.Deleted),
		len(filter.ModelNumbers),
	)

	pr, err := q.lister.List(ctx, filter, sort, page)
	if err != nil {
		log.Printf("[ListQuery] ERROR lister.List failed: %v", err)
		return listdom.PageResult[ListRowDTO]{}, err
	}
	if pr.Items == nil {
		pr.Items = []listdom.List{}
	}

	// request-scope cache（同じIDの多重解決を防ぐ）
	tokenNameCache := map[string]string{}
	memberNameCache := map[string]string{}
	productNameCache := map[string]string{}

	out := make([]ListRowDTO, 0, len(pr.Items))

	for _, it := range pr.Items {
		id := strings.TrimSpace(it.ID)

		// productName:
		// - 基本は title を採用（Create で title を入れているため最低限表示できる）
		productName := strings.TrimSpace(it.Title)

		invID := strings.TrimSpace(it.InventoryID)
		pbID, tbID, ok := parseInventoryIDForListQuery(invID)

		// ✅ 入力材料ログ（tokenName が空の原因特定用）
		log.Printf("[ListQuery] row input listID=%q invID=%q parsed={ok:%v pbID:%q tbID:%q} title=%q assigneeID=%q status=%q",
			id, invID, ok, pbID, tbID, strings.TrimSpace(it.Title), strings.TrimSpace(it.AssigneeID), strings.TrimSpace(string(it.Status)),
		)

		// 解析できない場合はログで可視化（tokenName が空の原因特定用）
		if !ok && invID != "" {
			log.Printf("[ListQuery] WARN inventoryID not parseable invID=%q listID=%q", invID, id)
		}

		// product name resolve
		if ok && pbID != "" && q.nameResolver != nil {
			if cached, ok := productNameCache[pbID]; ok {
				if cached != "" {
					productName = cached
				}
				log.Printf("[ListQuery] productName cache pbID=%q -> %q", pbID, cached)
			} else {
				resolved := strings.TrimSpace(q.nameResolver.ResolveProductName(ctx, pbID))
				productNameCache[pbID] = resolved
				log.Printf("[ListQuery] productName resolved pbID=%q -> %q", pbID, resolved)
				if resolved != "" {
					productName = resolved
				}
			}
		} else {
			log.Printf("[ListQuery] productName skipped ok=%v pbID=%q nameResolverNil=%v", ok, pbID, q.nameResolver == nil)
		}

		// tokenName resolve
		tokenName := ""
		if ok && tbID != "" && q.nameResolver != nil {
			if cached, ok := tokenNameCache[tbID]; ok {
				tokenName = cached
				log.Printf("[ListQuery] tokenName cache tbID=%q -> %q", tbID, cached)
			} else {
				resolved := strings.TrimSpace(q.nameResolver.ResolveTokenName(ctx, tbID))
				tokenNameCache[tbID] = resolved
				tokenName = resolved
				log.Printf("[ListQuery] tokenName resolved tbID=%q -> %q", tbID, resolved)
			}
		} else {
			// ✅ tokenName が空の「理由」
			log.Printf("[ListQuery] tokenName skipped ok=%v tbID=%q nameResolverNil=%v (invID=%q listID=%q)",
				ok, tbID, q.nameResolver == nil, invID, id,
			)
		}

		// assigneeName resolve
		assigneeName := ""
		assigneeID := strings.TrimSpace(it.AssigneeID)
		if assigneeID != "" && q.nameResolver != nil {
			if cached, ok := memberNameCache[assigneeID]; ok {
				assigneeName = cached
				log.Printf("[ListQuery] assigneeName cache assigneeID=%q -> %q", assigneeID, cached)
			} else {
				resolved := strings.TrimSpace(q.nameResolver.ResolveAssigneeName(ctx, assigneeID))
				memberNameCache[assigneeID] = resolved
				assigneeName = resolved
				log.Printf("[ListQuery] assigneeName resolved assigneeID=%q -> %q", assigneeID, resolved)
			}
		} else {
			log.Printf("[ListQuery] assigneeName skipped assigneeID=%q nameResolverNil=%v", assigneeID, q.nameResolver == nil)
		}
		if assigneeName == "" {
			assigneeName = "未設定"
		}

		row := ListRowDTO{
			ID:           nonEmpty(id, "(missing id)"),
			ProductName:  strings.TrimSpace(productName),
			TokenName:    strings.TrimSpace(tokenName),
			AssigneeName: assigneeName,
			Status:       strings.TrimSpace(string(it.Status)),
		}

		// ✅ 画面へ渡す 1 行分の最終ログ
		log.Printf("[ListQuery] row output listID=%q productName=%q tokenName=%q assigneeName=%q status=%q",
			row.ID, row.ProductName, row.TokenName, row.AssigneeName, row.Status,
		)

		out = append(out, row)
	}

	// ✅ まとめログ（空が多いかを見る）
	emptyToken := 0
	for _, r := range out {
		if strings.TrimSpace(r.TokenName) == "" {
			emptyToken++
		}
	}
	log.Printf("[ListQuery] ListRows EXIT items=%d emptyTokenName=%d page=%d perPage=%d total=%d totalPages=%d",
		len(out), emptyToken, pr.Page, pr.PerPage, pr.TotalCount, pr.TotalPages,
	)

	return listdom.PageResult[ListRowDTO]{
		Items:      out,
		Page:       pr.Page,
		PerPage:    pr.PerPage,
		TotalCount: pr.TotalCount,
		TotalPages: pr.TotalPages,
	}, nil
}

// ============================================================
// helpers
// ============================================================

func nonEmpty(v string, fallback string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return fallback
	}
	return v
}

func ptrStr(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

func ptrBool(p *bool) any {
	if p == nil {
		return nil
	}
	return *p
}

func ptrStatus(p *listdom.ListStatus) any {
	if p == nil {
		return nil
	}
	return string(*p)
}

// parseInventoryIDForListQuery は List.InventoryID の表記揺れに強いパーサです。
// 期待： "{pbId}__{tbId}"
// 実際：pbId / tbId 単体や、区切りが違うケースを拾えるようにしておく。
func parseInventoryIDForListQuery(invID string) (pbID string, tbID string, ok bool) {
	invID = strings.TrimSpace(invID)
	if invID == "" {
		return "", "", false
	}

	// 1) 正：pb__tb
	if strings.Contains(invID, "__") {
		parts := strings.Split(invID, "__")
		// pb__tb__... のように増えても先頭2つを採用
		if len(parts) >= 2 {
			pb := strings.TrimSpace(parts[0])
			tb := strings.TrimSpace(parts[1])
			if pb != "" && tb != "" {
				return pb, tb, true
			}
		}
	}

	// 2) 念のため：pb|tb
	if strings.Contains(invID, "|") {
		parts := strings.Split(invID, "|")
		if len(parts) >= 2 {
			pb := strings.TrimSpace(parts[0])
			tb := strings.TrimSpace(parts[1])
			if pb != "" && tb != "" {
				return pb, tb, true
			}
		}
	}

	// 3) ここまで来るなら「解析できない」扱い
	return "", "", false
}
