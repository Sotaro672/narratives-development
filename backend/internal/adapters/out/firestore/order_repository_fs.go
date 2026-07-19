// backend/internal/adapters/out/firestore/order_repository_fs.go
package firestore

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	firestorepb "cloud.google.com/go/firestore/apiv1/firestorepb"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	fscommon "narratives/internal/adapters/out/firestore/common"
	common "narratives/internal/domain/common"
	orderdom "narratives/internal/domain/order"
)

var (
	ErrOrderRepositoryNotConfigured = errors.New(
		"order_repository_fs: not configured",
	)
	ErrInvalidOrderDocumentData = errors.New(
		"order_repository_fs: invalid order document data",
	)
)

// OrderRepositoryFS is the Firestore implementation of orderdom.Repository.
// Order writes and orderTransferItems projection writes share one transaction.
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

func (r *OrderRepositoryFS) orderTransferItemsCol() *firestore.CollectionRef {
	return r.Client.Collection("orderTransferItems")
}

func (r *OrderRepositoryFS) orderTransferItemDoc(
	orderID string,
	itemIndex int,
) *firestore.DocumentRef {
	return r.orderTransferItemsCol().Doc(
		orderID + "__" + strconv.Itoa(itemIndex),
	)
}

func (r *OrderRepositoryFS) GetByID(
	ctx context.Context,
	id string,
) (orderdom.Order, error) {
	if r == nil || r.Client == nil {
		return orderdom.Order{}, ErrOrderRepositoryNotConfigured
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

	return docToOrder(snap)
}

func (r *OrderRepositoryFS) ListByAvatarID(
	ctx context.Context,
	avatarID string,
	sort common.Sort,
	page common.Page,
) (common.PageResult[orderdom.Order], error) {
	if r == nil || r.Client == nil {
		return common.PageResult[orderdom.Order]{},
			ErrOrderRepositoryNotConfigured
	}

	pageNum, perPage, offset := fscommon.NormalizePage(
		page.Number,
		page.PerPage,
		50,
		200,
	)

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

	baseQuery := r.ordersCol().
		Where("avatarId", "==", avatarID)

	total, err := countOrderQuery(ctx, baseQuery)
	if err != nil {
		return common.PageResult[orderdom.Order]{}, err
	}

	query := applyOrderSort(baseQuery, sort).
		Offset(offset).
		Limit(perPage)

	iter := query.Documents(ctx)
	defer iter.Stop()

	items := make([]orderdom.Order, 0, perPage)

	for {
		snap, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return common.PageResult[orderdom.Order]{}, err
		}

		order, err := docToOrder(snap)
		if err != nil {
			return common.PageResult[orderdom.Order]{}, err
		}
		if order.AvatarID != avatarID {
			return common.PageResult[orderdom.Order]{},
				ErrInvalidOrderDocumentData
		}

		items = append(items, order)
	}

	return common.PageResult[orderdom.Order]{
		Items:      items,
		TotalCount: total,
		TotalPages: fscommon.ComputeTotalPages(total, perPage),
		Page:       pageNum,
		PerPage:    perPage,
	}, nil
}

func (r *OrderRepositoryFS) Create(
	ctx context.Context,
	o orderdom.Order,
) (orderdom.Order, error) {
	if r == nil || r.Client == nil {
		return orderdom.Order{}, ErrOrderRepositoryNotConfigured
	}
	if err := o.Validate(); err != nil {
		return orderdom.Order{}, err
	}

	orderRef := r.ordersCol().Doc(o.ID)
	orderData := orderToDoc(o)

	projectionData, err := orderTransferItemDocuments(o)
	if err != nil {
		return orderdom.Order{}, err
	}

	err = r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			tx *firestore.Transaction,
		) error {
			if err := tx.Create(orderRef, orderData); err != nil {
				return err
			}

			for itemIndex, data := range projectionData {
				ref := r.orderTransferItemDoc(o.ID, itemIndex)
				if err := tx.Create(ref, data); err != nil {
					return err
				}
			}

			return nil
		},
	)
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return orderdom.Order{}, orderdom.ErrConflict
		}
		return orderdom.Order{}, err
	}

	return o, nil
}

func (r *OrderRepositoryFS) Update(
	ctx context.Context,
	o orderdom.Order,
	_ *common.SaveOptions,
) (orderdom.Order, error) {
	if r == nil || r.Client == nil {
		return orderdom.Order{}, ErrOrderRepositoryNotConfigured
	}
	if o.ID == "" {
		return orderdom.Order{}, orderdom.ErrNotFound
	}
	if err := o.Validate(); err != nil {
		return orderdom.Order{}, err
	}

	orderRef := r.ordersCol().Doc(o.ID)
	orderData := orderToDoc(o)

	newProjectionData, err := orderTransferItemDocuments(o)
	if err != nil {
		return orderdom.Order{}, err
	}

	now := time.Now().UTC()

	err = r.Client.RunTransaction(
		ctx,
		func(
			ctx context.Context,
			tx *firestore.Transaction,
		) error {
			existingSnap, err := tx.Get(orderRef)
			if err != nil {
				if status.Code(err) == codes.NotFound {
					return orderdom.ErrNotFound
				}
				return err
			}

			existingOrder, err := docToOrder(existingSnap)
			if err != nil {
				return err
			}

			existingProjections := make(
				[]orderTransferItemProjection,
				len(existingOrder.Items),
			)

			for itemIndex := range existingOrder.Items {
				ref := r.orderTransferItemDoc(o.ID, itemIndex)

				snap, err := tx.Get(ref)
				if err != nil {
					if status.Code(err) == codes.NotFound {
						return ErrInvalidOrderTransferItemData
					}
					return err
				}

				projection, err :=
					orderTransferItemFromSnapshot(snap)
				if err != nil {
					return err
				}

				existingProjections[itemIndex] = projection
			}

			for itemIndex, projection := range existingProjections {
				locked :=
					projection.TransferLockExpiresAt != nil &&
						projection.TransferLockExpiresAt.After(now)

				if !locked {
					continue
				}

				if itemIndex >= len(o.Items) ||
					o.AvatarID != projection.AvatarID ||
					!o.Paid ||
					o.Items[itemIndex].Transferred ||
					!orderItemMatchesProjection(
						o.Items[itemIndex],
						projection,
					) {
					return ErrTransferItemLocked
				}

				newProjectionData[itemIndex]["transferLockedAt"] =
					projection.TransferLockedAt.UTC()
				newProjectionData[itemIndex]["transferLockExpiresAt"] =
					projection.TransferLockExpiresAt.UTC()
			}

			if err := tx.Set(orderRef, orderData); err != nil {
				return err
			}

			for itemIndex, data := range newProjectionData {
				ref := r.orderTransferItemDoc(o.ID, itemIndex)
				if err := tx.Set(ref, data); err != nil {
					return err
				}
			}

			for itemIndex := len(o.Items); itemIndex < len(existingOrder.Items); itemIndex++ {
				ref := r.orderTransferItemDoc(o.ID, itemIndex)
				if err := tx.Delete(ref); err != nil {
					return err
				}
			}

			return nil
		},
	)
	if err != nil {
		return orderdom.Order{}, err
	}

	return o, nil
}

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
	Type string `firestore:"type"`

	ModelID     string `firestore:"modelId,omitempty"`
	InventoryID string `firestore:"inventoryId,omitempty"`
	ListID      string `firestore:"listId,omitempty"`

	ResaleID string `firestore:"resaleId,omitempty"`

	ProductID          string `firestore:"productId,omitempty"`
	ProductBlueprintID string `firestore:"productBlueprintId,omitempty"`
	TokenBlueprintID   string `firestore:"tokenBlueprintId,omitempty"`
	BrandID            string `firestore:"brandId,omitempty"`

	Qty   int `firestore:"qty"`
	Price int `firestore:"price"`

	IsCanceled   bool `firestore:"isCanceled"`
	IsDispatched bool `firestore:"isDispatched"`

	Transferred   bool       `firestore:"transferred"`
	TransferredAt *time.Time `firestore:"transferredAt,omitempty"`
}

func docToOrder(
	snap *firestore.DocumentSnapshot,
) (orderdom.Order, error) {
	if snap == nil || snap.Ref == nil || !snap.Exists() {
		return orderdom.Order{}, orderdom.ErrNotFound
	}

	if err := validateOrderDocumentShape(snap.Data()); err != nil {
		return orderdom.Order{}, fmt.Errorf(
			"order %s: %w",
			snap.Ref.ID,
			err,
		)
	}

	var doc orderDoc
	if err := snap.DataTo(&doc); err != nil {
		return orderdom.Order{}, err
	}

	items := make(
		[]orderdom.OrderItemSnapshot,
		0,
		len(doc.Items),
	)

	for _, item := range doc.Items {
		var transferredAt *time.Time
		if item.TransferredAt != nil {
			value := item.TransferredAt.UTC()
			transferredAt = &value
		}

		items = append(
			items,
			orderdom.OrderItemSnapshot{
				Type: orderdom.OrderItemType(item.Type),

				ModelID:     item.ModelID,
				InventoryID: item.InventoryID,
				ListID:      item.ListID,

				ResaleID: item.ResaleID,

				ProductID:          item.ProductID,
				ProductBlueprintID: item.ProductBlueprintID,
				TokenBlueprintID:   item.TokenBlueprintID,
				BrandID:            item.BrandID,

				Qty:   item.Qty,
				Price: item.Price,

				IsCanceled:   item.IsCanceled,
				IsDispatched: item.IsDispatched,

				Transferred:   item.Transferred,
				TransferredAt: transferredAt,
			},
		)
	}

	order := orderdom.Order{
		ID:       snap.Ref.ID,
		UserID:   doc.UserID,
		AvatarID: doc.AvatarID,
		CartID:   doc.CartID,

		ShippingSnapshot: orderdom.ShippingSnapshot{
			ZipCode: doc.ShippingSnapshot.ZipCode,
			State:   doc.ShippingSnapshot.State,
			City:    doc.ShippingSnapshot.City,
			Street:  doc.ShippingSnapshot.Street,
			Street2: doc.ShippingSnapshot.Street2,
			Country: doc.ShippingSnapshot.Country,
		},

		PaymentMethodSnapshot: orderdom.PaymentMethodSnapshot{
			CustomerID:     doc.PaymentMethodSnapshot.CustomerID,
			Brand:          doc.PaymentMethodSnapshot.Brand,
			Last4:          doc.PaymentMethodSnapshot.Last4,
			ExpMonth:       doc.PaymentMethodSnapshot.ExpMonth,
			ExpYear:        doc.PaymentMethodSnapshot.ExpYear,
			CardholderName: doc.PaymentMethodSnapshot.CardholderName,
			IsDefault:      doc.PaymentMethodSnapshot.IsDefault,
		},

		Paid:      doc.Paid,
		Items:     items,
		CreatedAt: doc.CreatedAt.UTC(),
	}

	if err := order.Validate(); err != nil {
		return orderdom.Order{}, fmt.Errorf(
			"order %s: %w",
			snap.Ref.ID,
			err,
		)
	}

	return order, nil
}

func orderToDoc(o orderdom.Order) map[string]any {
	items := make([]map[string]any, 0, len(o.Items))
	for _, item := range o.Items {
		items = append(items, orderItemToDocMap(item))
	}

	return map[string]any{
		"userId":   o.UserID,
		"avatarId": o.AvatarID,
		"cartId":   o.CartID,

		"shippingSnapshot": map[string]any{
			"zipCode": o.ShippingSnapshot.ZipCode,
			"state":   o.ShippingSnapshot.State,
			"city":    o.ShippingSnapshot.City,
			"street":  o.ShippingSnapshot.Street,
			"street2": o.ShippingSnapshot.Street2,
			"country": o.ShippingSnapshot.Country,
		},

		"paymentMethodSnapshot": map[string]any{
			"customerId":     o.PaymentMethodSnapshot.CustomerID,
			"brand":          o.PaymentMethodSnapshot.Brand,
			"last4":          o.PaymentMethodSnapshot.Last4,
			"expMonth":       o.PaymentMethodSnapshot.ExpMonth,
			"expYear":        o.PaymentMethodSnapshot.ExpYear,
			"cardholderName": o.PaymentMethodSnapshot.CardholderName,
			"isDefault":      o.PaymentMethodSnapshot.IsDefault,
		},

		"paid":      o.Paid,
		"items":     items,
		"createdAt": o.CreatedAt.UTC(),
	}
}

func orderItemToDocMap(
	item orderdom.OrderItemSnapshot,
) map[string]any {
	doc := map[string]any{
		"type":         string(item.Type),
		"qty":          item.Qty,
		"price":        item.Price,
		"isCanceled":   item.IsCanceled,
		"isDispatched": item.IsDispatched,
		"transferred":  item.Transferred,
	}

	switch item.Type {
	case orderdom.OrderItemTypeList:
		doc["modelId"] = item.ModelID
		doc["inventoryId"] = item.InventoryID
		doc["listId"] = item.ListID
		doc["productBlueprintId"] =
			item.ProductBlueprintID
		doc["tokenBlueprintId"] =
			item.TokenBlueprintID

	case orderdom.OrderItemTypeResale:
		doc["resaleId"] = item.ResaleID
		doc["productId"] = item.ProductID
		doc["productBlueprintId"] =
			item.ProductBlueprintID
		doc["tokenBlueprintId"] =
			item.TokenBlueprintID
		doc["brandId"] = item.BrandID
	}

	if item.Transferred && item.TransferredAt != nil {
		doc["transferredAt"] =
			item.TransferredAt.UTC()
	}

	return doc
}

func orderTransferItemDocuments(
	o orderdom.Order,
) ([]map[string]any, error) {
	documents := make(
		[]map[string]any,
		0,
		len(o.Items),
	)

	for itemIndex, item := range o.Items {
		eligible := orderdom.EligibleTransferItem{
			OrderID:   o.ID,
			ItemType:  item.Type,
			ItemIndex: itemIndex,

			ModelID:     item.ModelID,
			InventoryID: item.InventoryID,
			ListID:      item.ListID,

			ResaleID: item.ResaleID,

			ProductID:          item.ProductID,
			ProductBlueprintID: item.ProductBlueprintID,
			TokenBlueprintID:   item.TokenBlueprintID,
			BrandID:            item.BrandID,
		}

		if err := eligible.Validate(); err != nil {
			return nil, fmt.Errorf(
				"order %s item %d: %w",
				o.ID,
				itemIndex,
				err,
			)
		}

		doc := map[string]any{
			"orderId":     o.ID,
			"avatarId":    o.AvatarID,
			"itemType":    string(item.Type),
			"itemIndex":   itemIndex,
			"paid":        o.Paid,
			"transferred": item.Transferred,
			"createdAt":   o.CreatedAt.UTC(),
		}

		switch item.Type {
		case orderdom.OrderItemTypeList:
			doc["modelId"] = item.ModelID
			doc["inventoryId"] = item.InventoryID
			doc["listId"] = item.ListID
			doc["productBlueprintId"] =
				item.ProductBlueprintID
			doc["tokenBlueprintId"] =
				item.TokenBlueprintID

		case orderdom.OrderItemTypeResale:
			doc["resaleId"] = item.ResaleID
			doc["productId"] = item.ProductID
			doc["productBlueprintId"] =
				item.ProductBlueprintID
			doc["tokenBlueprintId"] =
				item.TokenBlueprintID
			doc["brandId"] = item.BrandID
		}

		if item.Transferred &&
			item.TransferredAt != nil {
			doc["transferredAt"] =
				item.TransferredAt.UTC()
		}

		documents = append(documents, doc)
	}

	return documents, nil
}

func validateOrderDocumentShape(
	raw map[string]any,
) error {
	if raw == nil {
		return ErrInvalidOrderDocumentData
	}

	if _, ok := requiredOrderString(raw, "userId"); !ok {
		return ErrInvalidOrderDocumentData
	}
	if _, ok := requiredOrderString(raw, "avatarId"); !ok {
		return ErrInvalidOrderDocumentData
	}
	if _, ok := requiredOrderString(raw, "cartId"); !ok {
		return ErrInvalidOrderDocumentData
	}
	if _, ok := requiredOrderBool(raw, "paid"); !ok {
		return ErrInvalidOrderDocumentData
	}

	createdAt, ok := requiredOrderTime(raw, "createdAt")
	if !ok || createdAt.IsZero() {
		return ErrInvalidOrderDocumentData
	}

	rawItems, ok := raw["items"].([]any)
	if !ok || len(rawItems) == 0 {
		return ErrInvalidOrderDocumentData
	}

	for _, rawItem := range rawItems {
		item, ok := rawItem.(map[string]any)
		if !ok || item == nil {
			return ErrInvalidOrderDocumentData
		}

		if err := validateOrderItemDocumentShape(item); err != nil {
			return err
		}
	}

	return nil
}

func validateOrderItemDocumentShape(
	raw map[string]any,
) error {
	itemType, ok := requiredOrderString(raw, "type")
	if !ok {
		return ErrInvalidOrderDocumentData
	}

	if _, ok := requiredOrderInt(raw, "qty"); !ok {
		return ErrInvalidOrderDocumentData
	}
	if _, ok := requiredOrderInt(raw, "price"); !ok {
		return ErrInvalidOrderDocumentData
	}
	if _, ok := requiredOrderBool(raw, "isCanceled"); !ok {
		return ErrInvalidOrderDocumentData
	}
	if _, ok := requiredOrderBool(raw, "isDispatched"); !ok {
		return ErrInvalidOrderDocumentData
	}

	transferred, ok :=
		requiredOrderBool(raw, "transferred")
	if !ok {
		return ErrInvalidOrderDocumentData
	}

	_, transferredAtExists, err :=
		optionalOrderTime(raw, "transferredAt")
	if err != nil ||
		transferred != transferredAtExists {
		return ErrInvalidOrderDocumentData
	}

	switch orderdom.OrderItemType(itemType) {
	case orderdom.OrderItemTypeList:
		for _, field := range []string{
			"modelId",
			"inventoryId",
			"listId",
			"productBlueprintId",
			"tokenBlueprintId",
		} {
			if _, ok := requiredOrderString(raw, field); !ok {
				return ErrInvalidOrderDocumentData
			}
		}

	case orderdom.OrderItemTypeResale:
		for _, field := range []string{
			"resaleId",
			"productId",
			"productBlueprintId",
			"tokenBlueprintId",
			"brandId",
		} {
			if _, ok := requiredOrderString(raw, field); !ok {
				return ErrInvalidOrderDocumentData
			}
		}

	default:
		return ErrInvalidOrderDocumentData
	}

	return nil
}

func requiredOrderString(
	raw map[string]any,
	field string,
) (string, bool) {
	value, exists := raw[field]
	if !exists || value == nil {
		return "", false
	}

	result, ok := value.(string)
	return result, ok && result != ""
}

func requiredOrderBool(
	raw map[string]any,
	field string,
) (bool, bool) {
	value, exists := raw[field]
	if !exists || value == nil {
		return false, false
	}

	result, ok := value.(bool)
	return result, ok
}

func requiredOrderInt(
	raw map[string]any,
	field string,
) (int, bool) {
	value, exists := raw[field]
	if !exists || value == nil {
		return 0, false
	}

	switch result := value.(type) {
	case int:
		return result, true

	case int64:
		return int(result), true

	default:
		return 0, false
	}
}

func requiredOrderTime(
	raw map[string]any,
	field string,
) (time.Time, bool) {
	value, exists := raw[field]
	if !exists || value == nil {
		return time.Time{}, false
	}

	result, ok := value.(time.Time)
	return result, ok && !result.IsZero()
}

func optionalOrderTime(
	raw map[string]any,
	field string,
) (time.Time, bool, error) {
	value, exists := raw[field]
	if !exists || value == nil {
		return time.Time{}, false, nil
	}

	result, ok := value.(time.Time)
	if !ok || result.IsZero() {
		return time.Time{},
			false,
			ErrInvalidOrderDocumentData
	}

	return result.UTC(), true, nil
}

const orderCountAlias = "total"

func countOrderQuery(
	ctx context.Context,
	query firestore.Query,
) (int, error) {
	result, err := query.
		NewAggregationQuery().
		WithCount(orderCountAlias).
		Get(ctx)
	if err != nil {
		return 0, err
	}

	rawCount, ok := result[orderCountAlias]
	if !ok {
		return 0, errors.New(
			"firestore: order count result is missing",
		)
	}

	countValue, ok := rawCount.(*firestorepb.Value)
	if !ok || countValue == nil {
		return 0, fmt.Errorf(
			"firestore: invalid order count result type %T",
			rawCount,
		)
	}

	count64 := countValue.GetIntegerValue()
	if count64 < 0 {
		return 0, fmt.Errorf(
			"firestore: invalid negative order count: %d",
			count64,
		)
	}

	total := int(count64)
	if int64(total) != count64 {
		return 0, fmt.Errorf(
			"firestore: order count overflows int: %d",
			count64,
		)
	}

	return total, nil
}

func applyOrderSort(
	query firestore.Query,
	sort common.Sort,
) firestore.Query {
	direction := firestore.Desc
	if sort.Order == common.SortAsc {
		direction = firestore.Asc
	}

	if sort.Column != "" &&
		sort.Column != orderdom.SortByCreatedAt {
		direction = firestore.Desc
	}

	return query.
		OrderBy(orderdom.SortByCreatedAt, direction).
		OrderBy(firestore.DocumentID, direction)
}
