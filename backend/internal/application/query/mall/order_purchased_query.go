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
  items 内の「未transfer(transfer=false 相当)」の productId を抽出する。
- product テーブル(products/{productId})から modelId を取得する。
- token テーブル(tokens/{productId})から tokenBlueprintId を取得する。
- 上記を組み合わせて「modelId と tokenBlueprintId の組み合わせ」を返す。

前提（今回の方針）:
- products/{productId} に modelId がある
- tokens/{productId} に tokenBlueprintId がある（docId = productId）
*/

var (
	ErrOrderPurchasedQueryNotConfigured = errors.New("order_purchased_query: not configured")
	ErrInvalidAvatarID                  = errors.New("order_purchased_query: invalid avatarId")
)

type OrderPurchasedQuery struct {
	FS *firestore.Client

	OrdersCol   string
	ProductsCol string
	TokensCol   string

	now func() time.Time
}

func NewOrderPurchasedQuery(fs *firestore.Client) *OrderPurchasedQuery {
	return &OrderPurchasedQuery{
		FS:          fs,
		OrdersCol:   "orders",
		ProductsCol: "products",
		TokensCol:   "tokens",
		now:         time.Now,
	}
}

// PurchasedPair is a resolved (modelId, tokenBlueprintId) pair derived from an eligible order item.
type PurchasedPair struct {
	OrderID          string `json:"orderId"`
	ProductID        string `json:"productId"`
	ModelID          string `json:"modelId"`
	TokenBlueprintID string `json:"tokenBlueprintId"`
}

// Result is the query output.
// - Pairs は orderId/productId 単位で返す（同一 modelId/tokenBlueprintId が複数回出る可能性あり）
type OrderPurchasedResult struct {
	AvatarID string          `json:"avatarId"`
	Pairs    []PurchasedPair `json:"pairs"`
}

// ListEligiblePairsByAvatarID searches orders by:
// - avatarId == avatarID
// - paid == true
// and filters items by:
// - transfer/transferred == false
// then resolves modelId and tokenBlueprintId using products/tokens tables.
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
	productsCol := strings.TrimSpace(q.ProductsCol)
	if productsCol == "" {
		productsCol = "products"
	}
	tokensCol := strings.TrimSpace(q.TokensCol)
	if tokensCol == "" {
		tokensCol = "tokens"
	}

	start := time.Now()
	log.Printf("[order_purchased_query] start avatarId=%s ordersCol=%q productsCol=%q tokensCol=%q", mask(aid), ordersCol, productsCol, tokensCol)

	// 1) orders where avatarId == aid AND paid == true
	it := q.FS.Collection(ordersCol).
		Where("avatarId", "==", aid).
		Where("paid", "==", true).
		Documents(ctx)
	defer it.Stop()

	// cache to reduce reads
	modelByProduct := map[string]string{}
	tbByProduct := map[string]string{}

	var (
		pairs              []PurchasedPair
		ordersScanned      int
		ordersWithItems    int
		itemsScanned       int
		itemsEligible      int
		itemsMissingPID    int
		itemsNotEligible   int
		productsFetched    int
		tokensFetched      int
		productResolveFail int
		tokenResolveFail   int
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

			pid := strings.TrimSpace(getString(m, "productId", "productID", "product_id"))
			if pid == "" {
				itemsMissingPID++
				log.Printf("[order_purchased_query] WARN: eligible item missing productId orderId=%s", mask(orderID))
				continue
			}

			itemsEligible++
			log.Printf("[order_purchased_query] eligible item orderId=%s productId=%s", mask(orderID), mask(pid))

			// 3) product -> modelId
			modelID, ok := modelByProduct[pid]
			if !ok {
				mid, rerr := q.resolveModelIDByProductID(ctx, productsCol, pid)
				if rerr != nil {
					productResolveFail++
					log.Printf("[order_purchased_query] WARN: resolve modelId failed orderId=%s productId=%s err=%v", mask(orderID), mask(pid), rerr)
					continue
				}
				modelID = mid
				modelByProduct[pid] = modelID
				productsFetched++
				log.Printf("[order_purchased_query] resolved modelId orderId=%s productId=%s modelId=%s (fetched)", mask(orderID), mask(pid), mask(modelID))
			} else {
				log.Printf("[order_purchased_query] resolved modelId orderId=%s productId=%s modelId=%s (cache)", mask(orderID), mask(pid), mask(modelID))
			}
			if strings.TrimSpace(modelID) == "" {
				productResolveFail++
				log.Printf("[order_purchased_query] WARN: modelId empty after resolve orderId=%s productId=%s", mask(orderID), mask(pid))
				continue
			}

			// 4) token -> tokenBlueprintId (docId = productId)
			tbID, ok := tbByProduct[pid]
			if !ok {
				tbid, rerr := q.resolveTokenBlueprintIDByProductID(ctx, tokensCol, pid)
				if rerr != nil {
					tokenResolveFail++
					log.Printf("[order_purchased_query] WARN: resolve tokenBlueprintId failed orderId=%s productId=%s err=%v", mask(orderID), mask(pid), rerr)
					continue
				}
				tbID = tbid
				tbByProduct[pid] = tbID
				tokensFetched++
				log.Printf("[order_purchased_query] resolved tokenBlueprintId orderId=%s productId=%s tokenBlueprintId=%s (fetched)", mask(orderID), mask(pid), mask(tbID))
			} else {
				log.Printf("[order_purchased_query] resolved tokenBlueprintId orderId=%s productId=%s tokenBlueprintId=%s (cache)", mask(orderID), mask(pid), mask(tbID))
			}
			if strings.TrimSpace(tbID) == "" {
				tokenResolveFail++
				log.Printf("[order_purchased_query] WARN: tokenBlueprintId empty after resolve orderId=%s productId=%s", mask(orderID), mask(pid))
				continue
			}

			p := PurchasedPair{
				OrderID:          orderID,
				ProductID:        pid,
				ModelID:          modelID,
				TokenBlueprintID: tbID,
			}
			pairs = append(pairs, p)

			log.Printf("[order_purchased_query] pair added orderId=%s productId=%s modelId=%s tokenBlueprintId=%s",
				mask(orderID), mask(pid), mask(modelID), mask(tbID))
		}
	}

	elapsed := time.Since(start)
	log.Printf(
		"[order_purchased_query] done avatarId=%s ordersScanned=%d ordersWithItems=%d itemsScanned=%d itemsEligible=%d itemsNotEligible=%d itemsMissingProductId=%d pairs=%d productsFetched=%d tokensFetched=%d productResolveFail=%d tokenResolveFail=%d elapsed=%s",
		mask(aid),
		ordersScanned,
		ordersWithItems,
		itemsScanned,
		itemsEligible,
		itemsNotEligible,
		itemsMissingPID,
		len(pairs),
		productsFetched,
		tokensFetched,
		productResolveFail,
		tokenResolveFail,
		elapsed.String(),
	)

	return OrderPurchasedResult{
		AvatarID: aid,
		Pairs:    pairs,
	}, nil
}

// ------------------------------------------------------------
// Resolvers (Firestore)
// ------------------------------------------------------------

func (q *OrderPurchasedQuery) resolveModelIDByProductID(ctx context.Context, productsCol string, productID string) (string, error) {
	pid := strings.TrimSpace(productID)
	if pid == "" {
		return "", errors.New("productId is empty")
	}

	snap, err := q.FS.Collection(productsCol).Doc(pid).Get(ctx)
	if err != nil {
		return "", err
	}
	raw := snap.Data()
	if raw == nil {
		return "", fmt.Errorf("product doc is empty (productId=%s)", pid)
	}

	modelID := strings.TrimSpace(getString(raw, "modelId", "modelID", "model_id"))
	if modelID == "" {
		return "", fmt.Errorf("modelId not found in product (productId=%s)", pid)
	}
	return modelID, nil
}

func (q *OrderPurchasedQuery) resolveTokenBlueprintIDByProductID(ctx context.Context, tokensCol string, productID string) (string, error) {
	pid := strings.TrimSpace(productID)
	if pid == "" {
		return "", errors.New("productId is empty")
	}

	// tokens/{productId}
	snap, err := q.FS.Collection(tokensCol).Doc(pid).Get(ctx)
	if err != nil {
		return "", err
	}
	raw := snap.Data()
	if raw == nil {
		return "", fmt.Errorf("token doc is empty (productId=%s)", pid)
	}

	tbID := strings.TrimSpace(getString(raw, "tokenBlueprintId", "tokenBlueprintID", "token_blueprint_id"))
	if tbID == "" {
		return "", fmt.Errorf("tokenBlueprintId not found in token (productId=%s)", pid)
	}
	return tbID, nil
}

// ------------------------------------------------------------
// Helpers
// ------------------------------------------------------------

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
