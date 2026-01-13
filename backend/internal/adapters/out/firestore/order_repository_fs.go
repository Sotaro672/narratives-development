// backend/internal/adapters/out/firestore/order_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fscommon "narratives/internal/adapters/out/firestore/common"
	uc "narratives/internal/application/usecase"
	common "narratives/internal/domain/common"
	orderdom "narratives/internal/domain/order"
)

// Firestore implementation of usecase.OrderRepo
type OrderRepositoryFS struct {
	Client *firestore.Client
}

func NewOrderRepositoryFS(client *firestore.Client) *OrderRepositoryFS {
	return &OrderRepositoryFS{Client: client}
}

func (r *OrderRepositoryFS) ordersCol() *firestore.CollectionRef {
	return r.Client.Collection("orders")
}

// ========================
// RepositoryPort impl
// ========================

func (r *OrderRepositoryFS) GetByID(ctx context.Context, id string) (orderdom.Order, error) {
	if r.Client == nil {
		return orderdom.Order{}, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return orderdom.Order{}, orderdom.ErrNotFound
	}

	snap, err := r.ordersCol().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return orderdom.Order{}, orderdom.ErrNotFound
		}
		return orderdom.Order{}, err
	}

	o, err := docToOrder(snap) // ✅ order_mapper_fs.go を利用
	if err != nil {
		return orderdom.Order{}, err
	}
	return o, nil
}

// Exists is optional (dev/testing convenience)
func (r *OrderRepositoryFS) Exists(ctx context.Context, id string) (bool, error) {
	if r.Client == nil {
		return false, errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return false, nil
	}

	_, err := r.ordersCol().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *OrderRepositoryFS) List(
	ctx context.Context,
	filter uc.OrderFilter,
	sort common.Sort,
	page common.Page,
) (common.PageResult[orderdom.Order], error) {
	if r.Client == nil {
		return common.PageResult[orderdom.Order]{}, errors.New("firestore client is nil")
	}

	pageNum, perPage, offset := fscommon.NormalizePage(page.Number, page.PerPage, 50, 200)

	q := r.ordersCol().Query
	q = applyOrderSort(q, sort) // ✅ order_query_fs.go に存在するものを利用

	it := q.Documents(ctx)
	defer it.Stop()

	var all []orderdom.Order
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
		if matchOrderFilter(o, filter) { // ✅ order_query_fs.go に存在するものを利用
			all = append(all, o)
		}
	}

	total := len(all)
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
	end := offset + perPage
	if end > total {
		end = total
	}
	items := all[offset:end]

	return common.PageResult[orderdom.Order]{
		Items:      items,
		TotalCount: total,
		TotalPages: fscommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

// ListByCursor: ordered by ID ASC, cursor = last ID.
func (r *OrderRepositoryFS) ListByCursor(
	ctx context.Context,
	filter uc.OrderFilter,
	_ common.Sort,
	cpage common.CursorPage,
) (common.CursorPageResult[orderdom.Order], error) {
	if r.Client == nil {
		return common.CursorPageResult[orderdom.Order]{}, errors.New("firestore client is nil")
	}

	limit := cpage.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	q := r.ordersCol().OrderBy(firestore.DocumentID, firestore.Asc)
	if after := strings.TrimSpace(cpage.After); after != "" {
		q = q.StartAfter(after)
	}

	it := q.Documents(ctx)
	defer it.Stop()

	var (
		items []orderdom.Order
		last  string
	)
	for {
		if len(items) >= limit {
			break
		}
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return common.CursorPageResult[orderdom.Order]{}, err
		}

		o, err := docToOrder(doc)
		if err != nil {
			return common.CursorPageResult[orderdom.Order]{}, err
		}
		if matchOrderFilter(o, filter) {
			items = append(items, o)
			last = o.ID
		}
	}

	var next *string
	if last != "" && len(items) == limit {
		next = &last
	}

	return common.CursorPageResult[orderdom.Order]{
		Items:      items,
		NextCursor: next,
		Limit:      limit,
	}, nil
}

func (r *OrderRepositoryFS) Count(ctx context.Context, filter uc.OrderFilter) (int, error) {
	if r.Client == nil {
		return 0, errors.New("firestore client is nil")
	}

	it := r.ordersCol().Documents(ctx)
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
		o, err := docToOrder(doc)
		if err != nil {
			return 0, err
		}
		if matchOrderFilter(o, filter) {
			total++
		}
	}
	return total, nil
}

func (r *OrderRepositoryFS) Create(ctx context.Context, o orderdom.Order) (orderdom.Order, error) {
	if r.Client == nil {
		return orderdom.Order{}, errors.New("firestore client is nil")
	}

	id := strings.TrimSpace(o.ID)
	now := time.Now().UTC()
	if o.CreatedAt.IsZero() {
		o.CreatedAt = now
	}

	// ✅ 起票時は必ず paid=false（orderレベル）
	o.Paid = false

	// ✅ item-level transferred defaults（安全側で初期化）
	for i := range o.Items {
		o.Items[i].Transferred = false
		o.Items[i].TransferredAt = nil
	}

	var docRef *firestore.DocumentRef
	if id == "" {
		docRef = r.ordersCol().NewDoc()
		o.ID = docRef.ID
	} else {
		docRef = r.ordersCol().Doc(id)
		o.ID = id
	}

	data := orderToDoc(o) // ✅ mapper を利用（item transferred を含む）

	_, err := docRef.Create(ctx, data)
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return orderdom.Order{}, orderdom.ErrConflict
		}
		return orderdom.Order{}, err
	}

	snap, err := docRef.Get(ctx)
	if err != nil {
		return orderdom.Order{}, err
	}
	out, err := docToOrder(snap)
	if err != nil {
		return orderdom.Order{}, err
	}
	return out, nil
}

func (r *OrderRepositoryFS) Save(ctx context.Context, o orderdom.Order, _ *common.SaveOptions) (orderdom.Order, error) {
	if r.Client == nil {
		return orderdom.Order{}, errors.New("firestore client is nil")
	}

	id := strings.TrimSpace(o.ID)
	now := time.Now().UTC()

	// upsert (new doc)
	if id == "" {
		if o.CreatedAt.IsZero() {
			o.CreatedAt = now
		}

		// ✅ upsert(new) も起票扱い: paid=false
		o.Paid = false

		// ✅ item defaults
		for i := range o.Items {
			o.Items[i].Transferred = false
			o.Items[i].TransferredAt = nil
		}

		docRef := r.ordersCol().NewDoc()
		o.ID = docRef.ID
		if _, err := docRef.Set(ctx, orderToDoc(o)); err != nil {
			return orderdom.Order{}, err
		}
		snap, err := docRef.Get(ctx)
		if err != nil {
			return orderdom.Order{}, err
		}
		return docToOrder(snap)
	}

	o.ID = id

	// preserve CreatedAt if missing
	if o.CreatedAt.IsZero() {
		if snap, err := r.ordersCol().Doc(id).Get(ctx); err == nil {
			if existing, err2 := docToOrder(snap); err2 == nil && !existing.CreatedAt.IsZero() {
				o.CreatedAt = existing.CreatedAt
			}
		}
	}
	if o.CreatedAt.IsZero() {
		o.CreatedAt = now
	}

	docRef := r.ordersCol().Doc(id)
	data := orderToDoc(o)

	_, err := docRef.Set(ctx, data, firestore.MergeAll)
	if err != nil {
		return orderdom.Order{}, err
	}

	snap, err := docRef.Get(ctx)
	if err != nil {
		return orderdom.Order{}, err
	}
	return docToOrder(snap)
}

func (r *OrderRepositoryFS) Delete(ctx context.Context, id string) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return orderdom.ErrNotFound
	}

	_, err := r.ordersCol().Doc(id).Delete(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return orderdom.ErrNotFound
		}
		return err
	}
	return nil
}

// Reset deletes all orders using Transactions instead of WriteBatch.
func (r *OrderRepositoryFS) Reset(ctx context.Context) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	it := r.ordersCol().Documents(ctx)
	defer it.Stop()

	var refs []*firestore.DocumentRef
	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		refs = append(refs, doc.Ref)
	}

	if len(refs) == 0 {
		return nil
	}

	const chunkSize = 400
	for start := 0; start < len(refs); start += chunkSize {
		end := start + chunkSize
		if end > len(refs) {
			end = len(refs)
		}
		chunk := refs[start:end]

		err := r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
			for _, ref := range chunk {
				if err := tx.Delete(ref); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	log.Printf("[order_repo_fs] Reset OK deleted=%d", len(refs))
	return nil
}

// ============================================================
// ✅ Transfer flow helpers (item-level transferred)
// ============================================================

// ListPaidUntransferredByAvatarID returns orders where:
// - avatarId == avatarID
// - paid == true
// - and (in application-side check) at least one item has transferred == false
//
// NOTE: Firestore cannot filter "array element has field == false" without restructuring data.
// So we fetch paid orders by avatarId and then filter in memory.
func (r *OrderRepositoryFS) ListPaidUntransferredByAvatarID(ctx context.Context, avatarID string, limit int) ([]orderdom.Order, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	avatarID = strings.TrimSpace(avatarID)
	if avatarID == "" {
		return []orderdom.Order{}, nil
	}

	if limit <= 0 || limit > 200 {
		limit = 50
	}

	q := r.ordersCol().
		Where("avatarId", "==", avatarID).
		Where("paid", "==", true).
		OrderBy("createdAt", firestore.Desc).
		OrderBy(firestore.DocumentID, firestore.Desc).
		Limit(limit)

	it := q.Documents(ctx)
	defer it.Stop()

	out := make([]orderdom.Order, 0, 10)
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

		// in-memory filter: at least one item not transferred
		if hasUntransferredItem(o) {
			out = append(out, o)
		}
	}
	return out, nil
}

func hasUntransferredItem(o orderdom.Order) bool {
	for _, it := range o.Items {
		if !it.Transferred {
			return true
		}
	}
	return false
}

// LockTransferItem atomically checks eligibility and sets item-level:
// - items[itemIndex].transferred = true
// - items[itemIndex].transferredAt = now
//
// Eligibility:
// - paid == true
// - item exists
// - items[itemIndex].transferred == false
func (r *OrderRepositoryFS) LockTransferItem(ctx context.Context, orderID string, itemIndex int, now time.Time) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return orderdom.ErrNotFound
	}
	if itemIndex < 0 {
		return fmt.Errorf("invalid itemIndex: %d", itemIndex)
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	docRef := r.ordersCol().Doc(orderID)

	return r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(docRef)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return orderdom.ErrNotFound
			}
			return err
		}

		paid, _ := mapGetBool(snap.Data(), "paid")
		if !paid {
			return fmt.Errorf("order %s: not paid", orderID)
		}

		o, err := docToOrder(snap)
		if err != nil {
			return err
		}

		if itemIndex >= len(o.Items) {
			return fmt.Errorf("order %s: itemIndex out of range: %d", orderID, itemIndex)
		}
		if o.Items[itemIndex].Transferred {
			return fmt.Errorf("order %s: item already transferred (idx=%d)", orderID, itemIndex)
		}

		o.Items[itemIndex].Transferred = true
		t := now.UTC()
		o.Items[itemIndex].TransferredAt = &t

		updates := []firestore.Update{
			{Path: "items", Value: orderToDoc(o)["items"]},
		}
		return tx.Update(docRef, updates)
	})
}

// UnlockTransferItem rolls back item-level lock by setting:
// - items[itemIndex].transferred = false
// - items[itemIndex].transferredAt deleted (nil)
func (r *OrderRepositoryFS) UnlockTransferItem(ctx context.Context, orderID string, itemIndex int) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return orderdom.ErrNotFound
	}
	if itemIndex < 0 {
		return fmt.Errorf("invalid itemIndex: %d", itemIndex)
	}

	docRef := r.ordersCol().Doc(orderID)

	return r.Client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		snap, err := tx.Get(docRef)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				return orderdom.ErrNotFound
			}
			return err
		}

		o, err := docToOrder(snap)
		if err != nil {
			return err
		}

		if itemIndex >= len(o.Items) {
			return fmt.Errorf("order %s: itemIndex out of range: %d", orderID, itemIndex)
		}

		o.Items[itemIndex].Transferred = false
		o.Items[itemIndex].TransferredAt = nil

		updates := []firestore.Update{
			{Path: "items", Value: orderToDoc(o)["items"]},
		}
		return tx.Update(docRef, updates)
	})
}
