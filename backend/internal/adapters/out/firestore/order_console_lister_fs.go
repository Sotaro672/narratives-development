// backend/internal/adapters/out/firestore/order_console_lister_fs.go
package firestore

import (
	"context"
	"errors"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	fscommon "narratives/internal/adapters/out/firestore/common"
	applicationport "narratives/internal/application/port"
	common "narratives/internal/domain/common"
	orderdom "narratives/internal/domain/order"
)

// OrderConsoleListerFS is a console/query-side order lister.
//
// NOTE:
// This is intentionally separated from OrderRepositoryFS.
//   - OrderRepositoryFS is used by mall/usecase flows and exposes ListByAvatarID only.
//   - OrderConsoleListerFS is used by console read models that need broader search.
//
// Company-bound inventory filtering is intentionally handled by
// OrderManagementQuery at item level.
type OrderConsoleListerFS struct {
	Client *firestore.Client
}

func NewOrderConsoleListerFS(client *firestore.Client) *OrderConsoleListerFS {
	return &OrderConsoleListerFS{Client: client}
}

func (r *OrderConsoleListerFS) ordersCol() *firestore.CollectionRef {
	return r.Client.Collection("orders")
}

// ListByInventoryIDs lists orders for console query processing.
//
// NOTE:
// allowedInventoryIDs and filter are intentionally kept in the signature for
// OrderLister interface compatibility.
// This lister does not apply company-bound inventory filtering.
// OrderManagementQuery applies allowed inventory filtering at item level.
func (r *OrderConsoleListerFS) ListByInventoryIDs(
	ctx context.Context,
	allowedInventoryIDs map[string]struct{},
	filter orderdom.Filter,
	sort common.Sort,
	page common.Page,
) (common.PageResult[orderdom.Order], error) {
	if r == nil || r.Client == nil {
		return common.PageResult[orderdom.Order]{}, errors.New("firestore client is nil")
	}

	page = normalizeOrderConsolePage(page)
	pageNum, perPage, offset := fscommon.NormalizePage(page.Number, page.PerPage, 50, 200)

	q := r.ordersCol().Query
	q = applyOrderSort(q, sort)

	it := q.Documents(ctx)
	defer it.Stop()

	targetEnd := offset + perPage
	matched := make([]orderdom.Order, 0, targetEnd)

	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return common.PageResult[orderdom.Order]{}, err
		}

		o, err := docToOrder(doc)
		if err != nil {
			return common.PageResult[orderdom.Order]{}, err
		}

		matched = append(matched, o)
	}

	total := len(matched)
	if total == 0 {
		return common.PageResult[orderdom.Order]{
			Items:      []orderdom.Order{},
			TotalCount: 0,
			TotalPages: 0,
			Page:       pageNum,
			PerPage:    perPage,
		}, nil
	}

	if offset > total {
		offset = total
	}

	end := targetEnd
	if end > total {
		end = total
	}

	return common.PageResult[orderdom.Order]{
		Items:      matched[offset:end],
		TotalCount: total,
		TotalPages: fscommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

// ListByListIDs returns cumulative paid order statistics for the requested lists.
//
// Aggregation rules:
// - Paid == true orders only.
// - type == "list" items only.
// - cancelled items are excluded.
// - items outside allowedInventoryIDs are excluded.
// - TotalOrderCount counts each Order only once per List.
// - TotalSalesAmount is the sum of Price * Qty and excludes tax and shipping.
func (r *OrderConsoleListerFS) ListByListIDs(
	ctx context.Context,
	listIDs []string,
	allowedInventoryIDs map[string]struct{},
) (
	map[string]applicationport.ListSalesSummary,
	error,
) {
	if r == nil || r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	result := make(
		map[string]applicationport.ListSalesSummary,
	)

	if len(listIDs) == 0 || len(allowedInventoryIDs) == 0 {
		return result, nil
	}

	targetListIDs := make(
		map[string]struct{},
		len(listIDs),
	)

	for _, listID := range listIDs {
		if listID == "" {
			continue
		}

		targetListIDs[listID] = struct{}{}
	}

	if len(targetListIDs) == 0 {
		return result, nil
	}

	it := r.ordersCol().Documents(ctx)
	defer it.Stop()

	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		o, err := docToOrder(doc)
		if err != nil {
			return nil, err
		}

		if !o.Paid {
			continue
		}

		countedListIDs := make(
			map[string]struct{},
		)

		for _, item := range o.Items {
			if item.Type != orderdom.OrderItemTypeList {
				continue
			}

			if item.IsCancelled {
				continue
			}

			if item.ListID == "" || item.InventoryID == "" {
				continue
			}

			if _, ok := targetListIDs[item.ListID]; !ok {
				continue
			}

			if _, ok := allowedInventoryIDs[item.InventoryID]; !ok {
				continue
			}

			summary := result[item.ListID]

			summary.TotalSalesAmount +=
				int64(item.Price) *
					int64(item.Qty)

			if _, counted := countedListIDs[item.ListID]; !counted {
				summary.TotalOrderCount++
				countedListIDs[item.ListID] = struct{}{}
			}

			result[item.ListID] = summary
		}
	}

	return result, nil
}

func normalizeOrderConsolePage(p common.Page) common.Page {
	if p.Number <= 0 {
		p.Number = 1
	}

	if p.PerPage <= 0 {
		p.PerPage = 20
	}

	if p.PerPage > 200 {
		p.PerPage = 200
	}

	return p
}
