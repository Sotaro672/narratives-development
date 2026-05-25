// backend/internal/adapters/out/firestore/order_repository_fs.go
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
	"google.golang.org/protobuf/types/known/timestamppb"

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

// Exists is optional (dev/testing convenience)
func (r *OrderRepositoryFS) Exists(ctx context.Context, id string) (bool, error) {
	if r.Client == nil {
		return false, errors.New("firestore client is nil")
	}

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

func (r *OrderRepositoryFS) ListByUserID(
	ctx context.Context,
	userID string,
	sort common.Sort,
	page common.Page,
) (common.PageResult[orderdom.Order], error) {
	if r.Client == nil {
		return common.PageResult[orderdom.Order]{}, errors.New("firestore client is nil")
	}

	filter := uc.OrderFilter{
		UserID: &userID,
	}
	return r.List(ctx, filter, sort, page)
}

// ListByCursor currently paginates by Firestore document ID ascending only.
// The sort argument is ignored.
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
	if after := cpage.After; after != "" {
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

func (r *OrderRepositoryFS) Create(ctx context.Context, o orderdom.Order) (orderdom.Order, error) {
	if r.Client == nil {
		return orderdom.Order{}, errors.New("firestore client is nil")
	}

	id := o.ID
	now := time.Now().UTC()
	if o.CreatedAt.IsZero() {
		o.CreatedAt = now
	}

	// 起票時は必ず paid=false（orderレベル）
	o.Paid = false

	// item-level transferred defaults（安全側で初期化）
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

	id := o.ID
	now := time.Now().UTC()

	if id == "" {
		if o.CreatedAt.IsZero() {
			o.CreatedAt = now
		}
		return r.Create(ctx, o)
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

	if id == "" {
		return orderdom.ErrNotFound
	}

	exists, err := r.Exists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return orderdom.ErrNotFound
	}

	_, err = r.ordersCol().Doc(id).Delete(ctx)
	if err != nil {
		return err
	}
	return nil
}

// ============================================================
// Transfer flow helpers (item-level transferred)
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

// ========================
// Mapper
// ========================

// docToOrder converts a Firestore document snapshot to orderdom.Order (NEW schema only).
// NEW schema:
// - paid is on order root
// - transferred/transferredAt are on each item (items[].transferred / items[].transferredAt)
func docToOrder(doc *firestore.DocumentSnapshot) (orderdom.Order, error) {
	data := doc.Data()
	if data == nil {
		return orderdom.Order{}, fmt.Errorf("empty order document: %s", doc.Ref.ID)
	}

	getStr := func(key string) string {
		if v, ok := data[key].(string); ok {
			return v
		}
		return ""
	}

	getTime := func(key string) time.Time {
		if v, ok := data[key].(time.Time); ok && !v.IsZero() {
			return v.UTC()
		}
		return time.Time{}
	}

	getBool := func(key string) bool {
		if v, ok := data[key].(bool); ok {
			return v
		}
		return false
	}

	var ship orderdom.ShippingSnapshot
	if v, ok := data["shippingSnapshot"]; ok {
		if s, ok2 := decodeShippingSnapshot(v); ok2 {
			ship = s
		}
	}

	var paymentMethod orderdom.PaymentMethodSnapshot
	if v, ok := data["paymentMethodSnapshot"]; ok {
		if p, ok2 := decodePaymentMethodSnapshot(v); ok2 {
			paymentMethod = p
		}
	}

	items, ok := decodeItems(data["items"])
	if !ok {
		items = nil
	}

	createdAt := getTime("createdAt")
	paid := getBool("paid")
	avatarID := getStr("avatarId")

	if avatarID == "" {
		return orderdom.Order{}, fmt.Errorf("order %s: missing avatarId", doc.Ref.ID)
	}
	if ship.State == "" ||
		ship.City == "" ||
		ship.Street == "" ||
		ship.Country == "" {
		return orderdom.Order{}, fmt.Errorf("order %s: missing shippingSnapshot", doc.Ref.ID)
	}
	if paymentMethod.CustomerID == "" ||
		paymentMethod.Brand == "" ||
		paymentMethod.Last4 == "" ||
		paymentMethod.ExpMonth < 1 ||
		paymentMethod.ExpMonth > 12 ||
		paymentMethod.ExpYear < 2000 ||
		paymentMethod.ExpYear > 9999 ||
		paymentMethod.CardholderName == "" {
		return orderdom.Order{}, fmt.Errorf("order %s: missing paymentMethodSnapshot", doc.Ref.ID)
	}
	if len(items) == 0 {
		return orderdom.Order{}, fmt.Errorf("order %s: missing items", doc.Ref.ID)
	}

	return orderdom.Order{
		ID:       doc.Ref.ID,
		UserID:   getStr("userId"),
		AvatarID: avatarID,
		CartID:   getStr("cartId"),

		ShippingSnapshot:      ship,
		PaymentMethodSnapshot: paymentMethod,

		Paid: paid,

		Items:     items,
		CreatedAt: createdAt,
	}, nil
}

// orderToDoc converts orderdom.Order into a Firestore-storable map (NEW schema only).
// NEW schema:
// - paid is on order root
// - transferred/transferredAt are on each item (items[].transferred / items[].transferredAt)
func orderToDoc(o orderdom.Order) map[string]any {
	ship := map[string]any{
		"zipCode": o.ShippingSnapshot.ZipCode,
		"state":   o.ShippingSnapshot.State,
		"city":    o.ShippingSnapshot.City,
		"street":  o.ShippingSnapshot.Street,
		"street2": o.ShippingSnapshot.Street2,
		"country": o.ShippingSnapshot.Country,
	}
	paymentMethod := map[string]any{
		"customerId":     o.PaymentMethodSnapshot.CustomerID,
		"brand":          o.PaymentMethodSnapshot.Brand,
		"last4":          o.PaymentMethodSnapshot.Last4,
		"expMonth":       o.PaymentMethodSnapshot.ExpMonth,
		"expYear":        o.PaymentMethodSnapshot.ExpYear,
		"cardholderName": o.PaymentMethodSnapshot.CardholderName,
		"isDefault":      o.PaymentMethodSnapshot.IsDefault,
	}

	items := make([]map[string]any, 0, len(o.Items))
	for _, it := range o.Items {
		im := map[string]any{
			"modelId":      it.ModelID,
			"inventoryId":  it.InventoryID,
			"listId":       it.ListID,
			"qty":          it.Qty,
			"price":        it.Price,
			"isCanceled":   it.IsCanceled,
			"isDispatched": it.IsDispatched,
			"transferred":  it.Transferred,
		}

		if it.Transferred && it.TransferredAt != nil && !it.TransferredAt.IsZero() {
			im["transferredAt"] = it.TransferredAt.UTC()
		}

		items = append(items, im)
	}

	m := map[string]any{
		"userId":   o.UserID,
		"avatarId": o.AvatarID,
		"cartId":   o.CartID,

		"shippingSnapshot":      ship,
		"paymentMethodSnapshot": paymentMethod,

		"paid": o.Paid,

		"items": items,
	}

	if !o.CreatedAt.IsZero() {
		m["createdAt"] = o.CreatedAt.UTC()
	}

	return m
}

// ========================
// Decode helpers
// ========================

func asMapAny(v any) map[string]any {
	if v == nil {
		return nil
	}
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return nil
}

func mapGetStr(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

func mapGetInt(m map[string]any, key string) int {
	if m == nil {
		return 0
	}
	v, ok := m[key]
	if !ok || v == nil {
		return 0
	}
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case int32:
		return int(t)
	case float64:
		return int(t)
	case float32:
		return int(t)
	default:
		return 0
	}
}

func mapGetBool(m map[string]any, key string) (bool, bool) {
	if m == nil {
		return false, false
	}
	v, ok := m[key]
	if !ok || v == nil {
		return false, false
	}
	switch t := v.(type) {
	case bool:
		return t, true
	case string:
		s := strings.ToLower(strings.TrimSpace(t))
		if s == "true" {
			return true, true
		}
		if s == "false" {
			return false, true
		}
		return false, false
	default:
		return false, false
	}
}

// mapGetTimeBestEffort reads Firestore timestamp wobble.
// - time.Time
// - *timestamppb.Timestamp
func mapGetTimeBestEffort(m map[string]any, key string) (time.Time, bool) {
	if m == nil {
		return time.Time{}, false
	}
	v, ok := m[key]
	if !ok || v == nil {
		return time.Time{}, false
	}

	switch t := v.(type) {
	case time.Time:
		if t.IsZero() {
			return time.Time{}, false
		}
		return t.UTC(), true
	case *timestamppb.Timestamp:
		if t == nil {
			return time.Time{}, false
		}
		tt := t.AsTime()
		if tt.IsZero() {
			return time.Time{}, false
		}
		return tt.UTC(), true
	default:
		return time.Time{}, false
	}
}

func decodeShippingSnapshot(v any) (orderdom.ShippingSnapshot, bool) {
	m := asMapAny(v)
	if m == nil {
		return orderdom.ShippingSnapshot{}, false
	}
	return orderdom.ShippingSnapshot{
		ZipCode: mapGetStr(m, "zipCode"),
		State:   mapGetStr(m, "state"),
		City:    mapGetStr(m, "city"),
		Street:  mapGetStr(m, "street"),
		Street2: mapGetStr(m, "street2"),
		Country: mapGetStr(m, "country"),
	}, true
}

func decodePaymentMethodSnapshot(v any) (orderdom.PaymentMethodSnapshot, bool) {
	m := asMapAny(v)
	if m == nil {
		return orderdom.PaymentMethodSnapshot{}, false
	}

	isDefault, _ := mapGetBool(m, "isDefault")

	return orderdom.PaymentMethodSnapshot{
		CustomerID:     mapGetStr(m, "customerId"),
		Brand:          mapGetStr(m, "brand"),
		Last4:          mapGetStr(m, "last4"),
		ExpMonth:       mapGetInt(m, "expMonth"),
		ExpYear:        mapGetInt(m, "expYear"),
		CardholderName: mapGetStr(m, "cardholderName"),
		IsDefault:      isDefault,
	}, true
}

func decodeItems(v any) ([]orderdom.OrderItemSnapshot, bool) {
	if v == nil {
		return nil, false
	}

	build := func(m map[string]any) orderdom.OrderItemSnapshot {
		if m == nil {
			return orderdom.OrderItemSnapshot{}
		}

		transferred, _ := mapGetBool(m, "transferred")
		isCanceled, _ := mapGetBool(m, "isCanceled")
		isDispatched, _ := mapGetBool(m, "isDispatched")

		var transferredAt *time.Time
		if t, ok := mapGetTimeBestEffort(m, "transferredAt"); ok {
			tt := t.UTC()
			transferredAt = &tt
		}

		return orderdom.OrderItemSnapshot{
			ModelID:       mapGetStr(m, "modelId"),
			InventoryID:   mapGetStr(m, "inventoryId"),
			ListID:        mapGetStr(m, "listId"),
			Qty:           mapGetInt(m, "qty"),
			Price:         mapGetInt(m, "price"),
			IsCanceled:    isCanceled,
			IsDispatched:  isDispatched,
			Transferred:   transferred,
			TransferredAt: transferredAt,
		}
	}

	switch raw := v.(type) {
	case []any:
		out := make([]orderdom.OrderItemSnapshot, 0, len(raw))
		for _, x := range raw {
			out = append(out, build(asMapAny(x)))
		}
		return out, true

	case []map[string]any:
		out := make([]orderdom.OrderItemSnapshot, 0, len(raw))
		for _, m := range raw {
			out = append(out, build(m))
		}
		return out, true

	default:
		return nil, false
	}
}

// ========================
// Query helpers
// ========================

func applyOrderSort(q firestore.Query, sort common.Sort) firestore.Query {
	dir := firestore.Desc
	if sort.Order == common.SortAsc {
		dir = firestore.Asc
	}

	// absolute source of truth: createdAt only
	if sort.Column != "" && sort.Column != "createdAt" {
		return q.OrderBy("createdAt", firestore.Desc).
			OrderBy(firestore.DocumentID, firestore.Desc)
	}

	return q.OrderBy("createdAt", dir).
		OrderBy(firestore.DocumentID, dir)
}

func matchOrderFilter(o orderdom.Order, f uc.OrderFilter) bool {
	if f.ID != "" && o.ID != f.ID {
		return false
	}

	if f.UserID != nil && *f.UserID != "" && o.UserID != *f.UserID {
		return false
	}

	if f.AvatarID != nil && *f.AvatarID != "" && o.AvatarID != *f.AvatarID {
		return false
	}

	if f.CartID != nil && *f.CartID != "" && o.CartID != *f.CartID {
		return false
	}

	if f.CreatedFrom != nil {
		if o.CreatedAt.IsZero() || o.CreatedAt.Before(f.CreatedFrom.UTC()) {
			return false
		}
	}

	if f.CreatedTo != nil {
		// upper bound exclusive
		if o.CreatedAt.IsZero() || !o.CreatedAt.Before(f.CreatedTo.UTC()) {
			return false
		}
	}

	return true
}
