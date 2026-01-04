// backend/internal/adapters/out/firestore/order_repository_fs.go
package firestore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
// RepositoryPort impl (usecase.OrderRepo)
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

	o, err := docToOrder(snap)
	if err != nil {
		return orderdom.Order{}, err
	}
	return o, nil
}

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
	q = applyOrderSort(q, sort)

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
		if matchOrderFilter(o, filter) {
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

// ListByCursor follows the PG behavior: ordered by ID ASC, cursor = last ID.
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
		if len(items) > limit {
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
	if len(items) > limit {
		items = items[:limit]
		if last != "" {
			next = &last
		}
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
	if o.UpdatedAt.IsZero() {
		o.UpdatedAt = now
	}

	var docRef *firestore.DocumentRef
	if id == "" {
		docRef = r.ordersCol().NewDoc()
		o.ID = docRef.ID
	} else {
		docRef = r.ordersCol().Doc(id)
		o.ID = id
	}

	data := orderToDoc(o)

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

	if id == "" {
		// Behave like Create with auto ID.
		if o.CreatedAt.IsZero() {
			o.CreatedAt = now
		}
		if o.UpdatedAt.IsZero() {
			o.UpdatedAt = now
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

	// Preserve CreatedAt if absent by trying to load existing.
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
	if o.UpdatedAt.IsZero() {
		o.UpdatedAt = now
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

	return nil
}

// ========================
// Helpers
// ========================

// Firestore/encoding/json decode often uses map[string]interface{}.
// NOTE: any == interface{} in Go, so map[string]any == map[string]interface{} (identical type).
func asMapAny(v any) map[string]any {
	if v == nil {
		return nil
	}
	if m, ok := v.(map[string]interface{}); ok {
		// identical type to map[string]any due to any alias
		return m
	}
	return nil
}

func mapGetStr(m map[string]any, keys ...string) string {
	if m == nil {
		return ""
	}
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if s, ok2 := v.(string); ok2 {
				return strings.TrimSpace(s)
			}
			if v != nil {
				return strings.TrimSpace(fmt.Sprint(v))
			}
		}
	}
	return ""
}

func decodeShippingSnapshot(v any) (orderdom.ShippingSnapshot, bool) {
	m := asMapAny(v)
	if m == nil {
		return orderdom.ShippingSnapshot{}, false
	}
	return orderdom.ShippingSnapshot{
		ZipCode: mapGetStr(m, "zipCode", "zip_code", "postalCode", "postal_code"),
		State:   mapGetStr(m, "state", "prefecture", "region"),
		City:    mapGetStr(m, "city", "locality"),
		Street:  mapGetStr(m, "street", "address1", "address_1"),
		Street2: mapGetStr(m, "street2", "address2", "address_2"),
		Country: mapGetStr(m, "country", "countryCode", "country_code"),
	}, true
}

func decodeBillingSnapshot(v any) (orderdom.BillingSnapshot, bool) {
	m := asMapAny(v)
	if m == nil {
		return orderdom.BillingSnapshot{}, false
	}
	holder := mapGetStr(m, "cardHolderName", "cardholderName", "holderName", "card_holder_name")
	last4 := mapGetStr(m, "last4", "cardLast4", "card_last4")
	return orderdom.BillingSnapshot{
		Last4:          last4,
		CardHolderName: holder,
	}, true
}

// docToOrder converts a Firestore document snapshot to orderdom.Order.
// NOTE: legacy fields (orderNumber/status/fulfillment/tracking/deleted*) are ignored safely.
func docToOrder(doc *firestore.DocumentSnapshot) (orderdom.Order, error) {
	data := doc.Data()
	if data == nil {
		return orderdom.Order{}, fmt.Errorf("empty order document: %s", doc.Ref.ID)
	}

	getStr := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := data[k].(string); ok {
				return strings.TrimSpace(v)
			}
		}
		return ""
	}
	getStrPtr := func(keys ...string) *string {
		for _, k := range keys {
			if v, ok := data[k].(string); ok {
				s := strings.TrimSpace(v)
				if s != "" {
					return &s
				}
			}
		}
		return nil
	}
	getTime := func(keys ...string) time.Time {
		for _, k := range keys {
			if v, ok := data[k].(time.Time); ok && !v.IsZero() {
				return v.UTC()
			}
		}
		return time.Time{}
	}
	getTimePtr := func(keys ...string) *time.Time {
		for _, k := range keys {
			if v, ok := data[k].(time.Time); ok && !v.IsZero() {
				t := v.UTC()
				return &t
			}
		}
		return nil
	}
	getItems := func() []string {
		raw, ok := data["items"]
		if !ok || raw == nil {
			return nil
		}
		switch vv := raw.(type) {
		case []interface{}:
			out := make([]string, 0, len(vv))
			for _, x := range vv {
				switch s := x.(type) {
				case string:
					out = append(out, s)
				default:
					out = append(out, fmt.Sprint(s))
				}
			}
			return out
		case []string:
			return vv
		case string:
			if vv == "" {
				return nil
			}
			var arr []string
			if err := json.Unmarshal([]byte(vv), &arr); err == nil {
				return arr
			}
		}
		return nil
	}

	// ✅ snapshots (required by domain validation)
	var ship orderdom.ShippingSnapshot
	if v, ok := data["shippingSnapshot"]; ok {
		if s, ok2 := decodeShippingSnapshot(v); ok2 {
			ship = s
		}
	} else if v, ok := data["shipping_snapshot"]; ok {
		if s, ok2 := decodeShippingSnapshot(v); ok2 {
			ship = s
		}
	}

	var bill orderdom.BillingSnapshot
	if v, ok := data["billingSnapshot"]; ok {
		if b, ok2 := decodeBillingSnapshot(v); ok2 {
			bill = b
		}
	} else if v, ok := data["billing_snapshot"]; ok {
		if b, ok2 := decodeBillingSnapshot(v); ok2 {
			bill = b
		}
	}

	// If snapshots are missing but legacy IDs exist, fail loudly (migration needed).
	legacyShipID := getStr("shippingAddressId", "shipping_address_id")
	legacyBillID := getStr("billingAddressId", "billing_address_id")

	if strings.TrimSpace(ship.State) == "" &&
		strings.TrimSpace(ship.City) == "" &&
		strings.TrimSpace(ship.Street) == "" &&
		strings.TrimSpace(ship.Country) == "" {
		if legacyShipID != "" {
			return orderdom.Order{}, fmt.Errorf("order %s: missing shippingSnapshot (legacy shippingAddressId=%q present)", doc.Ref.ID, legacyShipID)
		}
		return orderdom.Order{}, fmt.Errorf("order %s: missing shippingSnapshot", doc.Ref.ID)
	}

	if strings.TrimSpace(bill.Last4) == "" {
		if legacyBillID != "" {
			return orderdom.Order{}, fmt.Errorf("order %s: missing billingSnapshot.last4 (legacy billingAddressId=%q present)", doc.Ref.ID, legacyBillID)
		}
		return orderdom.Order{}, fmt.Errorf("order %s: missing billingSnapshot.last4", doc.Ref.ID)
	}

	return orderdom.Order{
		ID:     doc.Ref.ID,
		UserID: getStr("userId", "user_id"),
		CartID: getStr("cartId", "cart_id"),

		ShippingSnapshot: ship,
		BillingSnapshot:  bill,

		ListID:         getStr("listId", "list_id"),
		Items:          getItems(),
		InvoiceID:      getStr("invoiceId", "invoice_id"),
		PaymentID:      getStr("paymentId", "payment_id"),
		TransferedDate: getTimePtr("transferedDate", "transfered_date"),
		CreatedAt:      getTime("createdAt", "created_at"),
		UpdatedAt:      getTime("updatedAt", "updated_at"),
		UpdatedBy:      getStrPtr("updatedBy", "updated_by"),
	}, nil
}

// orderToDoc converts orderdom.Order into a Firestore-storable map.
func orderToDoc(o orderdom.Order) map[string]any {
	ship := map[string]any{
		"zipCode": strings.TrimSpace(o.ShippingSnapshot.ZipCode),
		"state":   strings.TrimSpace(o.ShippingSnapshot.State),
		"city":    strings.TrimSpace(o.ShippingSnapshot.City),
		"street":  strings.TrimSpace(o.ShippingSnapshot.Street),
		"street2": strings.TrimSpace(o.ShippingSnapshot.Street2),
		"country": strings.TrimSpace(o.ShippingSnapshot.Country),
	}
	bill := map[string]any{
		"last4":          strings.TrimSpace(o.BillingSnapshot.Last4),
		"cardHolderName": strings.TrimSpace(o.BillingSnapshot.CardHolderName),
	}

	m := map[string]any{
		"userId": strings.TrimSpace(o.UserID),
		"cartId": strings.TrimSpace(o.CartID),

		// ✅ snapshots
		"shippingSnapshot": ship,
		"billingSnapshot":  bill,

		"listId":    strings.TrimSpace(o.ListID),
		"invoiceId": strings.TrimSpace(o.InvoiceID),
		"paymentId": strings.TrimSpace(o.PaymentID),
	}

	if len(o.Items) > 0 {
		m["items"] = o.Items
	}

	if o.TransferedDate != nil && !o.TransferedDate.IsZero() {
		m["transferedDate"] = o.TransferedDate.UTC()
	}

	if !o.CreatedAt.IsZero() {
		m["createdAt"] = o.CreatedAt.UTC()
	}
	if !o.UpdatedAt.IsZero() {
		m["updatedAt"] = o.UpdatedAt.UTC()
	}
	if o.UpdatedBy != nil {
		if s := strings.TrimSpace(*o.UpdatedBy); s != "" {
			m["updatedBy"] = s
		}
	}

	return m
}

// matchOrderFilter applies uc.OrderFilter in-memory.
func matchOrderFilter(o orderdom.Order, f uc.OrderFilter) bool {
	if f.UserID != nil {
		if strings.TrimSpace(o.UserID) != strings.TrimSpace(*f.UserID) {
			return false
		}
	}

	// Time ranges
	if f.CreatedFrom != nil {
		if o.CreatedAt.IsZero() || o.CreatedAt.Before(f.CreatedFrom.UTC()) {
			return false
		}
	}
	if f.CreatedTo != nil {
		if o.CreatedAt.IsZero() || !o.CreatedAt.Before(f.CreatedTo.UTC()) {
			return false
		}
	}
	if f.UpdatedFrom != nil {
		if o.UpdatedAt.IsZero() || o.UpdatedAt.Before(f.UpdatedFrom.UTC()) {
			return false
		}
	}
	if f.UpdatedTo != nil {
		if o.UpdatedAt.IsZero() || !o.UpdatedAt.Before(f.UpdatedTo.UTC()) {
			return false
		}
	}
	if f.TransferedFrom != nil {
		if o.TransferedDate == nil || o.TransferedDate.Before(f.TransferedFrom.UTC()) {
			return false
		}
	}
	if f.TransferedTo != nil {
		if o.TransferedDate == nil || !o.TransferedDate.Before(f.TransferedTo.UTC()) {
			return false
		}
	}

	return true
}

// applyOrderSort maps common.Sort to Firestore orderBy.
func applyOrderSort(q firestore.Query, sort common.Sort) firestore.Query {
	col := strings.ToLower(strings.TrimSpace(string(sort.Column)))
	var field string

	switch col {
	case "createdat", "created_at":
		field = "createdAt"
	case "updatedat", "updated_at":
		field = "updatedAt"
	case "transfereddate", "transfered_date":
		field = "transferedDate"
	default:
		// Default: createdAt DESC, id DESC (to match PG default)
		return q.OrderBy("createdAt", firestore.Desc).
			OrderBy(firestore.DocumentID, firestore.Desc)
	}

	dir := firestore.Desc
	if strings.EqualFold(string(sort.Order), "asc") {
		dir = firestore.Asc
	}

	return q.OrderBy(field, dir).OrderBy(firestore.DocumentID, dir)
}
