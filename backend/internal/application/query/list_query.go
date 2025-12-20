// backend/internal/application/query/list_query.go
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

// ✅ NEW: ListCreateSeedDTO は「list新規作成画面」に必要な材料のみを返す DTO。
// - ここでは「画面情報を揃える」だけが責務で、永続化(Create)は usecase に移譲する。
// - prices は [modelId: priceValue] の map を返す（値は初期値 0）。
type ListCreateSeedDTO struct {
	InventoryID        string           `json:"inventoryId"`
	ProductBlueprintID string           `json:"productBlueprintId"`
	TokenBlueprintID   string           `json:"tokenBlueprintId"`
	ProductName        string           `json:"productName"`
	TokenName          string           `json:"tokenName"`
	Prices             map[string]int64 `json:"prices"` // modelId -> price value
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
		pbID, tbID, ok := parseInventoryIDStrict(invID)

		// 入力材料ログ
		log.Printf("[ListQuery] row input listID=%q invID=%q parsed={ok:%v pbID:%q tbID:%q} title=%q assigneeID=%q status=%q",
			id, invID, ok, pbID, tbID, strings.TrimSpace(it.Title), strings.TrimSpace(it.AssigneeID), strings.TrimSpace(string(it.Status)),
		)

		if !ok && invID != "" {
			log.Printf("[ListQuery] WARN inventoryID not parseable (expected {pbId}__{tbId}) invID=%q listID=%q", invID, id)
		}

		// product name resolve
		if ok && pbID != "" && q.nameResolver != nil {
			if cached, ok := productNameCache[pbID]; ok {
				if cached != "" {
					productName = cached
				}
			} else {
				resolved := strings.TrimSpace(q.nameResolver.ResolveProductName(ctx, pbID))
				productNameCache[pbID] = resolved
				if resolved != "" {
					productName = resolved
				}
			}
		}

		// tokenName resolve
		tokenName := ""
		if ok && tbID != "" && q.nameResolver != nil {
			if cached, ok := tokenNameCache[tbID]; ok {
				tokenName = cached
			} else {
				resolved := strings.TrimSpace(q.nameResolver.ResolveTokenName(ctx, tbID))
				tokenNameCache[tbID] = resolved
				tokenName = resolved
			}
		}

		// assigneeName resolve
		assigneeName := ""
		assigneeID := strings.TrimSpace(it.AssigneeID)
		if assigneeID != "" && q.nameResolver != nil {
			if cached, ok := memberNameCache[assigneeID]; ok {
				assigneeName = cached
			} else {
				resolved := strings.TrimSpace(q.nameResolver.ResolveAssigneeName(ctx, assigneeID))
				memberNameCache[assigneeID] = resolved
				assigneeName = resolved
			}
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

		out = append(out, row)
	}

	log.Printf("[ListQuery] ListRows EXIT items=%d page=%d perPage=%d total=%d totalPages=%d",
		len(out), pr.Page, pr.PerPage, pr.TotalCount, pr.TotalPages,
	)

	return listdom.PageResult[ListRowDTO]{
		Items:      out,
		Page:       pr.Page,
		PerPage:    pr.PerPage,
		TotalCount: pr.TotalCount,
		TotalPages: pr.TotalPages,
	}, nil
}

// ✅ NEW: BuildCreateSeed は list新規作成画面に必要な情報を揃えて返します。
// - ここでは永続化(Create)は行いません（usecase に移譲）。
// - inventoryID は "{pbId}__{tbId}" のみ許可します（名揺れ吸収しない）。
// - prices は [modelId: priceValue] の map を返します。初期値は 0。
func (q *ListQuery) BuildCreateSeed(ctx context.Context, inventoryID string, modelIDs []string) (ListCreateSeedDTO, error) {
	inventoryID = strings.TrimSpace(inventoryID)

	pbID, tbID, ok := parseInventoryIDStrict(inventoryID)
	if !ok {
		// ここは "画面情報を揃える" 以前の入力エラーなので明示的にログ
		log.Printf("[ListQuery] BuildCreateSeed invalid inventoryID (expected {pbId}__{tbId}) inventoryID=%q", inventoryID)
		return ListCreateSeedDTO{}, listdom.ErrInvalidInventoryID
	}

	productName := ""
	tokenName := ""

	if q != nil && q.nameResolver != nil {
		productName = strings.TrimSpace(q.nameResolver.ResolveProductName(ctx, pbID))
		tokenName = strings.TrimSpace(q.nameResolver.ResolveTokenName(ctx, tbID))
	}

	// prices: modelId -> 0 (初期値)
	prices := map[string]int64{}
	for _, mid := range modelIDs {
		mid = strings.TrimSpace(mid)
		if mid == "" {
			continue
		}
		// 重複は上書きでOK（同じキーは1つ）
		prices[mid] = 0
	}

	out := ListCreateSeedDTO{
		InventoryID:        inventoryID,
		ProductBlueprintID: pbID,
		TokenBlueprintID:   tbID,
		ProductName:        productName,
		TokenName:          tokenName,
		Prices:             prices,
	}

	log.Printf("[ListQuery] BuildCreateSeed ok inventoryID=%q pbID=%q tbID=%q modelIDs=%d pricesKeys=%d",
		inventoryID, pbID, tbID, len(modelIDs), len(prices),
	)

	return out, nil
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

// parseInventoryIDStrict は List.InventoryID を厳密にパースします。
// 期待： "{pbId}__{tbId}"
// 名揺れ吸収はしません（正規フォーマットのみ許可）。
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
