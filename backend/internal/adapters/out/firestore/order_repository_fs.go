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

	fscommon "narratives/internal/adapters/out/firestore/common"
	common "narratives/internal/domain/common"
	orderdom "narratives/internal/domain/order"
)

// Firestore implementation of usecase.OrderRepo
type OrderRepositoryFS struct {
	Client *firestore.Client
}

var _ orderdom.Repository = (*OrderRepositoryFS)(nil)

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

func (r *OrderRepositoryFS) ListByAvatarID(
	ctx context.Context,
	avatarID string,
	sort common.Sort,
	page common.Page,
) (common.PageResult[orderdom.Order], error) {
	if r.Client == nil {
		return common.PageResult[orderdom.Order]{}, errors.New("firestore client is nil")
	}

	pageNum, perPage, offset := fscommon.NormalizePage(page.Number, page.PerPage, 50, 200)

	avatarID = strings.TrimSpace(avatarID)
	if avatarID == "" {
		return common.PageResult[orderdom.Order]{
			Items:      []orderdom.Order{},
			TotalCount: 0,
			TotalPages: 0,
			Page:       pageNum,
			PerPage:    perPage,
		}, nil
	}

	q := r.ordersCol().
		Where("avatarId", "==", avatarID)

	q = applyOrderSort(q, sort)
	q = q.Offset(offset).Limit(perPage)

	it := q.Documents(ctx)
	defer it.Stop()

	items := make([]orderdom.Order, 0, perPage)
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

		// Firestore query already filters avatarId, but keep this as a defensive check.
		if o.AvatarID == avatarID {
			items = append(items, o)
		}
	}

	total := len(items)

	return common.PageResult[orderdom.Order]{
		Items:      items,
		TotalCount: total,
		TotalPages: fscommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

// ListTransferredByAvatarIDModelIDAndTransferredAt returns orders that contain
// transferred items matching avatarId, modelId, and transferredAt.
//
// Query condition:
// - order.avatarId == avatarID
// - order.paid == true
//
// In-memory item filter:
// - item.modelId == modelID
// - item.transferred == true
// - item.transferredAt == transferredAt
//
// Firestore cannot reliably query nested array map fields with this full condition,
// so this repository queries by avatarId/paid first and filters order items here.
func (r *OrderRepositoryFS) ListTransferredByAvatarIDModelIDAndTransferredAt(
	ctx context.Context,
	avatarID string,
	modelID string,
	transferredAt time.Time,
	sort common.Sort,
	page common.Page,
) (common.PageResult[orderdom.Order], error) {
	if r.Client == nil {
		return common.PageResult[orderdom.Order]{}, errors.New("firestore client is nil")
	}

	pageNum, perPage, offset := fscommon.NormalizePage(page.Number, page.PerPage, 50, 200)

	avatarID = strings.TrimSpace(avatarID)
	modelID = strings.TrimSpace(modelID)
	transferredAt = transferredAt.UTC()

	if avatarID == "" || modelID == "" || transferredAt.IsZero() {
		return common.PageResult[orderdom.Order]{
			Items:      []orderdom.Order{},
			TotalCount: 0,
			TotalPages: 0,
			Page:       pageNum,
			PerPage:    perPage,
		}, nil
	}

	q := r.ordersCol().
		Where("avatarId", "==", avatarID).
		Where("paid", "==", true)

	q = applyOrderSort(q, sort)

	it := q.Documents(ctx)
	defer it.Stop()

	matched := make([]orderdom.Order, 0)

	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return common.PageResult[orderdom.Order]{}, err
		}
		if doc == nil || doc.Ref == nil {
			continue
		}

		o, err := docToOrder(doc)
		if err != nil {
			return common.PageResult[orderdom.Order]{}, err
		}

		// Firestore query already filters avatarId and paid,
		// but keep these as defensive checks.
		if o.AvatarID != avatarID {
			continue
		}
		if !o.Paid {
			continue
		}

		filteredItems := filterTransferredOrderItemsByModelIDAndTransferredAt(
			o.Items,
			modelID,
			transferredAt,
		)
		if len(filteredItems) == 0 {
			continue
		}

		o.Items = filteredItems
		matched = append(matched, o)
	}

	total := len(matched)
	paged := paginateOrders(matched, offset, perPage)

	return common.PageResult[orderdom.Order]{
		Items:      paged,
		TotalCount: total,
		TotalPages: fscommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

// ListEligibleTransferItemsByAvatarID returns paid and untransferred order items for transfer verification.
//
// Query condition:
// - order.avatarId == avatarID
// - order.paid == true
//
// In-memory item filter:
// - item.transferred == false
// - item.modelId is not empty
// - item.inventoryId is not empty
func (r *OrderRepositoryFS) ListEligibleTransferItemsByAvatarID(
	ctx context.Context,
	avatarID string,
) ([]orderdom.EligibleTransferItem, error) {
	if r.Client == nil {
		return nil, errors.New("firestore client is nil")
	}

	avatarID = strings.TrimSpace(avatarID)
	if avatarID == "" {
		return []orderdom.EligibleTransferItem{}, nil
	}

	it := r.ordersCol().
		Where("avatarId", "==", avatarID).
		Where("paid", "==", true).
		Documents(ctx)
	defer it.Stop()

	out := make([]orderdom.EligibleTransferItem, 0)

	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		if doc == nil || doc.Ref == nil {
			continue
		}

		o, err := docToOrder(doc)
		if err != nil {
			return nil, err
		}

		// Firestore query already filters avatarId and paid,
		// but keep these as defensive checks.
		if o.AvatarID != avatarID {
			continue
		}
		if !o.Paid {
			continue
		}

		for _, item := range o.Items {
			if item.Transferred {
				continue
			}
			if item.ModelID == "" {
				continue
			}
			if item.InventoryID == "" {
				continue
			}

			out = append(out, orderdom.EligibleTransferItem{
				OrderID:     o.ID,
				ModelID:     item.ModelID,
				InventoryID: item.InventoryID,
			})
		}
	}

	return out, nil
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

func (r *OrderRepositoryFS) Update(ctx context.Context, o orderdom.Order, _ *common.SaveOptions) (orderdom.Order, error) {
	if r.Client == nil {
		return orderdom.Order{}, errors.New("firestore client is nil")
	}

	id := o.ID
	if id == "" {
		return orderdom.Order{}, orderdom.ErrNotFound
	}

	docRef := r.ordersCol().Doc(id)

	snap, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return orderdom.Order{}, orderdom.ErrNotFound
		}
		return orderdom.Order{}, err
	}

	o.ID = id

	// preserve CreatedAt if missing
	if o.CreatedAt.IsZero() {
		existing, err := docToOrder(snap)
		if err != nil {
			return orderdom.Order{}, err
		}
		if !existing.CreatedAt.IsZero() {
			o.CreatedAt = existing.CreatedAt
		}
	}

	if o.CreatedAt.IsZero() {
		o.CreatedAt = time.Now().UTC()
	}

	data := orderToDoc(o)

	_, err = docRef.Set(ctx, data, firestore.MergeAll)
	if err != nil {
		return orderdom.Order{}, err
	}

	updatedSnap, err := docRef.Get(ctx)
	if err != nil {
		return orderdom.Order{}, err
	}
	return docToOrder(updatedSnap)
}

// ========================
// Firestore DTO
// ========================

type orderDoc struct {
	UserID   string    `firestore:"userId"`
	AvatarID string    `firestore:"avatarId"`
	CartID   string    `firestore:"cartId"`
	Paid     bool      `firestore:"paid"`
	Items    []itemDoc `firestore:"items"`

	ShippingSnapshot      shippingSnapshotDoc      `firestore:"shippingSnapshot"`
	PaymentMethodSnapshot paymentMethodSnapshotDoc `firestore:"paymentMethodSnapshot"`

	CreatedAt time.Time `firestore:"createdAt"`
}

type shippingSnapshotDoc struct {
	ZipCode string `firestore:"zipCode"`
	State   string `firestore:"state"`
	City    string `firestore:"city"`
	Street  string `firestore:"street"`
	Street2 string `firestore:"street2"`
	Country string `firestore:"country"`
}

type paymentMethodSnapshotDoc struct {
	CustomerID     string `firestore:"customerId"`
	Brand          string `firestore:"brand"`
	Last4          string `firestore:"last4"`
	ExpMonth       int    `firestore:"expMonth"`
	ExpYear        int    `firestore:"expYear"`
	CardholderName string `firestore:"cardholderName"`
	IsDefault      bool   `firestore:"isDefault"`
}

type itemDoc struct {
	ModelID       string     `firestore:"modelId"`
	InventoryID   string     `firestore:"inventoryId"`
	ListID        string     `firestore:"listId"`
	Qty           int        `firestore:"qty"`
	Price         int        `firestore:"price"`
	IsCanceled    bool       `firestore:"isCanceled"`
	IsDispatched  bool       `firestore:"isDispatched"`
	Transferred   bool       `firestore:"transferred"`
	TransferredAt *time.Time `firestore:"transferredAt,omitempty"`
}

// ========================
// Mapper
// ========================

// docToOrder converts a Firestore document snapshot to orderdom.Order (NEW schema only).
// NEW schema:
// - paid is on order root
// - transferred/transferredAt are on each item (items[].transferred / items[].transferredAt)
func docToOrder(doc *firestore.DocumentSnapshot) (orderdom.Order, error) {
	if doc == nil {
		return orderdom.Order{}, fmt.Errorf("nil order document")
	}

	var d orderDoc
	if err := doc.DataTo(&d); err != nil {
		return orderdom.Order{}, err
	}

	items := make([]orderdom.OrderItemSnapshot, 0, len(d.Items))
	for _, it := range d.Items {
		var transferredAt *time.Time
		if it.TransferredAt != nil && !it.TransferredAt.IsZero() {
			t := it.TransferredAt.UTC()
			transferredAt = &t
		}

		items = append(items, orderdom.OrderItemSnapshot{
			ModelID:       it.ModelID,
			InventoryID:   it.InventoryID,
			ListID:        it.ListID,
			Qty:           it.Qty,
			Price:         it.Price,
			IsCanceled:    it.IsCanceled,
			IsDispatched:  it.IsDispatched,
			Transferred:   it.Transferred,
			TransferredAt: transferredAt,
		})
	}

	o := orderdom.Order{
		ID:       doc.Ref.ID,
		UserID:   d.UserID,
		AvatarID: d.AvatarID,
		CartID:   d.CartID,

		ShippingSnapshot: orderdom.ShippingSnapshot{
			ZipCode: d.ShippingSnapshot.ZipCode,
			State:   d.ShippingSnapshot.State,
			City:    d.ShippingSnapshot.City,
			Street:  d.ShippingSnapshot.Street,
			Street2: d.ShippingSnapshot.Street2,
			Country: d.ShippingSnapshot.Country,
		},
		PaymentMethodSnapshot: orderdom.PaymentMethodSnapshot{
			CustomerID:     d.PaymentMethodSnapshot.CustomerID,
			Brand:          d.PaymentMethodSnapshot.Brand,
			Last4:          d.PaymentMethodSnapshot.Last4,
			ExpMonth:       d.PaymentMethodSnapshot.ExpMonth,
			ExpYear:        d.PaymentMethodSnapshot.ExpYear,
			CardholderName: d.PaymentMethodSnapshot.CardholderName,
			IsDefault:      d.PaymentMethodSnapshot.IsDefault,
		},

		Paid:      d.Paid,
		Items:     items,
		CreatedAt: d.CreatedAt.UTC(),
	}

	if err := validateDecodedOrder(o); err != nil {
		return orderdom.Order{}, fmt.Errorf("order %s: %w", doc.Ref.ID, err)
	}

	return o, nil
}

func validateDecodedOrder(o orderdom.Order) error {
	if o.AvatarID == "" {
		return fmt.Errorf("missing avatarId")
	}
	if o.ShippingSnapshot.State == "" ||
		o.ShippingSnapshot.City == "" ||
		o.ShippingSnapshot.Street == "" ||
		o.ShippingSnapshot.Country == "" {
		return fmt.Errorf("missing shippingSnapshot")
	}
	if o.PaymentMethodSnapshot.CustomerID == "" ||
		o.PaymentMethodSnapshot.Brand == "" ||
		o.PaymentMethodSnapshot.Last4 == "" ||
		o.PaymentMethodSnapshot.ExpMonth < 1 ||
		o.PaymentMethodSnapshot.ExpMonth > 12 ||
		o.PaymentMethodSnapshot.ExpYear < 2000 ||
		o.PaymentMethodSnapshot.ExpYear > 9999 ||
		o.PaymentMethodSnapshot.CardholderName == "" {
		return fmt.Errorf("missing paymentMethodSnapshot")
	}
	if len(o.Items) == 0 {
		return fmt.Errorf("missing items")
	}
	return nil
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

func filterTransferredOrderItemsByModelIDAndTransferredAt(
	items []orderdom.OrderItemSnapshot,
	modelID string,
	transferredAt time.Time,
) []orderdom.OrderItemSnapshot {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" || transferredAt.IsZero() || len(items) == 0 {
		return []orderdom.OrderItemSnapshot{}
	}

	expected := transferredAt.UTC()

	out := make([]orderdom.OrderItemSnapshot, 0, len(items))
	for _, item := range items {
		if item.ModelID != modelID {
			continue
		}
		if !item.Transferred {
			continue
		}
		if item.TransferredAt == nil || item.TransferredAt.IsZero() {
			continue
		}

		actual := item.TransferredAt.UTC()
		if !actual.Equal(expected) {
			continue
		}

		out = append(out, item)
	}

	return out
}

func paginateOrders(items []orderdom.Order, offset int, perPage int) []orderdom.Order {
	if len(items) == 0 {
		return []orderdom.Order{}
	}

	if perPage <= 0 {
		perPage = len(items)
	}

	if offset < 0 {
		offset = 0
	}

	if offset >= len(items) {
		return []orderdom.Order{}
	}

	end := offset + perPage
	if end > len(items) {
		end = len(items)
	}

	return items[offset:end]
}
