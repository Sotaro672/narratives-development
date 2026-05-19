// backend/internal/application/query/mall/order_purchased_query.go
package mall

import (
	"context"
	"errors"
	"strings"

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
}

func NewOrderPurchasedQuery(fs *firestore.Client) *OrderPurchasedQuery {
	return &OrderPurchasedQuery{
		FS:        fs,
		OrdersCol: "orders",
	}
}

// PurchasedPair is a resolved (modelId, tokenBlueprintId) pair derived from an eligible order item.
type PurchasedPair struct {
	OrderID          string `json:"orderId"`
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

	it := q.FS.Collection(ordersCol).
		Where("avatarId", "==", aid).
		Where("paid", "==", true).
		Documents(ctx)
	defer it.Stop()

	var pairs []PurchasedPair

	for {
		doc, err := it.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}
			return OrderPurchasedResult{}, err
		}
		if doc == nil || doc.Ref == nil {
			continue
		}

		orderID := doc.Ref.ID
		raw := doc.Data()
		if raw == nil {
			continue
		}

		itemsAny, ok := raw["items"]
		if !ok {
			continue
		}

		items, ok := itemsAny.([]any)
		if !ok || len(items) == 0 {
			continue
		}

		for _, one := range items {
			m, ok := one.(map[string]any)
			if !ok || m == nil {
				continue
			}

			tv, ok := m["transferred"]
			if !ok {
				continue
			}

			transferred, ok := tv.(bool)
			if !ok {
				continue
			}
			if transferred {
				continue
			}

			modelID, _ := m["modelId"].(string)
			if modelID == "" {
				continue
			}

			invID, _ := m["inventoryId"].(string)
			if invID == "" {
				continue
			}

			parts := strings.Split(invID, "__")
			if len(parts) < 2 || parts[1] == "" {
				continue
			}
			tbID := parts[1]

			pairs = append(pairs, PurchasedPair{
				OrderID:          orderID,
				ModelID:          modelID,
				TokenBlueprintID: tbID,
			})
		}
	}

	return OrderPurchasedResult{
		AvatarID: aid,
		Pairs:    pairs,
	}, nil
}
