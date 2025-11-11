// backend/internal/adapters/out/firestore/orderItem_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fscommon "narratives/internal/adapters/out/firestore/common"
	orderitemdom "narratives/internal/domain/orderItem"
)

// Firestore implementation of orderitemdom.RepositoryPort-style interface.
type OrderItemRepositoryFS struct {
	Client *firestore.Client
}

func NewOrderItemRepositoryFS(client *firestore.Client) *OrderItemRepositoryFS {
	return &OrderItemRepositoryFS{Client: client}
}

func (r *OrderItemRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("order_items")
}

// ========================
// RepositoryPort impl
// ========================

func (r *OrderItemRepositoryFS) GetByID(ctx context.Context, id string) (*orderitemdom.OrderItem, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return nil, orderitemdom.ErrNotFound
	}

	snap, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, orderitemdom.ErrNotFound
		}
		return nil, err
	}

	oi, err := docToOrderItem(snap)
	if err != nil {
		return nil, err
	}
	return &oi, nil
}

func (r *OrderItemRepositoryFS) List(
	ctx context.Context,
	filter orderitemdom.Filter,
	sort orderitemdom.Sort,
	page orderitemdom.Page,
) (orderitemdom.PageResult, error) {
	if r.Client == nil {
		return orderitemdom.PageResult{}, errors.New("firestore client is nil")
	}

	pageNum, perPage, offset := fscommon.NormalizePage(page.Number, page.PerPage, 50, 200)

	q := r.col().Query
	q = applyOrderItemSort(q, sort)

	it := q.Documents(ctx)
	defer it.Stop()

	var all []orderitemdom.OrderItem
	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return orderitemdom.PageResult{}, err
		}
		oi, err := docToOrderItem(doc)
		if err != nil {
			return orderitemdom.PageResult{}, err
		}
		if matchOrderItemFilter(oi, filter) {
			all = append(all, oi)
		}
	}

	total := len(all)
	if total == 0 {
		return orderitemdom.PageResult{
			Items:      []orderitemdom.OrderItem{},
			TotalCount: 0,
			TotalPages: 0,
			Page:       pageNum,
			PerPage:    perPage,
		}, nil
	}

	if offset > total {
		offset = total
	}
	end := offset + perPage
	if end > total {
		end = total
	}
	items := all[offset:end]

	return orderitemdom.PageResult{
		Items:      items,
		TotalCount: total,
		TotalPages: fscommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

func (r *OrderItemRepositoryFS) Count(ctx context.Context, filter orderitemdom.Filter) (int, error) {
	if r.Client == nil {
		return 0, errors.New("firestore client is nil")
	}

	it := r.col().Documents(ctx)
	defer it.Stop()

	total := 0
	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, err
		}
		oi, err := docToOrderItem(doc)
		if err != nil {
			return 0, err
		}
		if matchOrderItemFilter(oi, filter) {
			total++
		}
	}
	return total, nil
}

func (r *OrderItemRepositoryFS) Create(ctx context.Context, in orderitemdom.CreateOrderItemInput) (*orderitemdom.OrderItem, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	docRef := r.col().NewDoc()

	oi := orderitemdom.OrderItem{
		ID:          docRef.ID,
		ModelID:     strings.TrimSpace(in.ModelID),
		SaleID:      strings.TrimSpace(in.SaleID),
		InventoryID: strings.TrimSpace(in.InventoryID),
		Quantity:    in.Quantity,
	}

	data := orderItemToDoc(oi)

	if _, err := docRef.Create(ctx, data); err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return nil, orderitemdom.ErrConflict
		}
		return nil, err
	}

	snap, err := docRef.Get(ctx)
	if err != nil {
		return nil, err
	}
	created, err := docToOrderItem(snap)
	if err != nil {
		return nil, err
	}
	return &created, nil
}

func (r *OrderItemRepositoryFS) Update(ctx context.Context, id string, patch orderitemdom.UpdateOrderItemInput) (*orderitemdom.OrderItem, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return nil, orderitemdom.ErrNotFound
	}

	docRef := r.col().Doc(id)
	var updates []firestore.Update

	if patch.ModelID != nil {
		updates = append(updates, firestore.Update{
			Path:  "modelId",
			Value: strings.TrimSpace(*patch.ModelID),
		})
	}
	if patch.SaleID != nil {
		updates = append(updates, firestore.Update{
			Path:  "saleId",
			Value: strings.TrimSpace(*patch.SaleID),
		})
	}
	if patch.InventoryID != nil {
		updates = append(updates, firestore.Update{
			Path:  "inventoryId",
			Value: strings.TrimSpace(*patch.InventoryID),
		})
	}
	if patch.Quantity != nil {
		updates = append(updates, firestore.Update{
			Path:  "quantity",
			Value: *patch.Quantity,
		})
	}

	if len(updates) == 0 {
		// nothing to update; just return current
		return r.GetByID(ctx, id)
	}

	_, err := docRef.Update(ctx, updates)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, orderitemdom.ErrNotFound
		}
		if status.Code(err) == codes.AlreadyExists {
			return nil, orderitemdom.ErrConflict
		}
		return nil, err
	}

	return r.GetByID(ctx, id)
}

func (r *OrderItemRepositoryFS) Delete(ctx context.Context, id string) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return orderitemdom.ErrNotFound
	}

	_, err := r.col().Doc(id).Delete(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return orderitemdom.ErrNotFound
		}
		return err
	}
	return nil
}

func (r *OrderItemRepositoryFS) Reset(ctx context.Context) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	it := r.col().Documents(ctx)
	b := r.Client.Batch()
	count := 0

	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		b.Delete(doc.Ref)
		count++
		if count%400 == 0 {
			if _, err := b.Commit(ctx); err != nil {
				return err
			}
			b = r.Client.Batch()
		}
	}
	if count > 0 {
		if _, err := b.Commit(ctx); err != nil {
			return err
		}
	}
	return nil
}

// ========================
// Helpers
// ========================

func docToOrderItem(doc *firestore.DocumentSnapshot) (orderitemdom.OrderItem, error) {
	data := doc.Data()
	if data == nil {
		return orderitemdom.OrderItem{}, fmt.Errorf("empty order_item document: %s", doc.Ref.ID)
	}

	getStr := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := data[k].(string); ok {
				return strings.TrimSpace(v)
			}
		}
		return ""
	}

	var quantity int
	if v, ok := data["quantity"]; ok {
		switch n := v.(type) {
		case int64:
			quantity = int(n)
		case int:
			quantity = n
		case float64:
			quantity = int(n)
		}
	}

	return orderitemdom.OrderItem{
		ID:          doc.Ref.ID,
		ModelID:     getStr("modelId", "model_id"),
		SaleID:      getStr("saleId", "sale_id"),
		InventoryID: getStr("inventoryId", "inventory_id"),
		Quantity:    quantity,
	}, nil
}

func orderItemToDoc(oi orderitemdom.OrderItem) map[string]any {
	m := map[string]any{
		"modelId":     strings.TrimSpace(oi.ModelID),
		"saleId":      strings.TrimSpace(oi.SaleID),
		"inventoryId": strings.TrimSpace(oi.InventoryID),
		"quantity":    oi.Quantity,
	}
	return m
}

// matchOrderItemFilter applies Filter in-memory (Firestore-friendly mirror of buildOrderItemWhere).
func matchOrderItemFilter(oi orderitemdom.OrderItem, f orderitemdom.Filter) bool {
	if strings.TrimSpace(f.ID) != "" &&
		strings.TrimSpace(oi.ID) != strings.TrimSpace(f.ID) {
		return false
	}
	if strings.TrimSpace(f.ModelID) != "" &&
		strings.TrimSpace(oi.ModelID) != strings.TrimSpace(f.ModelID) {
		return false
	}
	if strings.TrimSpace(f.SaleID) != "" &&
		strings.TrimSpace(oi.SaleID) != strings.TrimSpace(f.SaleID) {
		return false
	}
	if strings.TrimSpace(f.InventoryID) != "" &&
		strings.TrimSpace(oi.InventoryID) != strings.TrimSpace(f.InventoryID) {
		return false
	}

	if f.MinQuantity != nil && oi.Quantity < *f.MinQuantity {
		return false
	}
	if f.MaxQuantity != nil && oi.Quantity > *f.MaxQuantity {
		return false
	}

	return true
}

// applyOrderItemSort maps Sort to Firestore orderBy.
func applyOrderItemSort(q firestore.Query, s orderitemdom.Sort) firestore.Query {
	col := strings.ToLower(strings.TrimSpace(string(s.Column)))
	var field string

	switch col {
	case "id":
		// use DocumentID
		field = ""
	case "quantity":
		field = "quantity"
	default:
		// default: id ASC
		return q.OrderBy(firestore.DocumentID, firestore.Asc)
	}

	dir := firestore.Asc
	if strings.EqualFold(string(s.Order), "desc") {
		dir = firestore.Desc
	}

	if field == "" {
		return q.OrderBy(firestore.DocumentID, dir)
	}
	// stable secondary sort by id
	return q.OrderBy(field, dir).OrderBy(firestore.DocumentID, dir)
}
