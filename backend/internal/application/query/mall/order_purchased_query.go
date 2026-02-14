// backend/internal/application/query/mall/order_purchased_query.go
package mall

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
)

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
// - transferred == false (backend/internal/domain/order/entity.go の transferred:bool のみを正とする)
// then derives:
// - modelId from items[*].modelId
// - tokenBlueprintId from items[*].inventoryId (2nd segment)
func (q *OrderPurchasedQuery) ListEligiblePairsByAvatarID(ctx context.Context, avatarID string) (OrderPurchasedResult, error) {
	if q == nil || q.FS == nil {
		return OrderPurchasedResult{}, ErrOrderPurchasedQueryNotConfigured
	}

	aid := avatarID
	if aid == "" {
		return OrderPurchasedResult{}, ErrInvalidAvatarID
	}

	ordersCol := q.OrdersCol
	if ordersCol == "" {
		ordersCol = "orders"
	}

	start := time.Now()
	log.Printf("[order_purchased_query] start avatarId=%s ordersCol=%q", aid, ordersCol)

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
			log.Printf("[order_purchased_query] ERROR: orders iterate failed avatarId=%s err=%v", aid, err)
			return OrderPurchasedResult{}, err
		}
		if doc == nil || doc.Ref == nil {
			continue
		}

		ordersScanned++

		orderID := doc.Ref.ID
		raw := doc.Data()
		if raw == nil {
			log.Printf("[order_purchased_query] WARN: order doc data is nil orderId=%s", orderID)
			continue
		}

		// items must exist (array)
		itemsAny, ok := raw["items"]
		if !ok {
			log.Printf("[order_purchased_query] skip: items missing orderId=%s", orderID)
			continue
		}

		items, ok := itemsAny.([]any)
		if !ok || len(items) == 0 {
			log.Printf("[order_purchased_query] skip: items empty or not array orderId=%s type=%T", orderID, itemsAny)
			continue
		}
		ordersWithItems++

		// 2) filter items by transferred == false (bool only; fail-closed)
		for _, one := range items {
			itemsScanned++

			m, ok := one.(map[string]any)
			if !ok || m == nil {
				itemsNotEligible++
				continue
			}

			// ✅ Only trust "transferred" bool from domain order entity
			tv, ok := m["transferred"]
			if !ok {
				// fail-closed
				itemsNotEligible++
				continue
			}
			transferred, ok := tv.(bool)
			if !ok {
				// bool only; fail-closed
				itemsNotEligible++
				continue
			}
			if transferred {
				itemsNotEligible++
				continue
			}

			// ✅ modelId is required（正スキーマ: modelId）
			modelID, _ := m["modelId"].(string)
			if modelID == "" {
				itemsMissingModelID++
				log.Printf("[order_purchased_query] WARN: eligible item missing modelId orderId=%s", orderID)
				continue
			}

			// ✅ tokenBlueprintId is derived from inventoryId (2nd segment)
			invID, _ := m["inventoryId"].(string)
			if invID == "" {
				itemsMissingInventoryID++
				log.Printf("[order_purchased_query] WARN: eligible item missing inventoryId orderId=%s modelId=%s", orderID, modelID)
				continue
			}

			parts := strings.Split(invID, "__")
			if len(parts) < 2 || parts[1] == "" {
				itemsMissingInventoryID++
				log.Printf("[order_purchased_query] WARN: tokenBlueprintId not derivable from inventoryId orderId=%s modelId=%s inventoryId=%s",
					orderID, modelID, invID)
				continue
			}
			tbID := parts[1]

			// optional (compat): productId may not exist in real data
			pid, _ := m["productId"].(string)

			itemsEligible++
			log.Printf("[order_purchased_query] eligible item orderId=%s modelId=%s tokenBlueprintId=%s productId=%s",
				orderID, modelID, tbID, pid)

			p := PurchasedPair{
				OrderID:          orderID,
				ProductID:        pid,
				ModelID:          modelID,
				TokenBlueprintID: tbID,
			}
			pairs = append(pairs, p)
			pairsAdded++

			log.Printf("[order_purchased_query] pair added orderId=%s modelId=%s tokenBlueprintId=%s productId=%s",
				orderID, modelID, tbID, pid)
		}
	}

	elapsed := time.Since(start)
	log.Printf(
		"[order_purchased_query] done avatarId=%s ordersScanned=%d ordersWithItems=%d itemsScanned=%d itemsEligible=%d itemsNotEligible=%d itemsMissingModelId=%d itemsMissingInventoryId=%d pairs=%d elapsed=%s",
		aid,
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
