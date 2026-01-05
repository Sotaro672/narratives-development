// backend/internal/adapters/out/firestore/invoice_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fscommon "narratives/internal/adapters/out/firestore/common"
	uc "narratives/internal/application/usecase"
	common "narratives/internal/domain/common"
	invoicedom "narratives/internal/domain/invoice"
)

// Firestore implementation of usecase.InvoiceRepo
// ✅ docId = orderId（invoice doc内の orderId フィールドは不要）
type InvoiceRepositoryFS struct {
	Client *firestore.Client
}

func NewInvoiceRepositoryFS(client *firestore.Client) *InvoiceRepositoryFS {
	return &InvoiceRepositoryFS{Client: client}
}

func (r *InvoiceRepositoryFS) invoicesCol() *firestore.CollectionRef {
	return r.Client.Collection("invoices")
}

// ========================
// usecase.InvoiceRepo impl
// ========================

func (r *InvoiceRepositoryFS) GetByOrderID(ctx context.Context, orderID string) (invoicedom.Invoice, error) {
	if r.Client == nil {
		return invoicedom.Invoice{}, errors.New("firestore client is nil")
	}

	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return invoicedom.Invoice{}, invoicedom.ErrNotFound
	}

	snap, err := r.invoicesCol().Doc(orderID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return invoicedom.Invoice{}, invoicedom.ErrNotFound
		}
		return invoicedom.Invoice{}, err
	}

	return docToInvoice(snap)
}

func (r *InvoiceRepositoryFS) Exists(ctx context.Context, orderID string) (bool, error) {
	if r.Client == nil {
		return false, errors.New("firestore client is nil")
	}

	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return false, nil
	}

	_, err := r.invoicesCol().Doc(orderID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *InvoiceRepositoryFS) List(
	ctx context.Context,
	filter uc.InvoiceFilter,
	sort common.Sort,
	page common.Page,
) (common.PageResult[invoicedom.Invoice], error) {
	if r.Client == nil {
		return common.PageResult[invoicedom.Invoice]{}, errors.New("firestore client is nil")
	}

	pageNum, perPage, offset := fscommon.NormalizePage(page.Number, page.PerPage, 50, 200)

	q := r.invoicesCol().Query
	q = applyInvoiceSort(q, sort)

	it := q.Documents(ctx)
	defer it.Stop()

	var all []invoicedom.Invoice
	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return common.PageResult[invoicedom.Invoice]{}, err
		}

		inv, err := docToInvoice(doc)
		if err != nil {
			return common.PageResult[invoicedom.Invoice]{}, err
		}
		if matchInvoiceFilter(inv, filter) {
			all = append(all, inv)
		}
	}

	total := len(all)
	if total == 0 {
		return common.PageResult[invoicedom.Invoice]{
			Items:      []invoicedom.Invoice{},
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

	return common.PageResult[invoicedom.Invoice]{
		Items:      all[offset:end],
		TotalCount: total,
		TotalPages: fscommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

func (r *InvoiceRepositoryFS) ListByCursor(
	ctx context.Context,
	filter uc.InvoiceFilter,
	_ common.Sort,
	cpage common.CursorPage,
) (common.CursorPageResult[invoicedom.Invoice], error) {
	if r.Client == nil {
		return common.CursorPageResult[invoicedom.Invoice]{}, errors.New("firestore client is nil")
	}

	limit := cpage.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	q := r.invoicesCol().OrderBy(firestore.DocumentID, firestore.Asc)
	if after := strings.TrimSpace(cpage.After); after != "" {
		q = q.StartAfter(after)
	}

	it := q.Documents(ctx)
	defer it.Stop()

	var (
		items []invoicedom.Invoice
		last  string
	)
	for {
		if len(items) > limit {
			break
		}
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return common.CursorPageResult[invoicedom.Invoice]{}, err
		}

		inv, err := docToInvoice(doc)
		if err != nil {
			return common.CursorPageResult[invoicedom.Invoice]{}, err
		}
		if matchInvoiceFilter(inv, filter) {
			items = append(items, inv)
			last = strings.TrimSpace(doc.Ref.ID) // ✅ cursor=docId
		}
	}

	var next *string
	if len(items) > limit {
		items = items[:limit]
		if strings.TrimSpace(last) != "" {
			next = &last
		}
	}

	return common.CursorPageResult[invoicedom.Invoice]{
		Items:      items,
		NextCursor: next,
		Limit:      limit,
	}, nil
}

func (r *InvoiceRepositoryFS) Count(ctx context.Context, filter uc.InvoiceFilter) (int, error) {
	if r.Client == nil {
		return 0, errors.New("firestore client is nil")
	}

	it := r.invoicesCol().Documents(ctx)
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
		inv, err := docToInvoice(doc)
		if err != nil {
			return 0, err
		}
		if matchInvoiceFilter(inv, filter) {
			total++
		}
	}
	return total, nil
}

func (r *InvoiceRepositoryFS) Create(ctx context.Context, inv invoicedom.Invoice) (invoicedom.Invoice, error) {
	if r.Client == nil {
		return invoicedom.Invoice{}, errors.New("firestore client is nil")
	}

	orderID := strings.TrimSpace(inv.OrderID)
	if orderID == "" {
		return invoicedom.Invoice{}, errors.New("invoice: invalid orderId")
	}
	inv.OrderID = orderID

	data := invoiceToDoc(inv)

	docRef := r.invoicesCol().Doc(orderID)
	_, err := docRef.Create(ctx, data)
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return invoicedom.Invoice{}, invoicedom.ErrConflict
		}
		return invoicedom.Invoice{}, err
	}

	snap, err := docRef.Get(ctx)
	if err != nil {
		return invoicedom.Invoice{}, err
	}
	return docToInvoice(snap)
}

func (r *InvoiceRepositoryFS) Save(ctx context.Context, inv invoicedom.Invoice, _ *common.SaveOptions) (invoicedom.Invoice, error) {
	if r.Client == nil {
		return invoicedom.Invoice{}, errors.New("firestore client is nil")
	}

	orderID := strings.TrimSpace(inv.OrderID)
	if orderID == "" {
		return invoicedom.Invoice{}, errors.New("invoice: invalid orderId")
	}
	inv.OrderID = orderID

	docRef := r.invoicesCol().Doc(orderID)
	data := invoiceToDoc(inv)

	if _, err := docRef.Set(ctx, data, firestore.MergeAll); err != nil {
		return invoicedom.Invoice{}, err
	}

	snap, err := docRef.Get(ctx)
	if err != nil {
		return invoicedom.Invoice{}, err
	}
	return docToInvoice(snap)
}

func (r *InvoiceRepositoryFS) DeleteByOrderID(ctx context.Context, orderID string) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return invoicedom.ErrNotFound
	}

	_, err := r.invoicesCol().Doc(orderID).Delete(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return invoicedom.ErrNotFound
		}
		return err
	}
	return nil
}

func (r *InvoiceRepositoryFS) Reset(ctx context.Context) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	it := r.invoicesCol().Documents(ctx)
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
	return nil
}

// ========================
// Helpers
// ========================

func docToInvoice(doc *firestore.DocumentSnapshot) (invoicedom.Invoice, error) {
	data := doc.Data()
	if data == nil {
		return invoicedom.Invoice{}, fmt.Errorf("empty invoice document: %s", doc.Ref.ID)
	}

	getInt := func(key string) int {
		v, ok := data[key]
		if !ok || v == nil {
			return 0
		}
		switch t := v.(type) {
		case int:
			return t
		case int64:
			return int(t)
		case float64:
			return int(t)
		default:
			return 0
		}
	}

	getBool := func(key string) bool {
		v, ok := data[key]
		if !ok || v == nil {
			return false
		}
		if b, ok := v.(bool); ok {
			return b
		}
		s := strings.ToLower(strings.TrimSpace(fmt.Sprint(v)))
		return s == "true" || s == "1" || s == "yes"
	}

	getTimePtr := func(key string) *time.Time {
		v, ok := data[key]
		if !ok || v == nil {
			return nil
		}
		switch t := v.(type) {
		case time.Time:
			tt := t.UTC()
			if tt.IsZero() {
				return nil
			}
			return &tt
		case *time.Time:
			if t == nil || t.IsZero() {
				return nil
			}
			tt := t.UTC()
			return &tt
		case interface{ AsTime() time.Time }:
			tt := t.AsTime().UTC()
			if tt.IsZero() {
				return nil
			}
			return &tt
		default:
			s := strings.TrimSpace(fmt.Sprint(v))
			if s == "" {
				return nil
			}
			if tt, err := time.Parse(time.RFC3339, s); err == nil {
				u := tt.UTC()
				if u.IsZero() {
					return nil
				}
				return &u
			}
			return nil
		}
	}

	// prices
	var prices []int
	switch raw := data["prices"].(type) {
	case []any:
		prices = make([]int, 0, len(raw))
		for _, x := range raw {
			switch t := x.(type) {
			case int:
				prices = append(prices, t)
			case int64:
				prices = append(prices, int(t))
			case float64:
				prices = append(prices, int(t))
			default:
				prices = append(prices, 0)
			}
		}
	case []int:
		prices = raw
	default:
		prices = nil
	}

	// ✅ docId=orderId を正とする（invoice doc内に orderId は持たない）
	docID := strings.TrimSpace(doc.Ref.ID)
	if docID == "" {
		return invoicedom.Invoice{}, invoicedom.ErrNotFound
	}

	inv := invoicedom.Invoice{
		OrderID:     docID,
		Prices:      prices,
		Tax:         getInt("tax"),
		ShippingFee: getInt("shippingFee"),
		Paid:        getBool("paid"),
		UpdatedAt:   getTimePtr("updatedAt"),
	}

	if err := inv.Validate(); err != nil {
		return invoicedom.Invoice{}, err
	}
	return inv, nil
}

func invoiceToDoc(inv invoicedom.Invoice) map[string]any {
	prices := make([]int, 0, len(inv.Prices))
	for _, p := range inv.Prices {
		prices = append(prices, p)
	}

	doc := map[string]any{
		// ✅ orderId は保持しない（docId=orderId のため）
		"prices":      prices,
		"tax":         inv.Tax,
		"shippingFee": inv.ShippingFee,
		"paid":        inv.Paid,
	}

	// ✅ paid:false->true の瞬間だけ保持
	if inv.UpdatedAt != nil && !inv.UpdatedAt.IsZero() {
		doc["updatedAt"] = inv.UpdatedAt.UTC()
	}

	return doc
}

// matchInvoiceFilter is reflection-based so adapter compiles even if uc.InvoiceFilter shape changes.
// It tries to apply: OrderID, Paid.
func matchInvoiceFilter(inv invoicedom.Invoice, f uc.InvoiceFilter) bool {
	return matchInvoiceFilterAny(inv, any(f))
}

func matchInvoiceFilterAny(inv invoicedom.Invoice, fv any) bool {
	if orderID, ok := getFilterString(fv, "OrderID"); ok {
		if strings.TrimSpace(orderID) != "" && strings.TrimSpace(inv.OrderID) != strings.TrimSpace(orderID) {
			return false
		}
	}
	if paidPtr, ok := getFilterBoolPtr(fv, "Paid"); ok && paidPtr != nil {
		if inv.Paid != *paidPtr {
			return false
		}
	}
	return true
}

func getFilterBoolPtr(v any, field string) (*bool, bool) {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return nil, false
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil, false
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil, false
	}
	f := rv.FieldByName(field)
	if !f.IsValid() {
		f = rv.FieldByName(lowerFirst(field))
		if !f.IsValid() {
			return nil, false
		}
	}

	if f.Kind() == reflect.Pointer {
		if f.IsNil() {
			return nil, true
		}
		if b, ok := f.Interface().(*bool); ok {
			return b, true
		}
	}
	if f.CanInterface() {
		if b, ok := f.Interface().(bool); ok {
			return &b, true
		}
	}
	return nil, false
}

func applyInvoiceSort(q firestore.Query, sort common.Sort) firestore.Query {
	col := strings.ToLower(strings.TrimSpace(string(sort.Column)))

	dir := firestore.Asc
	if strings.EqualFold(string(sort.Order), "desc") {
		dir = firestore.Desc
	}

	switch col {
	case "updatedat", "updated_at":
		return q.OrderBy("updatedAt", dir).OrderBy(firestore.DocumentID, dir)

	case "orderid", "order_id", "order":
		// ✅ orderId フィールドは無いので docId でソート
		return q.OrderBy(firestore.DocumentID, dir)

	default:
		return q.OrderBy(firestore.DocumentID, firestore.Desc)
	}
}
