// backend/internal/application/query/console/order_management_query.go
package query

//
// 機能: OrderManagementQuery (console)
//   - currentCompany 境界（inventory_query 相当）で許可された inventoryId のみを対象に
//     Order.Items[].InventoryID をフラットに列挙する
//   - orderRepository の List をスキャンし、allowed items を集約してから再ページングする
//
// 目的:
// - order テーブルの items に記載された inventoryId を、company 境界に従って安全に一覧できるようにする
//
// ✅ DI整合のための方針:
//   - Firestore OrderRepositoryFS は usecase.OrderFilter / common.Sort / common.Page を引数に取るため、
//     Query側の port もそれに合わせる（domain/order.Filter は使わない）
import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	querydto "narratives/internal/application/query/console/dto"
	uc "narratives/internal/application/usecase"
	common "narratives/internal/domain/common"
	orderdom "narratives/internal/domain/order"
)

// ============================================================
// Ports (read-only)
// ============================================================

// ✅ Firestore repo (usecase.OrderRepo) の List シグネチャに合わせる
type OrderLister interface {
	List(ctx context.Context, filter uc.OrderFilter, sort common.Sort, page common.Page) (common.PageResult[orderdom.Order], error)
}

type InventoryRowsLister interface {
	ListByCurrentCompany(ctx context.Context) ([]querydto.InventoryManagementRowDTO, error)
}

// ============================================================
// DTO
// ============================================================

// OrderItemInventoryRowDTO
// - Order.Items をフラット化した 1行 DTO
// - UI はこれをテーブル表示すればよい
type OrderItemInventoryRowDTO struct {
	OrderID string `json:"orderId"`

	UserID   string `json:"userId,omitempty"`
	AvatarID string `json:"avatarId,omitempty"`
	CartID   string `json:"cartId,omitempty"`

	// order-level
	Paid      bool   `json:"paid"`
	CreatedAt string `json:"createdAt,omitempty"` // RFC3339(UTC)

	// item-level
	InventoryID string `json:"inventoryId"`
	ListID      string `json:"listId,omitempty"`
	ModelID     string `json:"modelId,omitempty"`
	Qty         int    `json:"qty,omitempty"`
	Price       int    `json:"price,omitempty"`

	Transferred   bool   `json:"transferred"`
	TransferredAt string `json:"transferredAt,omitempty"` // RFC3339(UTC)
}

// （任意）inventoryId だけ欲しい画面向け（distinct）
type InventoryIDDTO struct {
	InventoryID string `json:"inventoryId"`
}

// ============================================================
// Query
// ============================================================

type OrderManagementQuery struct {
	lister  OrderLister
	invRows InventoryRowsLister // REQUIRED
}

type NewOrderManagementQueryParams struct {
	Lister  OrderLister
	InvRows InventoryRowsLister // REQUIRED
}

func NewOrderManagementQuery(p NewOrderManagementQueryParams) *OrderManagementQuery {
	return &OrderManagementQuery{
		lister:  p.Lister,
		invRows: p.InvRows,
	}
}

// ============================================================
// Public APIs
// ============================================================

// ListItemInventoryRows
// - order をスキャンし、items をフラット化して返す
// - company boundary により許可された inventoryId のみ返す
//
// ページング方針:
// - repository.List は “order単位” でページングされるため、items単位で正しいページを作るには
//  1. orderをスキャン
//  2. allowed item を集約
//  3. item単位で再ページング
//     が必要（listManagement と同様の安全側実装）
func (q *OrderManagementQuery) ListItemInventoryRows(
	ctx context.Context,
	filter uc.OrderFilter,
	sort common.Sort,
	page common.Page,
) (common.PageResult[OrderItemInventoryRowDTO], error) {

	page = normalizePage(page)

	if q == nil || q.lister == nil || q.invRows == nil {
		return common.PageResult[OrderItemInventoryRowDTO]{}, errors.New("OrderManagementQuery.ListItemInventoryRows: wiring is incomplete (lister/invRows required)")
	}

	allowedSet, err := allowedInventoryIDSetFromContext(ctx, q.invRows)
	if err != nil {
		log.Printf("[OrderManagementQuery] ERROR company boundary (inventory_query) failed: %v", err)
		return common.PageResult[OrderItemInventoryRowDTO]{}, err
	}
	if len(allowedSet) == 0 {
		return common.PageResult[OrderItemInventoryRowDTO]{
			Items:      []OrderItemInventoryRowDTO{},
			Page:       page.Number,
			PerPage:    page.PerPage,
			TotalCount: 0,
			TotalPages: 0,
		}, nil
	}

	allowedAll := make([]OrderItemInventoryRowDTO, 0, page.PerPage)

	// スキャン上限（無限ループや巨大テーブルを避ける）
	const maxScanPages = 500
	srcPage := 1

	for {
		if srcPage > maxScanPages {
			log.Printf("[OrderManagementQuery] WARN scan page limit reached (max=%d). results may be truncated.", maxScanPages)
			break
		}

		pr, e := q.lister.List(ctx, filter, sort, common.Page{Number: srcPage, PerPage: page.PerPage})
		if e != nil {
			log.Printf("[OrderManagementQuery] ERROR lister.List failed (scan page=%d): %v", srcPage, e)
			return common.PageResult[OrderItemInventoryRowDTO]{}, e
		}
		if pr.Items == nil {
			pr.Items = []orderdom.Order{}
		}

		for _, ord := range pr.Items {
			orderID := nonEmpty(ord.ID, "(missing order id)")

			createdAt := ""
			if !ord.CreatedAt.IsZero() {
				createdAt = ord.CreatedAt.UTC().Format(time.RFC3339)
			}

			userID := strings.TrimSpace(ord.UserID)
			avatarID := strings.TrimSpace(ord.AvatarID)
			cartID := strings.TrimSpace(ord.CartID)

			// items をフラット化
			for _, it := range ord.Items {
				invID := strings.TrimSpace(it.InventoryID)
				if !inventoryAllowed(allowedSet, invID) {
					continue
				}

				transferredAt := ""
				if it.TransferredAt != nil && !it.TransferredAt.IsZero() {
					transferredAt = it.TransferredAt.UTC().Format(time.RFC3339)
				}

				allowedAll = append(allowedAll, OrderItemInventoryRowDTO{
					OrderID: orderID,

					UserID:   userID,
					AvatarID: avatarID,
					CartID:   cartID,

					Paid:      ord.Paid,
					CreatedAt: createdAt,

					InventoryID: invID,
					ListID:      strings.TrimSpace(it.ListID),
					ModelID:     strings.TrimSpace(it.ModelID),
					Qty:         it.Qty,
					Price:       it.Price,

					Transferred:   it.Transferred,
					TransferredAt: transferredAt,
				})
			}
		}

		// 終端判定
		if len(pr.Items) == 0 {
			break
		}
		if pr.TotalPages > 0 {
			if srcPage >= pr.TotalPages {
				break
			}
		} else {
			// TotalPages が未設定の場合のフォールバック
			if len(pr.Items) < page.PerPage {
				break
			}
		}

		srcPage++
	}

	// item単位で再ページング
	totalCount := len(allowedAll)
	tp := totalPages(totalCount, page.PerPage)

	start := (page.Number - 1) * page.PerPage
	if start < 0 {
		start = 0
	}
	if start >= totalCount {
		return common.PageResult[OrderItemInventoryRowDTO]{
			Items:      []OrderItemInventoryRowDTO{},
			Page:       page.Number,
			PerPage:    page.PerPage,
			TotalCount: totalCount,
			TotalPages: tp,
		}, nil
	}
	end := minInt(start+page.PerPage, totalCount)

	return common.PageResult[OrderItemInventoryRowDTO]{
		Items:      allowedAll[start:end],
		Page:       page.Number,
		PerPage:    page.PerPage,
		TotalCount: totalCount,
		TotalPages: tp,
	}, nil
}

// ListDistinctInventoryIDs
// - 許可された inventoryId を distinct で返す（※このメソッド内では「返却ページ内でdistinct」）
// - 画面が「inventoryId一覧」だけ欲しい場合に使用
func (q *OrderManagementQuery) ListDistinctInventoryIDs(
	ctx context.Context,
	filter uc.OrderFilter,
	sort common.Sort,
	page common.Page,
) (common.PageResult[InventoryIDDTO], error) {

	pr, err := q.ListItemInventoryRows(ctx, filter, sort, page)
	if err != nil {
		return common.PageResult[InventoryIDDTO]{}, err
	}

	seen := map[string]struct{}{}
	out := make([]InventoryIDDTO, 0, len(pr.Items))
	for _, row := range pr.Items {
		id := strings.TrimSpace(row.InventoryID)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, InventoryIDDTO{InventoryID: id})
	}

	return common.PageResult[InventoryIDDTO]{
		Items:      out,
		Page:       pr.Page,
		PerPage:    pr.PerPage,
		TotalCount: len(out),
		TotalPages: totalPages(len(out), pr.PerPage),
	}, nil
}

// ============================================================
// local helpers
// ============================================================

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

func normalizePage(p common.Page) common.Page {
	if p.Number <= 0 {
		p.Number = 1
	}
	if p.PerPage <= 0 {
		p.PerPage = 20
	}
	return p
}

func totalPages(totalCount int, perPage int) int {
	if perPage <= 0 || totalCount <= 0 {
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
