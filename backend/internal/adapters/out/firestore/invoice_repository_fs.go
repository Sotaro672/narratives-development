// backend/internal/adapters/out/firestore/invoice_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fscommon "narratives/internal/adapters/out/firestore/common"
	usecase "narratives/internal/application/usecase"
	invdom "narratives/internal/domain/invoice"
)

// InvoiceRepositoryFS implements usecase.InvoiceRepo using Firestore.
type InvoiceRepositoryFS struct {
	Client *firestore.Client
}

func NewInvoiceRepositoryFS(client *firestore.Client) *InvoiceRepositoryFS {
	return &InvoiceRepositoryFS{Client: client}
}

func (r *InvoiceRepositoryFS) col() *firestore.CollectionRef {
	return r.Client.Collection("invoices")
}

func (r *InvoiceRepositoryFS) orderItemInvoiceCol() *firestore.CollectionRef {
	return r.Client.Collection("order_item_invoices")
}

// Compile-time check: ensure InvoiceRepositoryFS satisfies usecase.InvoiceRepo.
var _ usecase.InvoiceRepo = (*InvoiceRepositoryFS)(nil)

// =======================
// Queries (Invoice)
// =======================

// GetByID は InvoiceRepo インターフェース準拠用。
// 本実装では orderID をそのままドキュメントIDとして扱う。
func (r *InvoiceRepositoryFS) GetByID(ctx context.Context, id string) (invdom.Invoice, error) {
	return r.GetByOrderID(ctx, id)
}

func (r *InvoiceRepositoryFS) GetByOrderID(ctx context.Context, orderID string) (invdom.Invoice, error) {
	if r.Client == nil {
		return invdom.Invoice{}, errors.New("firestore client is nil")
	}

	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return invdom.Invoice{}, invdom.ErrNotFound
	}

	snap, err := r.col().Doc(orderID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return invdom.Invoice{}, invdom.ErrNotFound
		}
		return invdom.Invoice{}, err
	}

	return docToInvoice(snap)
}

func (r *InvoiceRepositoryFS) Exists(ctx context.Context, id string) (bool, error) {
	if r.Client == nil {
		return false, errors.New("firestore client is nil")
	}

	orderID := strings.TrimSpace(id)
	if orderID == "" {
		return false, nil
	}

	_, err := r.col().Doc(orderID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Count: best-effort via scanning and applying Filter in-memory.
func (r *InvoiceRepositoryFS) Count(ctx context.Context, filter invdom.Filter) (int, error) {
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

func (r *InvoiceRepositoryFS) List(
	ctx context.Context,
	filter invdom.Filter,
	sort invdom.Sort,
	page invdom.Page,
) (invdom.PageResult[invdom.Invoice], error) {
	if r.Client == nil {
		return invdom.PageResult[invdom.Invoice]{}, errors.New("firestore client is nil")
	}

	pageNum, perPage, _ := fscommon.NormalizePage(page.Number, page.PerPage, 50, 0)

	q := r.col().Query
	q = applyInvoiceSort(q, sort)

	it := q.Documents(ctx)
	defer it.Stop()

	var all []invdom.Invoice
	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return invdom.PageResult[invdom.Invoice]{}, err
		}
		inv, err := docToInvoice(doc)
		if err != nil {
			return invdom.PageResult[invdom.Invoice]{}, err
		}
		if matchInvoiceFilter(inv, filter) {
			all = append(all, inv)
		}
	}

	total := len(all)
	if total == 0 {
		return invdom.PageResult[invdom.Invoice]{
			Items:      []invdom.Invoice{},
			TotalCount: 0,
			TotalPages: 0,
			Page:       pageNum,
			PerPage:    perPage,
		}, nil
	}

	offset := (pageNum - 1) * perPage
	if offset > total {
		offset = total
	}
	end := offset + perPage
	if end > total {
		end = total
	}
	items := all[offset:end]

	totalPages := fscommon.ComputeTotalPages(total, perPage)

	return invdom.PageResult[invdom.Invoice]{
		Items:      items,
		TotalCount: total,
		TotalPages: totalPages,
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

func (r *InvoiceRepositoryFS) ListByCursor(
	ctx context.Context,
	filter invdom.Filter,
	_ invdom.Sort,
	cpage invdom.CursorPage,
) (invdom.CursorPageResult[invdom.Invoice], error) {
	if r.Client == nil {
		return invdom.CursorPageResult[invdom.Invoice]{}, errors.New("firestore client is nil")
	}

	limit := cpage.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	// Cursor by orderId ascending.
	q := r.col().OrderBy("orderId", firestore.Asc)

	it := q.Documents(ctx)
	defer it.Stop()

	after := strings.TrimSpace(cpage.After)
	skipping := after != ""

	var (
		items []invdom.Invoice
		last  string
	)

	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return invdom.CursorPageResult[invdom.Invoice]{}, err
		}
		inv, err := docToInvoice(doc)
		if err != nil {
			return invdom.CursorPageResult[invdom.Invoice]{}, err
		}
		if !matchInvoiceFilter(inv, filter) {
			continue
		}

		if skipping {
			if inv.OrderID <= after {
				continue
			}
			skipping = false
		}

		items = append(items, inv)
		last = inv.OrderID

		if len(items) >= limit+1 {
			break
		}
	}

	var next *string
	if len(items) > limit {
		items = items[:limit]
		next = &last
	}

	return invdom.CursorPageResult[invdom.Invoice]{
		Items:      items,
		NextCursor: next,
		Limit:      limit,
	}, nil
}

// =======================
// Mutations (InvoiceRepo required methods)
// =======================

func (r *InvoiceRepositoryFS) Create(ctx context.Context, inv invdom.Invoice) (invdom.Invoice, error) {
	if r.Client == nil {
		return invdom.Invoice{}, errors.New("firestore client is nil")
	}

	id := strings.TrimSpace(inv.OrderID)
	if id == "" {
		return invdom.Invoice{}, errors.New("missing orderID")
	}

	now := time.Now().UTC()
	if inv.CreatedAt.IsZero() {
		inv.CreatedAt = now
	}
	if inv.UpdatedAt.IsZero() {
		inv.UpdatedAt = now
	}

	inv.OrderID = id
	docRef := r.col().Doc(id)

	data := invoiceToDocData(inv)

	_, err := docRef.Create(ctx, data)
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return invdom.Invoice{}, invdom.ErrConflict
		}
		return invdom.Invoice{}, err
	}

	snap, err := docRef.Get(ctx)
	if err != nil {
		return invdom.Invoice{}, err
	}
	return docToInvoice(snap)
}

// Save upserts an Invoice. (Matches usecase.InvoiceRepo.Save)
func (r *InvoiceRepositoryFS) Save(
	ctx context.Context,
	inv invdom.Invoice,
) (invdom.Invoice, error) {
	if r.Client == nil {
		return invdom.Invoice{}, errors.New("firestore client is nil")
	}

	id := strings.TrimSpace(inv.OrderID)
	if id == "" {
		return invdom.Invoice{}, errors.New("missing orderID")
	}

	now := time.Now().UTC()
	if inv.CreatedAt.IsZero() {
		inv.CreatedAt = now
	}
	if inv.UpdatedAt.IsZero() {
		inv.UpdatedAt = now
	}

	inv.OrderID = id
	docRef := r.col().Doc(id)

	data := invoiceToDocData(inv)

	_, err := docRef.Set(ctx, data, firestore.MergeAll)
	if err != nil {
		return invdom.Invoice{}, err
	}

	snap, err := docRef.Get(ctx)
	if err != nil {
		return invdom.Invoice{}, err
	}
	return docToInvoice(snap)
}

func (r *InvoiceRepositoryFS) Delete(ctx context.Context, id string) error {
	if r.Client == nil {
		return errors.New("firestore client is nil")
	}

	orderID := strings.TrimSpace(id)
	if orderID == "" {
		return invdom.ErrNotFound
	}

	_, err := r.col().Doc(orderID).Delete(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return invdom.ErrNotFound
		}
		return err
	}
	return nil
}

// =======================
// Order item level invoices (extra helpers)
// =======================

func (r *InvoiceRepositoryFS) GetOrderItemInvoiceByOrderItemID(
	ctx context.Context,
	orderItemID string,
) (invdom.OrderItemInvoice, error) {
	if r.Client == nil {
		return invdom.OrderItemInvoice{}, errors.New("firestore client is nil")
	}

	orderItemID = strings.TrimSpace(orderItemID)
	if orderItemID == "" {
		return invdom.OrderItemInvoice{}, invdom.ErrNotFound
	}

	q := r.orderItemInvoiceCol().
		Where("orderItemId", "==", orderItemID).
		OrderBy("updatedAt", firestore.Desc).
		Limit(1)

	it := q.Documents(ctx)
	defer it.Stop()

	doc, err := it.Next()
	if err == iterator.Done {
		return invdom.OrderItemInvoice{}, invdom.ErrNotFound
	}
	if err != nil {
		return invdom.OrderItemInvoice{}, err
	}

	return docToOrderItemInvoice(doc)
}

func (r *InvoiceRepositoryFS) ListOrderItemInvoicesByOrderItemIDs(
	ctx context.Context,
	orderItemIDs []string,
) ([]invdom.OrderItemInvoice, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	ids := make([]string, 0, len(orderItemIDs))
	for _, id := range orderItemIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return []invdom.OrderItemInvoice{}, nil
	}

	var out []invdom.OrderItemInvoice

	// Firestore "in" は最大 10 要素まで。それ以内ならそのまま問い合わせ。
	if len(ids) <= 10 {
		q := r.orderItemInvoiceCol().
			Where("orderItemId", "in", ids).
			OrderBy("orderItemId", firestore.Asc).
			OrderBy("updatedAt", firestore.Desc)

		it := q.Documents(ctx)
		defer it.Stop()

		for {
			doc, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return nil, err
			}
			oi, err := docToOrderItemInvoice(doc)
			if err != nil {
				return nil, err
			}
			out = append(out, oi)
		}
		return out, nil
	}

	// 10超の場合は全件走査して id セットでフィルタ
	idset := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		idset[id] = struct{}{}
	}

	it := r.orderItemInvoiceCol().
		OrderBy("orderItemId", firestore.Asc).
		OrderBy("updatedAt", firestore.Desc).
		Documents(ctx)
	defer it.Stop()

	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		oi, err := docToOrderItemInvoice(doc)
		if err != nil {
			return nil, err
		}
		if _, ok := idset[oi.OrderItemID]; ok {
			out = append(out, oi)
		}
	}

	return out, nil
}

// =======================
// Helpers
// =======================

func invoiceToDocData(inv invdom.Invoice) map[string]any {
	return map[string]any{
		"orderId":          strings.TrimSpace(inv.OrderID),
		"subtotal":         inv.Subtotal,
		"discountAmount":   inv.DiscountAmount,
		"taxAmount":        inv.TaxAmount,
		"shippingCost":     inv.ShippingCost,
		"totalAmount":      inv.TotalAmount,
		"currency":         strings.TrimSpace(inv.Currency),
		"createdAt":        inv.CreatedAt.UTC(),
		"updatedAt":        inv.UpdatedAt.UTC(),
		"billingAddressId": strings.TrimSpace(inv.BillingAddressID),
	}
}

func docToInvoice(doc *firestore.DocumentSnapshot) (invdom.Invoice, error) {
	data := doc.Data()
	if data == nil {
		return invdom.Invoice{}, fmt.Errorf("empty invoice document: %s", doc.Ref.ID)
	}

	getStr := func(keys ...string) string {
		for _, key := range keys {
			if v, ok := data[key].(string); ok {
				return strings.TrimSpace(v)
			}
		}
		return ""
	}
	getInt := func(keys ...string) int {
		for _, key := range keys {
			if v, ok := data[key]; ok {
				switch n := v.(type) {
				case int:
					return n
				case int32:
					return int(n)
				case int64:
					return int(n)
				case float64:
					return int(n)
				}
			}
		}
		return 0
	}
	getTime := func(keys ...string) (time.Time, bool) {
		for _, key := range keys {
			if v, ok := data[key].(time.Time); ok {
				return v.UTC(), !v.IsZero()
			}
		}
		return time.Time{}, false
	}

	var inv invdom.Invoice

	inv.OrderID = getStr("orderId", "order_id")
	if inv.OrderID == "" {
		inv.OrderID = doc.Ref.ID
	}

	inv.Subtotal = getInt("subtotal")
	inv.DiscountAmount = getInt("discountAmount", "discount_amount")
	inv.TaxAmount = getInt("taxAmount", "tax_amount")
	inv.ShippingCost = getInt("shippingCost", "shipping_cost")
	inv.TotalAmount = getInt("totalAmount", "total_amount")
	inv.Currency = getStr("currency")
	inv.BillingAddressID = getStr("billingAddressId", "billing_address_id")

	if t, ok := getTime("createdAt", "created_at"); ok {
		inv.CreatedAt = t
	}
	if t, ok := getTime("updatedAt", "updated_at"); ok {
		inv.UpdatedAt = t
	}

	inv.OrderItemInvoices = nil

	return inv, nil
}

func docToOrderItemInvoice(doc *firestore.DocumentSnapshot) (invdom.OrderItemInvoice, error) {
	data := doc.Data()
	if data == nil {
		return invdom.OrderItemInvoice{}, fmt.Errorf("empty order_item_invoice document: %s", doc.Ref.ID)
	}

	getStr := func(keys ...string) string {
		for _, key := range keys {
			if v, ok := data[key].(string); ok {
				return strings.TrimSpace(v)
			}
		}
		return ""
	}
	getInt := func(keys ...string) int {
		for _, key := range keys {
			if v, ok := data[key]; ok {
				switch n := v.(type) {
				case int:
					return n
				case int32:
					return int(n)
				case int64:
					return int(n)
				case float64:
					return int(n)
				}
			}
		}
		return 0
	}
	getTime := func(keys ...string) (time.Time, bool) {
		for _, key := range keys {
			if v, ok := data[key].(time.Time); ok {
				return v.UTC(), !v.IsZero()
			}
		}
		return time.Time{}, false
	}

	var oi invdom.OrderItemInvoice

	oi.ID = getStr("id")
	if oi.ID == "" {
		oi.ID = doc.Ref.ID
	}
	oi.OrderItemID = getStr("orderItemId", "order_item_id")
	oi.UnitPrice = getInt("unitPrice", "unit_price")
	oi.TotalPrice = getInt("totalPrice", "total_price")

	if t, ok := getTime("createdAt", "created_at"); ok {
		oi.CreatedAt = t
	}
	if t, ok := getTime("updatedAt", "updated_at"); ok {
		oi.UpdatedAt = t
	}

	return oi, nil
}

// matchInvoiceFilter applies invdom.Filter in-memory.
func matchInvoiceFilter(inv invdom.Invoice, f invdom.Filter) bool {
	// Free text search: orderID or currency
	if sq := strings.TrimSpace(f.SearchQuery); sq != "" {
		lq := strings.ToLower(sq)
		haystack := strings.ToLower(inv.OrderID + " " + inv.Currency)
		if !strings.Contains(haystack, lq) {
			return false
		}
	}

	// OrderID exact
	if f.OrderID != nil && strings.TrimSpace(*f.OrderID) != "" {
		if inv.OrderID != strings.TrimSpace(*f.OrderID) {
			return false
		}
	}

	// OrderIDs IN (...)
	if len(f.OrderIDs) > 0 {
		found := false
		for _, id := range f.OrderIDs {
			if strings.TrimSpace(id) == inv.OrderID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Currency
	if f.Currency != nil && strings.TrimSpace(*f.Currency) != "" {
		if inv.Currency != strings.TrimSpace(*f.Currency) {
			return false
		}
	}

	// Amount ranges helper
	inRange := func(v int, min, max *int) bool {
		if min != nil && v < *min {
			return false
		}
		if max != nil && v > *max {
			return false
		}
		return true
	}

	if !inRange(inv.Subtotal, f.MinSubtotal, f.MaxSubtotal) {
		return false
	}
	if !inRange(inv.DiscountAmount, f.MinDiscountAmount, f.MaxDiscountAmount) {
		return false
	}
	if !inRange(inv.TaxAmount, f.MinTaxAmount, f.MaxTaxAmount) {
		return false
	}
	if !inRange(inv.ShippingCost, f.MinShippingCost, f.MaxShippingCost) {
		return false
	}
	if !inRange(inv.TotalAmount, f.MinTotalAmount, f.MaxTotalAmount) {
		return false
	}

	// Date ranges
	if f.CreatedFrom != nil && inv.CreatedAt.Before(f.CreatedFrom.UTC()) {
		return false
	}
	if f.CreatedTo != nil && !inv.CreatedAt.Before(f.CreatedTo.UTC()) {
		return false
	}
	if f.UpdatedFrom != nil && inv.UpdatedAt.Before(f.UpdatedFrom.UTC()) {
		return false
	}
	if f.UpdatedTo != nil && !inv.UpdatedAt.Before(f.UpdatedTo.UTC()) {
		return false
	}

	return true
}

func applyInvoiceSort(q firestore.Query, sort invdom.Sort) firestore.Query {
	col, dir := mapInvoiceSort(sort)
	if col == "" {
		// default: updatedAt DESC, orderId DESC
		return q.OrderBy("updatedAt", firestore.Desc).OrderBy("orderId", firestore.Desc)
	}
	return q.OrderBy(col, dir).OrderBy("orderId", firestore.Asc)
}

func mapInvoiceSort(sort invdom.Sort) (string, firestore.Direction) {
	col := strings.ToLower(string(sort.Column))
	var field string

	switch col {
	case "orderid", "order_id":
		field = "orderId"
	case "subtotal":
		field = "subtotal"
	case "discountamount", "discount_amount":
		field = "discountAmount"
	case "taxamount", "tax_amount":
		field = "taxAmount"
	case "shippingcost", "shipping_cost":
		field = "shippingCost"
	case "totalamount", "total_amount":
		field = "totalAmount"
	case "currency":
		field = "currency"
	case "createdat", "created_at":
		field = "createdAt"
	case "updatedat", "updated_at":
		field = "updatedAt"
	default:
		return "", firestore.Desc
	}

	dir := firestore.Asc
	if strings.EqualFold(string(sort.Order), "desc") {
		dir = firestore.Desc
	}
	return field, dir
}
