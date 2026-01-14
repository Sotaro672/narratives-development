// backend/internal/application/query/mall/order_purchased_query.go
package mall

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

/*
責任と機能:
- avatarId を起点に「購入済み(paid=true)」の orders を検索し、
  items 内の「未transfer(transfer=false 相当)」の (modelId, tokenBlueprintId) を抽出して返す。

今回の実データ前提（あなたの orders.items の形）:
- items[*].modelId が入っている（productId は無い）
- items[*].inventoryId が入っており、以下の shape:
    "<productBlueprintId>__<tokenBlueprintId>__<...>__<modelId>"
  → tokenBlueprintId は 2つめのセグメント（index=1）

そのため、この Query は:
- productId で products/tokens を引きに行かない
- modelId は items[*].modelId をそのまま使う
- tokenBlueprintId は inventoryId の2セグメント目から取り出す
*/

var (
	ErrOrderPurchasedQueryNotConfigured = errors.New("order_purchased_query: not configured")
	ErrInvalidAvatarID                  = errors.New("order_purchased_query: invalid avatarId")
)

type OrderPurchasedQuery struct {
	FS *firestore.Client

	OrdersCol string

	now func() time.Time
}

func NewOrderPurchasedQuery(fs *firestore.Client) *OrderPurchasedQuery {
	return &OrderPurchasedQuery{
		FS:        fs,
		OrdersCol: "orders",
		now:       time.Now,
	}
}

// PurchasedPair is a resolved (modelId, tokenBlueprintId) pair derived from an eligible order item.
// NOTE: ProductID は互換のため残す（実データに無ければ空）。
type PurchasedPair struct {
	OrderID          string `json:"orderId"`
	ProductID        string `json:"productId"`
	ModelID          string `json:"modelId"`
	TokenBlueprintID string `json:"tokenBlueprintId"`
}

// Result is the query output.
// - Pairs は orderId 単位で返す（同一 modelId/tokenBlueprintId が複数回出る可能性あり）
type OrderPurchasedResult struct {
	AvatarID string          `json:"avatarId"`
	Pairs    []PurchasedPair `json:"pairs"`
}

// ListEligiblePairsByAvatarID searches orders by:
// - avatarId == avatarID
// - paid == true
// and filters items by:
// - transfer/transferred == false
// then derives:
// - modelId from items[*].modelId
// - tokenBlueprintId from items[*].inventoryId (2nd segment)
func (q *OrderPurchasedQuery) ListEligiblePairsByAvatarID(ctx context.Context, avatarID string) (OrderPurchasedResult, error) {
	if q == nil || q.FS == nil {
		return OrderPurchasedResult{}, ErrOrderPurchasedQueryNotConfigured
	}

	aid := strings.TrimSpace(avatarID)
	if aid == "" {
		return OrderPurchasedResult{}, ErrInvalidAvatarID
	}

	ordersCol := strings.TrimSpace(q.OrdersCol)
	if ordersCol == "" {
		ordersCol = "orders"
	}

	start := time.Now()
	log.Printf("[order_purchased_query] start avatarId=%s ordersCol=%q", mask(aid), ordersCol)

	// 1) orders where avatarId == aid AND paid == true
	it := q.FS.Collection(ordersCol).
		Where("avatarId", "==", aid).
		Where("paid", "==", true).
		Documents(ctx)
	defer it.Stop()

	var (
		pairs                   []PurchasedPair
		ordersScanned           int
		ordersWithItems         int
		itemsScanned            int
		itemsEligible           int
		itemsNotEligible        int
		itemsMissingModelID     int
		itemsMissingInventoryID int
		pairsAdded              int
	)

	for {
		doc, err := it.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			log.Printf("[order_purchased_query] ERROR: orders iterate failed avatarId=%s err=%v", mask(aid), err)
			return OrderPurchasedResult{}, err
		}
		if doc == nil || doc.Ref == nil {
			continue
		}

		ordersScanned++

		orderID := strings.TrimSpace(doc.Ref.ID)
		raw := doc.Data()
		if raw == nil {
			log.Printf("[order_purchased_query] WARN: order doc data is nil orderId=%s", mask(orderID))
			continue
		}

		// items must exist (array)
		itemsAny, ok := raw["items"]
		if !ok {
			log.Printf("[order_purchased_query] skip: items missing orderId=%s", mask(orderID))
			continue
		}

		items, ok := itemsAny.([]any)
		if !ok || len(items) == 0 {
			log.Printf("[order_purchased_query] skip: items empty or not array orderId=%s type=%T", mask(orderID), itemsAny)
			continue
		}
		ordersWithItems++

		// 2) filter items by transfer=false (or transferred=false)
		for _, one := range items {
			itemsScanned++

			m, ok := one.(map[string]any)
			if !ok || m == nil {
				itemsNotEligible++
				continue
			}

			// transfer flag (compat keys)
			if !isItemUntransferred(m) {
				itemsNotEligible++
				continue
			}

			// ✅ modelId is required (productId is NOT used in your real data)
			modelID := strings.TrimSpace(getString(m, "modelId", "modelID", "model_id"))
			if modelID == "" {
				itemsMissingModelID++
				log.Printf("[order_purchased_query] WARN: eligible item missing modelId orderId=%s", mask(orderID))
				continue
			}

			// ✅ tokenBlueprintId is derived from inventoryId (2nd segment)
			invID := strings.TrimSpace(getString(m, "inventoryId", "inventoryID", "inventory_id"))
			if invID == "" {
				itemsMissingInventoryID++
				log.Printf("[order_purchased_query] WARN: eligible item missing inventoryId orderId=%s modelId=%s", mask(orderID), mask(modelID))
				continue
			}

			tbID := strings.TrimSpace(tokenBlueprintIDFromInventoryID(invID))
			if tbID == "" {
				itemsMissingInventoryID++
				log.Printf("[order_purchased_query] WARN: tokenBlueprintId not derivable from inventoryId orderId=%s modelId=%s inventoryId=%s",
					mask(orderID), mask(modelID), mask(invID))
				continue
			}

			// optional (compat): productId may not exist in real data
			pid := strings.TrimSpace(getString(m, "productId", "productID", "product_id"))

			itemsEligible++
			log.Printf("[order_purchased_query] eligible item orderId=%s modelId=%s tokenBlueprintId=%s productId=%s",
				mask(orderID), mask(modelID), mask(tbID), mask(pid))

			p := PurchasedPair{
				OrderID:          orderID,
				ProductID:        pid,
				ModelID:          modelID,
				TokenBlueprintID: tbID,
			}
			pairs = append(pairs, p)
			pairsAdded++

			log.Printf("[order_purchased_query] pair added orderId=%s modelId=%s tokenBlueprintId=%s productId=%s",
				mask(orderID), mask(modelID), mask(tbID), mask(pid))
		}
	}

	elapsed := time.Since(start)
	log.Printf(
		"[order_purchased_query] done avatarId=%s ordersScanned=%d ordersWithItems=%d itemsScanned=%d itemsEligible=%d itemsNotEligible=%d itemsMissingModelId=%d itemsMissingInventoryId=%d pairs=%d elapsed=%s",
		mask(aid),
		ordersScanned,
		ordersWithItems,
		itemsScanned,
		itemsEligible,
		itemsNotEligible,
		itemsMissingModelID,
		itemsMissingInventoryID,
		len(pairs),
		elapsed.String(),
	)

	return OrderPurchasedResult{
		AvatarID: aid,
		Pairs:    pairs,
	}, nil
}

// ------------------------------------------------------------
// Helpers
// ------------------------------------------------------------

// tokenBlueprintIDFromInventoryID derives tokenBlueprintId from inventoryId.
// Expected shape: "<productBlueprintId>__<tokenBlueprintId>__...__<modelId>"
func tokenBlueprintIDFromInventoryID(inventoryID string) string {
	s := strings.TrimSpace(inventoryID)
	if s == "" {
		return ""
	}
	parts := strings.Split(s, "__")
	if len(parts) < 2 {
		return ""
	}
	// ✅ tokenBlueprintId is the 2nd segment
	tb := strings.TrimSpace(parts[1])
	if tb == "" {
		return ""
	}
	return tb
}

// isItemUntransferred checks item transfer flag.
// Accepts both shapes:
// - item.transfer == false
// - item.transferred == false
// If no flag exists, it returns false (fail-closed).
func isItemUntransferred(item map[string]any) bool {
	// prefer explicit keys
	if v, ok := item["transfer"]; ok {
		return isFalse(v)
	}
	if v, ok := item["transferred"]; ok {
		return isFalse(v)
	}
	if v, ok := item["isTransferred"]; ok {
		// isTransferred=true/false
		return isFalse(v)
	}
	return false
}

func isFalse(v any) bool {
	switch t := v.(type) {
	case bool:
		return t == false
	case string:
		s := strings.ToLower(strings.TrimSpace(t))
		return s == "false" || s == "0" || s == "no"
	case int:
		return t == 0
	case int64:
		return t == 0
	case float64:
		return t == 0
	default:
		return false
	}
}

func getString(m map[string]any, keys ...string) string {
	if m == nil {
		return ""
	}
	for _, k := range keys {
		if v, ok := m[k]; ok {
			s := strings.TrimSpace(fmt.Sprint(v))
			if s != "" && s != "<nil>" {
				return s
			}
		}
	}
	return ""
}
