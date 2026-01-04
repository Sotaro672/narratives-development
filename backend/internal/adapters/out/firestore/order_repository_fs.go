// backend/internal/adapters/out/firestore/order_repository_fs.go
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

	// upsert (new doc)
	if id == "" {
		if o.CreatedAt.IsZero() {
			o.CreatedAt = now
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

	return nil
}

// ========================
// Helpers (NEW schema; entity.go as source of truth)
// ========================

func asMapAny(v any) map[string]any {
	if v == nil {
		return nil
	}
	if m, ok := v.(map[string]any); ok {
		return m
	}
	if m, ok := v.(map[string]interface{}); ok {
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
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(fmt.Sprint(v))
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
	case float64:
		return int(t)
	default:
		// Firestore decode の揺れがあっても落とさず 0 に寄せる（domain が弾く）
		return 0
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

func decodeBillingSnapshot(v any) (orderdom.BillingSnapshot, bool) {
	m := asMapAny(v)
	if m == nil {
		return orderdom.BillingSnapshot{}, false
	}
	return orderdom.BillingSnapshot{
		Last4:          mapGetStr(m, "last4"),
		CardHolderName: mapGetStr(m, "cardHolderName"),
	}, true
}

func decodeItems(v any) ([]orderdom.OrderItemSnapshot, bool) {
	if v == nil {
		return nil, false
	}

	switch raw := v.(type) {
	case []any:
		out := make([]orderdom.OrderItemSnapshot, 0, len(raw))
		for _, x := range raw {
			m := asMapAny(x)
			if m == nil {
				out = append(out, orderdom.OrderItemSnapshot{})
				continue
			}
			out = append(out, orderdom.OrderItemSnapshot{
				ModelID:     strings.TrimSpace(mapGetStr(m, "modelId")),
				InventoryID: strings.TrimSpace(mapGetStr(m, "inventoryId")),
				Qty:         mapGetInt(m, "qty"),
				Price:       mapGetInt(m, "price"),
			})
		}
		return out, true
	case []map[string]any:
		out := make([]orderdom.OrderItemSnapshot, 0, len(raw))
		for _, m := range raw {
			out = append(out, orderdom.OrderItemSnapshot{
				ModelID:     strings.TrimSpace(mapGetStr(m, "modelId")),
				InventoryID: strings.TrimSpace(mapGetStr(m, "inventoryId")),
				Qty:         mapGetInt(m, "qty"),
				Price:       mapGetInt(m, "price"),
			})
		}
		return out, true
	default:
		return nil, false
	}
}

// docToOrder converts a Firestore document snapshot to orderdom.Order (NEW schema only).
func docToOrder(doc *firestore.DocumentSnapshot) (orderdom.Order, error) {
	data := doc.Data()
	if data == nil {
		return orderdom.Order{}, fmt.Errorf("empty order document: %s", doc.Ref.ID)
	}

	getStr := func(key string) string {
		if v, ok := data[key].(string); ok {
			return strings.TrimSpace(v)
		}
		if v, ok := data[key]; ok && v != nil {
			return strings.TrimSpace(fmt.Sprint(v))
		}
		return ""
	}

	getTime := func(key string) time.Time {
		if v, ok := data[key].(time.Time); ok && !v.IsZero() {
			return v.UTC()
		}
		// 念のため protobuf Timestamp も受ける（環境差の吸収）
		if ts, ok := data[key].(*timestamppb.Timestamp); ok && ts != nil {
			t := ts.AsTime()
			if !t.IsZero() {
				return t.UTC()
			}
		}
		return time.Time{}
	}

	// snapshots
	var ship orderdom.ShippingSnapshot
	if v, ok := data["shippingSnapshot"]; ok {
		if s, ok2 := decodeShippingSnapshot(v); ok2 {
			ship = s
		}
	}
	var bill orderdom.BillingSnapshot
	if v, ok := data["billingSnapshot"]; ok {
		if b, ok2 := decodeBillingSnapshot(v); ok2 {
			bill = b
		}
	}

	items, ok := decodeItems(data["items"])
	if !ok {
		items = nil
	}

	createdAt := getTime("createdAt")
	if createdAt.IsZero() && !doc.CreateTime.IsZero() {
		createdAt = doc.CreateTime.UTC()
	}

	// Strict minimums (entity validate 前に、アダプタとして最低限守る)
	avatarID := getStr("avatarId")
	if avatarID == "" {
		// 旧データ救済（念のため）
		avatarID = getStr("avatarID")
	}

	if strings.TrimSpace(avatarID) == "" {
		return orderdom.Order{}, fmt.Errorf("order %s: missing avatarId", doc.Ref.ID)
	}
	if strings.TrimSpace(ship.State) == "" ||
		strings.TrimSpace(ship.City) == "" ||
		strings.TrimSpace(ship.Street) == "" ||
		strings.TrimSpace(ship.Country) == "" {
		return orderdom.Order{}, fmt.Errorf("order %s: missing shippingSnapshot", doc.Ref.ID)
	}
	if strings.TrimSpace(bill.Last4) == "" {
		return orderdom.Order{}, fmt.Errorf("order %s: missing billingSnapshot.last4", doc.Ref.ID)
	}
	if len(items) == 0 {
		return orderdom.Order{}, fmt.Errorf("order %s: missing items", doc.Ref.ID)
	}

	return orderdom.Order{
		ID:       doc.Ref.ID,
		UserID:   getStr("userId"),
		AvatarID: avatarID,
		CartID:   getStr("cartId"),

		ShippingSnapshot: ship,
		BillingSnapshot:  bill,

		Items:     items,
		CreatedAt: createdAt,
	}, nil
}

// orderToDoc converts orderdom.Order into a Firestore-storable map (NEW schema only).
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

	items := make([]map[string]any, 0, len(o.Items))
	for _, it := range o.Items {
		items = append(items, map[string]any{
			"modelId":     strings.TrimSpace(it.ModelID),
			"inventoryId": strings.TrimSpace(it.InventoryID),
			"qty":         it.Qty,
			"price":       it.Price,
		})
	}

	m := map[string]any{
		"userId":   strings.TrimSpace(o.UserID),
		"avatarId": strings.TrimSpace(o.AvatarID),
		"cartId":   strings.TrimSpace(o.CartID),

		"shippingSnapshot": ship,
		"billingSnapshot":  bill,

		"items": items,
	}

	if !o.CreatedAt.IsZero() {
		m["createdAt"] = o.CreatedAt.UTC()
	}

	return m
}

// --- filter/sort helpers ---

// matchOrderFilter is reflection-based so adapter compiles even if uc.OrderFilter shape changes.
// It tries to apply: ID, UserID, AvatarID, CartID, CreatedFrom/CreatedTo.
func matchOrderFilter(o orderdom.Order, f uc.OrderFilter) bool {
	return matchOrderFilterAny(o, any(f))
}

func matchOrderFilterAny(o orderdom.Order, fv any) bool {
	// ID
	if id, ok := getFilterString(fv, "ID"); ok {
		if strings.TrimSpace(id) != "" && strings.TrimSpace(o.ID) != strings.TrimSpace(id) {
			return false
		}
	}
	// UserID
	if uid, ok := getFilterString(fv, "UserID"); ok {
		if strings.TrimSpace(uid) != "" && strings.TrimSpace(o.UserID) != strings.TrimSpace(uid) {
			return false
		}
	}

	// AvatarID (filter 側の命名揺れも吸収)
	if aid, ok := getFilterString(fv, "AvatarID"); ok {
		if strings.TrimSpace(aid) != "" && strings.TrimSpace(o.AvatarID) != strings.TrimSpace(aid) {
			return false
		}
	} else if aid, ok := getFilterString(fv, "AvatarId"); ok {
		if strings.TrimSpace(aid) != "" && strings.TrimSpace(o.AvatarID) != strings.TrimSpace(aid) {
			return false
		}
	}

	// CartID
	if cid, ok := getFilterString(fv, "CartID"); ok {
		if strings.TrimSpace(cid) != "" && strings.TrimSpace(o.CartID) != strings.TrimSpace(cid) {
			return false
		}
	}

	// CreatedFrom / CreatedTo
	if from, ok := getFilterTimePtr(fv, "CreatedFrom"); ok && from != nil {
		if o.CreatedAt.IsZero() || o.CreatedAt.Before(from.UTC()) {
			return false
		}
	}
	if to, ok := getFilterTimePtr(fv, "CreatedTo"); ok && to != nil {
		// "to" は Upper bound exclusive に寄せる（以前の実装踏襲）
		if o.CreatedAt.IsZero() || !o.CreatedAt.Before(to.UTC()) {
			return false
		}
	}

	return true
}

func getFilterString(v any, field string) (string, bool) {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return "", false
	}
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return "", false
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return "", false
	}
	f := rv.FieldByName(field)
	if !f.IsValid() {
		// try lowerCamel (e.g., userId / avatarId)
		f = rv.FieldByName(lowerFirst(field))
		if !f.IsValid() {
			return "", false
		}
	}
	// string
	if f.Kind() == reflect.String {
		return f.String(), true
	}
	// *string
	if f.Kind() == reflect.Pointer && f.Type().Elem().Kind() == reflect.String {
		if f.IsNil() {
			return "", true
		}
		return f.Elem().String(), true
	}
	return "", false
}

func getFilterTimePtr(v any, field string) (*time.Time, bool) {
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

	// *time.Time
	if f.Kind() == reflect.Pointer {
		if f.IsNil() {
			return nil, true
		}
		if t, ok := f.Interface().(*time.Time); ok {
			return t, true
		}
	}
	// time.Time
	if f.CanInterface() {
		if t, ok := f.Interface().(time.Time); ok {
			return &t, true
		}
	}
	return nil, false
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

func applyOrderSort(q firestore.Query, sort common.Sort) firestore.Query {
	col := strings.ToLower(strings.TrimSpace(string(sort.Column)))

	// entity.go に合わせて createdAt のみ許可
	field := ""
	switch col {
	case "createdat", "created_at", "created":
		field = "createdAt"
	default:
		// default: newest first
		return q.OrderBy("createdAt", firestore.Desc).
			OrderBy(firestore.DocumentID, firestore.Desc)
	}

	dir := firestore.Desc
	if strings.EqualFold(string(sort.Order), "asc") {
		dir = firestore.Asc
	}

	return q.OrderBy(field, dir).OrderBy(firestore.DocumentID, dir)
}
